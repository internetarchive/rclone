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
	defaultUploadChunkSize    = 1 << 24 // 16M
	defaultMaxParallelChunks  = 2
	defaultMaxParallelUploads = 2
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
				Name:    "suppress_progress_bar",
				Help:    "Suppress deposit progress bar",
				Default: false,
				Hide:    fs.OptionHideConfigurator,
			},
			{
				Name:    "resume_deposit_id",
				Help:    "Resume a deposit",
				Default: 0,
				Hide:    fs.OptionHideConfigurator,
			},
			{
				Name:     "chunk_size",
				Help:     "Upload chunk size in bytes (limited)",
				Default:  defaultUploadChunkSize,
				Advanced: true,
			},
		},
		CommandHelp: []fs.CommandHelp{
			fs.CommandHelp{
				Name:  "status",
				Short: "show deposit status",
				Long: `Display status of deposit, pass deposit id (e.g. 742) as argument, e.g.:

    $ rclone backend ds vault: 742

Will return a JSON like this:

    {
      "assembled_files": 6,
      "errored_files": 0,
      "file_queue": 0,
      "in_storage_files": 0,
      "total_files": 6,
      "uploaded_files": 0
    }
`,
			},
		},
	})
}

// NewFS sets up a new filesystem for vault.
func NewFs(ctx context.Context, name, root string, m configmap.Mapper) (fs.Fs, error) {
	return v2.NewFs(ctx, name, root, m)
}
