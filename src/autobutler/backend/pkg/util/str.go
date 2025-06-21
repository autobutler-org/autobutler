package util

func TrimLeading(path string, c byte) string {
	for ;len(path) > 0 && path[0] == c; path = path[1:] { /* Trim leading slashes */}
	return path
}
