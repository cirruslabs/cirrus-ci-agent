// parseRange() test from Golang's source code tree (src/net/http/range_test.go).
//
// Copyright 2011 The Go Authors. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
// * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package httprange

import (
	"testing"
)

var ParseRangeTests = []struct {
	s      string
	length int64
	r      []httpRange
}{
	{"", 0, nil},
	{"", 1000, nil},
	{"foo", 0, nil},
	{"bytes=", 0, nil},
	{"bytes=7", 10, nil},
	{"bytes= 7 ", 10, nil},
	{"bytes=1-", 0, nil},
	{"bytes=5-4", 10, nil},
	{"bytes=0-2,5-4", 10, nil},
	{"bytes=2-5,4-3", 10, nil},
	{"bytes=--5,4--3", 10, nil},
	{"bytes=A-", 10, nil},
	{"bytes=A- ", 10, nil},
	{"bytes=A-Z", 10, nil},
	{"bytes= -Z", 10, nil},
	{"bytes=5-Z", 10, nil},
	{"bytes=Ran-dom, garbage", 10, nil},
	{"bytes=0x01-0x02", 10, nil},
	{"bytes=         ", 10, nil},
	{"bytes= , , ,   ", 10, nil},

	{"bytes=0-9", 10, []httpRange{{0, 10}}},
	{"bytes=0-", 10, []httpRange{{0, 10}}},
	{"bytes=5-", 10, []httpRange{{5, 5}}},
	{"bytes=0-20", 10, []httpRange{{0, 10}}},
	{"bytes=15-,0-5", 10, []httpRange{{0, 6}}},
	{"bytes=1-2,5-", 10, []httpRange{{1, 2}, {5, 5}}},
	{"bytes=-2 , 7-", 11, []httpRange{{9, 2}, {7, 4}}},
	{"bytes=0-0 ,2-2, 7-", 11, []httpRange{{0, 1}, {2, 1}, {7, 4}}},
	{"bytes=-5", 10, []httpRange{{5, 5}}},
	{"bytes=-15", 10, []httpRange{{0, 10}}},
	{"bytes=0-499", 10000, []httpRange{{0, 500}}},
	{"bytes=500-999", 10000, []httpRange{{500, 500}}},
	{"bytes=-500", 10000, []httpRange{{9500, 500}}},
	{"bytes=9500-", 10000, []httpRange{{9500, 500}}},
	{"bytes=0-0,-1", 10000, []httpRange{{0, 1}, {9999, 1}}},
	{"bytes=500-600,601-999", 10000, []httpRange{{500, 101}, {601, 399}}},
	{"bytes=500-700,601-999", 10000, []httpRange{{500, 201}, {601, 399}}},

	// Match Apache laxity:
	{"bytes=   1 -2   ,  4- 5, 7 - 8 , ,,", 11, []httpRange{{1, 2}, {4, 2}, {7, 2}}},
}

func TestParseRange(t *testing.T) {
	for _, test := range ParseRangeTests {
		r := test.r
		ranges, err := parseRange(test.s, test.length)
		if err != nil && r != nil {
			t.Errorf("parseRange(%q) returned error %q", test.s, err)
		}
		if len(ranges) != len(r) {
			t.Errorf("len(parseRange(%q)) = %d, want %d", test.s, len(ranges), len(r))
			continue
		}
		for i := range r {
			if ranges[i].start != r[i].start {
				t.Errorf("parseRange(%q)[%d].start = %d, want %d", test.s, i, ranges[i].start, r[i].start)
			}
			if ranges[i].length != r[i].length {
				t.Errorf("parseRange(%q)[%d].length = %d, want %d", test.s, i, ranges[i].length, r[i].length)
			}
		}
	}
}
