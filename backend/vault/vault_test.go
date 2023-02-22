package vault

import (
	"context"
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

// TestCreateCollection tests collection creation.
func TestCreateCollection(t *testing.T) {
	api := api.New("http://localhost:8000/api", "admin", "admin")
	err := api.Login()
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	ctx := context.Background()
	name := fmt.Sprintf("test-create-collection-%024d", rand.Int63())
	err = api.CreateCollection(ctx, name)
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
}

func TestCreateFolder(t *testing.T) {
	api := api.New("http://localhost:8000/api", "admin", "admin")
	err := api.Login()
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	ctx := context.Background()
	// create collection
	name := fmt.Sprintf("test-create-collection-%024d", rand.Int63())
	err = api.CreateCollection(ctx, name)
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
	// find treenode for collection
	vs = url.Values{}
	vs.Set("id", fmt.Sprintf("%d", result[0].TreeNodeIdentifier()))
	t.Logf("finding treenode: %v", result[0].TreeNodeIdentifier())
	ts, err := api.FindTreeNodes(vs)
	if err != nil {
		t.Fatalf("failed to get treenode: %v", err)
	}
	if len(ts) != 1 {
		t.Fatalf("expected single result, got %v", len(ts))
	}
	// create folder
	folder := fmt.Sprintf("test-folder-%024d", rand.Int63())
	err = api.CreateFolder(ctx, ts[0], folder)
	if err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}
	t.Logf("created collection and folder")
}

// TODO:
//
// [ ] register deposit
// [ ] upload file
// [ ] rename file
// [ ] move file
// [ ] move folder
// [ ] delete folder
