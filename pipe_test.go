// from https://github.com/bradfitz/http2/pull/8/files
//
// motivation: https://groups.google.com/forum/#!topic/golang-dev/k0bSal8eDyE
//
// Copyright 2014 The Go Authors.
// See https://code.google.com/p/go/source/browse/CONTRIBUTORS
// Licensed under the same terms as Go itself:
// https://code.google.com/p/go/source/browse/LICENSE

package lcon

import (
	"errors"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestPipeClose(t *testing.T) {
	var p Pipe
	p.c.L = &p.m
	a := errors.New("a")
	b := errors.New("b")
	p.SetErrorAndClose(a)
	p.SetErrorAndClose(b)
	_, err := p.Read(make([]byte, 1))
	if err != a {
		t.Errorf("err = %v want %v", err, a)
	}
}

func TestPipeAsNetConn(t *testing.T) {

	var nc net.Conn = NewPipe(make([]byte, 100))

	msg := "hello-world"
	ms2 := "finkleworms"
	for i := 0; i < 2; i++ {
		if i == 1 {
			msg = ms2
		}

		n, err := nc.Write([]byte(msg))
		if err != nil {
			t.Errorf("err = %v", err)
		}
		if n != len(msg) {
			t.Errorf("Write truncated at %v < %v", n, len(msg))
		}

		readbuf := make([]byte, len(msg))
		m, err := nc.Read(readbuf)
		if err != nil {
			t.Errorf("err = %v", err)
		}
		if m != n {
			t.Errorf("Read truncated at %v !=n %v", m, n)
		}
		back := string(readbuf[:m])
		if back != msg {
			t.Errorf("msg corrupted, wrote '%v', read '%v'", msg, back)
		}
	}

	// write-write, read-read
	{
		n, err := nc.Write([]byte(msg))
		if err != nil {
			t.Errorf("err = %v", err)
		}
		if n != len(msg) {
			t.Errorf("Write truncated at %v < %v", n, len(msg))
		}

		n2, err := nc.Write([]byte(ms2))
		if err != nil {
			t.Errorf("err = %v", err)
		}
		if n2 != len(ms2) {
			t.Errorf("Write truncated at %v < %v", n, len(msg))
		}

		readbuf := make([]byte, len(msg))
		m, err := nc.Read(readbuf)
		if err != nil {
			t.Errorf("err = %v", err)
		}
		if m != n {
			t.Errorf("Read truncated at %v !=n %v", m, n)
		}
		back := string(readbuf[:m])
		if back != msg {
			t.Errorf("msg corrupted, wrote '%v', read '%v'", msg, back)
		}

		// 2nd read
		readbuf2 := make([]byte, len(msg))
		m2, err := nc.Read(readbuf2)
		if err != nil {
			t.Errorf("err = %v", err)
		}
		if m2 != n2 {
			t.Errorf("Read truncated at %v !=n %v", m2, n2)
		}
		back2 := string(readbuf2[:m2])
		if back != ms2 {
			t.Errorf("msg corrupted, wrote '%v', read '%v'", ms2, back2)
		}
	}

	// blocking read
	readdone := make(chan struct{})
	readbuf3 := make([]byte, len(msg))
	var m3 int
	var err error
	go func() {

		// should block
		m3, err = nc.Read(readbuf3)
		if err != nil {
			t.Errorf("err = %v", err)
		}
		close(readdone)
	}()
	select {
	case <-readdone:
		t.Fatal("read did not block")
	case <-time.After(60 * time.Millisecond):
		// good, read should have blocked.
	}

	msg3 := "heya"
	// write and see release
	n3, err := nc.Write([]byte(msg3))
	if err != nil {
		t.Errorf("err = %v", err)
	}
	if n3 != len(msg3) {
		t.Errorf("Write truncated at %v < %v", n3, len(msg3))
	}
	<-readdone
	got := string(readbuf3[:m3])
	if got != msg3 {
		t.Errorf(fmt.Errorf("msg corrupted, wrote '%v', read '%v'", msg3, got).Error())
	}
	//fmt.Printf("\n got = '%s'\n", got)
}
