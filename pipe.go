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
	b buffer
	c sync.Cond
	m sync.Mutex
}

// NewPipe must be given a buf of
// pre-allocated size to use as the
// internal buffer between reads
// and writes.
func NewPipe(buf []byte) *Pipe {
	p := &Pipe{
		b: buffer{buf: buf},
	}
	p.c = *sync.NewCond(&p.m)
	return p
}

// Read waits until data is available and copies bytes
// from the buffer into p.
func (r *Pipe) Read(p []byte) (n int, err error) {
	r.c.L.Lock()
	defer r.c.L.Unlock()
	for r.b.Len() == 0 && !r.b.closed {
		r.c.Wait()
	}
	return r.b.Read(p)
}

// Write copies bytes from p into the buffer and wakes a reader.
// It is an error to write more data than the buffer can hold.
func (w *Pipe) Write(p []byte) (n int, err error) {
	w.c.L.Lock()
	defer w.c.L.Unlock()
	defer w.c.Signal()
	return w.b.Write(p)
}

var ErrLconPipeClosed = fmt.Errorf("lcon pipe closed")

func (c *Pipe) Close() error {
	c.SetErrorAndClose(ErrLconPipeClosed)
	return nil
}

func (c *Pipe) SetErrorAndClose(err error) {
	c.c.L.Lock()
	defer c.c.L.Unlock()
	defer c.c.Signal()
	c.b.Close(err)
}

// Pipe technically fullfills the net.Conn interface
// with these next five methods, but they are no-ops.

func (c *Pipe) LocalAddr() net.Addr                { return nil }
func (c *Pipe) RemoteAddr() net.Addr               { return nil }
func (c *Pipe) SetDeadline(t time.Time) error      { return nil }
func (c *Pipe) SetReadDeadline(t time.Time) error  { return nil }
func (c *Pipe) SetWriteDeadline(t time.Time) error { return nil }
