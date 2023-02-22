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
	"github.com/rclone/rclone/fstest/fstests"
)

// TestIntegration runs integration tests against the remote. This is a set of
// test supplied by rclone, of which we still fail a lot.
//
//	$ VAULT_TEST_REMOTE_NAME=v: go test -v ./backend/vault/...
func TestIntegration(t *testing.T) {
	t.Skip("skipping integration tests temporarily")
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

func MustLogin(t *testing.T) *api.API {
	api := api.New("http://localhost:8000/api", "admin", "admin")
	err := api.Login()
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	return api
}

func MustCollection(t *testing.T, api *api.API, name string) *api.Collection {
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

// TestCreateCollection tests collection creation.
func TestCreateCollection(t *testing.T) {
	var (
		api  = MustLogin(t)
		name = fmt.Sprintf("test-create-collection-%024d", rand.Int63())
	)
	_ = MustCollection(t, api, name)
}

func MustTreeNodeForCollection(t *testing.T, api *api.API, c *api.Collection) *api.TreeNode {
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

func TestCreateFolder(t *testing.T) {
	var (
		api        = MustLogin(t)
		ctx        = context.Background()
		name       = fmt.Sprintf("test-create-collection-%024d", rand.Int63())
		collection = MustCollection(t, api, name)
		treeNode   = MustTreeNodeForCollection(t, api, collection)
		folder     = fmt.Sprintf("test-folder-%024d", rand.Int63())
	)
	err := api.CreateFolder(ctx, treeNode, folder)
	if err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}
	t.Logf("created collection and folder")
}

func TestRegisterDeposit(t *testing.T) {
	var (
		vapi = MustLogin(t)
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
	}
	// Test various incomplete register deposit requests.
	var apiError *api.Error
	for _, c := range errCases {
		_, err := vapi.RegisterDeposit(ctx, c.rdr)
		if !errors.As(err, &apiError) {
			t.Fatalf("register failed [%s]: got %v, want an api error", c.help, err)
		}
	}

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
