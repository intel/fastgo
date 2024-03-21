// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package flate

const (
	smallShortSymLen        = 9
	smallShortSymMask       = ((1 << smallShortSymLen) - 1)
	smallLongSymLen         = 9
	smallLongSymMask        = ((1 << smallLongSymLen) - 1)
	smallShortCodeLenOffset = 11
	smallLongCodeLenOffset  = 10
	smallFlagBitOffset      = 10
	smallFlagBit            = (1 << smallFlagBitOffset)
)

/* Small lookup table for decoding huffman codes */
type smallHuffCodeTable struct {
	ShortCodeLookup [1 << (10)]uint16
	LongCodeLookup  [80]uint16
}

// type smallHuffCodeTable = common.SmallHuffCodeTable

// ShortCode
// | Name        | Bits    |
// | ----------- | ------- |
// | Symbol      | 0-7     |
// | Flag        | 10      |
// | Code Length / Max Length | 12 - 16 |
func (t *smallHuffCodeTable) GenerateForHeader(codes []huffCode, count []uint16, maxSymbol uint32) {
	var countTotal, countTotalTmp [17]uint32
	shortCodeLookup := t.ShortCodeLookup[:]

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

	// /* Initialize shortCodeLookup, so invalid lookups process data */
	for ; lastLength <= distLookupBits; lastLength++ {
		copy(shortCodeLookup[copySize:], shortCodeLookup[:copySize])
		copySize *= 2

		for k := countTotal[lastLength]; k < countTotal[lastLength+1]; k++ {
			idx := codeList[k]

			if idx >= maxSymbol {
				continue
			}
			/* Set lookup table to return the current symbol concatenated
			 * with the code length when the first DECODE_LENGTH bits of the
			 * address are the same as the code for the current symbol. The
			 * first 9 bits are the code, bits 14:10 are the code length,
			 * bit 15 is a flag representing this is a symbol*/
			shortCodeLookup[codes[idx].Code()] = uint16(idx | (codes[idx].Length())<<smallShortCodeLenOffset)
		}
	}

	longCodeStart := countTotal[distLookupBits+1]
	longCodeList := codeList[longCodeStart:]
	longCodeLength := codeListLen - longCodeStart
	var longCodeLookupLength uint32
	var tempCodeList [1 << (15 - distLookupBits)]uint16
	for i := uint32(0); i < longCodeLength; i++ {
		/*Set the look up table to point to a hint where the symbol can be found
		 * in the list of long codes and add the current symbol to the list of
		 * long codes. */
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
				if maxLength < codes[longCodeList[j]].Length() {
					maxLength = codes[longCodeList[j]].Length()
				}
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
				t.LongCodeLookup[longCodeLookupLength+uint32(longBits)] = uint16(uint32(sym) | (codeLength << smallLongCodeLenOffset))
			}
			codes[sym].SetCode(0xFFFF)
		}
		t.ShortCodeLookup[firstBits] = uint16(longCodeLookupLength |
			(maxLength << smallShortCodeLenOffset) | smallFlagBit)
		longCodeLookupLength += 1 << (maxLength - distLookupBits)
	}
}

func indexToSym(index uint32) uint32 {
	if index != 513 {
		return index
	} else {
		return 512
	}
}

const (
	largeShortSymLen        = 25
	largeShortSymMask       = (1 << largeShortSymLen) - 1
	largeLongSymLen         = 10
	largeLongSymMask        = (1 << largeLongSymLen) - 1
	largeShortCodeLenOffset = 28
	largeLongCodeLenOffset  = 10
	largeFlagBitOffset      = 25
	largeFlagBit            = 1 << largeFlagBitOffset
	largeSymCountOffset     = 26
	largeSymCountLen        = 2
	largeSymCountMask       = (1 << largeSymCountLen) - 1
	largeShortMaxLenOffset  = 26
)

type largeHuffCodeTable struct {
	shortCodeLookup [1 << (12)]uint32 // sym1|sym2|sym3|code_len|sym_size
	longCodeLookup  [1264]uint16
}

