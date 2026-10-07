// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build aix

package syscall

import "internal/goexperiment"

// On IBMi PASE a program started by fork+exec can fail to load ("Could not
// load program ...: System error - error data is: -1 9") when the parent
// closes any file descriptor while the child is still starting. The child
// exits with status 255, or dies before closing the status pipe, in which
// case forkExec would block forever reading it. It showed up as the need to
// run the go command with GOMAXPROCS=1, which made "go build -p" start one
// child at a time.
//
// The failure disappears if the child closes every descriptor it does not
// need before it calls exec, so with GOEXPERIMENT=iseriesaix forkAndExecInChild
// does that (see closeAllInChild in exec_libc.go). A child then inherits only
// the descriptors listed in ProcAttr.Files, including ExtraFiles.
//
// Setting GOIBMI_CLOSEALL=0 in the environment switches this off.
func init() {
	if !goexperiment.ISeriesAix {
		return
	}
	closeAllInChild = true
	if v, _ := Getenv("GOIBMI_CLOSEALL"); v == "0" {
		closeAllInChild = false
	}
}
