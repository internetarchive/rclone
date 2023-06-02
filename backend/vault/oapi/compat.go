package oapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	"github.com/rclone/rclone/backend/vault/api"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/lib/rest"
)

// TODO(martin): use oapi generated code, not legacyAPI
//
// * [ ] keep only types from the API (e.g. for signatures)
// * [ ] remove all code from manual API except for the types
// * [ ] start to rewrite client code in terms of the new API
// * [ ] move legacy api types to generated types
// * [ ] once client side code uses only new API constructs, delete manual API completely
//

const (
	// VaultVersionHeader as served by vault site.
	VaultVersionHeader = "X-Vault-API-Version"
	// VersionSupported is the version of the vault API this package implements.
	VersionSupported = "2"
	// maxResponseBody limit in bytes when reading a response body.
	maxResponseBody = 1 << 24
)

var (
	// ErrUserNotFound when a username is not registered in vault.
	ErrUserNotFound = errors.New("user not found")
	// ErrAmbiguousQuery when we except 0 or 1 result in the result set, but get more.
	ErrAmbiguousQuery = errors.New("ambiguous query")
	// ErrMissingCSRFToken may occur, if site structure changes
	ErrMissingCSRFToken = errors.New("missing CSRF token")
	// VaultRcloneUserAgentString set the User-Agent string (for most requests)
	VaultRcloneUserAgentString = fmt.Sprintf("rclone/%s (vault-api v%s)", fs.Version, VersionSupported)
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
	// c is a vanilla http.Client for now, will be wrapped by
	// deepmap/oapi-codegen generated client.  On login, we need to set cookies
	// on the HTTP client, that's why we need to keep this around separately
	// from the higher-level client. We need the cookies as well, hence not
	// just the Doer interface.
	c *http.Client
	// client is an initialized client, wrapping vault endpoints.
	client *ClientWithResponses // OpenAPI client
	// csrfTokenPattern is how we find tokens in the HTML to supply any
	// operation. It would best, if we would not need this at all, but we do.
	// We use Django REST Framework (DRF), and SessionAuthentication; [...] "if
	// you're using SessionAuthentication you'll need to include valid CSRF
	// tokens for any POST, PUT, PATCH or DELETE operations" (DRF docs).
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
		// TODO: using vanilla client for now, but could upgrade to pester or something else
		c:                &http.Client{Timeout: 30 * time.Second},
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

// Intercept adds required headers to each request, namely a csrf token and
// referer. Some vault endpoints are exempt from CSRF, but that's not reflected
// here at the moment.
func (capi *CompatAPI) Intercept(ctx context.Context, req *http.Request) error {
	req.Header.Set("User-Agent", VaultRcloneUserAgentString)
	fs.Debugf(capi, "api CSRF intercept")
	// previously, we used api/collections or api/users, etc - but we don't get
	// any HTML back from resource endpoints; but just .../api works
	anyLink := capi.Endpoint
	fs.Debugf(capi, "using referer: %v", anyLink)
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
		fs.Debugf(capi, "set header: %v", req.Header)
		return nil
	}
	return ErrMissingCSRFToken
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
	// TODO: "deposit_status" is not covered by openapi schema
	// var body bytes.Buffer
	// err := json.NewEncoder(&body).Encode(struct {
	// 	ID string `json:"deposit_id"`
	// }{
	// 	ID: fmt.Sprintf("%v", id),
	// })
	// if err != nil {
	// 	return nil, err
	// }
	// req, err := http.NewRequest("GET", path.Join(capi.Endpoint, "deposit_status"), &body)
	// if err != nil {
	// 	return nil, err
	// }
	// resp, err := capi.c.Do(req)
	// if err != nil {
	// 	return nil, err
	// }
	// defer resp.Body.Close()
	// if resp.StatusCode >= 400 {
	// 	return nil, fmt.Errorf("deposit status: got %v", resp.StatusCode)
	// }
	// var ds api.DepositStatus
	// if err := json.NewDecoder(resp.Body).Decode(&ds); err != nil {
	// 	return nil, err
	// }
	// return &ds, nil
}

func (capi *CompatAPI) CreateCollection(ctx context.Context, name string) error {
	body := CollectionsCreateJSONRequestBody{
		Name: name,
	}
	resp, err := capi.client.CollectionsCreate(ctx, body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		b, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return err
		}
		fs.Debugf(capi, "create collection: got http %v: %s", resp.StatusCode, string(b))
		return fmt.Errorf("create collection: got http %v", resp.StatusCode)
	}
	return nil
}

