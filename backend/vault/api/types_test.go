package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCollectionStatsTotalSize(t *testing.T) {
	cs := &CollectionStats{
		Collections: []struct {
			FileCount int64  `json:"fileCount"`
			ID        int64  `json:"id"`
			Time      string `json:"time"`
			TotalSize int64  `json:"totalSize"`
		}{
			{TotalSize: 1},
			{TotalSize: 2},
		},
	}
	var want int64 = 3
	if cs.TotalSize() != want {
		t.Fatalf("got %v, want %v", cs.TotalSize(), want)
	}
}

func TestCollectionStatsNumFiles(t *testing.T) {
	cs := &CollectionStats{
		Collections: []struct {
			FileCount int64  `json:"fileCount"`
			ID        int64  `json:"id"`
			Time      string `json:"time"`
			TotalSize int64  `json:"totalSize"`
		}{
			{FileCount: 1},
			{FileCount: 2},
		},
	}
	var want int64 = 3
	if cs.NumFiles() != want {
		t.Fatalf("got %v, want %v", cs.NumFiles(), want)
	}
}

func TestTreeNodeContent(t *testing.T) {
	mockData := "hello from ts!"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, mockData)
	}))
	t.Logf("mock treenode content URL: %v", ts.URL)
	tno := &TreeNode{
		ContentURL: ts.URL,
	}
	rc, err := tno.Content(http.DefaultClient, "", nil)
	if err != nil {
		t.Fatalf("could not get content: %v", err)
	}
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	if string(b) != mockData {
		t.Fatalf("got %v, want %v", string(b), mockData)
	}
	defer ts.Close()
}

func TestTreeNodeSize(t *testing.T) {
	var cases = []struct {
		tno          *TreeNode
		expectedSize int64
	}{
		{&TreeNode{ObjectSize: nil}, 0},
		{&TreeNode{ObjectSize: 0}, 0},
		{&TreeNode{ObjectSize: 1}, 1},
		{&TreeNode{ObjectSize: 1024}, 1024},
	}
	for _, c := range cases {
		if got := c.tno.Size(); got != c.expectedSize {
			t.Fatalf("got %v, want %v", got, c.expectedSize)
		}
	}
}
