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

func TestTreeNodeMimeType(t *testing.T) {
	var cases = []struct {
		tno      *TreeNode
		expected string
	}{
		{&TreeNode{FileType: nil}, ""},
		{&TreeNode{FileType: "text/plain"}, "text/plain"},
		{&TreeNode{FileType: ""}, ""},
		{&TreeNode{FileType: "any"}, "any"},
	}
	for _, c := range cases {
		if got := c.tno.MimeType(); got != c.expected {
			t.Fatalf("got %v, want %v", got, c.expected)
		}
	}
}

func TestTreeNodeParentTreeNodeIdentifier(t *testing.T) {
	var cases = []struct {
		tno      *TreeNode
		expected string
	}{
		{&TreeNode{Parent: nil}, ""},
		{&TreeNode{Parent: "unchecked"}, "unchecked"},
		{&TreeNode{Parent: "http://invalid"}, ""},
		{&TreeNode{Parent: "http://vault.xyz/api/treenodes/invalid"}, ""},
		{&TreeNode{Parent: "http://vault.xyz/api/treenodes/123"}, "123"},
		{&TreeNode{Parent: "http://vault.xyz/api/treenodes/123X"}, ""},
	}
	for _, c := range cases {
		if got := c.tno.ParentTreeNodeIdentifier(); got != c.expected {
			t.Fatalf("got %v, want %v", got, c.expected)
		}
	}
}

func TestOrganizationPlanIdentifier(t *testing.T) {
	var cases = []struct {
		org      *Organization
		expected string
	}{
		{&Organization{Plan: ""}, ""},
		{&Organization{Plan: "xyz"}, "xyz"},
		{&Organization{Plan: "123"}, "123"},
		{&Organization{TreeNode: "", Plan: "http://vault.xyz/api/plans/123"}, "http://vault.xyz/api/plans/123"},
		{&Organization{TreeNode: "http://vault.xyz/...", Plan: "http://vault.xyz/api/plans/123"}, "123"},
	}
	for _, c := range cases {
		if got := c.org.PlanIdentifier(); got != c.expected {
			t.Fatalf("got %v, want %v", got, c.expected)
		}
	}
}
