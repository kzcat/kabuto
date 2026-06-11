//go:build !unix

package render

// ioctlSize は非Unix環境では未対応。0,0 を返す。
func ioctlSize() (cols, rows int) {
	return 0, 0
}

// ioctlWidth は非Unix環境では未対応。0 を返す。
func ioctlWidth() int {
	return 0
}
