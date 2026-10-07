// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build unix && !aix

package syscall

// Hooks used by forkExec on IBMi; no-ops on other systems.

func forkExecGateEnter()       {}
func forkExecGateLeave()       {}
func forkExecWaitDone(pid int) {}
func closeNoGate(fd int)       { Close(fd) }
