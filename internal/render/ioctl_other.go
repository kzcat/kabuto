//go:build !unix

package render

// ioctlSize is unsupported on non-Unix platforms. Returns 0,0.
func ioctlSize() (cols, rows int) {
	return 0, 0
}

// ioctlWidth is unsupported on non-Unix platforms. Returns 0.
func ioctlWidth() int {
	return 0
}
