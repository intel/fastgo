// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package deflate

import (
	"math/bits"
	"unsafe"
)

const (
	minMatchLength = 3
	maxMatchLength = 258
	hashMask       = 0xfffff
	minMatch       = 4
)

func getDistSymbol(dist uint32) (sym uint32, extraBits uint32) {
	if dist <= 2 {
		return dist - 1, 0
	}
	dist--
	msb := 32 - bits.LeadingZeros32(uint32(dist))
	numExtraBits := uint32(msb - 2)
	extraBits = dist & ((1 << numExtraBits) - 1)
	dist >>= numExtraBits
	sym = dist + 2*numExtraBits
	return sym, extraBits
}

func hash4(data uint32) uint32 {
	const prime = 0xB2D06057
	var hash uint64
	hash = uint64(data)
	hash *= prime
	hash >>= 16
	hash *= prime
	hash >>= 16
	return uint32(hash)
}

func lz77(flush bool, table []uint16, mask uint32, historySize int, hist *histogram, input []byte, processed int, offset int, tokens []token, maxToken int) (nOffset int, ntokens []token) {
	relative := processed - offset
	end := len(input) - 8
	for offset < end {
		fourBytes := loadU32(input, offset)
		hash := hash4(fourBytes) & mask
		lookup := table[hash]
		dist := uint32(uint16(relative+offset)-lookup) & 0xffff
		table[hash] = uint16(offset + relative)
		if uint32(dist-1) < uint32(historySize) {
			prev := offset - int(dist)
			var matchLength int
			test := loadU64(input, prev)
			test ^= loadU64(input, offset)

			if test != 0 {
				matchLength = bits.TrailingZeros64(test) / 8
			} else {
				matchLength = compare(input, prev+8, offset+8, end-offset-8) + 8
			}

			if matchLength > 258 {
				// only update next 3 hash
				for i := 0; i < 3; i++ {
					fourBytes := loadU32(input, offset+i)
					hash := hash4(fourBytes) & mask
					table[hash] = uint16(relative + offset + i)
				}
				lengthSymbol := 258 + 254
				distSymbol, extraBits := getDistSymbol(dist)
				token := newToken(uint32(lengthSymbol), uint32(distSymbol), uint32(extraBits))
				repeat := uint32(0)
				for matchLength > 258 {
					repeat++
					offset += 258

					tokens = append(tokens, token)
					if len(tokens) > maxToken {
						hist.literalCodes[lengthSymbol] += repeat
						hist.distanceCodes[distSymbol] += repeat
						return offset, tokens
					}
					matchLength -= 258
				}
				hist.literalCodes[lengthSymbol] += repeat
				hist.distanceCodes[distSymbol] += repeat
			}

			if matchLength >= minMatch {
				{
					// only update next 3 hash
					for i := 0; i < 3; i++ {
						fourBytes := loadU32(input, offset+i)
						hash := hash4(fourBytes) & mask
						table[hash] = uint16(relative + offset + i)
					}
				}
				lengthSymbol := matchLength + 254
				distSymbol, extraBits := getDistSymbol(dist)
				tokens = append(tokens, newToken(uint32(lengthSymbol), uint32(distSymbol), uint32(extraBits)))
				offset += matchLength

				hist.literalCodes[lengthSymbol]++
				hist.distanceCodes[distSymbol]++
				if len(tokens) > maxToken {
					return offset, tokens
				}
				continue
			}
		}
		offset++
		lit := fourBytes & 0xff
		tokens = append(tokens, newToken(lit, InvalidDist, 0))
		hist.literalCodes[lit]++
		if len(tokens) > maxToken {
			return offset, tokens
		}
	}
	if flush {
		for offset < len(input) {
			lit := input[offset]
			tokens = append(tokens, newToken(uint32(lit), InvalidDist, 0))
			hist.literalCodes[lit]++
			offset++
			if len(tokens) > maxToken {
				return offset, tokens
			}
		}
	}
	return offset, tokens
}

//go:linkname Prefetch runtime/internal/sys.Prefetch
func Prefetch(addr uintptr)

func compare(input []byte, prev, curr int, maxLength int) (match int) {
	if maxLength < 8 {
		return 0
	}
	max := maxLength & (^0x7)
	i := 0
	for ; i < max; i += 8 {
		test := loadU64(input, prev+i)
		test ^= loadU64(input, curr+i)
		if test != 0 {
			return i + bits.TrailingZeros64(test)/8
		}
	}
	temp := maxLength % 8
	// maxLength %= 8
	for ; temp > 0; temp-- {
		if input[prev+i] != input[curr+i] {
			return i
		}
		i++
	}
	return maxLength
}

func loadU32(input []byte, offset int) uint32 {
	return *(*uint32)(unsafe.Pointer(&input[offset]))
}

func loadU64(input []byte, offset int) uint64 {
	return *(*uint64)(unsafe.Pointer(&input[offset]))
}