func (capi *CompatAPI) CreateFolder(ctx context.Context, parent *api.TreeNode, name string) error {
	var (
		nodeType  = NodeTypeEnumFOLDER
		parentURL = parent.URL
	)
	body := TreenodesCreateJSONRequestBody{
		Name:     name,
		NodeType: &nodeType,
		Parent:   &parentURL,
	}
	resp, err := capi.client.TreenodesCreate(ctx, body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		b, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return err
		}
		fs.Debugf(capi, "create folder: got http %v: %s", resp.StatusCode, string(b))
		return fmt.Errorf("create folder: got http %v", resp.StatusCode)
	}
	return nil
}

// SetModTime is not naturally supported by vault. We do not alter the mod time for now.
func (capi *CompatAPI) SetModTime(ctx context.Context, t *api.TreeNode) error {
	// TODO: There may be an effect by doing a pseudo-change like setting the
	// name to name, and have "updated_at" reflect that.
	fs.Debugf(capi, "not changing immutable treenode.modified_at")
	return nil
}

// Rename a treenode.
func (capi *CompatAPI) Rename(ctx context.Context, t *api.TreeNode, name string) error {
	fs.Debugf(capi, "rename")
	var (
		payload = struct {
			Name string `json:"name"`
		}{name}
		buf bytes.Buffer
	)
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		return err
	}
	resp, err := capi.client.TreenodesPartialUpdateWithBody(
		ctx, int(t.ID), "application/json", &buf)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		b, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return err
		}
		fs.Debugf(capi, "move: got http %v: %s", resp.StatusCode, string(b))
		return fmt.Errorf("move: got http %v", resp.StatusCode)
	}
	return nil
}

func (capi *CompatAPI) Move(ctx context.Context, t, newParent *api.TreeNode) error {
	fs.Debugf(capi, "move %v => %v", t.Path, newParent.Path)
	// Payload is a minimal struct, not the generated PatchedTreeNodeRequest.
	// Reason is a mismatch in nullable field handling.
	//
	// When using PatchedTreeNodeRequest, we got:
	//
	// "Immutable field(s) can not be updated: {'pre_deposit_modified_at',
	// 'uploaded_at', 'size', 'file_type', 'comment', 'md5_sum',
	// 'sha1_sum', 'sha256_sum'}"
	//
	// The issue here is that "PatchedTreeNodeRequest" spec says "nullable",
	// which omits the "omitempty" tag, which in turn sends all fields in the
	// request (albeit being 'null'), which in turn results in a "immutable
	// fields can not be updated"
	//
	// There was no issue, when we just sent the one field to patch.
	//
	// For illustration, the last struct we tried.
	//
	//   body = PatchedTreeNodeRequest{
	//   	// Name:                 &node.Name,
	//   	// NodeType:             node.NodeType,
	//   	// PreDepositModifiedAt: node.PreDepositModifiedAt,
	//   	// UploadedAt:           node.UploadedAt,
	//   	// Size:                 node.Size,
	//   	// FileType:             node.FileType,
	//   	// Comment:              node.Comment,
	//   	// Md5Sum:               node.Md5Sum,
	//   	// Sha1Sum:              node.Sha1Sum,
	//   	// Sha256Sum:            node.Sha256Sum,
	//   	Parent: &parent,
	//   }
	payload := struct {
		Parent string `json:"parent"`
	}{
		newParent.URL,
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		return err
	}
	resp, err := capi.client.TreenodesPartialUpdateWithBody(
		ctx, int(t.ID), "application/json", &buf)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		b, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return err
		}
		fs.Debugf(capi, "move: got http %v: %s", resp.StatusCode, string(b))
		return fmt.Errorf("move: got http %v", resp.StatusCode)
	}
	return nil
}

func (capi *CompatAPI) Remove(ctx context.Context, t *api.TreeNode) error {
	resp, err := capi.client.TreenodesDestroy(ctx, int(t.ID))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("remove: got http %v", resp.StatusCode)
	}
	return nil
}

func (capi *CompatAPI) List(t *api.TreeNode) (result []*api.TreeNode, err error) {
	// TODO: this was the previous implementation; below is the OAPI generated
	// variant; to be used going forward
	// result, err = capi.legacyAPI.List(t)
	// TODO: legacyAPI had cache, which add noticable improvement
	var (
		ctx    = context.Background()
		parent = int(t.ID)
		limit  = 5000 // TODO: to match previous limit, may exceed some payload size
		params = &TreenodesListParams{
			Parent: &parent,
			Limit:  &limit,
		}
		resp *TreenodesListResponse
	)
	if resp, err = capi.client.TreenodesListWithResponse(ctx, params); err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, err
	}
	return toLegacyTreeNodes(resp.JSON200.Results), nil
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

