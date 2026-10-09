// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build unix && !aix

package syscall

// forkExecFDLock and forkExecFDUnlock are only used on AIX; see
// exec_aix_fdlock.go.
func forkExecFDLock()   {}
func forkExecFDUnlock() {}
