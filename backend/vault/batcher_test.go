package vault

import (
	"context"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/rclone/rclone/backend/vault/api"
)

func TestBatchItemToFile(t *testing.T) {
	var cases = []struct {
		item   *batchItem
		result *api.File
	}{
		{nil, nil},
		{&batchItem{}, nil},
		{
			&batchItem{
				root:     "/",
				filename: "a",
				src: &Object{
					treeNode: &api.TreeNode{},
				},
			}, &api.File{
				FlowIdentifier:       "rclone-vault-flow-6666cd76f96956469e7be39d750cc7d9",
				Name:                 ".",
				PreDepositModifiedAt: time.Unix(0, 0).Format("2006-01-02T03:04:05.000Z"),
			},
		},
		{
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
	}
	rand.Seed(0)
	ctx := context.TODO()
	for _, c := range cases {
		result := c.item.ToFile(ctx)
		if !reflect.DeepEqual(result, c.result) {
			t.Errorf("got %#v, want %#v", result, c.result)
		}
	}
}
