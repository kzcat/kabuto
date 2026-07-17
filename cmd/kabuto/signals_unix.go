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

// watchContChan returns a channel that receives on SIGCONT, i.e. when the
// process resumes after being stopped (Ctrl-Z / SIGTSTP) or after a system
// suspend on platforms that deliver SIGCONT on resume. Used to re-apply raw
// mode, which the shell/terminal may have reset while we were stopped.
func watchContChan() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGCONT)
	return ch
}
