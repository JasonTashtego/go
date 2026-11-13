// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build aix && goexperiment.iseriesaix

package user

import (
	"errors"
	"fmt"
	"internal/syscall/unix"
	"slices"
	"strconv"
	"strings"
)

// On IBM i PASE there is no /etc/group: users and groups come from the IBM i
// user profiles, which libc's getgrset reads.
func listGroups(u *User) ([]string, error) {
	if u.Username == "" {
		return nil, errors.New("user: list groups: empty username")
	}
	if _, err := strconv.Atoi(u.Gid); err != nil {
		return nil, fmt.Errorf("user: list groups for %s: invalid gid %q", u.Username, u.Gid)
	}
	list, err := unix.Getgrset(u.Username)
	if err != nil {
		return nil, fmt.Errorf("user: list groups for %s: %v", u.Username, err)
	}
	return groupsFromGetgrset(u.Gid, list), nil
}

// groupsFromGetgrset returns the primary gid followed by the other gids in
// list, a comma-separated list such as "102,103" as returned by getgrset.
// Empty and non-numeric entries and repeats are dropped.
func groupsFromGetgrset(primary, list string) []string {
	groups := []string{primary}
	for _, gid := range strings.Split(list, ",") {
		gid = strings.TrimSpace(gid)
		if gid == "" {
			continue
		}
		if _, err := strconv.Atoi(gid); err != nil {
			continue
		}
		if !slices.Contains(groups, gid) {
			groups = append(groups, gid)
		}
	}
	return groups
}
