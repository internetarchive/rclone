package oapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	"github.com/rclone/rclone/backend/vault/api"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/lib/rest"
)

const (
	// VaultVersionHeader as served by vault site.
	VaultVersionHeader = "X-Vault-API-Version"
	// VersionSupported is the version of the vault API this package implements.
	VersionSupported = "1"
	// maxResponseBody limit in bytes when reading a response body.
	maxResponseBody = 1 << 24
)

var (
	// ErrUserNotFound when a username is not registered in vault.
	ErrUserNotFound = errors.New("user not found")
	// ErrAmbiguousQuery when we except 0 or 1 result in the result set, but get more.
	ErrAmbiguousQuery = errors.New("ambiguous query")
)

// Error for failed api requests.
type Error struct {
	err error
}

// Error returns a string.
func (e *Error) Error() string {
	return fmt.Sprintf("api error: %v", e.err)
}

// CompatAPI is a compatibility layer and provides the exact same API to vault
// as the manually written one, but will only use the openapi-generated code
// after some transition period.
//
// Uglyness of two separate clients, a basic HTTP client that is wrapped by
// the OpenAPI client and that does authentication. Plus a legacy API that
// uses a different authentication mechanism and that we keep around for the
// transition period as fallback.
//
// TODO(martin): move all methods to use openapi client only
type CompatAPI struct {
	Endpoint string
	Username string
	Password string
	// VersionSupported by this implementation. This is should checked before
	// any other operation.
	VersionSupported string
	loginPath        string
	c                *http.Client         // vanilly HTTP client, will be wrapped by OpenAPI client
	client           *ClientWithResponses // OpenAPI client
	csrfTokenPattern *regexp.Regexp
	// legacyAPI, so we can replace and test one function at a time
	legacyAPI *api.API
}

func New(endpoint, username, password string) (*CompatAPI, error) {
	// TODO: need at least an HTTP client with cookie setup
	stripped := strings.TrimRight(strings.Replace(endpoint, "/api", "", 1), "/")
	capi := &CompatAPI{
		Endpoint:         endpoint,
		Username:         username,
		Password:         password,
		VersionSupported: VersionSupported,
		loginPath:        "/accounts/login/",
		c:                &http.Client{Timeout: 10 * time.Second},
		csrfTokenPattern: regexp.MustCompile(`csrfToken:[ ]*"([^"]*)"`),
		legacyAPI:        api.New(endpoint, username, password),
	}
	// NewClient wants the URL w/o the "/api" suffix by default.
	client, err := NewClientWithResponses(stripped,
		WithHTTPClient(capi.c),
		WithRequestEditorFn(capi.Intercept))
	if err != nil {
		return nil, err
	}
	capi.client = client
	return capi, nil
}

// Client returns the http client, which will have a session cookie after login.
func (capi *CompatAPI) Client() *http.Client {
	return capi.c
}

// Intercept adds required headers to each request, namely a csrf token and referer.
func (capi *CompatAPI) Intercept(ctx context.Context, req *http.Request) error {
	fs.Debugf(capi, "api CSRF intercept")
	// TODO: need to add cookie jar from capi
	anyLink, err := url.JoinPath(capi.Endpoint, "users") // any valid path will do
	if err != nil {
		return err
	}
	r, err := http.NewRequest("GET", anyLink, nil)
	if err != nil {
		return err
	}
	r.Header.Set("Accept", "text/html")
	resp, err := capi.c.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close() // nolint:errcheck
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if matches := capi.csrfTokenPattern.FindStringSubmatch(string(b)); len(matches) == 2 {
		req.Header.Set("X-CSRFTOKEN", matches[1])
		req.Header.Set("Referer", anyLink)
		return nil
	}
	return fmt.Errorf("could not set csrf token")
}

// Compatibility methods, from vault/api/api.go
// --------------------------------------------

func (capi *CompatAPI) Version(ctx context.Context) string {
	r, err := http.NewRequest("GET", capi.Endpoint, nil)
	if err != nil {
		return ""
	}
	resp, err := capi.c.Do(r)
	if err != nil {
		return ""
	}
	defer resp.Body.Close() // nolint:errcheck
	return resp.Header.Get(VaultVersionHeader)
}
func (capi *CompatAPI) String() string {
	return fmt.Sprintf("vault (v%s compat)", api.VersionSupported)
}

