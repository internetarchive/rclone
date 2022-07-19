package vault

import (
	"encoding/xml"
	"fmt"
	"strings"
	"unicode/utf8"
)

// IsValidPath returns true, if the path can be used in a petabox item using a
// set of predeclared prefixes for item names.
func IsValidPath(remote string) bool {
	for _, bucketPrefix := range VaultItemPrefixes {
		if !IsValidPathBucketPrefix(remote, bucketPrefix) {
			return false
		}
	}
	return true
}

// IsValidPath returns true, if the path can be used in a petabox item with a given name prefix.
func IsValidPathBucketPrefix(remote, bucketPrefix string) bool {
	if remote == "" {
		return false
	}
	invalidSuffixes := []string{
		"_files.xml",
		"_meta.xml",
		"_meta.sqlite",
		"_reviews.xml",
	}
	for _, suffix := range invalidSuffixes {
		if strings.HasPrefix(strings.TrimLeft(remote, "/"), bucketPrefix) && strings.HasSuffix(remote, suffix) {
			return false
		}
	}
	if remote == "/" {
		return false
	}
	if strings.Contains(remote, "//") {
		return false
	}
	if len(remote) > MaxPathLength {
		return false
	}
	segments := strings.Split(remote, "/")
	for _, s := range segments {
		if s == "." || s == ".." {
			return false
		}
		if len(s) > MaxNameLength {
			return false
		}
	}
	invalidChars := []string{string('\x00'), string('\x0a'), string('\x0d')}
	for _, c := range invalidChars {
		if strings.Contains(remote, c) {
			return false
		}
	}
	if !utf8.ValidString(remote) {
		return false
	}
	// Try to use path in XML, cf. self.contains_xml_incompatible_characters
	var dummy interface{}
	dec := xml.NewDecoder(strings.NewReader(fmt.Sprintf("<x>%s</x>", remote)))
	if err := dec.Decode(&dummy); err != nil {
		return false
	}
	return true
}
