package oapi

import "testing"

func TestCompatClient(t *testing.T) {
	client, err := New("http://localhost:8000/api", "admin", "admin")
	if err != nil {
		t.Fatalf("could not setup client: %v", err)
	}
	t.Logf("compat client: %v", client)
}
