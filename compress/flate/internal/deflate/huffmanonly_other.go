// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

//go:build !amd64 || noasmtest
// +build !amd64 noasmtest

// This file provides fallback implementations for non-AMD64 architectures
// or when assembly tests are disabled. It ensures the package works on all
// platforms while Intel optimizations are only available on supported hardware.
package deflate

// init sets up the standard encoding function for non-optimized platforms.
// This ensures consistent behavior across all supported architectures.
func init() {
	optimizedEncodeBytes = encodeBytes
}

func encodeBytes(hist *histogram, data []byte, buf *BitBuf) (num int) {
	// max bits write per loop = (15 + 15 + 5 + 13 ) = 48
	// 48 / 8 = 6
	buf.Sync()
	output := buf.output
	end := len(output) - 16
	var (
		idx    int    = buf.idx
		bits   uint64 = buf.bits
		bitLen int    = buf.bitLen
	)
	if end <= idx {
		return 0
	}
	endOfData := len(data) - 3
	for num < endOfData {
		lit := data[num]
		lit1 := data[num+1]
		lit2 := data[num+2]
		num += 3
		code, count := hist.litCode(uint32(lit))
		bits |= uint64(code) << bitLen
		bitLen += int(count)

		code, count = hist.litCode(uint32(lit1))
		bits |= uint64(code) << bitLen
		bitLen += int(count)

		code, count = hist.litCode(uint32(lit2))
		bits |= uint64(code) << bitLen
		bitLen += int(count)

		size := bitLen / 8
		for i := 0; i < size; i++ {
			output[idx+i] = byte(bits)
			bits >>= 8
		}

		idx += size
		bitLen = bitLen % 8

		if idx >= end {
			buf.idx = idx
			buf.bits = bits
			buf.bitLen = bitLen
			return num
		}
	}
	for num < len(data) {
		lit := data[num]
		code, count := hist.litCode(uint32(lit))
		bits |= uint64(code) << bitLen
		bitLen += int(count)
		num++
	}

	size := bitLen / 8
	for i := 0; i < size; i++ {
		output[idx+i] = byte(bits)
		bits >>= 8
	}

	idx += size
	bitLen = bitLen % 8

	code, count := hist.litCode(uint32(256))
	bits |= uint64(code) << bitLen
	bitLen += int(count)

	buf.idx = idx
	buf.bits = bits
	buf.bitLen = bitLen
	return num
}
