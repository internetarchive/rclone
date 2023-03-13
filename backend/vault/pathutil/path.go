package pathutil

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
	// archive.org/details/IA-DPS-VAULT-QA-... Filenames should not start with
	// any of these prefixes. As it is possible to extend this list, there is a
	// remote possibility that a once valid filename would become invalid.
	DefaultVaultItemPrefixes = []string{"DPS-VAULT", "IA-DPS-VAULT"}
)

// IsValidPath returns true, if the path can be used in a petabox item using a
// set of predeclared prefixes for item names.
func IsValidPath(remote string) bool {
	if !isValidPath(remote, DefaultVaultItemPrefixes...) {
		return false
	}
	return true
}

// isValidPath returns true, if the path can be used in a petabox item
// with a given bucket prefixes.
func isValidPath(remote string, prefixes ...string) bool {
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
	for _, prefix := range prefixes {
		hasInvalidPrefix := strings.HasPrefix(strings.TrimLeft(remote, "/"), prefix)
		for _, suffix := range invalidSuffixes {
			hasInvalidSuffix := strings.HasSuffix(remote, suffix)
			if hasInvalidSuffix && hasInvalidPrefix {
				return false
			}
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
