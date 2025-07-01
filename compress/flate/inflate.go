// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package flate

import (
	"encoding/binary"
)

// inflate represents the internal state of the Intel-optimized DEFLATE decompressor.
// It maintains all necessary state for streaming decompression including bit buffers,
// Huffman tables, and block processing state.
type inflate struct {
	input   []byte // Input data buffer
	bits    uint64 // Bit buffer for handling unaligned bit streams
	bitsLen int32  // Number of valid bits in the bit buffer

	// Overflow handling for output generation
	writeOverflowLits    int32 // Literal overflow count
	writeOverflowLen     int32 // Length overflow count
	copyOverflowLength   int32 // Copy length overflow
	copyOverflowDistance int32 // Copy distance overflow

	// Huffman decoding tables for literals/lengths and distances
	litLenTable largeHuffCodeTable // Table for literal/length symbols (larger symbol space)
	distTable   smallHuffCodeTable // Table for distance symbols (smaller symbol space)

	phase          int32  // Current decompression phase/state
	bfinal         uint32 // DEFLATE block final flag (1 if last block)
	litBlockLength int    // Length of literal (uncompressed) block

	// Header processing buffers and state
	headerBuffered int16               // Number of bytes accumulated in header buffer
	headerBuffer   [328]uint8          // Temporary buffer for header data accumulation
	dynHdr         dynamicHeaderReader // Dynamic header processing context

	roffset int64 // Read offset for position tracking
}

type dynamicHeaderReader struct {
	litAndDistHuff [litLenElems]huffCode
	clcTable       smallHuffCodeTable // Table for decoding codelencode symbols
	codeList       [litLenElems + 2]uint32
	litCount       [maxLitLenCount]uint16
	distCount      [maxHuffTreeDepth + 1]uint16
	litExpandCount [maxLitLenCount]uint16
	nextCode       [maxHuffTreeDepth + 1]uint32
	lenHuffCodes   [litLen - litSymbolsSize]huffCode
}

func (s *inflate) reset() {
	s.input = nil
	s.bits = 0    // Bits buffered to handle unaligned streams
	s.bitsLen = 0 // Bits in readIn
	s.phase = 0
	s.bfinal = 0
	s.litBlockLength = 0
	s.writeOverflowLits = 0
	s.writeOverflowLen = 0
	s.copyOverflowLength = 0
	s.copyOverflowDistance = 0
	s.headerBuffered = 0
	s.roffset = 0
}

const (
	phaseNewBlock = iota
	phaseDecodingHeader
	phaseLitBlock
	phaseHeaderDecoded
	phaseStreamEnd
	phaseFinish
)

func (state *inflate) nextBits(bitCount uint8) uint64 {
	ret := (state.bits) & ((1 << bitCount) - 1)
	state.bits = state.bits >> bitCount
	state.bitsLen -= int32(bitCount)
	return ret
}

func (state *inflate) readBits(bitCount uint8) uint64 {
	state.loadBits()
	return state.nextBits(bitCount)
}

func (state *inflate) loadBits() {
	if state.bitsLen < 57 {
		if len(state.input) >= 8 {
			/* If there is enough space to load a 64 bits, load the data and use
			* that to fill readIn */
			consumed := 8 - uint8(state.bitsLen+7)/8

			temp := binary.LittleEndian.Uint64(state.input)
			state.bits |= temp << state.bitsLen
			state.input = state.input[consumed:]
			state.bitsLen += int32(consumed) * 8
		} else {
			size := int(64-state.bitsLen) / 8

			if len(state.input) < size {
				size = len(state.input)
			}

			for _, b := range state.input[:size] {
				state.bits |= uint64(b) << state.bitsLen
				state.bitsLen += 8
			}
			state.input = state.input[size:]

		}
	}
}

func (state *inflate) tryDecodeHeader() (err error) {
	state.bfinal = uint32(state.readBits(1))
	btype := uint32(state.readBits(2))

	if state.bitsLen < 0 {
		return errEndInput
	}
	switch btype {
	case 0:
		return state.prepareForLitBlock()
	case 1:
		state.setupStaticHeader()
		return nil
	case 2:
		return state.setupDynamicHeader()
	default:
		return errInvalidBlock
	}
}

