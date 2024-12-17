package vault

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"testing"

	"github.com/rclone/rclone/backend/vault/api"
	"github.com/rclone/rclone/backend/vault/oapi"
	"github.com/rclone/rclone/fstest/fstests"
)

const (
	testEndpoint = "http://localhost:8000/api"
	testUsername = "admin"
	testPassword = "admin"
)

// TestIntegration runs integration tests against the remote. This is a set of
// test supplied by rclone, of which we still fail a lot.
//
//	$ VAULT_TEST_REMOTE_NAME=v: go test -v ./backend/vault/...
func TestIntegration(t *testing.T) {
	// t.Skip("skipping integration tests temporarily")
	var remoteName string
	if v := os.Getenv("VAULT_TEST_REMOTE_NAME"); v != "" {
		remoteName = v
	} else {
		t.Skip("VAULT_TEST_REMOTE_NAME env not set, skipping")
	}
	// TODO(martin): collection (top level dirs) cannot be deleted, but that
	// leads to failing tests; fix this.
	fstests.Run(t, &fstests.Opt{
		RemoteName:               remoteName,
		NilObject:                (*Object)(nil),
		SkipBadWindowsCharacters: true,
		SkipDirectoryCheckWrap:   true,
		SkipFsCheckWrap:          true,
		SkipInvalidUTF8:          true,
	})
}

// randomName returns a name that can be used for files, directories and
// collections.
func randomName(tag string) string {
	return fmt.Sprintf("%s-%024d", tag, rand.Int63())
}

// mustLogin returns an authenticated client.
func mustLogin(t *testing.T) *oapi.CompatAPI {
	api, err := oapi.New(testEndpoint, testUsername, testPassword)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if err = api.Login(); err != nil {
		t.Fatalf("login failed: %v", err)
	}
	return api
}

// mustCollection creates and returns a collection with a given name.
func mustCollection(t *testing.T, api *oapi.CompatAPI, name string) *api.Collection {
	ctx := context.Background()
	err := api.CreateCollection(ctx, name)
	if err != nil {
		t.Fatalf("failed to create collection: %v", err)
	}
	t.Logf("created collection %v", name)
	vs := url.Values{}
	vs.Set("name", name)
	result, err := api.FindCollections(vs)
	if err != nil {
		t.Fatalf("failed to query collections: %v, %v", result, err)
	}
	if len(result) != 1 {
		t.Fatalf("expected a single result, got %v", len(result))
	}
	return result[0]
}

func mustTreeNodeForCollection(t *testing.T, api *oapi.CompatAPI, c *api.Collection) *api.TreeNode {
	vs := url.Values{}
	vs.Set("id", fmt.Sprintf("%d", c.TreeNodeIdentifier()))
	t.Logf("finding treenode: %v", c.TreeNodeIdentifier())
	ts, err := api.FindTreeNodes(vs)
	if err != nil {
		t.Fatalf("failed to get treenode: %v", err)
	}
	if len(ts) != 1 {
		t.Fatalf("expected single result, got %v", len(ts))
	}
	return ts[0]
}

// TestCreateCollection tests collection creation.
func TestCreateCollection(t *testing.T) {
	var (
		api  = mustLogin(t)
		name = randomName("test-collection")
	)
	_ = mustCollection(t, api, name)
	t.Logf("created collection: %v", name)
}

func TestCreateFolder(t *testing.T) {
	var (
		ctx            = context.Background()
		api            = mustLogin(t)
		collectionName = randomName("test-collection")
		collection     = mustCollection(t, api, collectionName)
		treeNode       = mustTreeNodeForCollection(t, api, collection)
		folderName     = randomName("test-folder")
	)
	err := api.CreateFolder(ctx, treeNode, folderName)
	if err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}
	t.Logf("created collection and folder: %v/%v", collectionName, folderName)
}

func TestRegisterDeposit(t *testing.T) {
	t.Skip("obsolete")
}

