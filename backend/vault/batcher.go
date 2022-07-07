package vault

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/rclone/rclone/backend/vault/api"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/operations"
	"github.com/rclone/rclone/lib/rest"
	"github.com/schollz/progressbar/v3"
)

const defaultUploadChunkSize = 1 << 20 // 1M

// batcher is used to group files for a deposit.
type batcher struct {
	fs                  *Fs                 // fs.root will be the parent collection or folder
	parent              *api.TreeNode       // resolved and possibly new parent treenode
	showDepositProgress bool                // show progress bar
	chunkSize           int64               // upload unit size
	resumeDepositId     int64               // if non-zero, try to resume deposit
	shutOnce            sync.Once           // only shutdown once
	mu                  sync.Mutex          // protect items
	items               []*batchItem        // file metadata and content for deposit items
	seen                map[string]struct{} // avoid duplicates in batch
}

// batchItem for Put and Update requests, basically capturing those methods' arguments.
type batchItem struct {
	root     string // the fs root
	filename string // some (temporary) file with contents
	src      fs.ObjectInfo
	options  []fs.OpenOption
}

// ToFile turns a batch item into a File for a deposit request.  This method
// sets the flow identifier. TODO: For resumable uploads, we need to derive the
// flow identifier from the file itself.
func (item *batchItem) ToFile(ctx context.Context) *api.File {
	flowIdentifier, err := item.deriveFlowIdentifier()
	if err != nil {
		fs.Debugf(item, "falling back to synthetic flow id (deposit will not be resumable [err: %v])", err)
		var (
			randInt    = 100_000_000 + rand.Intn(899_999_999) // fixed length
			randSuffix = fmt.Sprintf("%s-%d", time.Now().Format("20060102030405"), randInt)
		)
		flowIdentifier = fmt.Sprintf("rclone-vault-flow-%s", randSuffix)
	}
	return &api.File{
		Name:                 path.Base(item.src.Remote()),
		FlowIdentifier:       flowIdentifier,
		RelativePath:         item.src.Remote(),
		Size:                 item.src.Size(),
		PreDepositModifiedAt: item.src.ModTime(ctx).Format("2006-01-02T03:04:05.000Z"),
		Type:                 item.contentType(),
	}
}

// contentType tries to sniff the content type, or returns the empty string.
func (item *batchItem) contentType() string {
	f, err := os.Open(item.filename)
	if err != nil {
		return ""
	}
	defer f.Close()
	buf := make([]byte, 512)
	if _, err := f.Read(buf); err != nil {
		return ""
	}
	switch v := http.DetectContentType(buf); v {
	case "application/octet-stream":
		// DetectContentType always returns a valid MIME type: if it cannot
		// determine a more specific one, it returns
		// "application/octet-stream".
		return ""
	default:
		return v
	}
}

