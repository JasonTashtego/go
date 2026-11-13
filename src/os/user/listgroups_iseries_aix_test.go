// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build aix && goexperiment.iseriesaix

package user

import (
	"slices"
	"testing"
)

func TestGroupsFromGetgrset(t *testing.T) {
	tests := []struct {
		primary, list string
		want          []string
	}{
		{"102", "102,103", []string{"102", "103"}},
		{"102", "103,102", []string{"102", "103"}},
		{"0", "", []string{"0"}},
		{"7", " 5 ,x,5,,6", []string{"7", "5", "6"}},
		{"7", "7", []string{"7"}},
	}
	for _, tt := range tests {
		if got := groupsFromGetgrset(tt.primary, tt.list); !slices.Equal(got, tt.want) {
			t.Errorf("groupsFromGetgrset(%q, %q) = %q, want %q", tt.primary, tt.list, got, tt.want)
		}
	}
}
