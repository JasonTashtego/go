// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build aix

package syscall

import (
	"internal/goexperiment"
	"sync"
)

// On IBMi PASE a program started by fork+exec or posix_spawn can fail to load
// ("Could not load program ...: System error - error data is: -1 9") when
// another thread of the parent creates or closes a file descriptor while the
// child is being created: the child then refers to a descriptor that is no
// longer what the parent had when the call began. A C program with eight
// threads starting a large executable with posix_spawn fails about a third of
// the time; the same program fails not at all when every pipe, open and close
// takes a read lock and the posix_spawn call takes the write lock. A single
// thread, or several processes of one thread each, never fail.
//
// ForkLock does not help: it is a read lock only for the few calls that need
// close-on-exec set atomically, and Close, Open, Accept and the like never
// take it. fdLock is a separate lock for every call that creates, duplicates
// or closes a descriptor (see the calls to fdLockR in zsyscall_aix_ppc64.go, and in Close in close_aix.go).
// forkExec holds it for writing while the child is created, which is the fork
// call only; the child's program load is not waited for.
//
// Lock order: ForkLock before fdLock. The wrappers hold fdLock only around the
// libc call and take no other lock inside it.
//
// Calls that can block (Open of a FIFO, Accept on a blocking socket) hold the
// read lock while they block, which stalls a fork until they return. Go's own
// sockets are non-blocking, so this affects only an Open of a FIFO that has no
// writer yet, and programs that call syscall.Accept on a blocking socket.
//
// Setting GOIBMI_FDLOCK=0 in the environment switches this off.
var (
	fdLock   sync.RWMutex
	fdLockOn bool
)

func init() {
	if !goexperiment.ISeriesAix {
		return
	}
	fdLockOn = true
	if v, _ := Getenv("GOIBMI_FDLOCK"); v == "0" {
		fdLockOn = false
	}
}

func fdLockR() {
	if fdLockOn {
		fdLock.RLock()
	}
}

func fdUnlockR() {
	if fdLockOn {
		fdLock.RUnlock()
	}
}

func forkExecFDLock() {
	if fdLockOn {
		fdLock.Lock()
	}
}

func forkExecFDUnlock() {
	if fdLockOn {
		fdLock.Unlock()
	}
}
