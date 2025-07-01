// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

// This file implements DEFLATE header parsing and Huffman table construction
// for Intel-optimized decompression. It handles both static and dynamic
// Huffman block headers as specified in RFC 1951.
package flate

import (
	"encoding/binary"
	"math/bits"
)

// Huffman table lookup constants
const (
	litLenLookupBits = 12 // Bits used for literal/length table lookup
	distLookupBits   = 10 // Bits used for distance table lookup

	invalidSymbolValue = 0x1FFF   // Marker for invalid symbol
	invalidCodeValue   = 0xFFFFFF // Marker for invalid code
)

// setupStaticHeader configures the decompressor for a static Huffman block.
// Static blocks use predefined Huffman tables as specified in RFC 1951.
func (state *inflate) setupStaticHeader() {
	state.litLenTable = staticLitHuffCode // Use predefined literal/length table
	state.distTable = staticDistHuffCode  // Use predefined distance table
	state.phase = phaseHeaderDecoded      // Move to data decompression phase
}

// codeLenCodes processes the code length codes section of a dynamic Huffman header.
// This builds the Huffman table used to decode the main literal/length and distance tables.
func (state *inflate) codeLenCodes(hclen int) error {
	var codeHuff [codeLenCodes]huffCode
	var codeCount [16]uint16
	// Code length order as defined in RFC 1951 section 3.2.7
	codeLengthOrder := [codeLenCodes]uint8{
		0x10, 0x11, 0x12, 0x00, 0x08, 0x07, 0x09, 0x06,
		0x0a, 0x05, 0x0b, 0x04, 0x0c, 0x03, 0x0d, 0x02, 0x0e, 0x01, 0x0f,
	}

	// /* Create the code huffman code for decoding the lit/len and dist huffman codes */
	for i := 0; i < 4; i++ {
		code := &codeHuff[codeLengthOrder[i]]
		length := uint32(state.nextBits(3))
		code.Set(0, length)
		codeCount[length] += 1
	}

	state.loadBits()

	for i := 4; i < int(hclen+4); i++ {
		code := &codeHuff[codeLengthOrder[i]]
		length := uint32(state.nextBits(3))
		code.Set(0, length)
		codeCount[length] += 1
	}

	if state.bitsLen < 0 {
		return errEndInput
	}

	if setCodes(codeHuff[:], codeCount[:]) != 0 {
		return errInvalidBlock
	}

	state.dynHdr.clcTable.GenerateForHeader(codeHuff[:], codeCount[:], codeLenCodes)
	return nil
}

func (state *inflate) setupDynamicHeader() error {
	state.dynHdr.litExpandCount = [maxLitLenCount]uint16{}
	state.dynHdr.litCount = [maxLitLenCount]uint16{}
	state.dynHdr.distCount = [maxHuffTreeDepth + 1]uint16{}
	state.dynHdr.litAndDistHuff = [litLenElems]huffCode{}
	ctx := &state.dynHdr
	var hclen, hdist, hlit uint64
	var multisym uint32 = defaultSymFlag

	if state.bfinal != 0 && len(state.input) <= singleSymThresh {
		multisym = singleSymFlag
	} else if state.bfinal != 0 && len(state.input) <= doubleSymThresh {
		multisym = doubleSymFlag
	}
	state.loadBits()
	if state.bitsLen < 14 {
		return errEndInput
	}

	hlit = state.nextBits(5)
	hdist = state.nextBits(5)
	hclen = state.nextBits(4)

	if hlit > 29 || hdist > 29 || hclen > 15 {
		return errInvalidBlock
	}

	err := state.codeLenCodes(int(hclen))
	if err != nil {
		return err
	}

	/* Decode the lit/len and dist huffman codes using the code huffman code */
	err = state.readLitDistLens(ctx, int(hdist), int(hlit))
	if err != nil {
		return err
	}
	if state.bitsLen < 0 {
		return errEndInput
	}

	if setCodes(ctx.litAndDistHuff[litLen:litLen+distLen], ctx.distCount[:]) != 0 {
		return errInvalidBlock
	}

	state.distTable.genForDists(ctx.litAndDistHuff[litLen:distLen+litLen], ctx.distCount[:], distLen)
	err = ctx.setAndExpandLitLenHuffCode()
	if err != nil {
		return err
	}
	state.litLenTable.genForLitLen(ctx, multisym)

	state.phase = phaseHeaderDecoded
	return nil
}

