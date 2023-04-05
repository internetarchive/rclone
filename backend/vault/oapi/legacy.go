package oapi

import (
	"strconv"
	"time"

	"github.com/rclone/rclone/backend/vault/api"
)

// toLegacyTreeNode is a transition helper, turns a oapi list of treenodes to
// api.TreeNode values.
func toLegacyTreeNode(vs *[]TreeNode) (result []*api.TreeNode) {
	if vs == nil {
		return
	}
	for _, t := range *vs {
		// UploadedBy is a potentially nil object, and we want the ID, so need
		// indirect once more.
		var (
			uploadedByID = 0
			path         = ""
			url          = ""
		)
		if t.UploadedBy != nil {
			if v := safeDereference(t.UploadedBy.Id); v != nil {
				uploadedByID = v.(int)
			}
		}
		if v := safeDereference(t.Path); v != nil {
			path = v.(string)
		}
		if v := safeDereference(t.Url); v != nil {
			url = v.(string)
		}
		result = append(result, &api.TreeNode{
			Comment:              safeDereference(t.Comment),
			ContentURL:           safeDereference(t.ContentUrl),
			FileType:             safeDereference(t.FileType),
			ID:                   int64(*t.Id),
			Md5Sum:               safeDereference(t.Md5Sum),
			ModifiedAt:           safeTimeFormat(t.ModifiedAt, time.RFC3339),
			Name:                 t.Name,
			NodeType:             string(*t.NodeType),
			Parent:               safeDereference(t.Parent),
			Path:                 path,
			PreDepositModifiedAt: safeTimeFormat(t.PreDepositModifiedAt, time.RFC3339),
			Sha1Sum:              safeDereference(t.Sha1Sum),
			Sha256Sum:            safeDereference(t.Sha256Sum),
			ObjectSize:           *t.Size,
			UploadedAt:           safeTimeFormat(t.UploadedAt, time.RFC3339),
			UploadedBy:           uploadedByID,
			URL:                  url,
		})
	}
	return
}

func toLegacyTargetGeolocation(vs *[]Geolocation) (result []struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}) {
	if vs == nil {
		return
	}
	for _, v := range *vs {
		result = append(result, struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		}{v.Name, *v.Url})
	}
	return
}

// toLegacyCollection is a helper to convert oapi values to legacy values.
func toLegacyCollection(vs *[]Collection) (result []*api.Collection) {
	if vs == nil {
		return
	}
	for _, v := range *vs {
		var targetReplication int64
		i, err := strconv.Atoi(string(*v.TargetReplication))
		if err == nil {
			targetReplication = int64(i)
		} else {
			// TODO: this may never happen
		}
		result = append(result, &api.Collection{
			FixityFrequency:    string(*v.FixityFrequency),
			Name:               v.Name,
			Organization:       v.Organization,
			TargetGeolocations: toLegacyTargetGeolocation(v.TargetGeolocations),
			TargetReplication:  targetReplication,
			TreeNode:           *v.TreeNode,
			URL:                *v.Url,
		})
	}
	return
}
