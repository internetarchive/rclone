package vault

import (
	"context"

	v2 "github.com/rclone/rclone/backend/vault/v2" // deposits/v2
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
)

// NewFS sets up a new filesystem for vault.
func NewFs(ctx context.Context, name, root string, m configmap.Mapper) (fs.Fs, error) {
	return v2.NewFs(ctx, name, root, m)
}
