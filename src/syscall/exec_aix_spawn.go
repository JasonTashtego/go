// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build aix && ppc64

package syscall

import (
	"internal/goexperiment"
	"unsafe"
)

// IBMi PASE cannot reliably fork a multi-threaded process. With
// GOEXPERIMENT=iseriesaix, processes are created with posix_spawn instead,
// which has no window in which a forked copy of the Go runtime must run
// Go code. Requests that need attributes posix_spawn cannot express here
// (chroot, credentials, sessions, process groups, controlling terminals)
// still use the fork path.

//go:cgo_import_dynamic libc_posix_spawn posix_spawn "libc.a/shr_64.o"
//go:cgo_import_dynamic libc_posix_spawn_file_actions_init posix_spawn_file_actions_init "libc.a/shr_64.o"
//go:cgo_import_dynamic libc_posix_spawn_file_actions_destroy posix_spawn_file_actions_destroy "libc.a/shr_64.o"
//go:cgo_import_dynamic libc_posix_spawn_file_actions_adddup2 posix_spawn_file_actions_adddup2 "libc.a/shr_64.o"
//go:cgo_import_dynamic libc_posix_spawn_file_actions_addclose posix_spawn_file_actions_addclose "libc.a/shr_64.o"

//go:linkname libc_posix_spawn libc_posix_spawn
//go:linkname libc_posix_spawn_file_actions_init libc_posix_spawn_file_actions_init
//go:linkname libc_posix_spawn_file_actions_destroy libc_posix_spawn_file_actions_destroy
//go:linkname libc_posix_spawn_file_actions_adddup2 libc_posix_spawn_file_actions_adddup2
//go:linkname libc_posix_spawn_file_actions_addclose libc_posix_spawn_file_actions_addclose

var (
	libc_posix_spawn,
	libc_posix_spawn_file_actions_init,
	libc_posix_spawn_file_actions_destroy,
	libc_posix_spawn_file_actions_adddup2,
	libc_posix_spawn_file_actions_addclose libcFunc
)

// spawnShell runs the program after changing directory, because PASE has no
// posix_spawn_file_actions_addchdir_np.
const spawnShell = "/QOpenSys/usr/bin/sh"

const spawnChdirScript = `cd "$1" || exit 126; shift; exec "$@"`

// spawnFileActions is storage for an opaque posix_spawn_file_actions_t. It is
// a package variable so that it never moves, and it is only used while the
// fork lock is held, which serializes callers. The size is generous because
// the real layout is not visible from here.
var spawnFileActions [256]uint64

func init() {
	// GOIBMI_FORK=1 forces the original fork path, as an escape hatch.
	if v, _ := Getenv("GOIBMI_FORK"); goexperiment.ISeriesAix && v != "1" {
		forkExecSpawn = spawnIBMi
	}
}

func spawnCall(fn *libcFunc, nargs uintptr, a1, a2, a3, a4, a5, a6 uintptr) Errno {
	r, _, _ := syscall6(uintptr(unsafe.Pointer(fn)), nargs, a1, a2, a3, a4, a5, a6)
	return Errno(r)
}

// fdInherited reports whether fd is open in this process and would be
// inherited across exec.
func fdInherited(fd int) bool {
	val, err := fcntl1(uintptr(fd), F_GETFD, 0)
	return err == 0 && val&FD_CLOEXEC == 0
}

// spawnIBMi implements forkAndExecInChild with posix_spawn. handled is false
// if the request must be served by the fork path instead.
func spawnIBMi(argv0 *byte, argv, envv []*byte, chroot, dir *byte, attr *ProcAttr, sys *SysProcAttr) (pid int, err Errno, handled bool) {
	if chroot != nil || sys.Credential != nil || sys.Setsid || sys.Setpgid ||
		sys.Foreground || sys.Setctty || sys.Noctty {
		return 0, 0, false
	}

	fa := uintptr(unsafe.Pointer(&spawnFileActions))
	if e := spawnCall(&libc_posix_spawn_file_actions_init, 1, fa, 0, 0, 0, 0, 0); e != 0 {
		return 0, e, true
	}
	defer spawnCall(&libc_posix_spawn_file_actions_destroy, 1, fa, 0, 0, 0, 0, 0)

	dup2 := func(from, to int) Errno {
		return spawnCall(&libc_posix_spawn_file_actions_adddup2, 3, fa, uintptr(from), uintptr(to), 0, 0, 0)
	}
	closefd := func(fd int) Errno {
		return spawnCall(&libc_posix_spawn_file_actions_addclose, 2, fa, uintptr(fd), 0, 0, 0, 0)
	}

	// Build the child's fd table the same way forkAndExecInChild does: move
	// sources that a later dup2 would clobber out of the way (pass 1), then
	// dup2 each source onto its target (pass 2). Temporary descriptors are
	// closed again at the end.
	fd := make([]int, len(attr.Files))
	nextfd := len(attr.Files)
	for i, ufd := range attr.Files {
		fd[i] = int(ufd)
		if nextfd < int(ufd) {
			nextfd = int(ufd)
		}
	}
	nextfd++

	var tmps []int
	for i := range fd {
		if fd[i] >= 0 && fd[i] < i {
			if e := dup2(fd[i], nextfd); e != 0 {
				return 0, e, true
			}
			tmps = append(tmps, nextfd)
			fd[i] = nextfd
			nextfd++
		}
	}
	for i := range fd {
		var e Errno
		switch {
		case fd[i] == -1:
			if fdInherited(i) {
				e = closefd(i)
			}
		case fd[i] == i:
			// dup2(i, i) does not reliably clear close-on-exec, so go
			// through a temporary.
			if e = dup2(i, nextfd); e == 0 {
				if e = dup2(nextfd, i); e == 0 {
					e = closefd(nextfd)
				}
			}
			nextfd++
		default:
			e = dup2(fd[i], i)
		}
		if e != 0 {
			return 0, e, true
		}
	}
	// By convention, close 0, 1, 2 if they were not given.
	for i := len(fd); i < 3; i++ {
		if fdInherited(i) {
			if e := closefd(i); e != 0 {
				return 0, e, true
			}
		}
	}
	for _, t := range tmps {
		if e := closefd(t); e != 0 {
			return 0, e, true
		}
	}

	path := argv0
	args := argv
	if dir != nil {
		sh, _ := BytePtrFromString(spawnShell)
		shname, _ := BytePtrFromString("sh")
		script, _ := BytePtrFromString(spawnChdirScript)
		dashc, _ := BytePtrFromString("-c")
		// sh -c script sh DIR PROG ARGS...; argv ends with its nil terminator.
		args = append([]*byte{shname, dashc, script, shname, dir, argv0}, argv[1:]...)
		path = sh
	}

	var child _Pid_t
	e := spawnCall(&libc_posix_spawn, 6,
		uintptr(unsafe.Pointer(&child)),
		uintptr(unsafe.Pointer(path)),
		fa,
		0, // default attributes
		uintptr(unsafe.Pointer(&args[0])),
		uintptr(unsafe.Pointer(&envv[0])))
	if e != 0 {
		return 0, e, true
	}
	return int(child), 0, true
}
