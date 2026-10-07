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
// child is still inside exec, or still starting up afterwards, makes the
// child's loader fail ("Could not load program ... error data is: -1 9").
// The child then exits with status 255, or dies before closing the status
// pipe, which would hang the parent in forkExec.
//
// With GOEXPERIMENT=iseriesaix, forkExec therefore holds execGate for writing
// from the fork until the child has finished exec (its SEXECING process flag,
// observable with getprocs64, is clear) plus a short settle time for its
// start-up, and Close takes execGate for reading. No descriptor is closed
// while a child is starting.

//go:cgo_import_dynamic libc_getprocs64 getprocs64 "libc.a/shr_64.o"
//go:cgo_import_dynamic libc_usleep usleep "libc.a/shr_64.o"

//go:linkname libc_getprocs64 libc_getprocs64
var libc_getprocs64 libcFunc

//go:linkname libc_usleep libc_usleep
var libc_usleep libcFunc

const (
	// Layout of struct procentry64 on PASE, from <procinfo.h>.
	procentry64Size = 5024
	procStateOff    = 52 // pi_state
	procFlagsOff    = 56 // pi_flags

	procSZOMB    = 5          // SZOMB
	procSEXECING = 0x01000000 // SEXECING: process is execing

	// Upper bound on getprocs64 polls while waiting for exec to finish.
	execWaitPolls = 5000

	// Default time to keep the gate closed after exec has finished, while the
	// new process starts up. Measured: 10 ms leaves ~0.5% of starts failing
	// and 20 ms none.
	defaultExecSettleUs = 30000

	// Pause between checks of the status pipe while the child is starting.
	statusPollUs = 200
)

var (
	execGate sync.RWMutex

	// Scratch state for procInfo. Only used with execGate held for writing,
	// so it needs no further locking and never moves.
	procBuf [procentry64Size/8 + 1]uint64
	procIdx _Pid_t

	// execSettleUs is how long to keep the gate closed after exec finishes.
	// GOIBMI_EXECWAIT_MS overrides it.
	execSettleUs = defaultExecSettleUs

	// execDebug makes forkExecWaitDone report on stderr how each wait ended.
	// GOIBMI_EXECDEBUG=1 enables it.
	execDebug bool

	// closeUngated makes Close ignore execGate (GOIBMI_NOCLOSEGATE=1). It exists
	// only to measure what the gate on Close contributes.
	closeUngated bool
)

func init() {
	if v, _ := Getenv("GOIBMI_EXECDEBUG"); v == "1" {
		execDebug = true
	}
	if v, _ := Getenv("GOIBMI_NOCLOSEGATE"); v == "1" {
		closeUngated = true
	}
	if v, _ := Getenv("GOIBMI_EXECWAIT_MS"); v != "" {
		n := 0
		for _, c := range v {
			if c < '0' || c > '9' {
				n = -1
				break
			}
			n = n*10 + int(c-'0')
		}
		if n >= 0 && n < 1000 { // usleep takes less than a second
			execSettleUs = n * 1000
		}
	}
}

func usleep(us int) {
	if us > 0 {
		syscall6(uintptr(unsafe.Pointer(&libc_usleep)), 1, uintptr(us), 0, 0, 0, 0, 0)
	}
}

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

// procInfo reports the state and flags of process pid. ok is false if the
// process no longer exists. It must be called with execGate held for writing.
func procInfo(pid int) (state, flags uint32, ok bool) {
	procIdx = _Pid_t(pid)
	r, _, _ := syscall6(uintptr(unsafe.Pointer(&libc_getprocs64)), 6,
		uintptr(unsafe.Pointer(&procBuf[0])), procentry64Size, 0, 0,
		uintptr(unsafe.Pointer(&procIdx)), 1)
	if int32(r) < 1 {
		return 0, 0, false
	}
	w := (*[procentry64Size / 4]uint32)(unsafe.Pointer(&procBuf[0]))
	if int(w[0]) != pid {
		return 0, 0, false // getprocs64 returned the next process: ours is gone
	}
	return w[procStateOff/4], w[procFlagsOff/4], true
}

// forkExecReadStatus reads the child's exec status from the status pipe. On
// IBMi it does not block in read: a child that dies before exec closes the
// pipe stays a zombie holding its write end, so a blocking read would never
// return, and with execGate held that would stop every Close in the process.
// It polls instead, and gives up if the child has gone.
func forkExecReadStatus(fd, pid int, p *byte, n int) (int, error) {
	if !goexperiment.ISeriesAix {
		return readlen(fd, p, n)
	}
	fcntl(fd, F_SETFL, O_NONBLOCK)
	for {
		m, err := readlen(fd, p, n)
		if err != EAGAIN {
			return m, err
		}
		// Nothing to read and the pipe is still open: the child is either
		// still before its exec, or dead without having closed the pipe.
		if state, _, ok := procInfo(pid); !ok || state == procSZOMB {
			return 0, EPIPE
		}
		usleep(statusPollUs)
	}
}

// forkExecWaitDone is called with execGate held, after the status pipe has
// reported that exec began. It returns once the child is no longer execing,
// has gone away, or the poll limit is reached, then lets its start-up settle.
func forkExecWaitDone(pid int) {
	if !goexperiment.ISeriesAix {
		return
	}
	reason := "limit"
	var state, flags uint32
	i := 0
	for ; i < execWaitPolls; i++ {
		var ok bool
		state, flags, ok = procInfo(pid)
		if !ok {
			reason = "gone"
			break
		}
		if state == procSZOMB {
			reason = "zombie"
			break
		}
		if flags&procSEXECING == 0 {
			reason = "cleared"
			break
		}
		runtime.Gosched()
	}
	if reason == "cleared" {
		usleep(execSettleUs)
	}
	if execDebug {
		println("execwait pid", pid, "reason", reason, "polls", i, "state", state, "flags", flags)
	}
}

// Close closes a file descriptor.
func Close(fd int) error {
	if goexperiment.ISeriesAix && !closeUngated {
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
