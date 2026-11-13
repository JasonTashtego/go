// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build aix

package syscall

import (
	"internal/goexperiment"
	"sync"
)

// On IBMi PASE, open calls that run at the same time in one process can hand
// out unusable descriptors: the descriptor open returns fails read with EBADF
// (fcntl(F_GETFL) fails too), and some opens fail outright. It needs several
// threads that ask for the same lowest free descriptor at once, which happens
// when a low descriptor is free: cmd/vet closes descriptor 1 when it replaces
// os.Stdout, and its analyzer goroutines then open files concurrently, one in
// fifty of them failing. A C program with four threads that open, read and
// close a file loses about 2% of its descriptors, with no loss when a mutex is
// held around open (and no change when one is held around close).
//
// Open therefore serializes the open call with openMu. A blocking open of a
// FIFO with no writer, or of a FIFO for writing with no reader, must not hold
// the mutex, since the other end may be opened by another goroutine of this
// process. So the open under the mutex always has O_NONBLOCK, which never
// waits. The flag is cleared again afterwards unless the caller asked for it.
// A FIFO opened for reading succeeds at once under O_NONBLOCK, without waiting
// for a writer; it is closed and opened again without the mutex and without
// O_NONBLOCK. So is a failure with ENXIO (a FIFO for writing with no reader)
// or EAGAIN. Opens of FIFOs are therefore not serialized against other opens.
//
// Lock order: openMu, then fdLock (taken by openRaw). Neither is held while
// waiting for anything but the other.
//
// Setting GOIBMI_OPENLOCK=0 in the environment switches this off.
var (
	openMu     sync.Mutex
	openSerial bool
)

func init() {
	if !goexperiment.ISeriesAix {
		return
	}
	openSerial = true
	if v, _ := Getenv("GOIBMI_OPENLOCK"); v == "0" {
		openSerial = false
	}
}

func Open(path string, mode int, perm uint32) (fd int, err error) {
	if !openSerial {
		return openRaw(path, mode, perm)
	}
	if mode&O_NONBLOCK != 0 {
		// Cannot wait for a FIFO; hold the mutex for the call only.
		openMu.Lock()
		fd, err = openRaw(path, mode, perm)
		openMu.Unlock()
		return fd, err
	}
	openMu.Lock()
	fd, err = openRaw(path, mode|O_NONBLOCK, perm)
	openMu.Unlock()
	if err != nil {
		if err == ENXIO || err == EAGAIN {
			// A FIFO with no other end, or a file that only waits
			// without O_NONBLOCK: do the blocking open outside the mutex.
			return openRaw(path, mode, perm)
		}
		return -1, err
	}
	if mode&O_ACCMODE == O_RDONLY {
		var st Stat_t
		if err = Fstat(fd, &st); err != nil {
			Close(fd)
			return -1, err
		}
		if st.Mode&S_IFMT == S_IFIFO {
			// The nonblocking open did not wait for a writer.
			Close(fd)
			return openRaw(path, mode, perm)
		}
	}
	flags, err := fcntl(fd, F_GETFL, 0)
	if err == nil {
		_, err = fcntl(fd, F_SETFL, flags&^O_NONBLOCK)
	}
	if err != nil {
		Close(fd)
		return -1, err
	}
	return fd, nil
}