func TestDeposit(t *testing.T)      {}
func TestFileRename(t *testing.T)   {}
func TestFileMove(t *testing.T)     {}
func TestFolderRename(t *testing.T) {}
func TestFolderMove(t *testing.T)   {}

// TODO:
//
// [ ] register deposit
// [ ] upload file
// [ ] rename file
// [ ] move file
// [ ] move folder
// [ ] delete folder

// FROM: VAULT_TEST_REMOTE_NAME=vo: go test -v ./backend/vault/...
//
// --- FAIL: TestIntegration (357.02s)
//     --- SKIP: TestIntegration/FsCheckWrap (0.00s)
//     --- SKIP: TestIntegration/FsCommand (0.00s)
//     --- PASS: TestIntegration/FsRmdirNotFound (0.57s)
//     --- PASS: TestIntegration/FsString (0.00s)
//     --- PASS: TestIntegration/FsName (0.00s)
//     --- PASS: TestIntegration/FsRoot (0.00s)
//     --- PASS: TestIntegration/FsRmdirEmpty (0.43s)
//     --- FAIL: TestIntegration/FsMkdir (351.98s)
//         --- PASS: TestIntegration/FsMkdir/FsMkdirRmdirSubdir (7.18s)
//         --- PASS: TestIntegration/FsMkdir/FsListEmpty (0.37s)
//         --- PASS: TestIntegration/FsMkdir/FsListDirEmpty (0.39s)
//         --- SKIP: TestIntegration/FsMkdir/FsListRDirEmpty (0.00s)
//         --- PASS: TestIntegration/FsMkdir/FsListDirNotFound (0.30s)
//         --- SKIP: TestIntegration/FsMkdir/FsListRDirNotFound (0.00s)
//         --- FAIL: TestIntegration/FsMkdir/FsEncoding (327.39s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/control_chars (4.36s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/dot (4.10s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/dot_dot (4.06s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/punctuation (3.82s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_space (6.58s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_tilde (22.93s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_CR (28.49s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_LF (33.09s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_HT (21.46s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_VT (22.07s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_dot (22.64s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_space (6.37s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_CR (22.76s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_LF (22.12s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_HT (27.66s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_VT (29.99s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_dot (21.99s)
//             --- SKIP: TestIntegration/FsMkdir/FsEncoding/invalid_UTF-8 (0.00s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/URL_encoding (22.43s)
//         --- PASS: TestIntegration/FsMkdir/FsNewObjectNotFound (0.79s)
//         --- PASS: TestIntegration/FsMkdir/FsPutError (0.32s)
//         --- FAIL: TestIntegration/FsMkdir/FsPutZeroLength (0.78s)
//         --- SKIP: TestIntegration/FsMkdir/FsOpenWriterAt (0.00s)
//         --- SKIP: TestIntegration/FsMkdir/FsOpenChunkWriter (0.00s)
//         --- SKIP: TestIntegration/FsMkdir/FsChangeNotify (0.00s)
//         --- FAIL: TestIntegration/FsMkdir/FsPutFiles (7.79s)
//         --- SKIP: TestIntegration/FsMkdir/FsPutChunked (0.00s)
//         --- SKIP: TestIntegration/FsMkdir/FsCopyChunked (0.00s)
//         --- FAIL: TestIntegration/FsMkdir/FsUploadUnknownSize (1.98s)
//             --- FAIL: TestIntegration/FsMkdir/FsUploadUnknownSize/FsPutUnknownSize (0.45s)
//             --- FAIL: TestIntegration/FsMkdir/FsUploadUnknownSize/FsUpdateUnknownSize (1.54s)
//         --- PASS: TestIntegration/FsMkdir/FsRootCollapse (3.33s)
//         --- SKIP: TestIntegration/FsMkdir/FsDirSetModTime (0.00s)
//         --- SKIP: TestIntegration/FsMkdir/FsMkdirMetadata (0.00s)
//         --- SKIP: TestIntegration/FsMkdir/FsDirectory (0.00s)
//     --- PASS: TestIntegration/FsShutdown (0.09s)
