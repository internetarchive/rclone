package vault

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rclone/rclone/backend/vault/api"
	"github.com/rclone/rclone/backend/vault/iotemp"
	"github.com/rclone/rclone/backend/vault/oapi"
	"github.com/rclone/rclone/backend/vault/retry"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/config/configstruct"
	"github.com/rclone/rclone/fs/hash"
	"github.com/rclone/rclone/lib/atexit"
)

const (
	// Note: the biggest increase in upload throughput so far came from
	// increasing the chunk size to 16M.
	//
	//  1M/1/1:  5M/s
	// 16M/1/1: 15M/s
	// 16M/2/2: 20M/s
	//
	// Target two-core QA machine was occassionally maxed out, not sure if
	// that's imposing a limit.
	//
	// ----8<----
	//
	// As of 05/2023, we reduced the default chunk size to 1M, temporarily
	// after encountering some issues.
	//
	// ----8<----
	//
	// @martin I'm just remembering to ask about this, but could you fill me in
	// on the rationale for decreasing the chunk sizes that rclone sends to
	// vault?
	//
	// yes, happily; in short: I was getting a HTTP 404 from chunk upload and I
	// believe it came from this line:
	// vault-site:19e168e41489e2278bb59c659489344857a04f4e/vault/deposit_api.py#L193-199
	// (not too many 404's possible otherwise) -- now the deposit is registered
	// successfully, so the deposit's state must have changed (to something
	// other than "REGISTERED") -- and that (only) happens, when a deposit is
	// "completed" -- which happens e.g. here:
	// vault-site:19e168e41489e2278bb59c659489344857a04f4e/vault/asynchronous/temporal/deposit/workflow.py#L62-66
	// -- so my wild guess ATM is that we encounter some break here:
	// vault-site:19e168e41489e2278bb59c659489344857a04f4e/vault/asynchronous/temporal/deposit/workflow.py#L24-60
	// -- and that causes the deposit to be "completed" early, which then causes
	// a 404 when trying to upload a chunk somehow, lowering the chunksize made
	// this problem go away, but there's probably a better way to handle this --
	// would be glad to have a short in person debug session (where we can try
	// to replicate the issue in prod together, or the like)
	defaultUploadChunkSize = 1 << 20 // 1M
)

func init() {
	fs.Register(&fs.RegInfo{
		Name:        "vault",
		Description: "Internet Archive Vault Digital Preservation System",
		NewFs:       NewFs,
		Options: []fs.Option{
			{
				Name:    "username",
				Help:    "Vault username",
				Default: "",
			},
			{
				Name:    "password",
				Help:    "Vault password",
				Default: "",
			},
			{
				Name:    "endpoint",
				Help:    "Vault API endpoint URL",
				Default: "http://127.0.0.1:8000/api",
			},
			{
				Name:     "chunk_size",
				Help:     "Upload chunk size in bytes (limited)",
				Default:  defaultUploadChunkSize,
				Advanced: true,
			},
		},
	})
}

const flowIdentifierPrefix = "rclone-vault-flow"

var (
	ErrCannotCopyToRoot         = errors.New("copying files to root is not supported in vault")
	ErrInvalidPath              = errors.New("invalid path")
	ErrVersionMismatch          = errors.New("api version mismatch")
	ErrMissingDepositIdentifier = errors.New("missing deposit identifier")
	ErrInvalidEndpoint          = errors.New("invalid endpoint")

	VersionMismatchMessage = `

 ██████  ██   ██     ███    ██  ██████
██    ██ ██   ██     ████   ██ ██    ██
██    ██ ███████     ██ ██  ██ ██    ██
██    ██ ██   ██     ██  ██ ██ ██    ██
 ██████  ██   ██     ██   ████  ██████

We detected a version mismatch between the Vault API (%v) and the version
supported by the currently installed rclone (%v). We kindly ask you to upgrade
to the latest rclone release to fix this problem.

You can download the latest release here: https://github.com/internetarchive/rclone/releases

Thank you for your understanding.

For more information about Vault visit https://webservices.archive.org/pages/vault

`

	UploadChunkTimeout     = 24 * time.Hour         // generous limit for single chunk upload time (should never be hit)
	UploadChunkBackoffBase = 100 * time.Millisecond // backoff base timeout
	UploadChunkBackoffCap  = 30 * time.Second       // max backoff interval
)