func (state *inflate) prepareForLitBlock() error {
	state.loadBits()
	bytes := uint8(state.bitsLen / 8)

	if bytes < 4 {
		return errEndInput
	}
	state.bits >>= state.bitsLen % 8
	state.bitsLen = int32(bytes * 8)

	len := uint16(state.bits & 0xFFFF)
	state.bits >>= 16
	nlen := uint16(state.bits & 0xFFFF)
	state.bits >>= 16
	state.bitsLen -= 32

	/* Check if len and nlen match */
	if len != (^nlen & 0xffff) {
		return errInvalidBlock
	}
	rest := state.bitsLen % 8
	if rest != 0 {
		state.bitsLen -= rest
		state.bits >>= uint64(rest)

	}
	state.bits = (state.bits) & ((1 << state.bitsLen) - 1)

	state.litBlockLength = int(len)
	state.phase = phaseLitBlock
	return nil
}

type huffCode struct {
	codeAndLength uint32
}

func (h *huffCode) Set(code uint32, length uint32) {
	h.codeAndLength = code | length<<24
}

const (
	lengthMask = uint32(1<<24 - 1)
	codeMask   = ^lengthMask
)

func (h *huffCode) SetCode(code uint32) {
	// clear code
	h.codeAndLength = codeMask & h.codeAndLength
	// set code
	h.codeAndLength |= code & lengthMask
}

func (h *huffCode) Code() uint32 {
	return h.codeAndLength & lengthMask
}

func (h *huffCode) Length() uint32 {
	return h.codeAndLength >> 24
}

const (
	codeLenCodes     = 19
	huffLen          = 19
	maxLitLenSym     = 512
	litLenElems      = 514
	maxLitLenCodeLen = 21
	maxLitLenCount   = (maxLitLenCodeLen + 2)

	tripleSymFlag  = 0
	doubleSymFlag  = tripleSymFlag + 1
	singleSymFlag  = doubleSymFlag + 1
	defaultSymFlag = tripleSymFlag

	distLen          = 30
	singleSymThresh  = (2 * 1024)
	doubleSymThresh  = (4 * 1024)
	maxHuffTreeDepth = 15
	litLen           = 257 + 29
	litTableSize     = 257

	maxHdrSize = 328
	maxCodeLen = 15

	historySize    = (32 * 1024)
	maxHistoryBits = 15

	maxMatch = 258
	minMatch = 3

	lookAhead = (maxMatch + 31) & ^31
)

func setCodes(table []huffCode, count []uint16) (ret int) {
	var max, code, length uint32
	var nextCode [maxHuffTreeDepth + 1]uint32
	for i := 2; i < maxHuffTreeDepth+1; i++ {
		nextCode[i] = (nextCode[i-1] + uint32(count[i-1])) << 1
	}
	max = nextCode[maxHuffTreeDepth] + uint32(count[maxHuffTreeDepth])
	if max > (1 << maxHuffTreeDepth) {
		return errorNoInvalidBlock
	}
	for i := 0; i < len(table); i++ {
		length = table[i].Length()
		if length == 0 {
			continue
		}
		code = bitReverse2(uint16(nextCode[length]), uint8(length))
		table[i].Set(code, length)
		nextCode[length]++
	}
	return 0
}

func (state *inflate) rOffset(inputSize, bitsLen int) {
	start := inputSize*8 + bitsLen
	end := len(state.input)*8 + int(state.bitsLen)
	size := (start - end) / 8
	state.roffset += int64(size)
}

func (state *inflate) readHeader() (err error) {
	bits := state.bits
	bitsLen := state.bitsLen
	input := state.input
	phase := state.phase

	var tempLen int
	if phase == phaseDecodingHeader {
		/* Setup so readHeader decodes data in tmpInBuffer */
		copySize := int(maxHdrSize - state.headerBuffered)
		if copySize > len(state.input) {
			copySize = len(state.input)
		}

		copy(state.headerBuffer[state.headerBuffered:], state.input[:copySize])
		tempLen = copySize + int(state.headerBuffered)
		state.input = state.headerBuffer[:tempLen]
	}

	err = state.tryDecodeHeader()

	if phase == phaseDecodingHeader {
		read := tempLen - len(state.input) - int(state.headerBuffered)
		state.input = input[read:]
	}

	if err == errEndInput {
		state.bits = bits
		state.bitsLen = bitsLen
		size := copy(state.headerBuffer[state.headerBuffered:], input)
		state.headerBuffered += int16(size)
		state.input = state.input[:0]
		state.phase = phaseDecodingHeader
	} else {
		state.headerBuffered = 0
	}
	return err
}
