package vault

import (
	"context"
	"errors"
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
		SkipFsCheckWrap:          true,
		SkipInvalidUTF8:          true,
		SkipBadWindowsCharacters: true,
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
		t.Fatalf("failed to query collections: %v", result)
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
	var (
		vapi = mustLogin(t)
		ctx  = context.Background()
	)
	// errCases are cases that should yield an api.Error of sorts
	var errCases = []struct {
		help string
		rdr  *api.RegisterDepositRequest
	}{
		{
			"empty request",
			&api.RegisterDepositRequest{},
		},
		{
			"incomplete, collection id only",
			&api.RegisterDepositRequest{CollectionID: 123},
		},
		{
			"invalid collection ids",
			&api.RegisterDepositRequest{
				CollectionID: 1234567,
				Files:        nil,
				ParentNodeID: 1,
				TotalSize:    0,
			},
		},
		{
			"invalid collection ids",
			&api.RegisterDepositRequest{
				CollectionID: 1234567,
				Files:        nil,
				ParentNodeID: 1,
				TotalSize:    0,
			},
		},
	}
	// Test various incomplete register deposit requests.
	var apiError *api.Error
	for _, c := range errCases {
		_, err := vapi.RegisterDeposit(ctx, c.rdr)
		if !errors.As(err, &apiError) {
			t.Fatalf("register failed [%s]: got %v, want an api error", c.help, err)
		}
	}
	// var okCases = []struct {
	// 	help string
	// 	rdr  *api.RegisterDepositRequest
	// }{
	// 	{
	// 		"complete request",
	// 		&api.RegisterDepositRequest{
	// 			CollectionID: 1,
	// 			Files:        nil,
	// 			ParentNodeID: 1,
	// 			TotalSize:    0,
	// 		},
	// 	},
	// }
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
// --- FAIL: TestIntegration (292.63s)
//     --- SKIP: TestIntegration/FsCheckWrap (0.00s)
//     --- SKIP: TestIntegration/FsCommand (0.00s)
//     --- PASS: TestIntegration/FsRmdirNotFound (0.54s)
//     --- PASS: TestIntegration/FsString (0.00s)
//     --- PASS: TestIntegration/FsName (0.00s)
//     --- PASS: TestIntegration/FsRoot (0.00s)
//     --- PASS: TestIntegration/FsRmdirEmpty (0.42s)
//     --- FAIL: TestIntegration/FsMkdir (289.13s)
//         --- PASS: TestIntegration/FsMkdir/FsMkdirRmdirSubdir (5.93s)
//         --- PASS: TestIntegration/FsMkdir/FsListEmpty (0.39s)
//         --- PASS: TestIntegration/FsMkdir/FsListDirEmpty (0.46s)
//         --- SKIP: TestIntegration/FsMkdir/FsListRDirEmpty (0.00s)
//         --- PASS: TestIntegration/FsMkdir/FsListDirNotFound (0.35s)
//         --- SKIP: TestIntegration/FsMkdir/FsListRDirNotFound (0.00s)
//         --- FAIL: TestIntegration/FsMkdir/FsEncoding (263.99s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/control_chars (2.36s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/dot (2.34s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/dot_dot (2.18s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/punctuation (2.29s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_space (5.82s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_tilde (20.21s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_CR (20.04s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_LF (19.52s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_HT (18.95s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_VT (20.17s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_dot (19.48s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_space (5.95s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_CR (20.73s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_LF (20.20s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_HT (20.86s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_VT (20.86s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_dot (20.42s)
//             --- SKIP: TestIntegration/FsMkdir/FsEncoding/invalid_UTF-8 (0.00s)
//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/URL_encoding (21.11s)
//         --- PASS: TestIntegration/FsMkdir/FsNewObjectNotFound (0.51s)
//         --- PASS: TestIntegration/FsMkdir/FsPutError (0.28s)
//         --- FAIL: TestIntegration/FsMkdir/FsPutZeroLength (5.42s)
//         --- SKIP: TestIntegration/FsMkdir/FsOpenWriterAt (0.00s)
//         --- SKIP: TestIntegration/FsMkdir/FsOpenChunkWriter (0.00s)
//         --- SKIP: TestIntegration/FsMkdir/FsChangeNotify (0.00s)
//         --- FAIL: TestIntegration/FsMkdir/FsPutFiles (7.20s)
//         --- SKIP: TestIntegration/FsMkdir/FsPutChunked (0.00s)
//         --- SKIP: TestIntegration/FsMkdir/FsCopyChunked (0.00s)
//         --- FAIL: TestIntegration/FsMkdir/FsUploadUnknownSize (1.20s)
//             --- FAIL: TestIntegration/FsMkdir/FsUploadUnknownSize/FsPutUnknownSize (0.34s)
//             --- FAIL: TestIntegration/FsMkdir/FsUploadUnknownSize/FsUpdateUnknownSize (0.86s)
//         --- PASS: TestIntegration/FsMkdir/FsRootCollapse (2.18s)
//     --- PASS: TestIntegration/FsShutdown (0.09s)
