// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package deflate

import "unsafe"

type BitBuf struct {
	output []byte
	idx    int
	bits   uint64
	bitLen int
}

func (b *BitBuf) reset() {
	b.idx = 0
	b.bits = 0
	b.bitLen = 0
}

func (b *BitBuf) writeEmptyBlock() {
	b.WriteBit(0b000, 3)
	b.flushLastByte()
	b.output[b.idx+0] = 0x00
	b.output[b.idx+1] = 0x00
	b.output[b.idx+2] = 0xff
	b.output[b.idx+3] = 0xff
	b.idx += 4
}

func (b *BitBuf) writeFinalEmptyBlock() {
	b.WriteBit(0b001, 3)
	b.flushLastByte()
	b.output[b.idx+0] = 0x00
	b.output[b.idx+1] = 0x00
	b.output[b.idx+2] = 0xff
	b.output[b.idx+3] = 0xff
	b.idx += 4
}

func (b *BitBuf) flushLastByte() {
	if b.bitLen == 0 {
		return
	}
	for b.bitLen > 0 {
		b.output[b.idx] = byte(b.bits & 0xff)
		b.bitLen -= 8
		b.bits >>= 8
		b.idx++
	}
	b.bitLen = 0
	b.bits = 0
}

func (b *BitBuf) Sync() {
	rest := b.bitLen % 8
	bytes := b.bitLen / 8
	for i := 0; i < bytes; i++ {
		b.output[b.idx+i] = byte(b.bits)
		b.bits >>= 8
	}
	b.idx += bytes
	b.bitLen = rest
}

func (b *BitBuf) WriteBit(code uint16, count uint8) {
	b.bits |= uint64(code) << b.bitLen
	b.bitLen += int(count)
	if b.bitLen > 64 {
		*(*uint64)(unsafe.Pointer(&b.output[b.idx])) = b.bits
		b.idx += 8
		b.bitLen -= 64
		b.bits = uint64(code) >> (uint64(count) - uint64(b.bitLen))
	}
}