// FindCollections returns a list of collections, typically given a treenode identifier.
func (capi *CompatAPI) FindCollections(vs url.Values) (result []*api.Collection, err error) {
	var (
		ctx    = context.Background()
		limit  = 5000 // TODO: switch to proper pagination
		params = &CollectionsListParams{
			Limit: &limit,
		}
		resp *CollectionsListResponse
	)
	for k, v := range vs {
		switch k {
		case "tree_node":
			i, err := strconv.Atoi(v[0])
			if err != nil {
				return nil, err
			}
			params.TreeNode = &i
		default:
			return nil, fmt.Errorf("compat missing legacy parameters: %v", k)
		}
	}
	if resp, err = capi.client.CollectionsListWithResponse(ctx, params); err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("collections: got http %v", resp.StatusCode())
	}
	return toLegacyCollection(resp.JSON200.Results), nil
}

// FindTreeNodes returns a list of treenodes given query parameters. We only
// deal with fields that we previously used. Anything else will fail noticably.
func (capi *CompatAPI) FindTreeNodes(vs url.Values) (result []*api.TreeNode, err error) {
	var (
		ctx    = context.Background()
		limit  = 5000 // TODO: to match previous limit, may exceed some payload size
		params = &TreenodesListParams{
			Limit: &limit,
		}
		resp *TreenodesListResponse
	)
	for k, v := range vs {
		// We only ever used "parent" and "name" as parameter. If we use
		// something else, we can err out.
		switch k {
		case "parent":
			if len(v) > 0 {
				i, err := strconv.Atoi(v[0])
				if err != nil {
					return nil, err
				}
				params.Parent = &i
			}
		case "name":
			if len(v) > 0 {
				params.Name = &v[0]
			}
		default:
			return nil, fmt.Errorf("compat missing legacy parameter: %v", k)
		}
	}
	if resp, err = capi.client.TreenodesListWithResponse(ctx, params); err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("treenode: got http %v", resp.StatusCode())
	}
	result = toLegacyTreeNodes(resp.JSON200.Results)
	return result, nil
}

// User returns the current user. This is an example of using the new API internally.
func (capi *CompatAPI) User() (*api.User, error) {
	// TODO: use cache
	ctx := context.Background()
	limit := 1
	params := &UsersListParams{
		Username: &capi.Username,
		Limit:    &limit,
	}
	r, err := capi.client.UsersListWithResponse(ctx, params)
	if err != nil {
		return nil, err
	}
	if r.StatusCode() != 200 {
		return nil, fmt.Errorf("user: got http %d", r.StatusCode())
	}
	if *r.JSON200.Count == 0 {
		return nil, fmt.Errorf("user not found: %s", capi.Username)
	}
	if *r.JSON200.Count > 1 {
		return nil, fmt.Errorf("ambiguous query")
	}
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
	org := r.JSON200
	return &api.Organization{
		Name:       org.Name,
		Plan:       org.Plan,
		QuotaBytes: *org.QuotaBytes,
		TreeNode:   *org.TreeNode,
		URL:        *org.Url,
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

// root returns the organization treenode for the current API user.
func (capi *CompatAPI) root() (*api.TreeNode, error) {
	organization, err := capi.Organization()
	if err != nil {
		return nil, err
	}
	id, err := strconv.Atoi(organization.TreeNodeIdentifier())
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	resp, err := capi.client.TreenodesRetrieveWithResponse(ctx, id)
	if err != nil {
		return nil, err
	}
	return toLegacyTreeNode(resp.JSON200), nil
}

// safeTimeFormat return a formatted time or the empty string.
func safeTimeFormat(t *time.Time, layout string) string {
	if t == nil {
		return ""
	}
	return t.Format(layout)
}

// safeDereference unwraps a pointer value. Either returns the dereferenced
// value or nil.
func safeDereference(ptr interface{}) interface{} {
	if ptr == nil {
		return nil
	}
	value := reflect.ValueOf(ptr)
	if value.Kind() != reflect.Ptr {
		return ptr
	}
	if value.IsNil() {
		return nil
	}
	return value.Elem().Interface()
}
