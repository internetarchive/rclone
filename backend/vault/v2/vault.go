// Package v2 implements changes proposed in MR362, namely v2 deposit API
// without treenode allocation.
package v2

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rclone/rclone/backend/vault/api"
	"github.com/rclone/rclone/backend/vault/iotemp"
	"github.com/rclone/rclone/backend/vault/oapi"
	"github.com/rclone/rclone/backend/vault/pathutil"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/config/configstruct"
	"github.com/rclone/rclone/fs/hash"
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
	defaultUploadChunkSize    = 1 << 24 // 16M
	defaultMaxParallelChunks  = 2
	defaultMaxParallelUploads = 2
	flowIdentifierPrefix      = "rclone-vault-flow"
)

var (
	ErrCannotCopyToRoot         = errors.New("copying files to root is not supported in vault")
	ErrInvalidPath              = errors.New("invalid path")
	ErrVersionMismatch          = errors.New("api version mismatch")
	ErrMissingDepositIdentifier = errors.New("missing deposit identifier")
)

// NewFS sets up a new filesystem for vault, with deposits/v2 support.
func NewFs(ctx context.Context, name, root string, m configmap.Mapper) (fs.Fs, error) {
	fs.Debugf(nil, "[exp] using experimental fs with deposits/v2")
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
		return nil, ErrVersionMismatch
	}
	// Setup v2 client, when requested, current endpoint:
	// /api/deposits/v2/
	//
	// May fail, if flag is used against endpoint w/o support for v2.
	// TODO: need to set auth handler for requests
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
	fs.Debugf(nil, "v2 client at %v", endpoint)
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
		Command:                 f.Command,
		DirMove:                 f.DirMove,
		Disconnect:              f.Disconnect,
		PublicLink:              f.PublicLink,
		Purge:                   f.Purge,
		PutStream:               f.PutStream,
		Shutdown:                f.Shutdown,
		UserInfo:                f.UserInfo,
	}).Fill(ctx, f)
	return f, nil
}

// Options for Vault.
type Options struct {
	Username                 string `config:"username"`
	Password                 string `config:"password"`
	Endpoint                 string `config:"endpoint"` // e.g. http://localhost:8000/api
	SuppressProgressBar      bool   `config:"suppress_progress_bar"`
	ResumeDepositId          int64  `config:"resume_deposit_id"`
	ChunkSize                int64  `config:"chunk_size"`
	MaxParallelChunks        int    `config:"max_parallel_chunks"`
	MaxParallelUploads       int    `config:"max_parallel_uploads"`
	SkipContentTypeDetection bool   `config:"skip_content_type_detection"`
}

// EndpointNormalized handles trailing slashes.
func (opt Options) EndpointNormalized() string {
	return strings.TrimRight(opt.Endpoint, "/")
}

// EndpointNormalizedDepositsV2 returns the deposits V2 endpoint. TODO(martin): move fixed value out.
func (opt Options) EndpointNormalizedDepositsV2() (string, error) {
	// return url.JoinPath(opt.EndpointNormalized(), "deposits/v2")
	return "http://localhost:8000/", nil
}

// Fs is the main Vault filesystem. Most operations are accessed through the
// api.
type Fs struct {
	name     string
	root     string
	opt      Options         // vault options
	api      *oapi.CompatAPI // compat api, wrapper around oapi, exposing legacy methods
	features *fs.Features    // optional features
	// On a first put, we are registering a deposit and retrieving a deposit
	// id. Any subsequent upload will be associated with that deposit id. On
	// shutdown, we send a finalize signal.
	depositsV2Client  *ClientWithResponses // v2 deposits API
	mu                sync.Mutex
	inflightDepositID int // inflight deposit id, empty if none inflight
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

// Hashes returns the supported hashes. Previously, we supported MD5, SHA1,
// SHA256 - but for large deposits, this would slow down uploads considerably.
// So for now, we do not want to support any hash.
func (f *Fs) Hashes() hash.Set { return hash.Set(hash.None) }

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
	if !pathutil.IsValidPath(remote) {
		return nil, ErrInvalidPath
	}
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
	fs.Debugf(f, "successfully registered deposit: %v", f.inflightDepositID)
	return nil
}

// getFlowIdentifier returns a flow identifier for an object.
func (f *Fs) getFlowIdentifier(src fs.ObjectInfo) (string, error) {
	var h = md5.New()
	if _, err := io.WriteString(h, f.root); err != nil {
		return nil, err
	}
	if _, err := io.WriteString(h, src.Remote()); err != nil {
		return nil, err
	}
	return fmt.Sprintf("%s-%x", flowIdentifierPrefix, h.Sum(nil)), nil
}

