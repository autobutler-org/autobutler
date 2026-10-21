package photoutil

import "strings"

// nameMatches reports whether fileName contains query, ignoring case.
// A blank query matches every name, so a listing that was not searched stays
// the whole library. The match is the file's own name, not a directory on
// its path.
func nameMatches(fileName, query string) bool {
	query = strings.TrimSpace(query)
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(fileName), strings.ToLower(query))
}
