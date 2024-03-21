// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

//go:build amd64 && !noasmtest

package deflate

import (
	"unsafe"

	"github.com/intel/fastgo/internal/cpu"
)

func init() {
	optimizedEncodeBytes = func(hist *histogram, data []byte, buf *BitBuf) (num int) {
		switch cpu.ArchLevel {
		case 4:
			num += encodeHuffmansArchV4(hist, data, buf)
		}
		num += encodeBytes(hist, data[num:], buf)
		return num
	}
}

func encodeBytes(hist *histogram, data []byte, buf *BitBuf) (num int) {
	// max bits write per loop = (15 + 15 + 5 + 13 ) = 48
	// 48 / 8 = 6
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
