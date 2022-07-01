package vault

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/rclone/rclone/backend/vault/api"
	"github.com/rclone/rclone/backend/vault/extra"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/config/configstruct"
	"github.com/rclone/rclone/fs/hash"
)

var (
	ErrNotImplemented  = errors.New("not implemented")
	ErrVersionMismatch = errors.New("api version mismatch")
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
		},
	})
}

// NewFS sets up a new filesystem.
func NewFs(ctx context.Context, name, root string, m configmap.Mapper) (fs.Fs, error) {
	var opt Options
	err := configstruct.Set(m, &opt)
	if err != nil {
		return nil, err
	}
	api := api.New(opt.EndpointNormalized(), opt.Username, opt.Password)
	if err := api.Login(); err != nil {
		return nil, err
	}
	if v := api.Version(ctx); v != "" && v != api.VersionSupported {
		return nil, ErrVersionMismatch
	}
	f := &Fs{
		name: name,
		root: root,
		opt:  opt,
		api:  api,
	}
	f.features = (&fs.Features{
		CaseInsensitive:         true,
		CanHaveEmptyDirectories: true,
		ReadMimeType:            true,
		PublicLink:              f.PublicLink,
		About:                   f.About,
		PutStream:               f.PutStream,
		UserInfo:                f.UserInfo,
		Disconnect:              f.Disconnect,
		DirMove:                 f.DirMove,
		Purge:                   f.Purge,
		Shutdown:                f.Shutdown,
	}).Fill(ctx, f)
	return f, nil
}

// Options for vault.
type Options struct {
	Username string `config:"username"`
	Password string `config:"password"`
	Endpoint string `config:"endpoint"`
}

// EndpointNormalized returns a normalized endpoint. We currently want no trailing slash.
func (opt Options) EndpointNormalized() string {
	return strings.TrimRight(opt.Endpoint, "/")
}

// Fs is the main vault filesystem. Most operations are accessed through the
// api. A batch helper is required to model the deposit-style upload of a
// potentially large set of files.
type Fs struct {
	name     string
	root     string
	opt      Options
	api      *api.Api     // vault api wrapper
	features *fs.Features // optional features
	mu       sync.Mutex   // protect batcher
	batcher  *batcher     // batching for deposits
}

// Fs Info
// -------

func (f *Fs) Name() string             { return f.name }
func (f *Fs) Root() string             { return f.root }
func (f *Fs) String() string           { return f.name }
func (f *Fs) Precision() time.Duration { return 1 * time.Second }
func (f *Fs) Hashes() hash.Set         { return hash.Set(hash.MD5 | hash.SHA1 | hash.SHA256) }
func (f *Fs) Features() *fs.Features   { return f.features }

// Fs Ops
// ------

// List the objects and directories in dir into entries.  The
// entries can be returned in any order but should be for a
// complete directory.
//
// dir should be "" to list the root, and should not have
// trailing slashes.
//
// This should return ErrDirNotFound if the directory isn't
// found.
func (f *Fs) List(ctx context.Context, dir string) (fs.DirEntries, error) {
	fs.Debugf(f, "list dir %v", dir)
	var (
		entries fs.DirEntries
		absPath = f.absPath(dir)
	)
	t, err := f.api.ResolvePath(absPath)
	if err != nil {
		if dir == "" && err == fs.ErrorObjectNotFound {
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

// Put uploads a new object. This does not upload content immediately, but save
// the source in a temporary file and add the file to the batcher, which will
// upload at rclone exit time.
func (f *Fs) Put(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) (fs.Object, error) {
	fs.Debugf(f, "put %v [%v]", src.Remote(), src.Size())
	var (
		filename string
		err      error
	)
	fs.Debugf(f, "fetching remote file temporarily")
	if filename, err = extra.TempFileFromReader(in); err != nil {
		return nil, err
	}
	f.mu.Lock()
	fs.Debugf(f, "fetched %v to %v", src.Remote(), filename)
	if f.batcher == nil {
		f.batcher, err = newBatcher(ctx, f)
		if err != nil {
			return nil, err
		}
		f.batcher.showDepositProgress = true
	}
	f.mu.Unlock()
	// TODO: with retries, we may add the same object twice or more; check that
	// each batch contains unique elements
	f.batcher.Add(&batchItem{
		root:     f.root,
		filename: filename,
		src:      src,
		options:  options,
	})
	return &Object{
		fs:     f,
		remote: src.Remote(),
		treeNode: &api.TreeNode{
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
	case t != nil && t.NodeType == "FOLDER":
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

// Rmdir deletes a folder.
func (f *Fs) Rmdir(ctx context.Context, dir string) error {
	t, err := f.api.ResolvePath(f.absPath(dir))
	if err != nil {
		return err
	}
	if t.NodeType != "FOLDER" {
		return fmt.Errorf("can only drop folders, not %v", t.NodeType)
	}
	return f.api.Remove(ctx, t)
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
		return v, nil
	default:
		return "", fmt.Errorf("link not available for treenode %v", t.Id)
	}
}

// About returns currently only the quota.
func (f *Fs) About(ctx context.Context) (*fs.Usage, error) {
	organization, err := f.api.Organization()
	if err != nil {
		return nil, err
	}
	stats, err := f.api.GetCollectionStats()
	if err != nil {
		return nil, err
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
	if srcDirParentNode.Id == dstDirParentNode.Id {
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

// Shutdown triggers the deposit upload.
func (f *Fs) Shutdown(ctx context.Context) error {
	fs.Debugf(f, "shutdown")
	if f.batcher != nil {
		return f.batcher.Shutdown()
	}
	return nil
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
		}
	case hash.SHA1:
		if v, ok := o.treeNode.Sha1Sum.(string); ok {
			return v, nil
		}
	case hash.SHA256:
		if v, ok := o.treeNode.Sha256Sum.(string); ok {
			return v, nil
		}
	}
	// TODO: we may want hash.ErrUnsupported, but we get an err, via:
	// https://github.com/rclone/rclone/blob/c85fbebce6f7166350c79e11fae763c8264ef865/fs/operations/operations.go#L105
	return "", nil
}
func (o *Object) Storable() bool { return true }

// Object Ops
// ----------

func (o *Object) SetModTime(ctx context.Context, t time.Time) error {
	fs.Debugf(o, "noop: set mod time")
	return nil
}
func (o *Object) Open(ctx context.Context, options ...fs.OpenOption) (io.ReadCloser, error) {
	fs.Debugf(o, "reading object contents")
	return o.treeNode.Content(options...)
}
func (o *Object) Update(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) error {
	fs.Debugf(o, "noop: update")
	return nil
}
func (o *Object) Remove(ctx context.Context) error {
	return o.fs.api.Remove(ctx, o.treeNode)
}

// Object extra
// ------------

func (o *Object) MimeType(ctx context.Context) string {
	return o.treeNode.MimeType()
}

// ID returns treenode path, which should be unique for any object in vault.
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
	_ fs.DirMover     = (*Fs)(nil)
	_ fs.Fs           = (*Fs)(nil)
	_ fs.PublicLinker = (*Fs)(nil)
	_ fs.PutStreamer  = (*Fs)(nil)
	_ fs.Shutdowner   = (*Fs)(nil)
	_ fs.UserInfoer   = (*Fs)(nil)
	_ fs.Disconnecter = (*Fs)(nil)
	_ fs.MimeTyper    = (*Object)(nil)
	_ fs.Object       = (*Object)(nil)
	_ fs.IDer         = (*Object)(nil)
	_ fs.Directory    = (*Dir)(nil)
)
