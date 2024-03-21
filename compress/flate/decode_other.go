// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

//go:build !amd64 || noasmtest

package flate

func decodeHuffman(state *inflate, output []byte, written int) (w int, err error) {
	return decodeHuffmanLargeLoop(state, output, written)
}
