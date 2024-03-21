// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package deflate

import (
	"unsafe"
)

// histogram is used to count the frequency of token occurrences.
//
// For performance reasons, we will also use the same memory to hold the huffman code table
// generated from the histogram data. You can assume that histogram and the following huffcodeTable
// are equivalent.
//
//	type huffcodeTable struct {
//		distanceCodes [31]distCode
//		literalCodes  [256 + 1 + (maxMatchLength - minMatchLength + 1)]litlenCode
//	}
//	type distCode struct {
//		code           [2]uint8
//		extraBitsCount uint8
//		length         uint8
//	}
//	type litlenCode struct {
//		code   [3]uint8
//		length uint8
//	}
type histogram struct {
	distanceCodes [31]uint32                                              // Distance Codes |0-15 code|16-23 extraCount|24-31 count|
	literalCodes  [256 + 1 + (maxMatchLength - minMatchLength + 1)]uint32 // Literal Codes  lit[0,255] | end_of_block[256] | len[3,258]
}

func (h *histogram) reset() {
	for i := range h.distanceCodes {
		h.distanceCodes[i] = 0
	}
	for i := range h.literalCodes {
		h.literalCodes[i] = 0
	}
}

func (h *histogram) reduceCounts() {
	curr := 265
	idx := 265
	for bits := 1; bits <= 5; bits++ {
		for i := 0; i < 4; i++ {
			val := uint32(0)
			for j := 0; j < 1<<bits; j++ {
				val += h.literalCodes[curr]
				curr++
			}
			h.literalCodes[idx] = val
			idx++
		}
	}
	h.literalCodes[285] = h.literalCodes[512]
}

func (h *histogram) expandCodes() {
	var origin [285 - 265 + 1]uint32
	copy(origin[:], h.literalCodes[265:])
	offset := 0
	idx := 0
	for bits := 1; bits <= 5; bits++ {
		for i := 0; i < 4; i++ {
			code := origin[idx]
			length := code >> 24
			code = code & 0xff_ff_ff
			idx++
			for j := 0; j < 1<<bits; j++ {
				ncode := code | uint32(j)<<length
				nlength := length + uint32(bits)
				h.setLitCode(uint32(265+offset), ncode, nlength)
				offset++
			}
		}
	}
	// write extraCount
	h.literalCodes[512] = origin[20]
	x := 4
	for i := uint32(1); i <= 13; i++ {
		h.distanceCodes[x] |= i << 16
		h.distanceCodes[x+1] |= i << 16
		x += 2
	}
	h.distanceCodes[30] = 0
}

func (h *histogram) litCode(litlen uint32) (code uint32, count uint32) {
	code = h.literalCodes[litlen]
	count = code >> 24
	code = code & 0xff_ff_ff
	return
}

func (h *histogram) setLitCode(litlen, code, count uint32) {
	h.literalCodes[litlen] = code | count<<24
}

func (h *histogram) distCode(dist uint32) (code, count, extraCount uint32) {
	// the code should be like this:
	// 	if dist >= 31 {
	// 		code, count = h.litCode(dist - 31)
	// 		return
	// 	}
	// 	code = h.DistanceCodes[dist]
	// for the performance reason, we have the code below:
	code = *(*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(&h.distanceCodes[0])) + uintptr(dist*4)))
	count = code >> 16
	extraCount = count & 0xff
	count = count >> 8
	code = code & 0xff_ff
	return
}
