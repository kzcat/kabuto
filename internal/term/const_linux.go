//go:build linux

package term

import "syscall"

const (
	ioctlGET = syscall.TCGETS
	ioctlSET = syscall.TCSETS
)
