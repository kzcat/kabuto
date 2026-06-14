//go:build unix

package render

import (
	"os"
	"unsafe"

	"syscall"
)

// ioctlSize retrieves terminal columns and rows via TIOCGWINSZ. Returns 0,0 on failure.
func ioctlSize() (cols, rows int) {
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
		return 0, 0
	}
	return int(ws.Col), int(ws.Row)
}

// ioctlWidth returns the terminal column count (for backward compatibility).
func ioctlWidth() int {
	c, _ := ioctlSize()
	return c
}