func (ctx *dynamicHeaderReader) setAndExpandLitLenHuffCode() error {
	// Setup for calculating huffman codes
	countTotal := uint32(0)
	countTmp := uint32(ctx.litExpandCount[1])
	ctx.nextCode[0] = 0
	ctx.nextCode[1] = 0
	ctx.litExpandCount[0] = 0
	ctx.litExpandCount[1] = 0

	for i := 1; i < maxHuffTreeDepth; i++ {
		countTotal = uint32(ctx.litCount[i]) + countTmp + countTotal
		countTmp = uint32(ctx.litExpandCount[i+1])
		ctx.litExpandCount[i+1] = uint16(countTotal)
		ctx.nextCode[i+1] = (ctx.nextCode[i] + uint32(ctx.litCount[i])) << 1
	}

	countTmp = uint32(ctx.litCount[maxHuffTreeDepth]) + countTmp

	for i := maxHuffTreeDepth; i < maxLitLenCount-1; i++ {
		countTotal = countTmp + countTotal
		countTmp = uint32(ctx.litExpandCount[i+1])
		ctx.litExpandCount[i+1] = uint16(countTotal)
	}

	max := ctx.nextCode[maxHuffTreeDepth] + uint32(ctx.litCount[maxHuffTreeDepth])

	if max > (1 << maxHuffTreeDepth) {
		return errInvalidBlock
	}

	copy(ctx.litCount[:maxLitLenCount], ctx.litExpandCount[:])

	copy(ctx.lenHuffCodes[:], ctx.litAndDistHuff[litSymbolsSize:litLen])

	// trigger memclr
	temp := ctx.litAndDistHuff[litSymbolsSize:litLenElems]
	for i := range temp {
		temp[i] = huffCode{}
	}
	ctx.calcCodeForLit()
	ctx.expandLenCodes()
	return nil
}

func (ctx *dynamicHeaderReader) calcCodeForLit() {
	// Calculate code corresponding to a given literal symbol
	for i := 0; i < litSymbolsSize; i++ {
		hcode := &ctx.litAndDistHuff[i]
		codeLen := hcode.Length()
		if codeLen == 0 {
			continue
		}

		code := bitReverse2(uint16(ctx.nextCode[codeLen]), uint8(codeLen))
		insertIndex := ctx.litExpandCount[codeLen]
		ctx.codeList[insertIndex] = uint32(i)
		ctx.litExpandCount[codeLen]++
		hcode.Set(code, codeLen)
		ctx.nextCode[codeLen] += 1
	}
}

// store len + bits to table
func (ctx *dynamicHeaderReader) expandLenCodes() {
	var (
		expandsIdx  int
		lenSym      int
		extraCount  int
		lenSize     int
		code        uint32
		expandLen   uint32
		insertIndex uint32
		extra       int
	)
	expandsIdx = litSymbolsSize
	// Calculate code corresponding to a given len symbol
	for lenSym = 0; lenSym < litLen-litSymbolsSize; lenSym++ {
		extraCount = int(rfcLookupTable.LenExtraBitCount[lenSym])
		lenSize = (1 << extraCount)

		codeLen := ctx.lenHuffCodes[lenSym].Length()
		if codeLen == 0 {
			expandsIdx += lenSize
			continue
		}

		code = bitReverse2(uint16(ctx.nextCode[codeLen]), uint8(codeLen))
		expandLen = codeLen + uint32(extraCount)
		ctx.nextCode[codeLen] += 1
		insertIndex = uint32(ctx.litExpandCount[expandLen])
		ctx.litExpandCount[expandLen] += uint16(lenSize)

		for extra = 0; extra < lenSize; extra++ {
			ctx.codeList[int(insertIndex)+extra] = uint32(expandsIdx)
			ctx.litAndDistHuff[expandsIdx].Set(code|(uint32(extra)<<codeLen), expandLen)
			expandsIdx++
		}
	}
}

const (
	litSymbolsSize    = 257                               // number of literal symbols (0-255 ,256)
	lenSymbolsSize    = 29                                // number of length  symbols
	distSymbolsSize   = 30                                // number of distance  symbols
	litLenSymbolsSize = (litSymbolsSize + lenSymbolsSize) // number of Literal/Length symbols
)

