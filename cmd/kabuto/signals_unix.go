//go:build unix

package main

import (
	"os"
	"os/signal"
	"syscall"
)

// watchResizeChan returns a channel that receives on terminal resize (SIGWINCH).
func watchResizeChan() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	return ch
}
