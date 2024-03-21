// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

//go:build !amd64 || noasmtest

package deflate

func (c *level1context) generate(flush bool, input []byte, processed int, offset int, tokens []token, maxToken int) (nOffset int, ntokens []token) {
	return lz77(flush, c.table[:], 1<<12-1, 1<<c.windowLevel, &c.hist, input, processed, offset, tokens, maxToken)
}

func (c *level2context) generate(flush bool, input []byte, processed int, offset int, tokens []token, maxToken int) (nOffset int, ntokens []token) {
	return lz77(flush, c.table[:], 1<<15-1, 1<<c.windowLevel, &c.hist, input, processed, offset, tokens, maxToken)
}
