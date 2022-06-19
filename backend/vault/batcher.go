package vault

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/rclone/rclone/backend/vault/api"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/operations"
	"github.com/rclone/rclone/lib/atexit"
	"github.com/rclone/rclone/lib/rest"
	"github.com/schollz/progressbar/v3"
)

// batcher is used to group files for a deposit.
type batcher struct {
	fs                  *Fs             // fs.root will be the parent collection or folder
	atexit              atexit.FnHandle // callback
	parent              *api.TreeNode   // resolved and possibly new parent treenode
	shutOnce            sync.Once       // only batch wrap up once
	mu                  sync.Mutex      // protect items
	items               []*batchItem    // file metadata and content for deposit items
	showDepositProgress bool            // show progress bar
}

// newBatcher creates a new batcher, which will execute most code at rclone
// exit time. Note: this will create the fs.Root if it does not exist, hence
// newBatcher should only be called, if we are actually performing a batch
// operation (e.g. in Put). We run "mkdir" here and not in the atexit handler,
// since here we can still return errors (which we currently cannot in the
// atexit handler).
func newBatcher(ctx context.Context, f *Fs) (*batcher, error) {
	t, err := f.api.ResolvePath(f.root)
	if err != nil {
		if err == fs.ErrorObjectNotFound {
			if err = f.mkdir(ctx, f.root); err != nil {
				return nil, err
			}
			if t, err = f.api.ResolvePath(f.root); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	b := &batcher{
		fs:     f,
		parent: t,
	}
	b.atexit = atexit.Register(b.Shutdown)
	// When interrupted, we may have items in the batch, clean these up before
	// the shutdown handler runs.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			b.mu.Lock()
			if len(b.items) > 0 {
				fs.Logf(b, "ignoring %d batch entries", len(b.items))
			}
			b.items = b.items[:0]
			b.mu.Unlock()
		}
	}()
	fs.Debugf(b, "initialized batcher")
	return b, nil
}

// batchItem for Put and Update requests, basically capturing those methods' arguments.
type batchItem struct {
	root     string // the fs root
	filename string // some temporary file with contents
	src      fs.ObjectInfo
	options  []fs.OpenOption
}

// ToFile turns a batch item into a File for a deposit request.
func (item *batchItem) ToFile(ctx context.Context) *api.File {
	var (
		randInt        = 1_000_000_000 + rand.Intn(8_999_999_999) // fixed length
		randSuffix     = fmt.Sprintf("%s-%d", time.Now().Format("20060102030405"), randInt)
		flowIdentifier = fmt.Sprintf("rclone-vault-flow-%s", randSuffix)
	)
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

// String will most likely show up in debug messages.
func (b *batcher) String() string {
	return "vault batcher"
}

// Add a single item to the batch.
func (b *batcher) Add(item *batchItem) {
	b.mu.Lock()
	b.items = append(b.items, item)
	b.mu.Unlock()
}

// Shutdown creates a new deposit request for all batch items and uploads them.
// This is the one of the last things rclone run before exiting. There is no
// way to relay an error to return from here, so we deliberately exit the
// process from here with an exit code of 1, if anything fails.
func (b *batcher) Shutdown() {
	fs.Debugf(b, "shutdown started")
	b.shutOnce.Do(func() {
		atexit.Unregister(b.atexit)
		signal.Reset(os.Interrupt)
		if len(b.items) == 0 {
			fs.Debugf(b, "nothing to deposit")
			return
		}
		var (
			// We do not want to be cancelled in Shutdown; or if we do, we want
			// to set our own timeout for deposit uploads.
			ctx             = context.Background()
			totalSize int64 = 0
			files     []*api.File
			bar       *progressbar.ProgressBar
		)
		fs.Debugf(b, "preparing %d files for deposit", len(b.items))
		for _, item := range b.items {
			totalSize += item.src.Size()
			files = append(files, item.ToFile(ctx))
		}
		// TODO: we may want to reuse a deposit to continue an interrupted deposit
		rdr := &api.RegisterDepositRequest{
			TotalSize: totalSize,
			Files:     files,
		}
		switch {
		case b.parent.NodeType == "COLLECTION":
			c, err := b.fs.api.TreeNodeToCollection(b.parent)
			if err != nil {
				log.Fatalf("failed to resolve treenode to collection")
			}
			rdr.CollectionId = c.Identifier()
		case b.parent.NodeType == "FOLDER":
			rdr.ParentNodeId = b.parent.Id
		}
		depositId, err := b.fs.api.RegisterDeposit(ctx, rdr)
		if err != nil {
			log.Fatalf("deposit failed: %v", err)
		}
		fs.Debugf(b, "created deposit %v", depositId)
		if b.showDepositProgress {
			bar = progressbar.DefaultBytes(totalSize, "<5>NOTICE: depositing")
		}
		for i, item := range b.items {
			// Upload file with a single chunk. First issue a GET, if that is a
			// 204 then follow up with a POST.
			fi, err := os.Stat(item.filename)
			if err != nil {
				log.Fatalf("stat: %v", err)
			}
			size := fi.Size()
			params := url.Values{
				"depositId":            []string{strconv.Itoa(int(depositId))},
				"flowChunkNumber":      []string{"1"},
				"flowChunkSize":        []string{strconv.Itoa(int(size))},
				"flowCurrentChunkSize": []string{strconv.Itoa(int(size))},
				"flowFilename":         []string{files[i].Name},
				"flowIdentifier":       []string{files[i].FlowIdentifier},
				"flowRelativePath":     []string{files[i].RelativePath},
				"flowTotalChunks":      []string{"1"},
				"flowTotalSize":        []string{strconv.Itoa(int(size))},
				"upload_token":         []string{"my_token"}, // just copy'n'pasting ...
			}
			opts := rest.Opts{
				Method:     "GET",
				Path:       "/flow_chunk",
				Parameters: params,
			}
			resp, err := b.fs.api.Call(ctx, &opts)
			if err != nil {
				log.Fatalf("call failed: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 204 {
				log.Fatalf("expected HTTP 204, got %v", resp.StatusCode)
			}
			itemf, err := os.Open(item.filename)
			if err != nil {
				log.Fatalf("failed to open temporary file: %v", err)
			}
			r := io.TeeReader(itemf, bar)
			opts = rest.Opts{
				Method:               "POST",
				Path:                 "/flow_chunk",
				MultipartParams:      params,
				ContentLength:        &size,
				MultipartContentName: "file",
				MultipartFileName:    path.Base(item.src.Remote()),
				Body:                 r,
			}
			resp, err = b.fs.api.CallJSON(ctx, &opts, nil, nil)
			if err != nil {
				log.Fatalf("upload failed: %v", err)
			}
			if err := resp.Body.Close(); err != nil {
				log.Fatalf("body close: %v", err)
			}
			if err := itemf.Close(); err != nil {
				log.Fatalf("file close: %v", err)
			}
			if err := os.Remove(item.filename); err != nil {
				log.Fatalf("cleanup: %v", err)
			}
		}
		fs.Logf(b, "upload done, deposited %s, %d item(s)",
			operations.SizeString(totalSize, true), len(b.items))
	})
}