func (t *largeHuffCodeTable) genForLitLen(ctx *dynamicHeaderReader, multisym uint32) {
	codeListLen := uint32(ctx.litCount[maxLitLenCount-1])
	if codeListLen == 0 {
		for i := range t.shortCodeLookup {
			t.shortCodeLookup[i] = 0
		}
		return
	}

	// Determine the length of the first code
	lastLen := ctx.litAndDistHuff[ctx.codeList[0]].Length()

	if lastLen > litLenLookupBits {
		lastLen = litLenLookupBits + 1
	}
	copySize := uint32(1 << (lastLen - 1))

	// Initialize shortCodeLookup, so invalid lookups process data
	for i := range t.shortCodeLookup[:copySize] {
		t.shortCodeLookup[i] = 0
	}

	minLen := lastLen
	for ; lastLen <= litLenLookupBits; lastLen++ {
		// Copy forward previously set codes
		copy(t.shortCodeLookup[copySize:], t.shortCodeLookup[:copySize])
		copySize *= 2
		t.encodeSingles(ctx, lastLen)

		// Continue if no pairs are possible
		if multisym >= singleSymFlag || lastLen < 2*minLen {
			continue
		}
		t.encodePairs(ctx, lastLen, minLen)

		// Continue if no triples are possible
		if multisym >= doubleSymFlag || lastLen < 3*minLen {
			continue
		}
		t.encodeTriples(ctx, lastLen, minLen)
	}
	t.encodeLongCodes(ctx, codeListLen)
}

func (t *largeHuffCodeTable) encodeSingles(ctx *dynamicHeaderReader, length uint32) {
	start := ctx.litCount[length]
	end := ctx.litCount[length+1]
	for _, index := range ctx.codeList[start:end] {
		sym := indexToSym(index)
		length := ctx.litAndDistHuff[index].Length()
		code := ctx.litAndDistHuff[index].Code()
		if sym > maxLitLenSym {
			continue
		}
		// Set new codes
		t.shortCodeLookup[code] = uint32(sym) | length<<largeShortCodeLenOffset | 1<<largeSymCountOffset
	}
}

func (t *largeHuffCodeTable) encodePairs(ctx *dynamicHeaderReader, length uint32, minLen uint32) {
	// Encode code pairs
	iend := ctx.litCount[length-minLen+1]
	for index1 := ctx.litCount[minLen]; index1 < iend; index1++ {
		sym1Index := ctx.codeList[index1]
		sym1 := indexToSym(sym1Index)
		code := ctx.litAndDistHuff[sym1Index]
		sym1Len := code.Length()
		sym1Code := code.Code()

		// Check that sym1 is a literal
		if sym1 >= 256 {
			index1 = ctx.litCount[sym1Len+1] - 1
			continue
		}

		sym2Len := length - uint32(sym1Len)
		start := ctx.litCount[sym2Len]
		end := ctx.litCount[sym2Len+1]
		list := ctx.codeList[start:end]
		for _, sym2Index := range list {
			sym2 := sym2Index
			if sym2Index == 513 {
				sym2 = 512
			}

			// Check that sym2 is an existing symbol
			if sym2 > maxLitLenSym {
				break
			}

			sym2Code := ctx.litAndDistHuff[sym2Index].Code()
			code := sym1Code | uint32(sym2Code)<<sym1Len
			codeLen := sym1Len + sym2Len
			t.shortCodeLookup[code] = uint32(sym1) | uint32(sym2)<<8 | uint32(codeLen)<<largeShortCodeLenOffset | 2<<largeSymCountOffset
		}
	}
}

