package vault

import (
	"context"
	"errors"

	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/rclone/rclone/backend/vault/api"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/operations"
	"github.com/rclone/rclone/lib/rest"
	"github.com/schollz/progressbar/v3"
)

const (
	defaultUploadChunkSize = 1 << 20 // 1M
	flowIdentifierPrefix   = "rclone-vault-flow"
)

var (
	ErrCannotCopyToRoot         = errors.New("copying files to root is not supported in vault")
	ErrMissingDepositIdentifier = errors.New("missing deposit identifier")
)

// batcher is used to group upload files (deposit).
type batcher struct {
	fs                  *Fs                 // fs.root will be the parent collection or folder
	parent              *api.TreeNode       // resolved and possibly new parent treenode
	showDepositProgress bool                // show progress bar
	chunkSize           int64               // upload unit size in bytes
	resumeDepositId     int64               // if non-zero, try to resume deposit
	shutOnce            sync.Once           // only shutdown once
	mu                  sync.Mutex          // protect items
	items               []*batchItem        // file metadata and content for deposit items
	seen                map[string]struct{} // avoid duplicates in batch items

	// The following fields are set up during processing, e.g. after the
	// deposit has been registered. The list of files is duplicating some
	// information in items; TODO: streamline.
	depositIdentifier int64                    // deposit ID, set while uploading
	files             []*api.File              // items, but represented as API items
	totalSize         int64                    // total upload size in bytes
	progressBar       *progressbar.ProgressBar // setup before upload starts
}

// batchItem for Put and Update requests, basically capturing those methods' arguments.
type batchItem struct {
	root                    string          // the fs root
	filename                string          // some file with contents, may be temporary
	src                     fs.ObjectInfo   // object info
	options                 []fs.OpenOption // open options
	deleteFileAfterTransfer bool            // if true, delete the file given in filename; only set this to true, if you are using temporary files
}