func bitReverse2(code uint16, length uint8) uint32 {
	return uint32(bits.Reverse16(code)) >> (16 - length)

	// var bitrev uint32
	// bitrev = uint32(bitrevTable[bits>>8])
	// bitrev |= uint32(bitrevTable[bits&0xFF]) << 8

	// return bitrev >> (16 - length)
}

func (state *inflate) readLitDistLens(ctx *dynamicHeaderReader, hdist, hlit int) (err error) {
	var (
		bits    = state.bits
		bitsLen = state.bitsLen
		input   = state.input
	)
	count := ctx.litCount[:]
	huffs := ctx.litAndDistHuff[:litLen+hdist+1]
	var curr, prev, end int
	end = litLen + hdist + 1
	prev = -1
	for curr < len(huffs) {
		var symbol uint16
		{
			var nextBits, nextSym uint16
			var bitCount, bitMask uint32

			if bitsLen <= maxCodeLen {
				if len(input) >= 8 {
					/* If there is enough space to load a 64 bits, load the data and use
					* that to fill bits */
					consumed := 8 - uint8(bitsLen+7)/8
					temp := binary.LittleEndian.Uint64(input)
					bits |= temp << bitsLen
					input = input[consumed:]
					bitsLen += int32(consumed) * 8

				} else {
					size := int(64-bitsLen) / 8
					if len(input) < size {
						size = len(input)
					}

					for _, b := range input[:size] {
						bits |= uint64(b) << bitsLen
						bitsLen += 8
					}
					input = input[size:]
				}
			}
			nextBits = uint16(bits & ((1 << distLookupBits) - 1))

			/* nextSym is a possible symbol decoded from nextBits. If bit 15 is 0,
			* nextCode is a symbol. Bits 9:0 represent the symbol, and bits 14:10
			* represent the length of that symbols huffman code. If nextSym is not
			* a symbol, it provides a hint of where the large symbols containing
			* this code are located. Note the hint is at largest the location the
			* first actual symbol in the long code list.*/
			nextSym = ctx.clcTable.ShortCodeLookup[nextBits]

			if (nextSym & smallFlagBit) == 0 {
				/* Return symbol found if nextCode is a complete huffman code
				* and shift in buffer over by the length of the nextCode */
				bitCount = uint32(nextSym >> smallShortCodeLenOffset)
				bits >>= bitCount
				bitsLen -= int32(bitCount)

				if bitCount == 0 {
					nextSym = invalidSymbolValue
				}
				symbol = nextSym & smallShortSymMask

			} else {
				/* If a symbol is not found, perform a linear search of the long code
				* list starting from the hint in nextSym */
				bitMask = uint32(nextSym-smallFlagBit) >> smallShortCodeLenOffset
				bitMask = (1 << bitMask) - 1
				nextBits = uint16(uint32(bits) & bitMask)
				nextSym = ctx.clcTable.LongCodeLookup[(nextSym&smallShortSymMask)+
					(nextBits>>distLookupBits)]
				bitCount = uint32(nextSym) >> smallLongCodeLenOffset
				bits >>= bitCount
				bitsLen -= int32(bitCount)
				symbol = nextSym & smallLongSymMask
			}
		}
		if bitsLen < 0 {
			if curr > 256 && huffs[256].Length() <= 0 {
				err = errInvalidBlock

				goto END
			}
			err = errEndInput

			goto END
		}
		switch {
		case symbol < 16:
			/* If a length is found, update the current lit/len/dist
			 * to have length symbol */
			if curr == int(litTableSize+hlit) {
				/* Switch code upon completion of litLen ctx.HuffCode */
				curr = litLen
				count = ctx.distCount[:]
			}
			count[symbol]++
			huffs[curr].Set(0, uint32(symbol))
			prev = curr
			curr++
			if symbol == 0 || // No symbol
				(prev >= int(litTableSize+hlit)) || // Dist ctx.HuffCode
				(prev < 264) { // Lit/Len with no extra bits
				continue
			}
			extraCount := int(rfcLookupTable.LenExtraBitCount[prev-litTableSize])
			ctx.litExpandCount[symbol]--
			ctx.litExpandCount[int(symbol)+extraCount] += (1 << extraCount)
		case symbol == 16:
			/* If a repeat length is found, update the next repeat
			 * length lit/len/dist elements to have the value of the
			 * repeated length */
			if len(input) >= 8 {
				/* If there is enough space to load a 64 bits, load the data and use
				* that to fill bits */
				consumed := 8 - uint8(bitsLen+7)/8
				temp := binary.LittleEndian.Uint64(input)
				bits |= temp << bitsLen
				input = input[consumed:]
				bitsLen += int32(consumed) * 8

			} else {
				size := int(64-bitsLen) / 8
				if len(input) < size {
					size = len(input)
				}

				for _, b := range input[:size] {
					bits |= uint64(b) << bitsLen
					bitsLen += 8
				}
				input = input[size:]
			}
			ret := (bits) & ((1 << 2) - 1)
			bits = bits >> 2
			bitsLen -= int32(2)

			i := int(3 + ret)

			if curr+i > end || prev == -1 {
				err = errInvalidBlock
				goto END
			}

			repCode := huffs[prev]
			for j := 0; j < i; j++ {
				if curr == int(litTableSize+hlit) {
					/* Switch code upon completion of litLen ctx.HuffCode */
					curr = litLen
					count = ctx.distCount[:]
				}

				huffs[curr] = repCode
				count[repCode.Length()]++
				prev = curr
				curr++

				if repCode.Length() == 0 || // No symbol
					(prev >= int(litTableSize+hlit)) || // Dist ctx.HuffCode
					(prev < +264) { // Lit/Len with no extra
					continue
				}

				extraCount := int(rfcLookupTable.LenExtraBitCount[prev-litTableSize])
				ctx.litExpandCount[repCode.Length()]--
				ctx.litExpandCount[int(repCode.Length())+extraCount] += (1 << extraCount)

			}
		case symbol == 17:
			/* If a repeat zeroes if found, update then next
			 * repeated zeroes length lit/len/dist elements to have
			 * length 0. */

			if len(input) >= 8 {
				/* If there is enough space to load a 64 bits, load the data and use
				* that to fill bits */
				consumed := 8 - uint8(bitsLen+7)/8
				temp := binary.LittleEndian.Uint64(input)
				bits |= temp << bitsLen
				input = input[consumed:]
				bitsLen += int32(consumed) * 8

			} else {
				size := int(64-bitsLen) / 8
				if len(input) < size {
					size = len(input)
				}

				for _, b := range input[:size] {
					bits |= uint64(b) << bitsLen
					bitsLen += 8
				}
				input = input[size:]
			}
			ret := (bits) & ((1 << 3) - 1)
			bits = bits >> 3
			bitsLen -= int32(3)

			i := int(3 + ret)

			curr = curr + i
			prev = curr - 1

			if &count[0] != &ctx.distCount[0] && curr > int(litTableSize+hlit) {
				/* Switch code upon completion of litLen ctx.HuffCode */
				curr += int(litLen - litTableSize - hlit)
				count = ctx.distCount[:]
				if curr > litLen {
					prev = curr - 1
				}
			}
		case symbol == 18:
			/* If a repeat zeroes if found, update then next
			 * repeated zeroes length lit/len/dist elements to have
			 * length 0. */
			if len(input) >= 8 {
				/* If there is enough space to load a 64 bits, load the data and use
				* that to fill bits */
				consumed := 8 - uint8(bitsLen+7)/8
				temp := binary.LittleEndian.Uint64(input)
				bits |= temp << bitsLen
				input = input[consumed:]
				bitsLen += int32(consumed) * 8

			} else {
				size := int(64-bitsLen) / 8
				if len(input) < size {
					size = len(input)
				}

				for _, b := range input[:size] {
					bits |= uint64(b) << bitsLen
					bitsLen += 8
				}
				input = input[size:]
			}
			ret := (bits) & ((1 << 7) - 1)
			bits = bits >> 7
			bitsLen -= int32(7)

			i := int(11 + ret)

			curr = curr + i
			prev = curr - 1

			if &count[0] != &ctx.distCount[0] && curr > int(litTableSize+hlit) {
				/* Switch code upon completion of litLen ctx.HuffCode */
				curr += int(litLen - litTableSize - hlit)
				count = ctx.distCount[:]
				if curr > litLen {
					prev = curr - 1
				}
			}
		default:
			err = errInvalidBlock
			goto END
		}
	}

	if curr > end || huffs[256].Length() <= 0 {
		err = errInvalidBlock
		goto END
	}
END:

	state.bits = bits
	state.bitsLen = bitsLen
	state.input = input

	return err
}

