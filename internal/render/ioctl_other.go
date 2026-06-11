//go:build !unix

package render

// ioctlWidth は非Unix環境では未対応。0 を返す。
func ioctlWidth() int {
	return 0
}
