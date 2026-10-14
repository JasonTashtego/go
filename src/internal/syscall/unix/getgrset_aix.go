// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unix

import (
	"errors"
	"sync"
	"syscall"
	"unsafe"
)

//go:cgo_import_dynamic libc_getgrset getgrset "libc.a/shr_64.o"
//go:linkname libc_getgrset libc_getgrset
var libc_getgrset uintptr

// getgrsetMu serializes calls to getgrset, which is not thread-safe and returns a static buffer.
var getgrsetMu sync.Mutex

// maxGetgrset bounds the length of the string that Getgrset will copy.
const maxGetgrset = 1 << 16

// Getgrset returns the comma-separated list of group IDs that libc's getgrset
// reports for the named user, for example "102,103". The list includes the
// user's primary group and is empty for a user in no group. On IBM i PASE the
// users and groups come from the IBM i user profiles; there is no /etc/group.
func Getgrset(user string) (string, error) {
	name, err := syscall.BytePtrFromString(user)
	if err != nil {
		return "", err
	}
	getgrsetMu.Lock()
	defer getgrsetMu.Unlock()
	r1, _, e1 := syscall6(uintptr(unsafe.Pointer(&libc_getgrset)), 1, uintptr(unsafe.Pointer(name)), 0, 0, 0, 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			return "", e1
		}
		return "", errors.New("getgrset failed")
	}
	// The string is in a static buffer owned by libc (on PASE two calls return the
	// same pointer), so it must not be freed, and it is copied while the lock is
	// still held.
	p := *(*unsafe.Pointer)(unsafe.Pointer(&r1))
	var buf []byte
	for i := 0; i < maxGetgrset; i++ {
		b := *(*byte)(unsafe.Add(p, i))
		if b == 0 {
			break
		}
		buf = append(buf, b)
	}
	return string(buf), nil
}