func (t *largeHuffCodeTable) encodeTriples(ctx *dynamicHeaderReader, length uint32, minLen uint32) {
	for index1 := ctx.litCount[minLen]; index1 < ctx.litCount[length-2*minLen+1]; index1++ {
		sym1Index := ctx.codeList[index1]
		sym1 := indexToSym(sym1Index)
		sym1Len := ctx.litAndDistHuff[sym1Index].Length()
		sym1Code := ctx.litAndDistHuff[sym1Index].Code()

		// Check that sym1 is a literal
		if sym1 >= 256 {
			index1 = ctx.litCount[sym1Len+1] - 1
			continue
		}

		if length-sym1Len < 2*minLen {
			break
		}

		for index2 := ctx.litCount[minLen]; index2 < ctx.litCount[length-sym1Len-minLen+1]; index2++ {
			sym2Index := (ctx.codeList[index2])
			sym2 := indexToSym(sym2Index)
			sym2Len := ctx.litAndDistHuff[sym2Index].Length()
			sym2Code := ctx.litAndDistHuff[sym2Index].Code()

			// Check that sym2 is a literal
			if sym2 >= 256 {
				index2 = ctx.litCount[sym2Len+1] - 1
				continue
			}

			sym3Len := (length - uint32(sym1Len) - uint32(sym2Len))
			for index3 := ctx.litCount[sym3Len]; index3 < ctx.litCount[sym3Len+1]; index3++ {
				sym3Index := (ctx.codeList[index3])
				sym3 := indexToSym(sym3Index)
				sym3Code := ctx.litAndDistHuff[sym3Index].Code()

				// Check that sym3 is writable existing symbol
				if sym3 > maxLitLenSym-1 {
					break
				}

				code := sym1Code | uint32(sym2Code)<<sym1Len | uint32(sym3Code)<<(sym2Len+sym1Len)
				codeLen := sym1Len + sym2Len + sym3Len
				t.shortCodeLookup[code] = sym1 | uint32(sym2)<<8 | uint32(sym3)<<16 | codeLen<<largeShortCodeLenOffset | 3<<largeSymCountOffset
			}

		}
	}
}

func (t *largeHuffCodeTable) encodeLongCodes(ctx *dynamicHeaderReader, codeListLen uint32) {
	idx := ctx.litCount[litLenLookupBits+1]
	longCodeLength := codeListLen - uint32(idx)
	longCodeList := ctx.codeList[idx:]

	var tempCodeList [1 << (maxLitLenCodeLen - litLenLookupBits)]uint16
	longCodeLookupLength := uint32(0)
	for i := 0; i < int(longCodeLength); i++ {
		if ctx.litAndDistHuff[longCodeList[i]].Code() == invalidCodeValue {
			continue
		}

		maxLen := ctx.litAndDistHuff[longCodeList[i]].Length()
		firstBits := (ctx.litAndDistHuff[longCodeList[i]].Code() & ((1 << litLenLookupBits) - 1))

		tempCodeList[0] = uint16(longCodeList[i])
		tempCodeLength := 1

		for j := i + 1; j < int(longCodeLength); j++ {
			if (ctx.litAndDistHuff[longCodeList[j]].Code() & ((1 << litLenLookupBits) - 1)) == firstBits {
				maxLen = ctx.litAndDistHuff[longCodeList[j]].Length()
				tempCodeList[tempCodeLength] = uint16(longCodeList[j])
				tempCodeLength++
			}
		}

		for j := 0; j < int(tempCodeLength); j++ {
			sym1Index := uint32(tempCodeList[j])
			sym1 := indexToSym(sym1Index)
			sym1Len := ctx.litAndDistHuff[sym1Index].Length()
			sym1Code := ctx.litAndDistHuff[sym1Index].Code()

			longBits := sym1Code >> litLenLookupBits
			minIncrement := uint32(1 << (sym1Len - litLenLookupBits))

			for ; longBits < (1 << (maxLen - litLenLookupBits)); longBits += minIncrement {
				t.longCodeLookup[longCodeLookupLength+longBits] = uint16(sym1 | uint32(sym1Len)<<largeLongCodeLenOffset)
			}
			ctx.litAndDistHuff[sym1Index].SetCode(invalidCodeValue)

		}
		t.shortCodeLookup[firstBits] = uint32(longCodeLookupLength) | uint32(maxLen)<<largeShortMaxLenOffset | largeFlagBit
		longCodeLookupLength += uint32(1 << (maxLen - litLenLookupBits))
	}
}