// deriveFlowIdentifier from a file, faster than a whole file fingerprint.
func (item *batchItem) deriveFlowIdentifier() (string, error) {
	f, err := os.Open(item.filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	const maxBytes = 1 << 30 // 1GB
	h := md5.New()
	if _, err := io.Copy(h, io.LimitReader(f, maxBytes)); err != nil {
		return "", err
	}
	// Filename and root would probably be enough. For the moment we include a
	// partial MD5 sum of the file.
	return fmt.Sprintf("rclone-vault-flow-%x-%s-%s",
		h.Sum(nil),
		hex.EncodeToString([]byte(item.root)),
		hex.EncodeToString([]byte(item.src.Remote()))), nil
}

// String will most likely show up in debug messages.
func (b *batcher) String() string {
	return "vault batcher"
}

// Add a single item to the batch.
func (b *batcher) Add(item *batchItem) {
	b.mu.Lock()
	if b.seen == nil {
		b.seen = make(map[string]struct{})
	}
	if _, ok := b.seen[item.filename]; !ok {
		b.items = append(b.items, item)
		b.seen[item.filename] = struct{}{}
	}
	b.mu.Unlock()
}

// Chunker allows to read file in chunks.
type Chunker struct {
	chunkSize int64
	fileSize  int64
	numChunks int64
	f         *os.File
}

// NewChunker sets up a new chunker. Caller will need to close this to close
// the associated file.
func NewChunker(filename string, chunkSize int64) (*Chunker, error) {
	if chunkSize < 1 {
		return nil, fmt.Errorf("chunk size must be positive")
	}
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	numChunks := int64(math.Ceil(float64(fi.Size()) / float64(chunkSize)))
	return &Chunker{
		f:         f,
		chunkSize: chunkSize,
		fileSize:  fi.Size(),
		numChunks: numChunks,
	}, nil
}

// FileSize returns the filesize.
func (c *Chunker) FileSize() int64 {
	return c.fileSize
}

// NumChunks returns the number of chunks this file is splitted to.
func (c *Chunker) NumChunks() int64 {
	return c.numChunks
}

// ChunkReader returns the reader over a section of the file. Counting starts at zero.
func (c *Chunker) ChunkReader(i int64) io.Reader {
	offset := i * c.chunkSize
	return io.NewSectionReader(c.f, offset, c.chunkSize)
}

// Close closes the wrapped file.
func (c *Chunker) Close() error {
	return c.f.Close()
}

// Shutdown creates a new deposit request for all batch items and uploads them.
// This is the one of the last things rclone run before exiting. There is no
// way to relay an error to return from here, so we deliberately exit the
// process from here with an exit code of 1, if anything fails.
func (b *batcher) Shutdown(ctx context.Context) (err error) {
	fs.Debugf(b, "shutdown started")
	b.shutOnce.Do(func() {
		if len(b.items) == 0 {
			fs.Debugf(b, "nothing to deposit")
			return
		}
		var (
			// We do not want to be cancelled in Shutdown; or if we do, we want
			// to set our own timeout for deposit uploads.
			ctx               = context.Background()
			totalSize   int64 = 0
			files       []*api.File
			progressBar *progressbar.ProgressBar
			t           *api.TreeNode
			depositId   int64
		)
		// Make sure the parent exists.
		t, err = b.fs.api.ResolvePath(b.fs.root)
		if err != nil {
			if err == fs.ErrorObjectNotFound {
				if err = b.fs.mkdir(ctx, b.fs.root); err != nil {
					return
				}
				if t, err = b.fs.api.ResolvePath(b.fs.root); err != nil {
					return
				}
			} else {
				return
			}
		}
		b.parent = t
		// Prepare deposit request.
		fs.Logf(b, "preparing %d file(s) for deposit", len(b.items))
		for _, item := range b.items {
			totalSize += item.src.Size()
			files = append(files, item.ToFile(ctx))
		}
		switch {
		case b.resumeDepositId > 0:
			depositId = b.resumeDepositId
			fs.Debugf(b, "trying to resume deposit %d", depositId)
		default:
			// TODO: we may want to reuse a deposit to continue an interrupted
			// deposit, e.g. --vault-resume-deposit-id 123
			rdr := &api.RegisterDepositRequest{
				TotalSize: totalSize,
				Files:     files,
			}
			switch {
			case b.parent.NodeType == "COLLECTION":
				c, err := b.fs.api.TreeNodeToCollection(b.parent)
				if err != nil {
					err = fmt.Errorf("failed to resolve treenode to collection: %w", err)
					return
				}
				rdr.CollectionId = c.Identifier()
			case b.parent.NodeType == "FOLDER":
				rdr.ParentNodeId = b.parent.Id
			}
			depositId, err = b.fs.api.RegisterDeposit(ctx, rdr)
			if err != nil {
				err = fmt.Errorf("deposit failed: %w", err)
				return
			}
			fs.Debugf(b, "created deposit %v", depositId)
		}
		if b.showDepositProgress {
			progressBar = progressbar.DefaultBytes(totalSize, "<5>NOTICE: depositing")
		}
		for i, item := range b.items {
			// Upload file with a single chunk. First issue a GET, if that is a
			// 204 then follow up with a POST.
			//
			// TODO: streamline the chunking part a bit
			// TODO: we could parallelize chunk uploads
			var (
				chunker *Chunker
				j       int64
				resp    *http.Response
			)
			chunker, err = NewChunker(item.filename, b.chunkSize)
			if err != nil {
				return
			}
			for j = 1; j <= chunker.NumChunks(); j++ {
				currentChunkSize := b.chunkSize
				if j == chunker.NumChunks() {
					currentChunkSize = chunker.FileSize() - ((j - 1) * b.chunkSize)
				}
				fs.Debugf(b, "[%d/%d] %d %d %s",
					j, chunker.NumChunks(), currentChunkSize, chunker.FileSize(), item.filename)
				params := url.Values{
					"depositId":            []string{strconv.Itoa(int(depositId))},
					"flowChunkNumber":      []string{strconv.Itoa(int(j))},
					"flowChunkSize":        []string{strconv.Itoa(int(b.chunkSize))},
					"flowCurrentChunkSize": []string{strconv.Itoa(int(currentChunkSize))},
					"flowFilename":         []string{files[i].Name},
					"flowIdentifier":       []string{files[i].FlowIdentifier},
					"flowRelativePath":     []string{files[i].RelativePath},
					"flowTotalChunks":      []string{strconv.Itoa(int(chunker.NumChunks()))},
					"flowTotalSize":        []string{strconv.Itoa(int(chunker.FileSize()))},
					"upload_token":         []string{"my_token"}, // just copy'n'pasting ...
				}
				fs.Debugf(b, "params: %v", params)
				opts := rest.Opts{
					Method:     "GET",
					Path:       "/flow_chunk",
					Parameters: params,
				}
				resp, err = b.fs.api.Call(ctx, &opts)
				if err != nil {
					fs.LogPrintf(fs.LogLevelError, b, "call (GET): %v", err)
					return
				}
				defer resp.Body.Close()
				if resp.StatusCode >= 300 {
					fs.LogPrintf(fs.LogLevelError, b, "expected HTTP < 300, got: %v", resp.StatusCode)
					err = fmt.Errorf("expected HTTP < 300, got %v", resp.StatusCode)
					return
				} else {
					fs.Debugf(b, "GET returned: %v", resp.StatusCode)
				}
				var r io.Reader
				if b.showDepositProgress {
					r = io.TeeReader(chunker.ChunkReader(j-1), progressBar)
				} else {
					r = chunker.ChunkReader(j - 1)
				}
				size := currentChunkSize
				opts = rest.Opts{
					Method:               "POST",
					Path:                 "/flow_chunk",
					MultipartParams:      params,
					ContentLength:        &size,
					MultipartContentName: "file",
					MultipartFileName:    path.Base(item.src.Remote()), // TODO: is it?
					Body:                 r,
				}
				resp, err = b.fs.api.CallJSON(ctx, &opts, nil, nil)
				if err != nil {
					fs.LogPrintf(fs.LogLevelError, b, "call (POST): %v", err)
					return
				}
				if err = resp.Body.Close(); err != nil {
					fs.LogPrintf(fs.LogLevelError, b, "body: %v", err)
					return
				}
			}
			if err = chunker.Close(); err != nil {
				fs.LogPrintf(fs.LogLevelError, b, "close: %v", err)
				return
			}
			if err = os.Remove(item.filename); err != nil {
				fs.LogPrintf(fs.LogLevelError, b, "remove: %v", err)
				return
			}
		}
		fs.Logf(b, "upload done (%d), deposited %s, %d item(s)",
			depositId, operations.SizeString(totalSize, true), len(b.items))
		return
	})
	return
}
