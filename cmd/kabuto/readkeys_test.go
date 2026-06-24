package main

import (
	"errors"
	"io"
	"testing"
	"time"
)

// scriptReader returns a scripted sequence of (data, err) on successive Reads.
type scriptReader struct {
	steps []struct {
		data []byte
		err  error
	}
	i int
}

func (s *scriptReader) Read(p []byte) (int, error) {
	if s.i >= len(s.steps) {
		return 0, io.EOF
	}
	st := s.steps[s.i]
	s.i++
	n := copy(p, st.data)
	return n, st.err
}

// With hasTimeout=true (raw tty + VMIN=0/VTIME), a 0-byte read surfaces as
// io.EOF but only means "no key yet" — it must NOT close the channel. This
// guards the -w watch regression where watch mode exited immediately.
func TestReadKeysIgnoresTimeoutEOF(t *testing.T) {
	errStop := errors.New("stop")
	r := &scriptReader{steps: []struct {
		data []byte
		err  error
	}{
		{nil, io.EOF},      // timeout
		{nil, io.EOF},      // timeout
		{nil, io.EOF},      // timeout
		{[]byte("q"), nil}, // actual key press
		{nil, errStop},     // real error -> close
	}}
	ch := make(chan Key, 8)
	go readKeys(r, true, ch)

	select {
	case k := <-ch:
		if k.R != 'q' {
			t.Fatalf("got %+v, want Key{R:'q'}", k)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout: readKeys treated VTIME EOF as close instead of skipping it")
	}

	select {
	case k, ok := <-ch:
		if ok {
			t.Fatalf("expected closed channel after real error, got %+v", k)
		}
	case <-time.After(time.Second):
		t.Fatal("channel not closed after a real (non-timeout) error")
	}
}

// With hasTimeout=false (non-tty/pipe), a real io.EOF must close the channel so
// piped input terminates the reader.
func TestReadKeysEOFClosesWhenNoTimeout(t *testing.T) {
	r := &scriptReader{steps: []struct {
		data []byte
		err  error
	}{
		{nil, io.EOF},
	}}
	ch := make(chan Key, 8)
	go readKeys(r, false, ch)

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected closed channel on EOF with hasTimeout=false")
		}
	case <-time.After(time.Second):
		t.Fatal("readKeys did not close on EOF with hasTimeout=false")
	}
}
