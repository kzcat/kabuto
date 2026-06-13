//go:build !unix

package main

import "os"

// watchResizeChan returns a no-op channel on non-Unix platforms (no SIGWINCH).
func watchResizeChan() <-chan os.Signal {
	return make(chan os.Signal, 1)
}
