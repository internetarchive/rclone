package vault

import (
	"context"
	"reflect"
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
