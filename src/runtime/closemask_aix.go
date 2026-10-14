// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build aix

package runtime

import _ "unsafe" // for go:linkname

// syscall_runtime_blockSignals blocks every signal on the calling thread and
// saves the previous mask in *old. It is used by package syscall around close;
// see syscall.Close in close_aix.go. The caller must stay on the same thread
// until it calls syscall_runtime_restoreSignals (runtime.LockOSThread).
//
//go:linkname syscall_runtime_blockSignals syscall.runtime_blockSignals
//go:nosplit
func syscall_runtime_blockSignals(old *sigset) {
	sigprocmask(_SIG_SETMASK, &sigset_all, old)
}

// syscall_runtime_restoreSignals restores the thread's signal mask saved by
// syscall_runtime_blockSignals.
//
//go:linkname syscall_runtime_restoreSignals syscall.runtime_restoreSignals
//go:nosplit
func syscall_runtime_restoreSignals(old *sigset) {
	sigprocmask(_SIG_SETMASK, old, nil)
}
