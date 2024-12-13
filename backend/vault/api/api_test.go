package api

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestApiVersion(t *testing.T) {
	var want = "3"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("R: %v", r.URL.String())
		w.Header().Set("X-Vault-API-Version", want)
	}))
	api := New(ts.URL, "xxx", "xxx")
	ctx := context.Background()
	t.Logf("api: %v", api)
	t.Logf("version: %v", api.Version(context.Background()))
	if got := api.Version(ctx); got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestApiLogin(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("R: %v", r.URL.String())
	}))
	client := ts.Client()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cannot create cookie jar")
	}
	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("cannot parse url")
	}
	jar.SetCookies(u, []*http.Cookie{&http.Cookie{
		Name:  "csrftoken",
		Value: "some-token-1234",
	},
		&http.Cookie{
			Name:  "session",
			Value: "some-session-id",
		},
	})
	client.Jar = jar
	api := New(ts.URL, "xxx", "xxx")
	err = api.Login()
	t.Logf("login err: %v", err)
	// TODO: check for cookies
}
