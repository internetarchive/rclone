package vault

import (
	"encoding/xml"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	MaxPathLength = 4096 // PATH_MAX
	MaxNameLength = 255  // NAME_MAX
)

var (
	// DefaultVaultItemPrefixes are expected petabox item name prefixes. If any
	// more prefixes are to be used, we need to add them here. Example:
	// archive.org/details/IA-DPS-VAULT-QA-... We need this to reject certain
	// prohibited filenames.
	DefaultVaultItemPrefixes = []string{"DPS-VAULT", "IA-DPS-VAULT"}
)

// IsValidPath returns true, if the path can be used in a petabox item using a
// set of predeclared prefixes for item names.
func IsValidPath(remote string) bool {
	for _, prefix := range DefaultVaultItemPrefixes {
		if !IsValidPathPrefix(remote, prefix) {
			return false
		}
	}
	return true
}

// IsValidPathPrefix returns true, if the path can be used in a petabox item
// with a given item name (bucket) prefix.
func IsValidPathPrefix(remote, bucketPrefix string) bool {
	if remote == "" {
		return false
	}
	if remote == "/" {
		return false
	}
	if len(remote) > MaxPathLength {
		return false
	}
	if strings.Contains(remote, "//") {
		return false
	}
	if !utf8.ValidString(remote) {
		return false
	}
	invalidSuffixes := []string{
		"_files.xml",
		"_meta.sqlite",
		"_meta.xml",
		"_reviews.xml",
	}
	for _, suffix := range invalidSuffixes {
		hasInvalidPrefix := strings.HasPrefix(strings.TrimLeft(remote, "/"), bucketPrefix)
		hasInvalidSuffix := strings.HasSuffix(remote, suffix)
		if hasInvalidPrefix && hasInvalidSuffix {
			return false
		}
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
	invalidChars := []string{
		string('\x00'),
		string('\x0a'),
		string('\x0d'),
	}
	for _, c := range invalidChars {
		if strings.Contains(remote, c) {
			return false
		}
	}
	// Try to use path in XML, cf. self.contains_xml_incompatible_characters
	var dummy interface{}
	dec := xml.NewDecoder(strings.NewReader(fmt.Sprintf("<x>%s</x>", remote)))
	if err := dec.Decode(&dummy); err != nil {
		return false
	}
	return true
}
