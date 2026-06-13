//go:build !unix

package term

// State is a placeholder for non-unix systems.
type State struct{}

// MakeRaw is a no-op on non-unix.
func MakeRaw(fd int) (*State, error) { return nil, nil }

// Restore is a no-op on non-unix.
func Restore(fd int, st *State) error { return nil }
