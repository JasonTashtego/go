// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ((((unix && !android) || (js && wasm) || wasip1) && ((!cgo && !darwin) || osusergo)) || aix) && !(aix && goexperiment.iseriesaix)

package user

import "os"

func listGroups(u *User) ([]string, error) {
	f, err := os.Open(groupFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return listGroupsFromReader(u, f)
}