const (
	distSymOffset      = 0
	distSymLen         = 5
	distSymMask        = ((1 << distSymLen) - 1)
	distSymExtraOffset = 5
	distSymExtraLen    = 4
	distSymExtraMask   = ((1 << distSymExtraLen) - 1)
	distSymLenOffset   = smallShortCodeLenOffset
)

func (t *smallHuffCodeTable) genForDists(codes []huffCode, count []uint16, maxSymbol uint32) {
	var countTotal, countTotalTmp [17]uint32

	for i := 2; i < 17; i++ {
		countTotal[i] = countTotal[i-1] + uint32(count[i-1])
	}
	copy(countTotalTmp[:], countTotal[:])

	codeListLen := countTotal[16]
	if codeListLen == 0 {
		return
	}
	var codeList [distLen + 2]uint32 /* The +2 is for the extra codes in the static header */
	for i, code := range codes {
		codeLength := code.Length()
		if codeLength == 0 {
			continue
		}
		insertIndex := countTotalTmp[codeLength]
		codeList[insertIndex] = uint32(i)
		countTotalTmp[codeLength]++
	}

	lastLength := codes[codeList[0]].Length()
	if lastLength > distLookupBits {
		lastLength = distLookupBits + 1
	}
	copySize := (1 << (lastLength - 1))

	for ; lastLength <= distLookupBits; lastLength++ {
		copy(t.ShortCodeLookup[copySize:], t.ShortCodeLookup[:copySize])
		copySize *= 2

		for k := countTotal[lastLength]; k < countTotal[lastLength+1]; k++ {
			idx := codeList[k]
			if idx >= maxSymbol {
				t.ShortCodeLookup[codes[idx].Code()] = uint16(codes[idx].Length())
				continue
			}
			t.ShortCodeLookup[codes[idx].Code()] = uint16(idx |
				uint32(rfcLookupTable.DistExtraBitCount[idx])<<distSymExtraOffset |
				(codes[idx].Length())<<smallShortCodeLenOffset)
		}
	}

	longCodeStart := countTotal[distLookupBits+1]
	longCodeList := codeList[longCodeStart:]
	longCodeLength := codeListLen - longCodeStart
	var longCodeLookupLength uint32
	var tempCodeList [1 << (15 - distLookupBits)]uint16
	for i := uint32(0); i < longCodeLength; i++ {
		if codes[longCodeList[i]].Code() == 0xFFFF {
			continue
		}

		maxLength := codes[longCodeList[i]].Length()
		firstBits := uint16(codes[longCodeList[i]].Code() & ((1 << distLookupBits) - 1))

		tempCodeList[0] = uint16(longCodeList[i])
		tempCodeLength := 1

		for j := i + 1; j < longCodeLength; j++ {
			if (codes[longCodeList[j]].Code() &
				((1 << distLookupBits) - 1)) == uint32(firstBits) {
				maxLength = codes[longCodeList[j]].Length()
				tempCodeList[tempCodeLength] = uint16(longCodeList[j])
				tempCodeLength++
			}
		}
		for x := longCodeLookupLength; x < longCodeLookupLength+2*(1<<(maxLength-distLookupBits)); x++ {
			t.LongCodeLookup[x] = 0
		}

		for j := 0; j < int(tempCodeLength); j++ {
			sym := tempCodeList[j]
			codeLength := codes[sym].Length()
			longBits := uint16(codes[sym].Code() >> distLookupBits)
			minIncrement := uint16(1 << (codeLength - distLookupBits))
			for ; longBits < (1 << (maxLength - distLookupBits)); longBits += minIncrement {
				if sym > uint16(maxSymbol) {
					t.LongCodeLookup[longCodeLookupLength+uint32(longBits)] = uint16(codeLength)
					continue
				}
				t.LongCodeLookup[longCodeLookupLength+uint32(longBits)] = uint16(uint32(sym) |
					uint32(rfcLookupTable.DistExtraBitCount[sym])<<distSymExtraOffset |
					(codeLength << smallLongCodeLenOffset))
			}
			codes[sym].SetCode(0xFFFF)
		}
		t.ShortCodeLookup[firstBits] = uint16(longCodeLookupLength |
			(maxLength << smallShortCodeLenOffset) | smallFlagBit)
		longCodeLookupLength += 1 << (maxLength - distLookupBits)
	}
}
