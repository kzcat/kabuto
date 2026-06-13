//go:build darwin

package term

import "syscall"

const (
	ioctlGET = syscall.TIOCGETA
	ioctlSET = syscall.TIOCSETA
)
