// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

//go:build amd64 && !noasmtest
// +build amd64,!noasmtest

// This file contains Intel AMD64-specific optimizations for compression levels 1 and 2.
// It provides assembly-accelerated LZ77 implementations that significantly improve
// compression performance while maintaining compression ratio.
package deflate

import (
	"github.com/intel/fastgo/internal/cpu"
)

// safeLZ77Boundary defines the safety margin for assembly LZ77 processing.
// This ensures the assembly code has sufficient buffer space for safe operation.
const safeLZ77Boundary = 4

// generate implements Level 1 compression with Intel optimizations.
// It automatically selects between assembly-optimized and standard implementations
// based on CPU capabilities and buffer constraints.
func (c *level1context) generate(flush bool, input []byte, processed int, offset int, tokens []token, maxToken int) (nOffset int, ntokens []token) {
	// Use standard implementation if CPU doesn't support optimizations or buffer is too small
	if cpu.ArchLevel < 1 || len(tokens)+safeLZ77Boundary > cap(tokens) {
		return lz77(flush, c.table[:], 1<<12-1, 1<<c.windowLevel, &c.hist, input, processed, offset, tokens, maxToken)
	}
	// Select appropriate assembly implementation based on window level
	if c.windowLevel == 12 {
		// Use 4K window assembly optimization
		nOffset, ntokens = lz77Asm4kL12V1(c, input, processed, offset, tokens, maxToken-safeLZ77Boundary)
	} else {
		// Use 32K window assembly optimization
		nOffset, ntokens = lz77Asm32kL12V1(c, input, processed, offset, tokens, maxToken-safeLZ77Boundary)
	}
	// Process any remaining data with standard implementation
	return lz77(flush, c.table[:], 1<<12-1, 1<<c.windowLevel, &c.hist, input, processed+nOffset-offset, nOffset, ntokens, maxToken)
}

// generate implements Level 2 compression with Intel optimizations.
// Similar to Level 1 but with different hash table size for better compression ratio.
func (c *level2context) generate(flush bool, input []byte, processed int, offset int, tokens []token, maxToken int) (nOffset int, ntokens []token) {
	// Use standard implementation if optimizations are not available
	if cpu.ArchLevel < 1 || len(tokens)+safeLZ77Boundary > cap(tokens) {
		return lz77(flush, c.table[:], 1<<15-1, 1<<c.windowLevel, &c.hist, input, processed, offset, tokens, maxToken)
	}
	if c.windowLevel == 12 {
		nOffset, ntokens = lz77Asm4kL15V1(c, input, processed, offset, tokens, maxToken-safeLZ77Boundary)
	} else {
		nOffset, ntokens = lz77Asm32kL15V1(c, input, processed, offset, tokens, maxToken-safeLZ77Boundary)
	}
	return lz77(true, c.table[:], 1<<15-1, 1<<c.windowLevel, &c.hist, input, processed+nOffset-offset, nOffset, ntokens, maxToken)
}