// Put uploads a new object, using v2 deposits. A new deposit is registered,
// once. Files are only written to a temporary file, if the remote does not
// support object size information.
func (f *Fs) Put(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) (fs.Object, error) {
	fs.Debugf(f, "put %v [%v]", src.Remote(), src.Size())
	if !pathutil.IsValidPath(src.Remote()) {
		return nil, ErrInvalidPath
	}
	// (1) Start a deposit, if not already started.
	if err := f.requestDeposit(ctx); err != nil {
		return nil, err
	}
	// (2) Get a flow identifier for file.
	flowIdentifier, err := f.getFlowIdentifier(src)
	if err != nil {
		return nil, err
	}
	// (3) Determine, whether we can get the size of the object.
	var (
		filename   string
		objectSize int
		err        error
	)
	switch {
	case src.Size() == -1: // https://is.gd/O7uQoq
		if filename, err = iotemp.TempFileFromReader(in); err != nil {
			return nil, err
		}
		fs.Debugf(f, "object does not support size, spooled to temp file: %v", filename)
		fi, err := os.Stat(filename)
		if err != nil {
			return nil, err
		}
		objectSize = int(fi.Size())
		f, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		in = f // breaks "accounting", does it affect anything?
		defer func() {
			_ = f.Close()
			_ = os.Remove(filename)
		}()
	default:
		objectSize = int(src.Size())
	}
	// (4) Need to get total size, and total number of chunks.
	var (
		flowTotalSize   = objectSize
		flowTotalChunks = int(math.Ceil(float64(flowTotalSize) / float64(f.opt.ChunkSize)))
	)
	// (5) Upload file in chunks. TODO: this can be parallelized as well.
	// We're loading a small (order 1M) chunk into memory, so we get the
	// correct total size of the chunk.
	for i := 1; i <= flowTotalChunks; i++ {
		fs.Debugf(f, "[>>>] uploading chunk %d/%d", i, flowTotalChunks)
		var (
			buf  bytes.Buffer                          // buffer for file data
			lr   = io.LimitReader(in, f.opt.ChunkSize) // chunk reader over stream
			wbuf = bytes.Buffer{}                      // buffer for multipart message
			w    = multipart.NewWriter(&wbuf)          // multipart writer
		)
		n, err = io.Copy(&buf, lr) // n <= opt.ChunkSize
		if err != nil {
			return nil, err
		}
		// (5a) write multipart fields
		mfw := &iotemp.MultipartFieldWriter{W: w}
		mfw.WriteField("depositId", fmt.Sprintf("%v", f.inflightDepositID))
		mfw.WriteField("flowChunkNumber", fmt.Sprintf("%v", i))
		mfw.WriteField("flowChunkSize", fmt.Sprintf("%v", f.opt.ChunkSize))
		mfw.WriteField("flowCurrentChunkSize", fmt.Sprintf("%v", n))
		mfw.WriteField("flowFilename", path.Base(src.Remote()))
		mfw.WriteField("flowIdentifier", flowIdentifier)
		mfw.WriteField("flowRelativePath", src.Remote())
		mfw.WriteField("flowTotalChunks", fmt.Sprintf("%v", flowTotalChunks))
		mfw.WriteField("flowTotalSize", fmt.Sprintf("%v", flowTotalSize))
		mfw.WriteField("flowMimetype", "application/octet-stream")
		mfw.WriteField("flowUserMtime", fmt.Sprintf("%v", time.Now().Format(time.RFC3339)))
		if mfw.Err() != nil {
			return nil, mfw.Err()
		}
		// (5b) write multipart file
		formFileName := fmt.Sprintf("%s-%016d", flowIdentifier, i)
		fw, err := w.CreateFormFile("file", formFileName) // can we use a random file name?
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(fw, &buf); err != nil {
			return nil, err
		}
		// (5c) finalize multipart writer
		if err := w.Close(); err != nil {
			return nil, err
		}
		fs.Debugf(f, "%s", string(wbuf.Bytes()))
		fs.Debugf(f, "content-type: %v", w.FormDataContentType())
		// (5d) send chunk
		resp, err := f.depositsV2Client.VaultDepositApiSendChunkWithBody(ctx, w.FormDataContentType(), &wbuf)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 400 {
			fs.Debugf(f, "got %v -- response dump follows", resp.Status)
			b, err := httputil.DumpResponse(resp, true)
			if err != nil {
				return nil, err
			}
			fs.Debugf(f, string(b))
		} else {
			fs.Debugf(f, "upload done")
		}
		fs.Debugf(f, "sent chunk, got: %v", resp.StatusCode)
	}
	fs.Debugf(f, "all chunks upload complete")
	return &Object{
		fs:     f,
		remote: src.Remote(),
		treeNode: &api.TreeNode{
			NodeType:   "FILE",
			ObjectSize: src.Size(),
		},
	}, nil
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
	if t.NodeType == "FOLDER" {
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

// Shutdown sends finalize signal.
func (f *Fs) Shutdown(ctx context.Context) error {
	fs.Debugf(f, "shutdown")
	fs.Debugf(f, "TODO: send finalize")
	if f.inflightDepositID == 0 {
		// nothing to be done
		return nil
	}
	body := VaultDepositApiFinalizeDepositJSONRequestBody{
		DepositId: f.inflightDepositID,
	}
	resp, err := f.depositsV2Client.VaultDepositApiFinalizeDepositWithResponse(ctx, body)
	if err != nil {
		return err
	}
	if resp.StatusCode() != 200 {
		return fmt.Errorf("finalize got: %v", resp.StatusCode())
	}
	return nil
}

// Command allows for custom commands. TODO(martin): We could have a cli dashboard or a deposit status command.
func (f *Fs) Command(ctx context.Context, name string, args []string, opt map[string]string) (out interface{}, err error) {
	// TODO: fixity reports, distribution, ...
	switch name {
	case "status", "st", "deposit-status", "ds", "dst":
		if len(args) == 0 {
			return nil, fmt.Errorf("deposit id required")
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return nil, fmt.Errorf("deposit id must be numeric")
		}
		ds, err := f.api.DepositStatus(int64(id))
		if err != nil {
			return nil, fmt.Errorf("failed to get deposit status")
		}
		return ds, nil
		// Add more custom commands here.
	}
	return nil, fmt.Errorf("command not found")
}

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
	const layout = "January 2, 2006 15:04:05 UTC"
	if t, err := time.Parse(layout, o.treeNode.ModifiedAt); err == nil {
		return t
	}
	return epoch
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
	_ fs.Abouter      = (*Fs)(nil)
	_ fs.Commander    = (*Fs)(nil)
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

// Experimental functions
// ----------------------
