package vault

import (
	"context"

	v2 "github.com/rclone/rclone/backend/vault/v2" // deposits/v2
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
)

const (
	// Note: the biggest increase in upload throughput so far came from
	// increasing the chunk size to 16M.
	//
	//  1M/1/1:  5M/s
	// 16M/1/1: 15M/s
	// 16M/2/2: 20M/s
	//
	// Target two-core QA machine was occassionally maxed out, not sure if
	// that's imposing a limit.
	//
	// ----8<----
	//
	// As of 05/2023, we reduced the default chunk size to 1M, temporarily
	// after encountering some issues.
	//
	// ----8<----
	//
	// @martin I'm just remembering to ask about this, but could you fill me in
	// on the rationale for decreasing the chunk sizes that rclone sends to
	// vault?
	//
	// yes, happily; in short: I was getting a HTTP 404 from chunk upload and I
	// believe it came from this line:
	// vault-site:19e168e41489e2278bb59c659489344857a04f4e/vault/deposit_api.py#L193-199
	// (not too many 404's possible otherwise) -- now the deposit is registered
	// successfully, so the deposit's state must have changed (to something
	// other than "REGISTERED") -- and that (only) happens, when a deposit is
	// "completed" -- which happens e.g. here:
	// vault-site:19e168e41489e2278bb59c659489344857a04f4e/vault/asynchronous/temporal/deposit/workflow.py#L62-66
	// -- so my wild guess ATM is that we encounter some break here:
	// vault-site:19e168e41489e2278bb59c659489344857a04f4e/vault/asynchronous/temporal/deposit/workflow.py#L24-60
	// -- and that causes the deposit to be "completed" early, which then causes
	// a 404 when trying to upload a chunk somehow, lowering the chunksize made
	// this problem go away, but there's probably a better way to handle this --
	// would be glad to have a short in person debug session (where we can try
	// to replicate the issue in prod together, or the like)
	defaultUploadChunkSize = 1 << 20 // 1M
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
			{
				Name:     "chunk_size",
				Help:     "Upload chunk size in bytes (limited)",
				Default:  defaultUploadChunkSize,
				Advanced: true,
			},
		},
	})
}

// NewFS sets up a new filesystem for vault.
func NewFs(ctx context.Context, name, root string, m configmap.Mapper) (fs.Fs, error) {
	return v2.NewFs(ctx, name, root, m)
}