// NewFS sets up a new filesystem for vault, with deposits/v2 support.
func NewFs(ctx context.Context, name, root string, m configmap.Mapper) (fs.Fs, error) {
	fs.Debugf(nil, "using deposits/v2")
	var opt Options
	err := configstruct.Set(m, &opt)
	if err != nil {
		return nil, err
	}
	api, err := oapi.New(opt.EndpointNormalized(), opt.Username, opt.Password)
	if err != nil {
		return nil, err
	}
	if err := api.Login(); err != nil {
		return nil, err
	}
	if v := api.Version(ctx); v != "" && v != api.VersionSupported {
		fmt.Fprintf(os.Stderr, VersionMismatchMessage, api.Version(ctx), api.VersionSupported)
		return nil, ErrVersionMismatch
	}
	// V2 is the current deposit API: /api/deposits/v2/
	var depositsV2Client *ClientWithResponses
	endpoint, err := opt.EndpointNormalizedDepositsV2()
	if err != nil {
		return nil, err
	}
	depositsV2Client, err = NewClientWithResponses(endpoint,
		WithHTTPClient(api.Client()))
	if err != nil {
		return nil, err
	}
	f := &Fs{
		name:             name,
		root:             root,
		opt:              opt,
		api:              api,
		depositsV2Client: depositsV2Client,
	}
	f.features = (&fs.Features{
		CanHaveEmptyDirectories: true,
		ReadMimeType:            true,
		SlowModTime:             true,
		About:                   f.About,
		DirMove:                 f.DirMove,
		Disconnect:              f.Disconnect,
		PublicLink:              f.PublicLink,
		Purge:                   f.Purge,
		PutStream:               f.PutStream,
		Shutdown:                f.Shutdown,
		UserInfo:                f.UserInfo,
	}).Fill(ctx, f)
	f.atexit = atexit.Register(f.Terminate)
	return f, nil
}

// Options for Vault.
type Options struct {
	Username        string `config:"username"`
	Password        string `config:"password"`
	Endpoint        string `config:"endpoint"` // e.g. http://localhost:8000/api
	ResumeDepositId int64  `config:"resume_deposit_id"`
	ChunkSize       int64  `config:"chunk_size"`
}

// EndpointNormalized handles trailing slashes.
func (opt Options) EndpointNormalized() string {
	return strings.TrimRight(opt.Endpoint, "/")
}

// EndpointNormalizedDepositsV2 returns the deposits V2 endpoint.
func (opt Options) EndpointNormalizedDepositsV2() (string, error) {
	u := opt.EndpointNormalized()
	if len(u) < 11 { // len("http://a.to")
		return "", ErrInvalidEndpoint
	}
	if strings.HasSuffix(u, "/api") {
		u = u[:len(u)-4]
	}
	return u, nil
}

// Fs is the main Vault filesystem. Most operations are accessed through the
// api.
type Fs struct {
	name     string
	root     string
	opt      Options         // vault options
	api      *oapi.CompatAPI // compat api, wrapper around oapi, exposing legacy methods; TODO: get rid of this
	features *fs.Features    // optional features
	// On a first put, we register a deposit to get a deposit id. Any
	// subsequent upload will be associated with that deposit id. On shutdown,
	// we send a finalize signal.
	depositsV2Client  *ClientWithResponses // v2 deposits API
	mu                sync.Mutex           // locks inflightDepositID
	inflightDepositID int                  // inflight deposit id, empty if none inflight
	started           time.Time            // registration time of the deposit
	atexit            atexit.FnHandle
}

// Fs Info
// -------

// Name returns the name of the filesystem.
func (f *Fs) Name() string { return f.name }

// Root returns the filesystem root.
func (f *Fs) Root() string { return f.root }

// String returns the name of the filesystem.
func (f *Fs) String() string { return f.name }

// Precision returns the support precision.
func (f *Fs) Precision() time.Duration { return 1 * time.Second }

// Hashes returns the supported hashes. Vault supports various hashes
// internally (availability may be delayed) and MD5 at upload time.
func (f *Fs) Hashes() hash.Set {
	return hash.Set(hash.MD5)
}

// Features returns optional features.
func (f *Fs) Features() *fs.Features { return f.features }

// Fs Ops
// ------

// List the objects and directories in dir into entries. The entries can be
// returned in any order but should be for a complete directory.
//
// dir should be "" to list the root, and should not have
// trailing slashes.
//
// This should return ErrDirNotFound if the directory isn't
// found.
func (f *Fs) List(ctx context.Context, dir string) (fs.DirEntries, error) {
	var (
		entries fs.DirEntries
		absPath = f.absPath(dir)
	)
	t, err := f.api.ResolvePath(absPath)
	if err != nil {
		if err == fs.ErrorObjectNotFound {
			return nil, fs.ErrorDirNotFound
		}
		return nil, err
	}
	switch {
	case dir == "" && t.NodeType == "FILE":
		obj := &Object{
			fs:       f,
			remote:   path.Join(dir, t.Name),
			treeNode: t,
		}
		entries = append(entries, obj)
	case t.NodeType == "ORGANIZATION" || t.NodeType == "COLLECTION" || t.NodeType == "FOLDER":
		nodes, err := f.api.List(t)
		if err != nil {
			return nil, err
		}
		for _, n := range nodes {
			switch {
			case n.NodeType == "COLLECTION" || n.NodeType == "FOLDER":
				dir := &Dir{
					fs:       f,
					remote:   path.Join(dir, n.Name),
					treeNode: n,
				}
				entries = append(entries, dir)
			case n.NodeType == "FILE":
				obj := &Object{
					fs:       f,
					remote:   path.Join(dir, n.Name),
					treeNode: n,
				}
				entries = append(entries, obj)
			default:
				return nil, fmt.Errorf("unknown node type: %v", n.NodeType)
			}
		}
	default:
		return nil, fs.ErrorDirNotFound
	}
	return entries, nil
}

