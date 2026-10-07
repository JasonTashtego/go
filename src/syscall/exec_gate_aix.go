// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build aix

package syscall

import (
	"internal/goexperiment"
	"runtime"
	"sync"
	"unsafe"
)

// On IBMi PASE, closing any descriptor in the parent while a freshly started
// child is still inside exec makes the child's loader fail ("Could not load
// program ... error data is: -1 9"), and the child then exits with status 255
// or dies before closing the status pipe, which hangs the parent in forkExec.
// The exec is in progress while the child's SEXECING process flag is set,
// which is observable with getprocs64.
//
// With GOEXPERIMENT=iseriesaix, forkExec therefore holds execGate for writing
// from the fork until the child has finished exec, and Close takes it for
// reading, so no descriptor is closed during that window.

//go:cgo_import_dynamic libc_getprocs64 getprocs64 "libc.a/shr_64.o"

//go:linkname libc_getprocs64 libc_getprocs64
var libc_getprocs64 libcFunc

const (
	// Layout of struct procentry64 on PASE, from <procinfo.h>.
	procentry64Size = 5024
	procStateOff    = 52 // pi_state
	procFlagsOff    = 56 // pi_flags

	procSZOMB    = 5          // SZOMB
	procSEXECING = 0x01000000 // SEXECING: process is execing

	// Upper bound on getprocs64 polls while waiting for exec to finish.
	execWaitPolls = 5000
)

var (
	execGate sync.RWMutex

	// Scratch state for forkExecWaitDone. Only used with execGate held for
	// writing, so it needs no further locking and never moves.
	procBuf [procentry64Size/8 + 1]uint64
	procIdx _Pid_t
)

func forkExecGateEnter() {
	if goexperiment.ISeriesAix {
		execGate.Lock()
	}
}

func forkExecGateLeave() {
	if goexperiment.ISeriesAix {
		execGate.Unlock()
	}
}

// forkExecWaitDone is called with execGate held, after the status pipe has
// reported that exec succeeded. It returns once the child is no longer
// execing, has gone away, or the poll limit is reached.
func forkExecWaitDone(pid int) {
	if !goexperiment.ISeriesAix {
		return
	}
	for i := 0; i < execWaitPolls; i++ {
		procIdx = _Pid_t(pid)
		r, _, _ := syscall6(uintptr(unsafe.Pointer(&libc_getprocs64)), 6,
			uintptr(unsafe.Pointer(&procBuf[0])), procentry64Size, 0, 0,
			uintptr(unsafe.Pointer(&procIdx)), 1)
		if int32(r) < 1 {
			return // no such process any more
		}
		w := (*[procentry64Size / 4]uint32)(unsafe.Pointer(&procBuf[0]))
		if int(w[0]) != pid {
			return // getprocs64 returned the next process: ours is gone
		}
		if w[procStateOff/4] == procSZOMB {
			return
		}
		if w[procFlagsOff/4]&procSEXECING == 0 {
			return
		}
		runtime.Gosched()
	}
}

// Close closes a file descriptor.
func Close(fd int) error {
	if goexperiment.ISeriesAix {
		execGate.RLock()
		defer execGate.RUnlock()
	}
	return closeRaw(fd)
}

// closeNoGate closes fd without taking execGate, for use by forkExec itself
// while it holds the gate.
func closeNoGate(fd int) {
	closeRaw(fd)
}
