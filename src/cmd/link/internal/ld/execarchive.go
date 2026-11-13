// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !wasm && !windows

package ld

import (
	"internal/buildcfg"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

// syscallExecSupported reports whether execArchive can use syscall.Exec. On
// IBMi PASE execve fails with EPERM ("Not owner") in a process that has more
// than one thread, as every Go program does, so the archiver is run as a
// subprocess there.
var syscallExecSupported = !(runtime.GOOS == "aix" && buildcfg.Experiment.ISeriesAix)

// execArchive invokes the archiver tool with syscall.Exec(), with
// the expectation that this is the last thing that takes place
// in the linking operation.
func (ctxt *Link) execArchive(argv []string) {
	var err error
	argv0 := argv[0]
	if filepath.Base(argv0) == argv0 {
		argv0, err = exec.LookPath(argv0)
		if err != nil {
			Exitf("cannot find %s: %v", argv[0], err)
		}
	}
	if ctxt.Debugvlog != 0 {
		ctxt.Logf("invoking archiver with syscall.Exec()\n")
	}
	err = syscall.Exec(argv0, argv, os.Environ())
	if err != nil {
		Exitf("running %s failed: %v", argv[0], err)
	}
}