// ToFile turns a batchItem value into a api.File for a deposit request. This
// method sets the flow identifier. Returns nil, when a batch item cannot be
// converted.
func (item *batchItem) ToFile(ctx context.Context) *api.File {
	if item == nil || item.src == nil {
		return nil
	}
	flowIdentifier, err := item.deriveFlowIdentifier()
	if err != nil {
		return nil
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

// contentType detects the content type. Returns the empty string, if no
// specific content type could be found. TODO(martin): This reads 512b from the
// file. May be a bottleneck when working with larger number of files.
func (item *batchItem) contentType() string {
	if item == nil {
		return ""
	}
	f, err := os.Open(item.filename)
	if err != nil {
		return ""
	}
	defer f.Close()
	buf := make([]byte, 512)
	if _, err := f.Read(buf); err != nil {
		return ""
	}
	if v := http.DetectContentType(buf); v == "application/octet-stream" {
		// DetectContentType always returns a valid MIME type: if it cannot
		// determine a more specific one, it returns
		// "application/octet-stream".
		return ""
	} else {
		return v
	}
}

// deriveFlowIdentifier derives a unique per file identifier from metadata (not
// content, for performance).
func (item *batchItem) deriveFlowIdentifier() (string, error) {
	if item == nil || item.src == nil {
		return "", nil
	}
	var h = md5.New()
	// Previously, we read up to 16M of the file and included that into the
	// hash, but for large number of files, this becomes a bottleneck. We want
	// this identifier to be stable and derived from the file, but we can use
	// the path as well (and be much faster).
	if _, err := io.WriteString(h, item.root); err != nil {
		return "", err
	}
	if _, err := io.WriteString(h, item.src.Remote()); err != nil {
		return "", err
	}
	// Filename and root may be enough. For the moment we include a partial MD5
	// sum of the file. We also want the filename length to be constant.
	return fmt.Sprintf("%s-%x", flowIdentifierPrefix, h.Sum(nil)), nil
}

// String will most likely show up in debug messages.
func (b *batcher) String() string {
	return fmt.Sprintf("vault batcher [%v]", len(b.items))
}

// Add a single item to the batch. If the item has been added before (same
// filename) it will be ignored. This is threadsafe, as rclone will be default
// run uploads concurrently.
func (b *batcher) Add(item *batchItem) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.seen == nil {
		b.seen = make(map[string]struct{})
	}
	if _, ok := b.seen[item.filename]; !ok {
		b.items = append(b.items, item)
		b.seen[item.filename] = struct{}{}
	} else {
		fs.Debugf(b, "ignoring already batched file: %v", item.filename)
	}
}

// files returns batch items as a list of vault API file objects and the total
// size of the objects. If an item cannot be converted, if will be ignored.
func (b *batcher) itemsToFiles(ctx context.Context) (files []*api.File, totalSize int64) {
	for _, item := range b.items {
		if f := item.ToFile(ctx); f != nil {
			totalSize += item.src.Size()
			files = append(files, f)
		}
	}
	return files, totalSize
}

// completeRegisterDepositRequest updates parent information of the
// registration request. Modifies request in place.
func (b *batcher) completeRegisterDepositRequest(rdr *api.RegisterDepositRequest) error {
	switch {
	case b.parent.NodeType == "COLLECTION":
		c, err := b.fs.api.TreeNodeToCollection(b.parent)
		if err != nil {
			err = fmt.Errorf("failed to resolve treenode to collection: %w", err)
			return err
		}
		rdr.CollectionId = c.Identifier()
	case b.parent.NodeType == "FOLDER":
		rdr.ParentNodeId = b.parent.Id
	default:
		return ErrCannotCopyToRoot
	}
	return nil
}

// ensureParentExists tries to create parent folder, if it does not exist.
func (b *batcher) ensureParentExists(ctx context.Context) error {
	var (
		t   *api.TreeNode
		err error
	)
	if b.parent != nil {
		return nil
	}
	t, err = b.fs.api.ResolvePath(b.fs.root)
	if err != nil {
		if err == fs.ErrorObjectNotFound {
			if err = b.fs.mkdir(ctx, b.fs.root); err != nil {
				return err
			}
			if t, err = b.fs.api.ResolvePath(b.fs.root); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	b.parent = t
	return nil
}

// Shutdown creates a new deposit request for all batch items and uploads them.
// This is the one of the last things rclone runs before exiting.
func (b *batcher) Shutdown(ctx context.Context) (err error) {
	fs.Debugf(b, "shutdown started")
	b.shutOnce.Do(func() {
		if len(b.items) == 0 {
			return
		}
		// We do not want to be cancelled in Shutdown; or if we do, we want
		// to set our own timeout for deposit uploads.
		var ctx = context.Background()
		// Make sure the parent exists.
		if err = b.ensureParentExists(ctx); err != nil {
			return
		}
		// Prepare deposit request.
		fs.Logf(b, "preparing %d file(s) for deposit", len(b.items))
		b.files, b.totalSize = b.itemsToFiles(ctx)
		if len(b.files) != len(b.items) {
			err = fmt.Errorf("not all items (%v) converted to files (%v)", len(b.items), len(b.files))
			return
		}
		// TODO: We want to clean any file from the deposit request, that
		// already exists on the remote until WT-1605 is resolved
		switch {
		case b.resumeDepositId > 0:
			b.depositIdentifier = b.resumeDepositId
			fs.Logf(b, "trying to resume deposit %d", b.depositIdentifier)
		default:
			rdr := &api.RegisterDepositRequest{
				TotalSize: b.totalSize,
				Files:     b.files,
			}
			// Complete parent information.
			b.completeRegisterDepositRequest(rdr)
			// Register deposit.
			b.depositIdentifier, err = b.fs.api.RegisterDeposit(ctx, rdr)
			if err != nil {
				err = fmt.Errorf("deposit failed: %w", err)
				return
			}
			fs.Debugf(b, "created deposit %v", b.depositIdentifier)
		}
		if b.showDepositProgress {
			b.progressBar = progressbar.DefaultBytes(b.totalSize, "<5>NOTICE: depositing")
		}
		for i, item := range b.items {
			if err = b.UploadItem(ctx, item, b.files[i]); err != nil {
				return
			}
			// // TODO: streamline the chunking part a bit
			// // TODO: we could parallelize chunk uploads
			// var (
			// 	chunker *Chunker
			// 	j       int64
			// 	resp    *http.Response
			// )
			// if chunker, err = NewChunker(item.filename, b.chunkSize); err != nil {
			// 	return
			// }
			// // We start at j = 1, since flowChunkNumber seems to start at 1.
			// for j = 1; j <= chunker.NumChunks(); j++ {
			// 	// Wrap upload into a function, so we can parallelize.
			// 	// TODO(martin): refactor this
			// 	currentChunkSize := chunker.ChunkSize(j - 1)
			// 	fs.Debugf(b, "[%d/%d] %d %d %s",
			// 		j,
			// 		chunker.NumChunks(),
			// 		currentChunkSize,
			// 		chunker.FileSize(),
			// 		item.filename,
			// 	)
			// 	params := url.Values{
			// 		"depositId":            []string{strconv.Itoa(int(b.depositIdentifier))},
			// 		"flowChunkNumber":      []string{strconv.Itoa(int(j))},
			// 		"flowChunkSize":        []string{strconv.Itoa(int(b.chunkSize))},
			// 		"flowCurrentChunkSize": []string{strconv.Itoa(int(currentChunkSize))},
			// 		"flowFilename":         []string{b.files[i].Name},
			// 		"flowIdentifier":       []string{b.files[i].FlowIdentifier},
			// 		"flowRelativePath":     []string{b.files[i].RelativePath},
			// 		"flowTotalChunks":      []string{strconv.Itoa(int(chunker.NumChunks()))},
			// 		"flowTotalSize":        []string{strconv.Itoa(int(chunker.FileSize()))},
			// 		"upload_token":         []string{"my_token"}, // TODO(martin): just copy'n'pasting ...
			// 	}
			// 	fs.Debugf(b, "params: %v", params)
			// 	opts := rest.Opts{
			// 		Method:     "GET",
			// 		Path:       "/flow_chunk",
			// 		Parameters: params,
			// 	}
			// 	resp, err = b.fs.api.Call(ctx, &opts)
			// 	if err != nil {
			// 		fs.LogPrintf(fs.LogLevelError, b, "call (GET): %v", err)
			// 		return
			// 	}
			// 	defer resp.Body.Close()
			// 	if resp.StatusCode >= 300 {
			// 		fs.LogPrintf(fs.LogLevelError, b, "expected HTTP < 300, got: %v", resp.StatusCode)
			// 		err = fmt.Errorf("expected HTTP < 300, got %v", resp.StatusCode)
			// 		return
			// 	} else {
			// 		fs.Debugf(b, "GET returned: %v", resp.StatusCode)
			// 	}
			// 	var (
			// 		r    io.Reader
			// 		chr  = chunker.ChunkReader(j - 1)
			// 		size = currentChunkSize // size will get mutated during request
			// 	)
			// 	if b.showDepositProgress {
			// 		r = io.TeeReader(chr, b.progressBar)
			// 	} else {
			// 		r = chr
			// 	}
			// 	opts = rest.Opts{
			// 		Method:               "POST",
			// 		Path:                 "/flow_chunk",
			// 		MultipartParams:      params,
			// 		ContentLength:        &size,
			// 		MultipartContentName: "file",
			// 		MultipartFileName:    path.Base(item.src.Remote()), // TODO: is it?
			// 		Body:                 r,
			// 	}
			// 	if resp, err = b.fs.api.CallJSON(ctx, &opts, nil, nil); err != nil {
			// 		fs.LogPrintf(fs.LogLevelError, b, "call (POST): %v", err)
			// 		return
			// 	}
			// 	if err = resp.Body.Close(); err != nil {
			// 		fs.LogPrintf(fs.LogLevelError, b, "body: %v", err)
			// 		return
			// 	}
			// }
			// if err = chunker.Close(); err != nil {
			// 	fs.LogPrintf(fs.LogLevelError, b, "close: %v", err)
			// 	return
			// }
			// if item.deleteFileAfterTransfer {
			// 	if err = os.Remove(item.filename); err != nil {
			// 		fs.LogPrintf(fs.LogLevelError, b, "remove: %v", err)
			// 		return
			// 	}
			// }
		}
		fs.Logf(b, "upload done (%d), deposited %s, %d item(s)",
			b.depositIdentifier, operations.SizeString(b.totalSize, true), len(b.items))
		return
	})
	return
}

// Upload a single item to vault, possibly in parallel.
func (b *batcher) UploadItem(ctx context.Context, item *batchItem, f *api.File) error {
	if b.depositIdentifier == 0 {
		return ErrMissingDepositIdentifier
	}
	if item == nil || f == nil {
		return nil
	}
	var (
		chunker *Chunker
		j       int64
		resp    *http.Response
		err     error
	)
	if chunker, err = NewChunker(item.filename, b.chunkSize); err != nil {
		return err
	}
	for j = 1; j <= chunker.NumChunks(); j++ {
		currentChunkSize := chunker.ChunkSize(j - 1)
		fs.Debugf(b, "[%d/%d] %d %d %s",
			j,
			chunker.NumChunks(),
			currentChunkSize,
			chunker.FileSize(),
			item.filename,
		)
		params := url.Values{
			"depositId":            []string{strconv.Itoa(int(b.depositIdentifier))},
			"flowChunkNumber":      []string{strconv.Itoa(int(j))},
			"flowChunkSize":        []string{strconv.Itoa(int(b.chunkSize))},
			"flowCurrentChunkSize": []string{strconv.Itoa(int(currentChunkSize))},
			"flowFilename":         []string{f.Name},
			"flowIdentifier":       []string{f.FlowIdentifier},
			"flowRelativePath":     []string{f.RelativePath},
			"flowTotalChunks":      []string{strconv.Itoa(int(chunker.NumChunks()))},
			"flowTotalSize":        []string{strconv.Itoa(int(chunker.FileSize()))},
			"upload_token":         []string{"my_token"}, // TODO(martin): just copy'n'pasting ...
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
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			fs.LogPrintf(fs.LogLevelError, b, "expected HTTP < 300, got: %v", resp.StatusCode)
			err = fmt.Errorf("expected HTTP < 300, got %v", resp.StatusCode)
			return err
		} else {
			fs.Debugf(b, "GET returned: %v", resp.StatusCode)
		}
		var (
			r    io.Reader
			chr  = chunker.ChunkReader(j - 1)
			size = currentChunkSize // size will get mutated during request
		)
		if b.showDepositProgress {
			r = io.TeeReader(chr, b.progressBar)
		} else {
			r = chr
		}
		opts = rest.Opts{
			Method:               "POST",
			Path:                 "/flow_chunk",
			MultipartParams:      params,
			ContentLength:        &size,
			MultipartContentName: "file",
			MultipartFileName:    path.Base(item.src.Remote()), // TODO: is it?
			Body:                 r,
		}
		if resp, err = b.fs.api.CallJSON(ctx, &opts, nil, nil); err != nil {
			fs.LogPrintf(fs.LogLevelError, b, "call (POST): %v", err)
			return err
		}
		if err = resp.Body.Close(); err != nil {
			fs.LogPrintf(fs.LogLevelError, b, "body: %v", err)
			return err
		}
	}
	if err = chunker.Close(); err != nil {
		fs.LogPrintf(fs.LogLevelError, b, "chunker close: %v", err)
		return err
	}
	if item.deleteFileAfterTransfer {
		if err = os.Remove(item.filename); err != nil {
			fs.LogPrintf(fs.LogLevelError, b, "remove: %v", err)
			return err
		}
	}
	return nil
}
