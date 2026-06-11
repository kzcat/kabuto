//go:build unix

package render

import (
	"os"
	"unsafe"

	"syscall"
)

// ioctlWidth は TIOCGWINSZ で端末の桁数を取得する。取得不可なら 0。
func ioctlWidth() int {
	type winsize struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}
	var ws winsize
	fd := os.Stdout.Fd()
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		fd,
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	)
	if errno != 0 {
		return 0
	}
	return int(ws.Col)
}
