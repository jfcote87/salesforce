// Copyright 2022 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jwt

import "strings"

// CheckTestHost validates nm is equal to h's host name
func CheckTestHost(h bool, nm string) (bool, string) {
	want := testHost(h).Host()
	return strings.HasPrefix(nm, want), want
}
