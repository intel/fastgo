// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

//go:build amd64 && !noasmtest
// +build amd64,!noasmtest

package deflate

import (
	"github.com/intel/fastgo/internal/cpu"
)

func (c *level1context) generate(flush bool, input []byte, processed int, offset int, tokens []token, maxToken int) (nOffset int, ntokens []token) {
	if cpu.ArchLevel < 1 {
		return lz77(flush, c.table[:], 1<<12-1, 1<<c.windowLevel, &c.hist, input, processed, offset, tokens, maxToken)
	}
	if c.windowLevel == 12 {
		nOffset, ntokens = lz77Asm4kL12V1(c, input, processed, offset, tokens, maxToken-2)
	} else {
		nOffset, ntokens = lz77Asm32kL12V1(c, input, processed, offset, tokens, maxToken-2)
	}
	return lz77(flush, c.table[:], 1<<12-1, 1<<c.windowLevel, &c.hist, input, processed+nOffset-offset, nOffset, ntokens, maxToken)
}

func (c *level2context) generate(flush bool, input []byte, processed int, offset int, tokens []token, maxToken int) (nOffset int, ntokens []token) {
	if cpu.ArchLevel < 1 {
		return lz77(flush, c.table[:], 1<<15-1, 1<<c.windowLevel, &c.hist, input, processed, offset, tokens, maxToken)
	} else {
		if c.windowLevel == 12 {
			nOffset, ntokens = lz77Asm4kL15V1(c, input, processed, offset, tokens, maxToken-2)
		} else {
			nOffset, ntokens = lz77Asm32kL15V1(c, input, processed, offset, tokens, maxToken-2)
		}
	}
	return lz77(true, c.table[:], 1<<15-1, 1<<c.windowLevel, &c.hist, input, processed+nOffset-offset, nOffset, ntokens, maxToken)
}
