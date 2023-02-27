package oapi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/antchfx/htmlquery"
	"github.com/rclone/rclone/fs/fshttp"
	"github.com/rclone/rclone/lib/rest"
)

// authenticatedClient returns an authenticated HTTP client. Plucked out of v1
// api client, for testing only.
func authenticatedClient(endpoint, username, password string) (HttpRequestDoer, error) {
	var (
		ctx       = context.Background()
		loginPath = "/accounts/login/"
		client    = rest.NewClient(fshttp.NewClient(ctx)).SetRoot(endpoint)
		u         *url.URL
		b         []byte
		err       error
	)
	if u, err = url.Parse(endpoint); err != nil {
		return nil, err
	}
	u.Path = strings.Replace(u.Path, "/api", loginPath, 1)
	loginPath = u.String()
	log.Printf("loginPath: %v", loginPath)
	resp, err := http.Get(loginPath)
	if err != nil {
		return nil, fmt.Errorf("cannot access login url: %w", err)
	}
	defer resp.Body.Close() // nolint:errcheck
	// Parse out the CSRF token: <input type="hidden"
	// name="csrfmiddlewaretoken"
	// value="CCBQ9qqG3ylgR1MaYBc6UCw4tlxR7rhP2Qs4uvIMAf1h7Dd4xtv5azTQJRgJ1y2I">
	//
	// TODO: move to a token based auth for the API:
	// https://stackoverflow.com/q/21317899/89391
	doc, err := htmlquery.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("html: %w", err)
	}
	token := htmlquery.SelectAttr(
		htmlquery.FindOne(doc, `//input[@name="csrfmiddlewaretoken"]`),
		"value",
	)
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	// Need to reparse, api may live on a different path.
	u, err = url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	jar.SetCookies(u, []*http.Cookie{&http.Cookie{
		Name:  "csrftoken",
		Value: token,
	}})
	cc := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
	}
	// We could use PostForm, but we need to set extra headers.
	data := url.Values{}
	data.Set("username", username)
	data.Set("password", password)
	data.Set("csrfmiddlewaretoken", token)
	req, err := http.NewRequest("POST", loginPath, strings.NewReader(data.Encode()))
	if err != nil {
		return cc, fmt.Errorf("login failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// You are seeing this message because this HTTPS site requires a "Referer
	// header" to be sent by your Web browser, but none was sent. This header
	// is required for security reasons, to ensure that your browser is not
	// being hijacked by third parties.
	req.Header.Set("Referer", loginPath)
	resp, err = cc.Do(req)
	if err != nil {
		return cc, fmt.Errorf("vault login: %w", err)
	}
	defer resp.Body.Close() // nolint:errcheck
	if resp.StatusCode >= 400 {
		b, _ = ioutil.ReadAll(resp.Body)
		return cc, fmt.Errorf("login failed with: %v (%s)", resp.StatusCode, string(b))
	}
	b, _ = httputil.DumpResponse(resp, true)
	if bytes.Contains(b, []byte(`Your username and password didn't match`)) {
		return cc, fmt.Errorf("username and password did not match")
	}
	log.Println(string(b))
	if len(jar.Cookies(u)) < 2 {
		msg := fmt.Sprintf("expected 2 cookies, got %v", len(jar.Cookies(u)))
		return cc, fmt.Errorf(msg)
	}
	client.SetCookie(jar.Cookies(u)...)
	return cc, nil
}

// vaultSecurityProvider injects a referer and a csrf token into every request.
// Needs a authenticated client.
func createSecurityProviderFunc(client HttpRequestDoer) func(context.Context, *http.Request) error {
	return func(ctx context.Context, req *http.Request) error {
		link := "http://localhost:8000/api/users"
		pageRequest, err := http.NewRequest("GET", link, nil)
		if err != nil {
			return err
		}
		pageRequest.Header.Set("Accept", "text/html")
		resp, err := client.Do(pageRequest)
		if err != nil {
			return err
		}
		defer resp.Body.Close() // nolint:errcheck
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		re := regexp.MustCompile(`csrfToken:[ ]*"([^"]*)"`)
		if matches := re.FindStringSubmatch(string(b)); len(matches) == 2 {
			req.Header.Set("X-CSRFTOKEN", matches[1])
			req.Header.Set("Referer", link)
			return nil
		}
		return fmt.Errorf("could not set csrf token")
	}
}

func TestNewClient(t *testing.T) {
	ctx := context.Background()
	c, err := authenticatedClient("http://localhost:8000/api", "admin", "admin")
	if err != nil {
		t.Fatalf("auth failed: %v", err)
	}
	t.Logf("auth ok: %v", c)
	// c := pester.New()
	// TODO: we need to setup our security provider, for csrf
	client, err := NewClient("http://localhost:8000/",
		WithHTTPClient(c),
		WithRequestEditorFn(createSecurityProviderFunc(c)))
	if err != nil {
		t.Fatalf("client failed: %v", client)
	}
	req := CollectionRequest{
		Name: "sample",
	}
	resp, err := client.CollectionsCreate(ctx, req)
	if err != nil {
		t.Fatalf("client failed: %v", err)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("client failed: %v", err)
	}
	defer resp.Body.Close()
	t.Logf("api: %v", resp.StatusCode)
	t.Logf("body: %v", string(b))
}
