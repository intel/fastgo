// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

//go:build amd64 && !noasmtest
// +build amd64,!noasmtest

// This file contains Intel AMD64-specific optimizations for token encoding.
// It provides assembly-accelerated implementations for encoding LZ77 tokens
// into DEFLATE bit streams with optimal performance on different Intel architectures.
package deflate

import (
	"unsafe"

	"github.com/intel/fastgo/internal/cpu"
)

// Assembly implementations for different Intel architecture levels
func encodeTokensArchV4(hist *histogram, tokens []token, buf *BitBuf) int // Level 4 optimizations
func encodeTokensArchV3(hist *histogram, tokens []token, buf *BitBuf) int // Level 3 optimizations

// asmTokenEncoder holds the selected assembly implementation based on CPU capabilities
var asmTokenEncoder func(hist *histogram, tokens []token, buf *BitBuf) int

// init selects the optimal token encoder implementation based on detected CPU architecture.
// Higher architecture levels provide better performance through advanced instruction sets.
func init() {
	switch cpu.ArchLevel {
	case 3:
		asmTokenEncoder = encodeTokensArchV3 // Use Level 3 optimizations
	case 4:
		asmTokenEncoder = encodeTokensArchV4 // Use Level 4 optimizations (highest performance)
	default:
		asmTokenEncoder = encodeTokens // Fall back to standard implementation
	}
}

func optimizedEncodeTokens(hist *histogram, tokens []token, buf *BitBuf) (tokenNum int) {
	buf.Sync()
	if cpu.ArchLevel > 0 {
		tokenNum += asmTokenEncoder(hist, tokens, buf)
	}

	tokenNum += encodeTokens(hist, tokens[tokenNum:], buf)
	return tokenNum
}

func encodeTokens(hist *histogram, tokens []token, buf *BitBuf) (tokenNum int) {
	if len(tokens) == 0 {
		return 0
	}
	// max bits write per loop = (15 + 15 + 5 + 13 ) = 48
	// 48 / 8 = 6
	output := buf.output
	end := len(output) - 8
	var (
		idx    int    = buf.idx
		bits   uint64 = buf.bits
		bitLen int    = buf.bitLen
	)
	if end <= idx {
		return 0
	}
	var extraCount uint32
	tokenIdx := 0

	for tokenIdx = range tokens {
		t := tokens[tokenIdx]
		litlen, dist, extra := t.Extract()
		code, count := hist.litCode(litlen)

		bits |= uint64(code) << bitLen
		bitLen += int(count)
		if bitLen > 64 {
			*(*uint64)(unsafe.Pointer(&output[idx])) = bits
			idx += 8
			bitLen -= 64
			bits = uint64(code) >> (uint64(count) - uint64(bitLen))
		}
		code, count, extraCount = hist.distCode(dist)

		bits |= uint64(code) << bitLen
		bitLen += int(count)
		if bitLen > 64 {
			*(*uint64)(unsafe.Pointer(&output[idx])) = bits
			idx += 8
			bitLen -= 64
			bits = uint64(code) >> (uint64(count) - uint64(bitLen))
		}
		code = extra
		count = extraCount

		bits |= uint64(code) << bitLen
		bitLen += int(count)
		if bitLen > 64 {
			*(*uint64)(unsafe.Pointer(&output[idx])) = bits
			idx += 8
			bitLen -= 64
			bits = uint64(code) >> (uint64(count) - uint64(bitLen))
		}
		if idx >= end {
			break
		}
	}

	buf.idx = idx
	buf.bits = bits
	buf.bitLen = bitLen
	return tokenIdx + 1
}
