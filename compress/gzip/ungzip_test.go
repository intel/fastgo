// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package gzip

import (
	"bufio"
	"bytes"
	_ "embed"
	"io"
	"reflect"
	"testing"
)

//go:embed testdata/multistream.gz
var multistreamTestFile []byte

var multistreamFileMap = map[string]int{
	"cpulvl_amd64.s": 1446,
	"decode_amd64.s": 5703,
	"hybrid.go":      1497,
	"litblock.go":    1266,
}

func testMultiStream(t *testing.T, compressed []byte) (res map[string]int) {
	res = map[string]int{}
	br := bufio.NewReader(bytes.NewReader(multistreamTestFile))
	var r Reader
	for {
		err := r.Reset(br)
		r.Multistream(false)
		if err != nil {
			return res
		}
		data, err := io.ReadAll(&r)
		if err != nil {
			t.Fatal(err)
		}
		res[r.Name] = len(data)
	}
}

func TestGunzipMultiStream(t *testing.T) {
	res := testMultiStream(t, multistreamTestFile)
	if !reflect.DeepEqual(res, multistreamFileMap) {
		t.Fatalf("expected %v got %v", multistreamFileMap, res)
	}
}

func TestGunzip(t *testing.T) {
	res := testMultiStream(t, multistreamTestFile)
	if !reflect.DeepEqual(res, multistreamFileMap) {
		t.Fatalf("expected %v got %v", multistreamFileMap, res)
	}
}
