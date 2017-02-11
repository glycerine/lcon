// from https://github.com/bradfitz/http2/pull/8/files
//
// motivation: https://groups.google.com/forum/#!topic/golang-dev/k0bSal8eDyE
//
// Copyright 2014 The Go Authors.
// See https://code.google.com/p/go/source/browse/CONTRIBUTORS
// Licensed under the same terms as Go itself:
// https://code.google.com/p/go/source/browse/LICENSE

// package pipe gives a local in-memory net.Conn
package lcon

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Pipe is buffered version of net.Pipe. Reads
// will block until data is available.
type Pipe struct {
	b       buffer
	rc      sync.Cond
	wc      sync.Cond
	rm      sync.Mutex
	wm      sync.Mutex
	Flushed chan bool

	readDeadline  time.Time
	writeDeadline time.Time
}

// NewPipe must be given a buf of
// pre-allocated size to use as the
// internal buffer between reads
// and writes.
func NewPipe(buf []byte) *Pipe {
	p := &Pipe{
		b:       buffer{buf: buf},
		Flushed: make(chan bool, 1),
	}
	p.rc = *sync.NewCond(&p.rm)
	return p
}

var ErrDeadline = fmt.Errorf("deadline exceeded")

// Read waits until data is available and copies bytes
// from the buffer into p.
func (r *Pipe) Read(p []byte) (n int, err error) {
	r.rc.L.Lock()
	defer r.rc.L.Unlock()
	if !r.readDeadline.IsZero() {
		now := time.Now()
		dur := r.readDeadline.Sub(now)
		if dur <= 0 {
			return 0, ErrDeadline
		}
		nextReadDone := make(chan struct{})
		defer close(nextReadDone)
		go func(dur time.Duration) {
			select {
			case <-time.After(dur):
				r.rc.L.Lock()
				r.b.late = true
				r.rc.L.Unlock()
				r.rc.Broadcast()
			case <-nextReadDone:
			}
		}(dur)
	}
	for r.b.Len() == 0 && !r.b.closed && !r.b.late {
		r.rc.Wait()
	}
	defer func() {
		// we already hold the lock
		r.b.late = false
		r.readDeadline = time.Time{}
	}()
	return r.b.Read(p)
}

// Write copies bytes from p into the buffer and wakes a reader.
// It is an error to write more data than the buffer can hold.
func (r *Pipe) Write(p []byte) (n int, err error) {
	r.rc.L.Lock()
	defer r.rc.L.Unlock()
	if !r.writeDeadline.IsZero() {
		now := time.Now()
		dur := r.writeDeadline.Sub(now)
		if dur <= 0 {
			return 0, ErrDeadline
		}
		nextWriteDone := make(chan struct{})
		defer close(nextWriteDone)
		go func(dur time.Duration) {
			select {
			case <-time.After(dur):
				r.rc.L.Lock()
				r.b.late = true
				r.rc.L.Unlock()
				r.rc.Broadcast()
			case <-nextWriteDone:
			}
		}(dur)
	}
	defer r.rc.Broadcast()
	defer r.flush()

	for r.b.freeBytes() < len(p) && !r.b.closed && !r.b.late {
		r.rc.Wait()
	}
	defer func() {
		// we already hold the lock
		r.b.late = false
		r.writeDeadline = time.Time{}
	}()

	return r.b.Write(p)
}

var ErrLconPipeClosed = fmt.Errorf("lcon pipe closed")

func (c *Pipe) Close() error {
	c.SetErrorAndClose(ErrLconPipeClosed)
	return nil
}

func (r *Pipe) SetErrorAndClose(err error) {
	r.rc.L.Lock()
	defer r.rc.L.Unlock()
	defer r.rc.Broadcast()
	r.b.Close(err)
}

// Pipe technically fullfills the net.Conn interface

func (r *Pipe) LocalAddr() net.Addr  { return addr{} }
func (r *Pipe) RemoteAddr() net.Addr { return addr{} }

func (r *Pipe) flush() {
	if len(r.Flushed) == 0 {
		r.Flushed <- true
	}
}

type addr struct{}

func (a addr) String() string  { return "memory.pipe:0" }
func (a addr) Network() string { return "in-process-internal" }

// SetDeadline implements the net.Conn method
func (r *Pipe) SetDeadline(t time.Time) error {
	err := r.SetReadDeadline(t)
	err2 := r.SetWriteDeadline(t)
	if err != nil {
		return err
	}
	return err2
}

// SetWriteDeadline implements the net.Conn method
func (r *Pipe) SetWriteDeadline(t time.Time) error {
	r.rc.L.Lock()
	r.writeDeadline = t
	r.rc.L.Unlock()
	return nil
}

// SetReadDeadline implements the net.Conn method
func (r *Pipe) SetReadDeadline(t time.Time) error {
	r.rc.L.Lock()
	r.readDeadline = t
	r.rc.L.Unlock()
	return nil
}
