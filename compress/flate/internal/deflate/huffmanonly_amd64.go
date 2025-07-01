// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

//go:build amd64 && !noasmtest
// +build amd64,!noasmtest

// This file contains Intel AMD64-specific optimizations for Huffman-only compression.
// It provides assembly-accelerated implementations that are automatically selected
// based on the detected CPU architecture level.
package deflate

import (
	"unsafe"

	"github.com/intel/fastgo/internal/cpu"
)

// init sets up the optimized encoding function based on CPU capabilities.
// It selects the appropriate assembly implementation for maximum performance.
func init() {
	optimizedEncodeBytes = func(hist *histogram, data []byte, buf *BitBuf) (num int) {
		switch cpu.ArchLevel {
		case 4:
			// Use Level 4 architecture optimizations (highest performance)
			num += encodeHuffmansArchV4(hist, data, buf)
		}
		// Fall back to standard implementation for remaining data
		num += encodeBytes(hist, data[num:], buf)
		return num
	}
}

// encodeBytes provides the fallback Huffman encoding implementation.
// This is used when assembly optimizations are not available or for remaining
// data after optimized processing.
func encodeBytes(hist *histogram, data []byte, buf *BitBuf) (num int) {
	// Maximum bits written per loop iteration = (15 + 15 + 5 + 13) = 48 bits
	// This ensures we don't overflow the bit buffer (48 / 8 = 6 bytes maximum)
	buf.Sync()
	output := buf.output
	end := len(output) - 16
	outputPtr := unsafe.Pointer(&buf.output[0])
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

		*(*uint64)(unsafe.Pointer(uintptr(outputPtr) + uintptr(idx))) = bits
		size := bitLen / 8
		idx += size
		bitLen = bitLen % 8
		bits >>= size * 8

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

	*(*uint64)(unsafe.Pointer(uintptr(outputPtr) + uintptr(idx))) = bits
	size := bitLen / 8
	idx += size
	bitLen = bitLen % 8
	bits >>= size * 8

	code, count := hist.litCode(uint32(256))
	bits |= uint64(code) << bitLen
	bitLen += int(count)

	buf.idx = idx
	buf.bits = bits
	buf.bitLen = bitLen
	return num
}

func encodeHuffmansArchV4(hist *histogram, input []byte, buf *BitBuf) int
