package vault

import (
	"testing"

	"github.com/rclone/rclone/fstest/fstests"
)

// TestIntegration runs integration tests against the remote
func TestIntegration(t *testing.T) {
	// TODO: Setup fresh vault, e.g. with testcontainers.
	fstests.Run(t, &fstests.Opt{
		RemoteName: "VaultTest:",
		NilObject:  (*Object)(nil),
	})
}
