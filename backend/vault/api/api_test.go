package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApiVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("R: %v", r.URL.String())
		w.Header().Set("X-Vault-API-Version", "3")
	}))
	api := New(ts.URL, "xxx", "xxx")
	t.Logf("api: %v", api)
	t.Logf("version: %v", api.Version(context.Background()))
	// XXX: follow up
}
