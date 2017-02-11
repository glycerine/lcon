package lcon

import (
	"net"
	"time"
)

type Bidir struct {
	Send *Pipe
	Recv *Pipe
}

func NewBidir(sz int) *Bidir {
	s := make([]byte, sz)
	r := make([]byte, sz)

	return &Bidir{
		Send: NewPipe(s),
		Recv: NewPipe(r),
	}
}

func (r *Bidir) Read(p []byte) (n int, err error) {
	return r.Recv.Read(p)
}

// Write copies bytes from p into the buffer and wakes a reader.
// It is an error to write more data than the buffer can hold.
func (r *Bidir) Write(p []byte) (n int, err error) {
	return r.Send.Write(p)
}

func (c *Bidir) Close() error {
	return c.Send.Close()
}

func (r *Bidir) SetErrorAndClose(err error) {
	r.Recv.SetErrorAndClose(err)
	r.Send.SetErrorAndClose(err)
}

// Bidir fullfills the net.Conn interface

func (r *Bidir) LocalAddr() net.Addr  { return addr{} }
func (r *Bidir) RemoteAddr() net.Addr { return addr{} }

// SetDeadline implements the net.Conn method
func (r *Bidir) SetDeadline(t time.Time) error {
	err := r.SetReadDeadline(t)
	err2 := r.SetWriteDeadline(t)
	if err != nil {
		return err
	}
	return err2
}

// SetWriteDeadline implements the net.Conn method
func (r *Bidir) SetWriteDeadline(t time.Time) error {
	return r.Send.SetWriteDeadline(t)
}

// SetReadDeadline implements the net.Conn method
func (r *Bidir) SetReadDeadline(t time.Time) error {
	return r.Recv.SetReadDeadline(t)
}
