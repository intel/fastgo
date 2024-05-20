// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

//go:build !amd64 || noasmtest
// +build !amd64 noasmtest

package deflate

func optimizedEncodeTokens(hist *histogram, tokens []token, buf *BitBuf) (tokenNum int) {
	return encodeTokens(hist, tokens, buf)
}

func encodeTokens(hist *histogram, tokens []token, buf *BitBuf) (tokenNum int) {
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
			output[idx+0] = byte(bits)
			output[idx+1] = byte(bits >> 8)
			output[idx+2] = byte(bits >> 16)
			output[idx+3] = byte(bits >> 24)
			output[idx+4] = byte(bits >> 32)
			output[idx+5] = byte(bits >> 40)
			output[idx+6] = byte(bits >> 48)
			output[idx+7] = byte(bits >> 56)

			idx += 8
			bitLen -= 64
			bits = uint64(code) >> (uint64(count) - uint64(bitLen))
		}
		code, count, extraCount = hist.distCode(dist)

		bits |= uint64(code) << bitLen
		bitLen += int(count)
		if bitLen > 64 {
			output[idx+0] = byte(bits)
			output[idx+1] = byte(bits >> 8)
			output[idx+2] = byte(bits >> 16)
			output[idx+3] = byte(bits >> 24)
			output[idx+4] = byte(bits >> 32)
			output[idx+5] = byte(bits >> 40)
			output[idx+6] = byte(bits >> 48)
			output[idx+7] = byte(bits >> 56)

			idx += 8
			bitLen -= 64
			bits = uint64(code) >> (uint64(count) - uint64(bitLen))
		}
		code = extra
		count = extraCount

		bits |= uint64(code) << bitLen
		bitLen += int(count)
		if bitLen > 64 {
			output[idx+0] = byte(bits)
			output[idx+1] = byte(bits >> 8)
			output[idx+2] = byte(bits >> 16)
			output[idx+3] = byte(bits >> 24)
			output[idx+4] = byte(bits >> 32)
			output[idx+5] = byte(bits >> 40)
			output[idx+6] = byte(bits >> 48)
			output[idx+7] = byte(bits >> 56)

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