// Login equips the HTTP client with a session cookie.
//
// Need to setup the cookie jar for the HTTP client as well as the cookie for
// the legacy client.
func (capi *CompatAPI) Login() error {
	if err := capi.legacyAPI.Login(); err != nil {
		return err
	}
	var (
		u   *url.URL
		b   []byte
		err error
	)
	if u, err = url.Parse(capi.Endpoint); err != nil {
		return err
	}
	u.Path = strings.Replace(u.Path, "/api", capi.loginPath, 1)
	loginPath := u.String()
	resp, err := http.Get(loginPath)
	if err != nil {
		return fmt.Errorf("cannot access login url: %w", err)
	}
	defer resp.Body.Close() // nolint:errcheck
	// Parse out the CSRF token: <input type="hidden"
	// name="csrfmiddlewaretoken"
	// value="CCBQ9qqG3ylgR1MaYBc6UCw4tlxR7rhP2Qs4uvIMAf1h7Dd4xtv5azTQJRgJ1y2I">
	doc, err := htmlquery.Parse(resp.Body)
	if err != nil {
		return fmt.Errorf("html: %w", err)
	}
	token := htmlquery.SelectAttr(
		htmlquery.FindOne(doc, `//input[@name="csrfmiddlewaretoken"]`),
		"value",
	)
	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	// Need to reparse, api may live on a different path.
	u, err = url.Parse(capi.Endpoint)
	if err != nil {
		return err
	}
	jar.SetCookies(u, []*http.Cookie{&http.Cookie{
		Name:  "csrftoken",
		Value: token,
	}})
	capi.c.Jar = jar
	data := url.Values{}
	data.Set("username", capi.Username)
	data.Set("password", capi.Password)
	data.Set("csrfmiddlewaretoken", token)
	req, err := http.NewRequest("POST", loginPath, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// You are seeing this message because this HTTPS site requires a "Referer
	// header" to be sent by your Web browser, but none was sent. This header
	// is required for security reasons, to ensure that your browser is not
	// being hijacked by third parties.
	req.Header.Set("Referer", loginPath)
	resp, err = capi.c.Do(req)
	if err != nil {
		return fmt.Errorf("vault login: %w", err)
	}
	defer resp.Body.Close() // nolint:errcheck
	if resp.StatusCode >= 400 {
		b, _ = ioutil.ReadAll(resp.Body)
		return fmt.Errorf("login failed with: %v (%s)", resp.StatusCode, string(b))
	}
	b, _ = httputil.DumpResponse(resp, true)
	if bytes.Contains(b, []byte(`Your username and password didn't match`)) {
		return fmt.Errorf("username and password did not match")
	}
	if len(jar.Cookies(u)) < 2 {
		msg := fmt.Sprintf("expected 2 cookies, got %v", len(jar.Cookies(u)))
		return fmt.Errorf(msg)
	}
	for i, c := range capi.c.Jar.Cookies(u) {
		fs.Debugf(capi, "cookie #%d: %v", i, c)
	}
	return nil
}

// Logout drops the session.
func (capi *CompatAPI) Logout() error {
	capi.legacyAPI.Logout()
	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	capi.c.Jar = jar
	return nil
}

func (capi *CompatAPI) Call(ctx context.Context, opts *rest.Opts) (*http.Response, error) {
	return capi.legacyAPI.Call(ctx, opts)
}
func (capi *CompatAPI) CallJSON(ctx context.Context, opts *rest.Opts, req, resp interface{}) (*http.Response, error) {
	return capi.legacyAPI.CallJSON(ctx, opts, req, resp)
}

// SplitPath turns an absolute path string into a PathInfo value.
func (capi *CompatAPI) SplitPath(p string) (*api.PathInfo, error) {
	return capi.legacyAPI.SplitPath(p)
}

// ResolvePath turns a path string into a treenode.
func (capi *CompatAPI) ResolvePath(p string) (*api.TreeNode, error) {
	return capi.legacyAPI.ResolvePath(p)
}
func (capi *CompatAPI) DepositStatus(id int64) (*api.DepositStatus, error) {
	return capi.legacyAPI.DepositStatus(id)
}
func (capi *CompatAPI) CreateCollection(ctx context.Context, name string) error {
	return capi.legacyAPI.CreateCollection(ctx, name)
}
func (capi *CompatAPI) CreateFolder(ctx context.Context, parent *api.TreeNode, name string) error {
	return capi.legacyAPI.CreateFolder(ctx, parent, name)
}
func (capi *CompatAPI) SetModTime(ctx context.Context, t *api.TreeNode) error {
	return capi.legacyAPI.SetModTime(ctx, t)
}
func (capi *CompatAPI) Rename(ctx context.Context, t *api.TreeNode, name string) error {
	return capi.legacyAPI.Rename(ctx, t, name)
}
func (capi *CompatAPI) Move(ctx context.Context, t, newParent *api.TreeNode) error {
	return capi.legacyAPI.Move(ctx, t, newParent)
}
func (capi *CompatAPI) Remove(ctx context.Context, t *api.TreeNode) error {
	return capi.legacyAPI.Remove(ctx, t)
}
func (capi *CompatAPI) List(t *api.TreeNode) ([]*api.TreeNode, error) {
	return capi.legacyAPI.List(t)
}
func (capi *CompatAPI) RegisterDeposit(ctx context.Context, rdr *api.RegisterDepositRequest) (id int64, err error) {
	return capi.legacyAPI.RegisterDeposit(ctx, rdr)
}
func (capi *CompatAPI) TreeNodeToCollection(t *api.TreeNode) (*api.Collection, error) {
	return capi.legacyAPI.TreeNodeToCollection(t)
}
func (capi *CompatAPI) GetCollectionStats() (*api.CollectionStats, error) {
	return capi.legacyAPI.GetCollectionStats()
}
func (capi *CompatAPI) FindCollections(vs url.Values) ([]*api.Collection, error) {
	return capi.legacyAPI.FindCollections(vs)
}
func (capi *CompatAPI) FindTreeNodes(vs url.Values) ([]*api.TreeNode, error) {
	return capi.legacyAPI.FindTreeNodes(vs)
}

// User returns the current user. This is an example of using the new API internally.
func (capi *CompatAPI) User() (*api.User, error) {
	ctx := context.Background()
	params := &UsersListParams{
		Username: &capi.Username,
	}
	r, err := capi.client.UsersListWithResponse(ctx, params)
	if err != nil {
		return nil, err
	}
	if r.StatusCode() != 200 {
		return nil, fmt.Errorf("got HTTP %d status from API", r.StatusCode())
	}
	if *r.JSON200.Count == 0 {
		return nil, fmt.Errorf("user not found: %s", capi.Username)
	}
	if *r.JSON200.Count > 1 {
		return nil, fmt.Errorf("ambiguous query")
	}
	// Translate API response to legacy user type.
	usr := (*r.JSON200.Results)[0]
	return &api.User{
		DateJoined:   usr.DateJoined.Format(time.RFC3339),
		FirstName:    *usr.FirstName,
		IsActive:     *usr.IsActive,
		IsStaff:      *usr.IsStaff,
		IsSuperuser:  *usr.IsSuperuser,
		LastLogin:    usr.LastLogin.Format(time.RFC3339),
		LastName:     *usr.LastName,
		Organization: *usr.Organization,
		URL:          *usr.Url,
		Username:     usr.Username,
	}, nil
}

// Organization returns the organization of the current user.
func (capi *CompatAPI) Organization() (*api.Organization, error) {
	ctx := context.Background()
	user, err := capi.User()
	if err != nil {
		return nil, err
	}
	sid := user.OrganizationIdentifier()
	id, err := strconv.Atoi(sid)
	if err != nil {
		return nil, err
	}
	r, err := capi.client.OrganizationsRetrieveWithResponse(ctx, id)
	if err != nil {
		return nil, err
	}
	if r.StatusCode() != 200 {
		return nil, fmt.Errorf("error retrieving organization: %v", r.StatusCode())
	}
	return &api.Organization{
		Name:       r.JSON200.Name,
		Plan:       r.JSON200.Plan,
		QuotaBytes: *r.JSON200.QuotaBytes,
		TreeNode:   *r.JSON200.TreeNode,
		URL:        *r.JSON200.Url,
	}, nil
}

// Plan returns the plan of the current user.
func (capi *CompatAPI) Plan() (*api.Plan, error) {
	ctx := context.Background()
	org, err := capi.Organization()
	if err != nil {
		return nil, err
	}
	pid := org.PlanIdentifier()
	id, err := strconv.Atoi(pid)
	if err != nil {
		return nil, err
	}
	r, err := capi.client.PlansRetrieveWithResponse(ctx, id)
	if r.StatusCode() != 200 {
		return nil, fmt.Errorf("error retrieving plan: %v", r.StatusCode())
	}
	return &api.Plan{
		DefaultFixityFrequency: string(*r.JSON200.DefaultFixityFrequency),
		DefaultGeolocations:    r.JSON200.DefaultGeolocations,
		DefaultReplication:     int64(*r.JSON200.DefaultReplication),
		Name:                   r.JSON200.Name,
		PricePerTerabyte:       r.JSON200.PricePerTerabyte,
		URL:                    *r.JSON200.Url,
	}, nil
}