// NewObject finds the Object at remote.  If it can't be found
// it returns the error ErrorObjectNotFound.
//
// If remote points to a directory then it should return
// ErrorIsDir if possible without doing any extra work,
// otherwise ErrorObjectNotFound.
func (f *Fs) NewObject(ctx context.Context, remote string) (fs.Object, error) {
	fs.Debugf(f, "new object at %v (%v)", remote, f.absPath(remote))
	t, err := f.api.ResolvePath(f.absPath(remote))
	if err != nil {
		return nil, err
	}
	switch {
	case t == nil:
		return nil, fs.ErrorObjectNotFound
	case t.NodeType == "ORGANIZATION" || t.NodeType == "COLLECTION" || t.NodeType == "FOLDER":
		return nil, fs.ErrorIsDir
	}
	return &Object{
		fs:       f,
		remote:   remote,
		treeNode: t,
	}, nil
}

// PutStream uploads a new object. Since we need to temporarily store files to upload, we can as well stream.
func (f *Fs) PutStream(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) (fs.Object, error) {
	fs.Debugf(f, "put stream %v [%v]", src.Remote(), src.Size())
	return f.Put(ctx, in, src, options...)
}

// requestDeposit attempts to start a new deposit. If a deposit is already
// inflight, this function returns immediately, without any error.
func (f *Fs) requestDeposit(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.inflightDepositID != 0 {
		return nil
	}
	t, err := f.api.ResolvePath(f.root)
	if err != nil {
		if err == fs.ErrorObjectNotFound {
			fs.Debugf(f, "root not found: %v", f.root)
			if err = f.mkdir(ctx, f.root); err != nil {
				return err
			}
			if t, err = f.api.ResolvePath(f.root); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	fs.Debugf(f, "root resolved: %s %v %v %T", f.root, t, err, err)
	var (
		parent = t
		body   = VaultDepositApiRegisterDepositJSONRequestBody{}
	)
	fs.Debugf(f, "request deposit: parent was %v", parent)
	switch {
	case parent.NodeType == "COLLECTION":
		c, err := f.api.TreeNodeToCollection(parent)
		if err != nil {
			return fmt.Errorf("failed to resolve treenode to collection: %w", err)
		}
		cid := int(c.Identifier())
		body.CollectionId = &cid
	case parent.NodeType == "FOLDER":
		pid := int(parent.ID)
		body.ParentNodeId = &pid
	default:
		return ErrCannotCopyToRoot
	}
	resp, err := f.depositsV2Client.VaultDepositApiRegisterDepositWithResponse(ctx, body)
	if err != nil {
		return err
	}
	if resp.StatusCode() != 200 {
		return fmt.Errorf("deposits/v2 registration failed with: %s", resp.HTTPResponse.Status)
	}
	if resp.JSON200.DepositId == 0 {
		return ErrMissingDepositIdentifier
	}
	f.inflightDepositID = resp.JSON200.DepositId
	f.started = time.Now()
	fs.Debugf(f, "successfully registered deposit: %v", f.inflightDepositID)
	return nil
}

// getFlowIdentifier returns a flow identifier for an object.
func (f *Fs) getFlowIdentifier(src fs.ObjectInfo) (s string, err error) {
	var h = md5.New()
	if _, err = io.WriteString(h, f.root); err != nil {
		return
	}
	if _, err = io.WriteString(h, src.Remote()); err != nil {
		return
	}
	return fmt.Sprintf("%s-%x", flowIdentifierPrefix, h.Sum(nil)), nil
}

// getFlowTotalChunks returns the number of chunks required to upload an object
// of a given size.
func getFlowTotalChunks(objectSize int, chunkSize int64) int {
	switch objectSize {
	case 0:
		return 1 // WT-2471
	default:
		return int(math.Ceil(float64(objectSize) / float64(chunkSize)))
	}
}

// Put uploads a new object, using v2 deposits. A new deposit is registered,
// once. Files are only written to a temporary file, if the remote does not
// support object size information.
func (f *Fs) Put(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) (fs.Object, error) {
	fs.Debugf(f, "put %v [%v]", src.Remote(), src.Size())
	var (
		flowIdentifier string
		err            error
	)
	// (1) Start a deposit, if not already started. TODO: support resuming a deposit.
	if err := f.requestDeposit(ctx); err != nil {
		return nil, err
	}
	// (2) Get a flow identifier for file.
	if flowIdentifier, err = f.getFlowIdentifier(src); err != nil {
		return nil, err
	}
	// (3) Determine, whether we can get the size of the object. Some backend
	// do not support size, then we have to move the data from the backend to a
	// temporary file first (which should rarely happen).
	var (
		tempfile   string
		objectSize int
	)
	if tempfile, objectSize, err = f.objectSize(in, src); err != nil {
		return nil, err
	}
	if tempfile != "" {
		f, err := os.Open(tempfile)
		if err != nil {
			return nil, err
		}
		in = f // breaks "accounting", does it affect anything?
		defer func() {
			_ = f.Close()
			// TODO: may be a problem on shutdown, as that will happen
			// elsewhere; TODO: move this into upload altogether
			_ = os.Remove(tempfile)
		}()
	}
	// (4) Need to get total size, and total number of chunks.
	var uploadInfo = &UploadInfo{
		flowTotalSize:   objectSize,
		flowTotalChunks: getFlowTotalChunks(objectSize, f.opt.ChunkSize),
		flowIdentifier:  flowIdentifier,
		in:              in,
		src:             src,
	}
	// (5) Upload file in chunks. TODO: this can be parallelized as well.
	// We're loading a small (order 1M) chunk into memory, so we get the
	// correct total size of the chunk.
	//
	// TODO: if we get interrupted inside this loop, we may not be able to
	// finalize the deposit, refs WT-2150, potentially related:
	// https://github.com/rclone/rclone/issues/966
	h, err := f.upload(ctx, uploadInfo)
	if err != nil {
		return nil, err
	}
	// We do not strictly need the hash sums, but we can compute the on the
	// fly, so we can augment the TreeNode value.
	sums := h.Sums()
	fs.Debugf(f, "chunk upload complete")
	return &Object{
		fs:     f,
		remote: src.Remote(),
		treeNode: &api.TreeNode{
			NodeType:   "FILE",
			ObjectSize: src.Size(),
			Md5Sum:     sums[hash.MD5],
			Sha1Sum:    sums[hash.SHA1],
			Sha256Sum:  sums[hash.SHA256],
		},
	}, nil
}

// objectSize tries to get the size of an object. If the object does not
// support reading its size, we spool the data into a temporary file and return
// the temporary filename. This may be necessary for rare cases, where the
// other backend does not support getting the size of an object before reading
// it in full.
func (f *Fs) objectSize(in io.Reader, src fs.ObjectInfo) (tempfile string, size int, err error) {
	switch {
	case src.Size() == -1:
		var (
			fi os.FileInfo
			f  *os.File
		)
		// Source does not support size, we stream to a temporary file and
		// return a reader of that file.
		if tempfile, err = iotemp.TempFileFromReader(in); err != nil {
			return "", 0, err
		}
		fs.Debugf(f, "object does not support size, spooled to temp file: %v", tempfile)
		if fi, err = os.Stat(tempfile); err != nil {
			return "", 0, err
		}
		size = int(fi.Size())
	default:
		size = int(src.Size()) // most objects will support size
	}
	return "", size, nil
}

// UploadInfo contains all information for a single file upload.
type UploadInfo struct {
	flowTotalChunks int
	flowTotalSize   int
	flowIdentifier  string
	in              io.Reader
	src             fs.ObjectInfo
	// i is the inflightChunkNumber keeps track of where we are with the
	// upload, modified during upload and only here, so we may pick up some
	// half-done work in the shutdown process, so we can get a HTTP 200 from
	// finalize
	i int
}

// IsDone returns the
func (info *UploadInfo) IsDone() bool {
	return info.i == info.flowTotalChunks
}

func (info *UploadInfo) resetStream() error {
	if info.src.Fs().Name() == "local" {
		filename := path.Join(info.src.Fs().Root(), info.src.String())
		f, err := os.Open(filename)
		if err != nil {
			return err
		}
		info.in = f
		info.i = 0
		return nil
	} else {
		return fmt.Errorf("cannot reset non-local stream")
	}
}

// upload is the main transfer function for a single file, which is wrapped in
// an UploadInfo value. Returns a hasher that contains the supported hashes of
// of the file object.
func (f *Fs) upload(ctx context.Context, info *UploadInfo) (hasher *hash.MultiHasher, err error) {
	hasher, err = hash.NewMultiHasherTypes(f.Hashes())
	if err != nil {
		return nil, err
	}
	for info.i < info.flowTotalChunks {
		info.i++
		fs.Infof(f, "[>>>] uploading file %v chunk %d/%d [%v]", info.src.Remote(), info.i, info.flowTotalChunks, time.Since(f.started))
		var (
			buf      bytes.Buffer                               // buffer for file data (we need the actual size at upload time)
			lr       = io.LimitReader(info.in, f.opt.ChunkSize) // chunk reader over stream
			wrapIn   = io.TeeReader(lr, hasher)                 // wrap input stream for hashing
			wbuf     = bytes.Buffer{}                           // buffer for multipart message
			w        = multipart.NewWriter(&wbuf)               // multipart writer
			mimeType = "application/octet-stream"               // file mime type
			n        int64                                      // actual length of this chunk
			err      error                                      // any error
			fw       io.Writer                                  // formfile writer
			resp     *http.Response                             // deposit API response
		)
		if n, err = io.Copy(&buf, wrapIn); err != nil { // n <= opt.ChunkSize
			return nil, err
		}
		// (5a) on first chunk, try to find mime type
		if info.i == 1 {
			mimeType = http.DetectContentType(buf.Bytes())
		}
		// (5b) write multipart fields
		mfw := &iotemp.MultipartFieldWriter{W: w}
		mfw.WriteField("depositId", fmt.Sprintf("%v", f.inflightDepositID))
		mfw.WriteField("flowChunkNumber", fmt.Sprintf("%v", info.i))
		mfw.WriteField("flowChunkSize", fmt.Sprintf("%v", f.opt.ChunkSize))
		mfw.WriteField("flowCurrentChunkSize", fmt.Sprintf("%v", n))
		mfw.WriteField("flowFilename", filepath.Base(info.src.Remote()))
		mfw.WriteField("flowIdentifier", info.flowIdentifier)
		mfw.WriteField("flowRelativePath", info.src.Remote())
		mfw.WriteField("flowTotalChunks", fmt.Sprintf("%v", info.flowTotalChunks))
		mfw.WriteField("flowTotalSize", fmt.Sprintf("%v", info.flowTotalSize))
		mfw.WriteField("flowMimetype", mimeType)
		mfw.WriteField("flowUserMtime", fmt.Sprintf("%v", info.src.ModTime(ctx).Format(time.RFC3339)))
		if err := mfw.Err(); err != nil {
			return nil, err
		}
		// (5c) write multipart file
		formFileName := fmt.Sprintf("%s-%016d", info.flowIdentifier, info.i)
		if fw, err = w.CreateFormFile("file", formFileName); err != nil { // can we use a random file name?
			return nil, err
		}
		if _, err := io.Copy(fw, &buf); err != nil {
			return nil, err
		}
		// (5d) finalize multipart writer
		if err := w.Close(); err != nil {
			return nil, err
		}
		// (5e) send chunk
		// The context passed may have a too eager deadline, so we give it a
		// fresh timeout per chunk upload request (note: this did not seem to
		// have been the cause of the previously encountered 404).
		ctx, cancel := context.WithTimeout(context.Background(), UploadChunkTimeout)
		defer cancel()
		backoff := retry.WithCappedDuration(UploadChunkBackoffCap, retry.NewFibonacci(UploadChunkBackoffBase))
		err = retry.Do(ctx, backoff, func(ctx context.Context) error {
			fs.Debugf(f, "starting upload... (buffer size: %v, [T=%v])", wbuf.Len(), time.Since(f.started))
			resp, err = f.depositsV2Client.VaultDepositApiSendChunkWithBody(ctx, w.FormDataContentType(), &wbuf)
			switch {
			case err != nil:
				// This may be cause by infrastructure errors, like DNS
				// failures, etc., so we can retry them as well. It's important
				// that we check this case first.
				return retry.RetryableError(err)
			case resp.StatusCode >= 500: // refs. VLT-518
				// We may recover from an HTTP 500 likely caused by a rare race
				// condition in a database trigger, encountered in 05/2023.
				fs.Debugf(f, "chunk upload retry: %v", resp.Status)
				return retry.RetryableError(err)
			case resp.StatusCode >= 400:
				// TODO: we get a HTTP 404 from prod, with message: {"detail": "Not Found"}
				// TODO: we get a 404 because deposit switches to "REPLICATED" quickly
				fs.Debugf(f, "chunk upload failed (deposit id=%v)", f.inflightDepositID)
				fs.Debugf(f, "got %v -- response dump follows", resp.Status)
				b, err := httputil.DumpResponse(resp, true)
				if err != nil {
					return err
				}
				fs.Debugf(f, string(b))
				// TODO: this can be triggered by running "sync", then
				// "CTRL-C", then without delay rerunning the "sync" command;
				// if the repeated command is issued after a delay, this issue
				// does not surface
				return fmt.Errorf("api responded with an HTTP %v, stopping chunk upload", resp.StatusCode)
			default:
				return nil
			}
		})
		// When chunk retry failed, we bail out.
		if err != nil {
			return nil, err
		}
	}
	return hasher, nil
}

// Mkdir creates a directory, if it does not exist.
func (f *Fs) Mkdir(ctx context.Context, dir string) error {
	return f.mkdir(ctx, f.absPath(dir))
}

// mkdir creates a directory, ignores the filesystem root and expects dir to be
// the absolute path. Will create parent directories if necessary.
func (f *Fs) mkdir(ctx context.Context, dir string) error {
	fs.Debugf(f, "mkdir: %v", dir)
	var t, _ = f.api.ResolvePath(dir)
	switch {
	case t != nil && (t.NodeType == "FOLDER" || t.NodeType == "COLLECTION"):
		return nil
	case t != nil:
		return fmt.Errorf("path already exists: %v [%s]", dir, t.NodeType)
	case f.root == "/" || strings.Count(dir, "/") == 1:
		return f.api.CreateCollection(ctx, path.Base(dir))
	default:
		segments := pathSegments(dir, "/")
		if len(segments) == 0 {
			return fmt.Errorf("broken path: %s", dir)
		}
		var (
			parent  *api.TreeNode
			current string
		)
		for i, s := range segments {
			fs.Debugf(f, "mkdir: %v %v %v", i, s, parent)
			current = path.Join(current, s)
			t, _ := f.api.ResolvePath(current)
			switch {
			case t != nil:
				parent = t
				continue
			case t == nil && i == 0:
				if err := f.api.CreateCollection(ctx, s); err != nil {
					return err
				}
			default:
				if err := f.api.CreateFolder(ctx, parent, s); err != nil {
					return err
				}
			}
			t, err := f.api.ResolvePath(current)
			if err != nil {
				return err
			}
			parent = t
		}
	}
	return nil
}

// Rmdir deletes a folder. Collections cannot be removed.
func (f *Fs) Rmdir(ctx context.Context, dir string) error {
	fs.Debugf(f, "rmdir %v", f.absPath(dir))
	t, err := f.api.ResolvePath(f.absPath(dir))
	if err != nil {
		return err
	}
	if t.NodeType == "FOLDER" || t.NodeType == "COLLECTION" {
		return f.api.Remove(ctx, t)
	}
	return fmt.Errorf("cannot delete node type %v", strings.ToLower(t.NodeType))
}

// Fs extra
// --------

// PublicLink returns the download link, if it exists.
func (f *Fs) PublicLink(ctx context.Context, remote string, expire fs.Duration, unlink bool) (link string, err error) {
	t, err := f.api.ResolvePath(f.absPath(remote))
	if err != nil {
		return "", err
	}
	switch v := t.ContentURL.(type) {
	case string:
		// TODO: may want to url encode
		u, err := url.Parse(v)
		if err != nil {
			return "", err
		}
		return u.String(), nil
	default:
		return "", fmt.Errorf("link not available for treenode %v", t.ID)
	}
}

// About returns currently only the quota.
func (f *Fs) About(ctx context.Context) (*fs.Usage, error) {
	organization, err := f.api.Organization()
	if err != nil {
		return nil, fmt.Errorf("api organization failed: %w", err)
	}
	stats, err := f.api.GetCollectionStats()
	if err != nil {
		return nil, fmt.Errorf("api collection failed: %w", err)
	}
	var (
		numFiles = stats.NumFiles()
		used     = stats.TotalSize()
		free     = organization.QuotaBytes - used
	)
	return &fs.Usage{
		Total:   &organization.QuotaBytes,
		Used:    &used,
		Free:    &free,
		Objects: &numFiles,
	}, nil
}

// UserInfo returns some information about the user, organization and plan.
func (f *Fs) UserInfo(ctx context.Context) (map[string]string, error) {
	u, err := f.api.User()
	if err != nil {
		return nil, err
	}
	organization, err := f.api.Organization()
	if err != nil {
		return nil, err
	}
	plan, err := f.api.Plan()
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"Username":               u.Username,
		"FirstName":              u.FirstName,
		"LastName":               u.LastName,
		"Organization":           organization.Name,
		"Plan":                   plan.Name,
		"DefaultFixityFrequency": plan.DefaultFixityFrequency,
		"QuotaBytes":             fmt.Sprintf("%d", organization.QuotaBytes),
		"LastLogin":              u.LastLogin,
	}, nil
}

// Disconnect logs out the current user.
func (f *Fs) Disconnect(ctx context.Context) error {
	fs.Debugf(f, "disconnect")
	f.api.Logout()
	return nil
}

// DirMove implements server side renames and moves.
func (f *Fs) DirMove(ctx context.Context, src fs.Fs, srcRemote, dstRemote string) error {
	fs.Debugf(f, "dir move: %v [%v] => %v", src.Root(), srcRemote, f.root)
	srcNode, err := f.api.ResolvePath(src.Root())
	if err != nil {
		return err
	}
	srcDirParent := path.Dir(src.Root())
	srcDirParentNode, err := f.api.ResolvePath(srcDirParent)
	if err != nil {
		return err
	}
	dstDirParent := path.Dir(f.root)
	dstDirParentNode, err := f.api.ResolvePath(dstDirParent)
	if err != nil {
		return err
	}
	if srcDirParentNode.ID == dstDirParentNode.ID {
		fs.Debugf(f, "move is a rename")
		t, err := f.api.ResolvePath(src.Root())
		if err != nil {
			return err
		}
		return f.api.Rename(ctx, t, path.Base(f.root))
	} else {
		switch {
		case srcNode.NodeType == "FILE":
			// If f.root exists and is a directory, we can move the file in
			// there; if f.root does not exists, we treat the parent as the dir
			// and the base as the file to copy to.
			rootNode, err := f.api.ResolvePath(f.root)
			if err == nil {
				if err := f.api.Move(ctx, srcNode, rootNode); err != nil {
					return err
				}
			} else {
				dstDir := path.Dir(f.root)
				if err := f.mkdir(ctx, dstDir); err != nil {
					return err
				}
				dstDirNode, err := f.api.ResolvePath(dstDir)
				if err != nil {
					return err
				}
				if err := f.api.Move(ctx, srcNode, dstDirNode); err != nil {
					return err
				}
				if path.Base(f.root) != path.Base(src.Root()) {
					return f.api.Rename(ctx, srcNode, path.Base(f.root))
				}
			}
		case srcNode.NodeType == "FOLDER" || srcNode.NodeType == "COLLECTION":
			fs.Debugf(f, "moving dir to %v", f.root)
			p, err := f.api.ResolvePath(f.root)
			if err != nil {
				return err
			}
			return f.api.Move(ctx, srcNode, p)
		}
	}
	return nil
}

// Purge remove a folder.
func (f *Fs) Purge(ctx context.Context, dir string) error {
	t, err := f.api.ResolvePath(f.absPath(dir))
	if err != nil {
		return err
	}
	if t.NodeType != "FOLDER" {
		return fmt.Errorf("can only purge folders, not %v", t.NodeType)
	}
	return f.api.Remove(ctx, t)
}

func (f *Fs) Shutdown(ctx context.Context) error {
	return f.finalize(ctx)
}

// Terminate the currently running deposit.
func (f *Fs) Terminate() {
	if f.inflightDepositID == 0 {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	body := TerminateDepositRequest{
		DepositId: f.inflightDepositID,
	}
	ctx := context.Background()
	resp, err := f.depositsV2Client.VaultDepositApiTerminateDeposit(ctx, body)
	if err != nil {
		fs.LogLevelPrintf(fs.LogLevelWarning, f, "terminate deposit failed: %v", err)
		return
	}
	if resp.StatusCode != 200 {
		fs.LogLevelPrintf(fs.LogLevelWarning, f, "terminate deposit failed: %v", resp.StatusCode)
		return
	}
	fs.Logf(f, "terminated deposit %d on user request", f.inflightDepositID)
}

// finalize sends finalize signal, only once, called on normal shutdown and on
// interrupted shutdown.
func (f *Fs) finalize(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.inflightDepositID == 0 {
		// nothing to be done
		return nil
	}
	fs.Debugf(f, "finalizing deposit %v", f.inflightDepositID)
	body := VaultDepositApiFinalizeDepositJSONRequestBody{
		DepositId: f.inflightDepositID,
	}
	resp, err := f.depositsV2Client.VaultDepositApiFinalizeDepositWithResponse(ctx, body)
	if err != nil {
		return err
	}
	if resp.StatusCode() != 200 {
		fs.Debugf(f, "[finalize] got %v -- response dump follows", resp.StatusCode())
		b, err := httputil.DumpResponse(resp.HTTPResponse, true)
		if err != nil {
			return err
		}
		fs.Debugf(f, string(b))
		return fmt.Errorf("finalize got: %v", resp.StatusCode())
	}
	fs.Debugf(f, "finalize done")
	f.inflightDepositID = 0
	return nil
}

// Command allows for custom commands. TODO(martin): We could have a cli dashboard or a deposit status command.
// func (f *Fs) Command(ctx context.Context, name string, args []string, opt map[string]string) (out interface{}, err error) {
// 	// TODO: fixity reports, distribution, ...
// 	switch name {
// 	default:
// 		return nil, fmt.Errorf("command not found")
// 	}
// }

// Fs helpers
// ----------

func (f *Fs) absPath(p string) string {
	return path.Join(f.root, p)
}

func pathSegments(p string, sep string) (result []string) {
	for _, v := range strings.Split(p, sep) {
		if strings.TrimSpace(v) == "" {
			continue
		}
		result = append(result, strings.Trim(v, sep))
	}
	return result
}

// Object
// ------

type Object struct {
	fs       *Fs
	remote   string
	treeNode *api.TreeNode
}

// Object DirEntry
// ---------------

func (o *Object) String() string {
	if o == nil {
		return "<nil>"
	}
	return o.remote
}
func (o *Object) Remote() string { return o.remote }
func (o *Object) ModTime(ctx context.Context) time.Time {
	epoch := time.Unix(0, 0)
	if o == nil || o.treeNode == nil {
		return epoch
	}
	layouts := []string{
		"January 2, 2006 15:04:05 UTC",
		"2006-01-02T15:04:05.99Z",
		"2006-01-02T15:04:05.999999Z",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, o.treeNode.ModifiedAt); err == nil {
			return t
		}
	}
	fs.Debugf(o, "failed to parse modification time layout: %v, falling back to epoch", o.treeNode.ModifiedAt)
	return epoch // TODO: that may cause unnecessary uploads, if T differs too much
}
func (o *Object) Size() int64 {
	return o.treeNode.Size()
}

// Object Info
// -----------

func (o *Object) Fs() fs.Info { return o.fs }
func (o *Object) Hash(ctx context.Context, ty hash.Type) (string, error) {
	if o.treeNode == nil {
		return "", nil
	}
	switch ty {
	case hash.MD5:
		if v, ok := o.treeNode.Md5Sum.(string); ok {
			return v, nil
		} else {
			return "", nil
		}
	case hash.SHA1:
		if v, ok := o.treeNode.Sha1Sum.(string); ok {
			return v, nil
		} else {
			return "", nil
		}
	case hash.SHA256:
		if v, ok := o.treeNode.Sha256Sum.(string); ok {
			return v, nil
		} else {
			return "", nil
		}
	case hash.None:
		// Testing systems sometimes miss a hash, so we just skip it.
		return "", nil
	}
	// TODO: we may want hash.ErrUnsupported, but we get an err, via:
	// https://github.com/rclone/rclone/blob/c85fbebce6f7166350c79e11fae763c8264ef865/fs/operations/operations.go#L105
	return "", hash.ErrUnsupported
}

// Storable returns true, since all we should be able to save all them.
func (o *Object) Storable() bool { return true }

// Object Ops
// ----------

// SetModTime set the modified at time to the current time.
func (o *Object) SetModTime(ctx context.Context, _ time.Time) error {
	fs.Debugf(o, "set mod time (now) for %v", o.ID())
	return o.fs.api.SetModTime(ctx, o.treeNode)
}
func (o *Object) Open(ctx context.Context, options ...fs.OpenOption) (io.ReadCloser, error) {
	fs.Debugf(o, "reading object contents from %v", o.ID())
	return o.treeNode.Content(options...)
}
func (o *Object) Update(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) error {
	fs.Debugf(o, "updating object contents at %v", o.ID())
	_, err := o.fs.Put(ctx, in, src, options...)
	return err
}
func (o *Object) Remove(ctx context.Context) error {
	fs.Debugf(o, "removing object: %v", o.ID())
	return o.fs.api.Remove(ctx, o.treeNode)
}

// Object extra
// ------------

func (o *Object) MimeType(ctx context.Context) string {
	return o.treeNode.MimeType()
}

// ID returns treenode path, which should be unique for any object in Vault.
func (o *Object) ID() string {
	if o.treeNode == nil {
		return ""
	}
	return o.treeNode.Path
}

func (o *Object) absPath() string {
	return path.Join(o.fs.Root(), o.remote)
}

// Dir
// ---

// Dir represents a collection or folder, something that can contain other
// objects.
type Dir struct {
	fs       *Fs
	remote   string
	treeNode *api.TreeNode
}

// Dir DirEntry
// ------------

func (dir *Dir) Fs() fs.Info    { return dir.fs }
func (dir *Dir) String() string { return dir.remote }
func (dir *Dir) Remote() string { return dir.remote }
func (dir *Dir) ModTime(ctx context.Context) time.Time {
	epoch := time.Unix(0, 0)
	if dir == nil || dir.treeNode == nil {
		return epoch
	}
	const layout = "January 2, 2006 15:04:05 UTC"
	if t, err := time.Parse(layout, dir.treeNode.ModifiedAt); err == nil {
		return t
	}
	return epoch
}
func (dir *Dir) Size() int64 { return 0 }

// Dir Ops
// -------

// Items returns the number of entries in this directory.
func (dir *Dir) Items() int64 {
	children, err := dir.fs.api.List(dir.treeNode)
	if err != nil {
		return 0
	}
	return int64(len(children))
}

// ID returns the treenode path. I believe most importantly, this needs to be
// unique (which path is).
func (dir *Dir) ID() string { return dir.treeNode.Path }

// Check if interfaces are satisfied
// ---------------------------------

var (
	_ fs.Abouter = (*Fs)(nil)
	// _ fs.Commander    = (*Fs)(nil)
	_ fs.DirMover     = (*Fs)(nil)
	_ fs.Disconnecter = (*Fs)(nil)
	_ fs.Fs           = (*Fs)(nil)
	_ fs.PublicLinker = (*Fs)(nil)
	_ fs.PutStreamer  = (*Fs)(nil)
	_ fs.Shutdowner   = (*Fs)(nil)
	_ fs.UserInfoer   = (*Fs)(nil)
	_ fs.MimeTyper    = (*Object)(nil)
	_ fs.Object       = (*Object)(nil)
	_ fs.IDer         = (*Object)(nil)
	_ fs.Directory    = (*Dir)(nil)
)
