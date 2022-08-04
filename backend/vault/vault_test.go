package vault

import (
	"os"
	"testing"

	"github.com/rclone/rclone/fstest/fstests"
)

// TestIntegration runs integration tests against the remote. Run:
//
//   $ VAULT_TEST_REMOTE_NAME=v: go test -v ./backend/vault/...
//
func TestIntegration(t *testing.T) {
	// TODO: Setup fresh vault, e.g. with testcontainers.
	remoteName := "VaultTest:"
	if v := os.Getenv("VAULT_TEST_REMOTE_NAME"); v != "" {
		remoteName = v
	}
	// TODO(martin): collection (top level dirs) cannot be deleted, but that
	// leads to failing tests; fix this.
	fstests.Run(t, &fstests.Opt{
		RemoteName:      remoteName,
		NilObject:       (*Object)(nil),
		SkipFsCheckWrap: true,
		SkipInvalidUTF8: true,
	})
}
