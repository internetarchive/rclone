package vault

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/rclone/rclone/backend/vault/api"
)

func TestBatchItemToFile(t *testing.T) {
	var cases = []struct {
		about  string
		item   *batchItem
		result *api.File
	}{
		{"nil item becomes nil", nil, nil},
		{"empty list becomes nil", &batchItem{}, nil},
		{
			"item an empty treenode",
			&batchItem{
				root:     "/",
				filename: "a",
				src: &Object{
					treeNode: &api.TreeNode{},
				},
			}, &api.File{
				FlowIdentifier:       "rclone-vault-flow-6666cd76f96956469e7be39d750cc7d9",
				Name:                 ".", // https://go.dev/play/p/PPSzc9GO4EJ
				PreDepositModifiedAt: time.Unix(0, 0).Format("2006-01-02T03:04:05.000Z"),
			},
		},
		{
			"item with a treenode",
			&batchItem{
				root:     "/",
				filename: "abc",
				src: &Object{
					treeNode: &api.TreeNode{
						Name: "abc",
					},
				},
			}, &api.File{
				FlowIdentifier:       "rclone-vault-flow-6666cd76f96956469e7be39d750cc7d9",
				Name:                 ".",
				PreDepositModifiedAt: time.Unix(0, 0).Format("2006-01-02T03:04:05.000Z"),
			},
		},
		{
			"item with a treenode with a path",
			&batchItem{
				root:     "/",
				filename: "abc",
				src: &Object{
					treeNode: &api.TreeNode{
						Name: "abc",
					},
					remote: "/a/b/c",
				},
			}, &api.File{
				FlowIdentifier:       "rclone-vault-flow-ea70257d5391fd2af2fbf70b1156dc19",
				Name:                 "c",
				RelativePath:         "/a/b/c",
				PreDepositModifiedAt: time.Unix(0, 0).Format("2006-01-02T03:04:05.000Z"),
			},
		},
	}
	ctx := context.TODO()
	for _, c := range cases {
		result := c.item.ToFile(ctx)
		if !reflect.DeepEqual(result, c.result) {
			t.Errorf("got %#v, want %#v", result, c.result)
		}
	}
}

func TestBatchItemContentType(t *testing.T) {
	var cases = []struct {
		about string
		item  *batchItem
		want  string // content-type
	}{
		{"nil result", nil, ""},
		{"empty item", &batchItem{}, ""},
		{"gzip", &batchItem{
			filename: mustWriteFileTemp([]byte{0x1f, 0x8b, 0x08}),
		}, "application/x-gzip"},
		{"zip", &batchItem{
			filename: mustWriteFileTemp([]byte{0x50, 0x4b, 0x03, 0x04}),
		}, "application/zip"},
		{"png", &batchItem{
			filename: mustWriteFileTemp([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}),
		}, "image/png"},
	}
	for _, c := range cases {
		got := c.item.contentType()
		if got != c.want {
			t.Fatalf("got %v, want %v", got, c.want)
		}
	}
}

func TestBatchItemDeriveFlowIdentifier(t *testing.T) {
	var cases = []struct {
		about string
		item  *batchItem
		want  string
		err   error
	}{
		{"nil item", nil, "", nil},
		{"empty item", &batchItem{}, "", nil},
		{"basic item", &batchItem{
			root: "/",
			src:  &Object{},
		}, "rclone-vault-flow-6666cd76f96956469e7be39d750cc7d9", nil},
		{"basic item", &batchItem{
			root: "/",
			src: &Object{
				remote: "abc",
			},
		}, "rclone-vault-flow-482a7143ac747eff5e5a5992a6016d65", nil},
	}
	for _, c := range cases {
		got, err := c.item.deriveFlowIdentifier()
		if c.err != err {
			t.Fatalf("got %v, want %v", err, c.err)
		}
		if got != c.want {
			t.Fatalf("got %v, want %v", got, c.want)
		}
	}
}

func TestBatcherAdd(t *testing.T) {
	b := &batcher{}
	var wg sync.WaitGroup
	N := 128
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			item := &batchItem{
				filename: fmt.Sprintf("%d.file", i),
			}
			b.Add(item)
		}(i)
	}
	wg.Wait()
	if len(b.items) != N {
		t.Fatalf("got %v, want %v", len(b.items), N)
	}
}

// mustWriteFile writes a temporary file and panics, if that fails. Returns the
// path to the temporary file.
func mustWriteFileTemp(data []byte) (filename string) {
	f, err := os.CreateTemp(os.TempDir(), "rclone-vault-test-*")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := os.WriteFile(f.Name(), data, 0644); err != nil {
		panic(err)
	}
	return f.Name()
}
