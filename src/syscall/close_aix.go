// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build aix

package syscall

import (
	"internal/goexperiment"
	"runtime"
)

// On IBMi PASE, a signal that interrupts close(2) of the write end of a pipe
// can leave the pipe with a writer: close returns 0 and the descriptor is
// gone, but a reader of the pipe never sees end-of-file. Go sends itself
// SIGURG to preempt goroutines, so on PASE this shows up as os/exec hanging
// in Wait with the child already gone, and in pipes that never reach EOF.
//
// A C program that blocks the signal on the calling thread around close and
// nowhere else never loses a pipe (100,000 pipes with half a million signals
// delivered to its other threads); with no blocking it loses one within a few
// dozen. With GOEXPERIMENT=iseriesaix Close therefore blocks all signals on the
// calling thread for the duration of the libc call. A signal that arrives
// meanwhile stays pending and is delivered when the mask is restored.
//
// Close also holds fdLock for reading; see exec_aix_fdlock.go. The lock is
// taken before the signals are blocked, because waiting for it may park the
// goroutine.
//
// Setting GOIBMI_CLOSEMASK=0 in the environment switches the masking off.
var closeMask bool

func init() {
	if !goexperiment.ISeriesAix {
		return
	}
	closeMask = true
	if v, _ := Getenv("GOIBMI_CLOSEMASK"); v == "0" {
		closeMask = false
	}
}

// Implemented in the runtime. The argument is a runtime.sigset.
func runtime_blockSignals(old *[4]uint64)
func runtime_restoreSignals(old *[4]uint64)

func Close(fd int) (err error) {
	fdLockR()
	if !closeMask {
		err = closeRaw(fd)
		fdUnlockR()
		return err
	}
	runtime.LockOSThread()
	var old [4]uint64
	runtime_blockSignals(&old)
	err = closeRaw(fd)
	runtime_restoreSignals(&old)
	runtime.UnlockOSThread()
	fdUnlockR()
	return err
}
