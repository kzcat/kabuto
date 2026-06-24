//go:build unix

package term

import (
	"syscall"
	"unsafe"
)

// State holds the original termios so it can be restored.
type State struct {
	termios syscall.Termios
}

// MakeRaw puts the terminal fd into raw mode and returns the previous state.
func MakeRaw(fd int) (*State, error) {
	var old syscall.Termios
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(ioctlGET), uintptr(unsafe.Pointer(&old))); errno != 0 {
		return nil, errno
	}
	raw := old
	raw.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	raw.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	raw.Cflag &^= syscall.CSIZE | syscall.PARENB
	raw.Cflag |= syscall.CS8
	// Keep OPOST so that \r\n works normally (render1 already emits \r\n).
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(ioctlSET), uintptr(unsafe.Pointer(&raw))); errno != 0 {
		return nil, errno
	}
	return &State{termios: old}, nil
}

// Restore restores the terminal to the given state.
func Restore(fd int, st *State) error {
	if st == nil {
		return nil
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(ioctlSET), uintptr(unsafe.Pointer(&st.termios)))
	if errno != 0 {
		return errno
	}
	return nil
}

// SetReadTimeout switches the fd's raw mode to a polling read (VMIN=0,
// VTIME=tenths*0.1s) so that Read returns after a short timeout even when no
// byte is available. This lets the key reader distinguish a lone ESC keypress
// from the start of an escape sequence (arrow keys). Unix only.
func SetReadTimeout(fd int, tenths byte) error {
	var t syscall.Termios
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(ioctlGET), uintptr(unsafe.Pointer(&t))); errno != 0 {
		return errno
	}
	t.Cc[syscall.VMIN] = 0
	t.Cc[syscall.VTIME] = tenths
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(ioctlSET), uintptr(unsafe.Pointer(&t))); errno != 0 {
		return errno
	}
	return nil
}
