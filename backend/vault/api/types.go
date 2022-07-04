// TODO(martin): pagination
//
package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/rclone/rclone/backend/vault/cache"
	"github.com/rclone/rclone/backend/vault/extra"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/lib/rest"
)

// defaultLimit is the limit used for queries againts rest API. We currently do
// not implement pagination, so we try to get all results at once.
const defaultLimit = "10000"

// Organization represents a single document.
type Organization struct {
	Name       string `json:"name"`
	Plan       string `json:"plan"`
	QuotaBytes int64  `json:"quota_bytes"`
	TreeNode   string `json:"tree_node"`
	URL        string `json:"url"`
}

// User represents a single user.
type User struct {
	DateJoined   string `json:"date_joined"`
	FirstName    string `json:"first_name"`
	IsActive     bool   `json:"is_active"`
	IsStaff      bool   `json:"is_staff"`
	IsSuperuser  bool   `json:"is_superuser"`
	LastLogin    string `json:"last_login"`
	LastName     string `json:"last_name"`
	Organization string `json:"organization"`
	URL          string `json:"url"`
	Username     string `json:"username"`
}

// Collection payload.
type Collection struct {
	FixityFrequency    string `json:"fixity_frequency"`
	Name               string `json:"name"`
	Organization       string `json:"organization"`
	TargetGeolocations []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"target_geolocations"`
	TargetReplication int64  `json:"target_replication"`
	TreeNode          string `json:"tree_node"`
	URL               string `json:"url"` // http://127.0.0.1:8000/api/collections/1/
}

// TreeNode is node in the filesystem tree.
type TreeNode struct {
	Comment              interface{} `json:"comment"`
	ContentURL           interface{} `json:"content_url"`
	FileType             interface{} `json:"file_type"`
	Id                   int64       `json:"id"`
	Md5Sum               interface{} `json:"md5_sum"`
	ModifiedAt           string      `json:"modified_at"`
	Name                 string      `json:"name"`
	NodeType             string      `json:"node_type"`
	Parent               interface{} `json:"parent"`
	Path                 string      `json:"path"`
	PreDepositModifiedAt string      `json:"pre_deposit_modified_at"`
	Sha1Sum              interface{} `json:"sha1_sum"`
	Sha256Sum            interface{} `json:"sha256_sum"`
	ObjectSize           interface{} `json:"size"`
	UploadedAt           string      `json:"uploaded_at"`
	UploadedBy           interface{} `json:"uploaded_by"`
	URL                  string      `json:"url"`
}

// DepositStatus response data.
type DepositStatus struct {
	AssembledFiles int64 `json:"assembled_files"`
	ErroredFiles   int64 `json:"errored_files"`
	FileQueue      int64 `json:"file_queue"`
	InStorageFiles int64 `json:"in_storage_files"`
	TotalFiles     int64 `json:"total_files"`
	UploadedFiles  int64 `json:"uploaded_files"`
}

// Auxiliary structures
// --------------------

// File passed e.g. in deposit requests.
type File struct {
	FlowIdentifier       string `json:"flow_identifier"`
	Name                 string `json:"name"`
	PreDepositModifiedAt string `json:"pre_deposit_modified_at"` // e.g. 2018-04-13T08:06:48.000Z
	RelativePath         string `json:"relative_path"`
	Size                 int64  `json:"size"`
	Type                 string `json:"type"`
}

// RegisterDepositRequest payload.
type RegisterDepositRequest struct {
	CollectionId int64   `json:"collection_id,omitempty"`
	Files        []*File `json:"files"`
	ParentNodeId int64   `json:"parent_node_id,omitempty"`
	TotalSize    int64   `json:"total_size"`
}

// RegisterDepositResponse is the response to a successful RegisterDepositRequest.
type RegisterDepositResponse struct {
	ID int64 `json:"deposit_id"`
}

// PathInfo can be obtained from an absolute path.
type PathInfo struct {
	CollectionTreeNode *TreeNode
	LeafTreeNode       *TreeNode
	RelativePath       string
}

// FlowChunk is used to post data to the server, like FlowChunkGet or
// FlowChunkPost in vault.
type FlowChunk struct {
	DepositIdentifier string
	FileIdentifier    string
	Filename          string
	RelativePath      string
	Number            int
	Size              int64
	TotalSize         int64
	NumChunks         int
	TargetChunkSize   int64
}

// CollectionStats from api/collections_stats.
type CollectionStats struct {
	Collections []struct {
		FileCount int64  `json:"fileCount"`
		Id        int64  `json:"id"`
		Time      string `json:"time"`
		TotalSize int64  `json:"totalSize"`
	} `json:"collections"`
	LatestReport struct {
	} `json:"latestReport"`
}

// Plan associated with an account.
type Plan struct {
	DefaultFixityFrequency string   `json:"default_fixity_frequency"`
	DefaultGeolocations    []string `json:"default_geolocations"`
	DefaultReplication     int64    `json:"default_replication"`
	Name                   string   `json:"name"`
	PricePerTerabyte       string   `json:"price_per_terabyte"`
	Url                    string   `json:"url"`
}

// Helper methods
// --------------

// TotalSize returns the sum of total collection sizes.
func (stats *CollectionStats) TotalSize() (result int64) {
	for _, c := range stats.Collections {
		result += c.TotalSize
	}
	return
}

// NumFiles returns the total number of files across all collections.
func (stats *CollectionStats) NumFiles() (result int64) {
	for _, c := range stats.Collections {
		result += c.FileCount
	}
	return
}

// Content either returns the real content or some dummy bytes of the size of
// the object. TODO: handle options
func (t *TreeNode) Content(options ...fs.OpenOption) (io.ReadCloser, error) {
	switch v := t.ContentURL.(type) {
	case string:
		resp, err := http.Get(v)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("open: %v", resp.StatusCode)
		}
		return resp.Body, nil
	case nil:
		r := &extra.DummyReader{N: t.Size(), C: 0x7c}
		return io.NopCloser(r), nil
	default:
		return nil, fmt.Errorf("invalid content url type: %T", v)
	}
}

// Size returns object size as int64.
func (t *TreeNode) Size() int64 {
	switch v := t.ObjectSize.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

// MimeType returns the mimetype for the treenode or the empty string.
func (t *TreeNode) MimeType() string {
	switch v := t.FileType.(type) {
	case string:
		return v
	default:
		return ""
	}
}

// ParentTreeNodeIdentifier returns the parent treenode id if found or the
// empty string.
func (t *TreeNode) ParentTreeNodeIdentifier() string {
	v, ok := t.Parent.(string)
	if !ok {
		return ""
	}
	switch {
	case v == "":
		return ""
	case !strings.HasPrefix(v, "http"):
		return v
	default:
		re := regexp.MustCompile(`^http.*/api/treenodes/([0-9]{1,})/?$`)
		matches := re.FindStringSubmatch(v)
		if len(matches) != 2 {
			return ""
		}
		return matches[1]
	}
}

// OrganizationIdentifier is a helper to get the organization id from a user.
func (u *User) OrganizationIdentifier() string {
	switch {
	case u.Organization == "":
		return ""
	case !strings.HasPrefix(u.Organization, "http"):
		// TODO: Check if this is a number.
		return u.Organization
	default:
		re := regexp.MustCompile(`^http.*/api/organizations/([0-9]{1,})/?$`)
		matches := re.FindStringSubmatch(u.Organization)
		if len(matches) != 2 {
			return ""
		}
		return matches[1]
	}
}

// TreeNodeIdentifier parses out the treenode identifier for an organization.
func (o *Organization) TreeNodeIdentifier() string {
	switch {
	case o.TreeNode == "":
		return ""
	case !strings.HasPrefix(o.TreeNode, "http"):
		// TODO: Check if this is a number.
		return o.TreeNode
	default:
		re := regexp.MustCompile(`^http.*/api/treenodes/([0-9]{1,})/?$`)
		matches := re.FindStringSubmatch(o.TreeNode)
		if len(matches) != 2 {
			return ""
		}
		return matches[1]
	}
}

// PlanIdentifier parses out the plan identifier for an organization.
func (o *Organization) PlanIdentifier() string {
	switch {
	case !strings.HasPrefix(o.TreeNode, "http"):
		// TODO: Check if this is a number.
		return o.Plan
	default:
		re := regexp.MustCompile(`^http.*/api/plans/([0-9]{1,})/?$`)
		matches := re.FindStringSubmatch(o.Plan)
		if len(matches) != 2 {
			return ""
		}
		return matches[1]
	}
}

// Identifier returns the collection identifier.
func (c *Collection) Identifier() int64 {
	switch {
	case c.URL == "":
		return 0
	case !strings.HasPrefix(c.URL, "http"):
		return 0
	default:
		re := regexp.MustCompile(fmt.Sprintf(`^http.*/api/collections/([0-9]{1,})/?$`))
		matches := re.FindStringSubmatch(c.URL)
		if len(matches) != 2 {
			return 0
		}
		v, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0
		}
		return int64(v)
	}
}

// List objects
// ------------

type List struct {
	Count    int64       `json:"count"`
	Next     interface{} `json:"next"`
	Previous interface{} `json:"previous"`
}

// UserList from API, via JSONGen.
type UserList struct {
	List
	Result []*User `json:"results"`
}

// OrganizationList contains a list of organizations, e.g. from search.
type OrganizationList struct {
	List
	Result []*Organization `json:"results"`
}

// CollectionList contains a list of collections.
type CollectionList struct {
	List
	Result []*Collection `json:"results"`
}

// TreeNodeList for a list of treenodes.
type TreeNodeList struct {
	List
	Result []*TreeNode `json:"results"`
}

// Get methods
// -----------

// GetCollectionStats returns a summary.
func (api *Api) GetCollectionStats() (*CollectionStats, error) {
	var (
		opts = rest.Opts{
			Method: "GET",
			Path:   "/collections_stats",
		}
		doc CollectionStats
	)
	resp, err := api.client.CallJSON(context.TODO(), &opts, nil, &doc)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return &doc, nil
}

// GetUser returns the user for a given id.
func (api *Api) GetUser(id string) (*User, error) {
	if v := api.cache.GetGroup(id, "user"); v != nil {
		return v.(*User), nil
	}
	var (
		opts = rest.Opts{
			Method: "GET",
			Path:   fmt.Sprintf("/users/%v/", id),
		}
		doc User
	)
	resp, err := api.client.CallJSON(context.TODO(), &opts, nil, &doc)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("api: users got %v", resp.StatusCode)
	}
	api.cache.SetGroup(id, "user", &doc)
	return &doc, nil
}

// GetOrganization returns the organization for a given id.
func (api *Api) GetOrganization(id string) (*Organization, error) {
	if v := api.cache.GetGroup(id, "organization"); v != nil {
		return v.(*Organization), nil
	}
	var (
		opts = rest.Opts{
			Method: "GET",
			Path:   fmt.Sprintf("/organizations/%v/", id),
		}
		doc Organization
	)
	resp, err := api.client.CallJSON(context.TODO(), &opts, nil, &doc)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("api: organizations got %v", resp.StatusCode)
	}
	api.cache.SetGroup(id, "organization", &doc)
	return &doc, nil
}

// GetCollection returns the collection for a given id.
func (api *Api) GetCollection(id string) (*Collection, error) {
	if v := api.cache.GetGroup(id, "collection"); v != nil {
		return v.(*Collection), nil
	}
	var (
		opts = rest.Opts{
			Method: "GET",
			Path:   fmt.Sprintf("/collections/%v/", id),
		}
		doc Collection
	)
	resp, err := api.client.CallJSON(context.TODO(), &opts, nil, &doc)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("api: collections got %v", resp.StatusCode)
	}
	api.cache.SetGroup(id, "collection", &doc)
	return &doc, nil
}

// GetTreeNode returns the treenode for a given id.
func (api *Api) GetTreeNode(id string) (*TreeNode, error) {
	if v := api.cache.GetGroup(id, "treenode"); v != nil {
		return v.(*TreeNode), nil
	}
	var (
		opts = rest.Opts{
			Method: "GET",
			Path:   fmt.Sprintf("/treenodes/%v/", id),
		}
		doc TreeNode
	)
	resp, err := api.client.CallJSON(context.TODO(), &opts, nil, &doc)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("api: treenodes got %v", resp.StatusCode)
	}
	api.cache.SetGroup(id, "treenode", &doc)
	return &doc, nil
}

// GetPlan returns the plan for a given id.
func (api *Api) GetPlan(id string) (*Plan, error) {
	if v := api.cache.GetGroup(id, "plan"); v != nil {
		return v.(*Plan), nil
	}
	var (
		opts = rest.Opts{
			Method: "GET",
			Path:   fmt.Sprintf("/plans/%v/", id),
		}
		doc Plan
	)
	resp, err := api.client.CallJSON(context.TODO(), &opts, nil, &doc)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("api: plan got %v", resp.StatusCode)
	}
	api.cache.SetGroup(id, "plan", &doc)
	return &doc, nil
}

// Find methods
// ------------

func (api *Api) FindUsers(vs url.Values) (result []*User, err error) {
	if !vs.Has("limit") && !vs.Has("offset") {
		vs.Set("offset", "0")
		vs.Set("limit", defaultLimit) // TODO: implement pagination
	}
	if v := api.cache.GetGroup(cache.Atos(vs), "users"); v != nil {
		return v.([]*User), nil
	}
	var (
		opts = rest.Opts{
			Method:     "GET",
			Path:       "/users/",
			Parameters: vs,
		}
		doc UserList
	)
	resp, err := api.client.CallJSON(context.TODO(), &opts, nil, &doc)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("api: users got %v", resp.StatusCode)
	}
	for _, v := range doc.Result {
		result = append(result, v)
	}
	api.cache.SetGroup(cache.Atos(vs), "users", result)
	return result, nil
}

func (api *Api) FindOrganizations(vs url.Values) (result []*Organization, err error) {
	if !vs.Has("limit") && !vs.Has("offset") {
		vs.Set("offset", "0")
		vs.Set("limit", defaultLimit) // TODO: implement pagination
	}
	var (
		opts = rest.Opts{
			Method:     "GET",
			Path:       "/organizations/",
			Parameters: vs,
		}
		doc OrganizationList
	)
	resp, err := api.client.CallJSON(context.TODO(), &opts, nil, &doc)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("api: organizations got %v", resp.StatusCode)
	}
	for _, v := range doc.Result {
		result = append(result, v)
	}
	return result, nil
}

func (api *Api) FindCollections(vs url.Values) (result []*Collection, err error) {
	if !vs.Has("limit") && !vs.Has("offset") {
		vs.Set("offset", "0")
		vs.Set("limit", defaultLimit) // TODO: implement pagination
	}
	var (
		opts = rest.Opts{
			Method:     "GET",
			Path:       "/collections/",
			Parameters: vs,
		}
		doc CollectionList
	)
	resp, err := api.client.CallJSON(context.TODO(), &opts, nil, &doc)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("api: collections got %v", resp.StatusCode)
	}
	for _, v := range doc.Result {
		result = append(result, v)
	}
	return result, nil
}

func (api *Api) FindTreeNodes(vs url.Values) (result []*TreeNode, err error) {
	// ?id=1&id__gt=&id__gte=&id__lt=&id__lte=&node_type__contains=&node_type__
	// endswith=&node_type=&node_type__icontains=&node_type__iexact=&node_type__startsw
	// ith=&path__contains=&path__endswith=&path=&path__icontains=&path__iexact=&path__
	// startswith=&name__contains=&name__endswith=&name=&name__icontains=&name__iexact=
	// &name__startswith=&md5_sum__contains=&md5_sum__endswith=&md5_sum=&md5_sum__icont
	// ains=&md5_sum__iexact=&md5_sum__startswith=&sha1_sum__contains=&sha1_sum__endswi
	// th=&sha1_sum=&sha1_sum__icontains=&sha1_sum__iexact=&sha1_sum__startswith=&sha25
	// 6_sum__contains=&sha256_sum__endswith=&sha256_sum=&sha256_sum__icontains=&sha256
	// _sum__iexact=&sha256_sum__startswith=&size=&size__gt=&size__gte=&size__lt=&size_
	// _lte=&file_type__contains=&file_type__endswith=&file_type=&file_type__icontains=
	// &file_type__iexact=&file_type__startswith=&uploaded_at=&uploaded_at__gt=&uploade
	// d_at__gte=&uploaded_at__lt=&uploaded_at__lte=&pre_deposit_modified_at=&pre_depos
	// it_modified_at__gt=&pre_deposit_modified_at__gte=&pre_deposit_modified_at__lt=&p
	// re_deposit_modified_at__lte=&modified_at=&modified_at__gt=&modified_at__gte=&mod
	// ified_at__lt=&modified_at__lte=&uploaded_by=&comment__contains=&comment__endswit
	// h=&comment=&comment__icontains=&comment__iexact=&comment__startswith=&parent=
	if v := api.cache.GetGroup(cache.Atos(vs), "treenodes"); v != nil {
		return v.([]*TreeNode), nil
	}
	if !vs.Has("limit") && !vs.Has("offset") {
		vs.Set("offset", "0")
		vs.Set("limit", defaultLimit) // TODO: implement pagination
	}
	var (
		opts = rest.Opts{
			Method:     "GET",
			Path:       "/treenodes/",
			Parameters: vs,
		}
		doc TreeNodeList
	)
	resp, err := api.client.CallJSON(context.TODO(), &opts, nil, &doc)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("api: treenodes got %v", resp.StatusCode)
	}
	for _, v := range doc.Result {
		result = append(result, v)
	}
	api.cache.SetGroup(cache.Atos(vs), "treenodes", result)
	return result, nil
}
