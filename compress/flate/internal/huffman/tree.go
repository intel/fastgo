// Copyright (c) 2023, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package huffman

// TreeGenerator generate code's lengths from the histogram
// A huffman tree generator must be reused
// due to memory allocation overhead.
type TreeGenerator interface {
	// Generate huffman tree from histogram and write tree codes into codes.
	Generate(maxLens int, histogram []uint32, codeLens []uint32) (num int)
}
