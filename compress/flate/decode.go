// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package flate

import (
	"encoding/binary"

	xbits "math/bits"
)

func (state *inflate) decodeLiteralBlock(output []byte, written int) (w int, err error) {
	if state.bfinal != 0 {
		state.phase = phaseStreamEnd
	} else {
		state.phase = phaseNewBlock
	}
	if state.litBlockLength == 0 {
		return written, nil
	}
	length := state.litBlockLength
	rest := len(output) - written

	if length > rest {
		length = rest
		state.phase = phaseLitBlock
		err = errOutputOverflow
		if rest == 0 {
			return written, err
		}
	}
	avail := int(state.bitsLen/8) + len(state.input)
	if length > avail {
		length = avail
		state.phase = phaseLitBlock
		err = errEndInput
	}
	state.litBlockLength -= length

	count := 0
	for state.bitsLen != 0 {
		output[written] = byte(state.bits & 0xff)
		written++
		state.bits >>= 8
		state.bitsLen -= 8
		count++
		if count == length {
			return written, err
		}
	}

	num := copy(output[written:], state.input[:length-count])
	written += num
	state.input = state.input[num:]
	state.bits = 0
	return written, err
}

const (
	copyLenMax = 258
	// COPY_LEN_MAX = 0
	// SLOPs here are designed to avoid branches
	// So we can use `movdqu xmm` without lots of jumps, and load full 64 bits each time .
	inBufferSlop  = 8 * 3
	copySize      = 16
	outBufferSlop = copySize + copyLenMax
)

const (
	errorNoEndInput        = 1  /* End of input reached */
	errorNoOutOverflow     = 2  /* End of output reached */
	errorNoInvalidBlock    = -1 /* Invalid deflate block found */
	errorNoInvalidSymbol   = -2 /* Invalid deflate symbol found */
	errorNoInvalidLookback = -3 /* Invalid lookback distance found */
)

func decodeHuffmanLargeLoop(state *inflate, output []byte, written int) (finalWritten int, err error) {
	bitsTemp := uint64(0)
	bitsLenTemp := int32(0)

	nextLits := uint32(0)
	symCount := uint32(0)

	var (
		bits    = state.bits    // Bits buffered to handle unaligned streams
		bitsLen = state.bitsLen // Bits in readIn
		input   = state.input
	)

	var (
		inputTemp   []byte
		writtenTemp int
	)

	state.copyOverflowLength = 0
	state.copyOverflowDistance = 0
	for state.phase == phaseHeaderDecoded {

		// state.InLoad(0)
		if bitsLen < 57 {
			if len(input) >= 8 {
				/* If there is enough space to load a 64 bits, load the data and use
				* that to fill readIn */
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

		bitsTemp = bits
		bitsLenTemp = bitsLen
		inputTemp = input

		writtenTemp = written
		var nextSym uint32
		{

			if bitsLen <= maxCodeLen {
				if len(input) >= 8 {
					/* If there is enough space to load a 64 bits, load the data and use
					* that to fill readIn */
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

			nextBits := uint32(bits & ((1 << litLenLookupBits) - 1))

			nextSym = state.litLenTable.shortCodeLookup[nextBits]

			if nextSym&largeFlagBit == 0 {
				bitCount := nextSym >> largeShortCodeLenOffset
				bits >>= bitCount
				bitsLen -= int32(bitCount)

				if bitCount == 0 {
					nextSym = invalidSymbolValue
				}

				symCount = (nextSym >> largeSymCountOffset) & largeSymCountMask
				nextLits = nextSym & largeShortSymMask

			} else {
				bitMask := nextSym >> largeShortMaxLenOffset
				bitMask = (1 << bitMask) - 1
				nextBits = uint32(bits) & (bitMask)
				nextSym = uint32(state.litLenTable.longCodeLookup[(nextSym&largeShortSymMask)+(nextBits>>litLenLookupBits)])
				bitCount := nextSym >> largeLongCodeLenOffset
				bits >>= bitCount
				bitsLen -= int32(bitCount)

				if bitCount == 0 {
					nextSym = invalidSymbolValue
				}

				symCount = 1
				nextLits = nextSym & largeLongSymMask
			}

		}

		if symCount == 0 {
			err = errInvalidSymbol
			goto FINISH
		}

		if bitsLen < 0 {
			bits = bitsTemp
			bitsLen = bitsLenTemp
			input = inputTemp
			err = errEndInput
			goto FINISH
		}

		for symCount > 0 {
			nextLit := uint16(nextLits & 0xffff)
			if nextLit < 256 || symCount > 1 {
				if len(output) == written {
					state.writeOverflowLits = int32(nextLits)
					state.writeOverflowLen = int32(symCount)
					nextLits = nextLits >> (8 * (symCount - 1))
					symCount = 1

					if nextLits < 256 {
						err = errOutputOverflow
						goto FINISH
					} else if nextLits == 256 {
						state.writeOverflowLen -= 1
						if state.bfinal == 1 {
							state.phase = phaseStreamEnd
						} else {
							state.phase = phaseNewBlock
						}
						err = errOutputOverflow
						goto FINISH
					} else {
						state.writeOverflowLen -= 1
						continue
					}
				}
				output[written] = byte(nextLit)
				written++

			} else if nextLit == 256 {
				if state.bfinal == 1 {
					state.phase = phaseStreamEnd
				} else {
					state.phase = phaseNewBlock
				}
			} else if nextLit <= maxLitLenSym {
				repeatLength := int(nextLit - 254)
				nextDist := uint8(0)

				if bitsLen <= maxCodeLen {
					if len(input) >= 8 {
						/* If there is enough space to load a 64 bits, load the data and use
						* that to fill readIn */
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

				nextBits := uint16(bits & ((1 << distLookupBits) - 1))
				nextSym = uint32(state.distTable.ShortCodeLookup[nextBits])
				if nextSym&smallFlagBit == 0 {

					bitCount := nextSym >> smallShortCodeLenOffset
					bits >>= bitCount
					bitsLen -= int32(bitCount)

					if bitCount == 0 {
						bitsLen -= int32(nextSym)
						nextSym = invalidSymbolValue
					}
					nextDist = uint8(nextSym & distSymMask)

				} else {
					bitMask := (uint32(nextSym) - smallFlagBit) >> smallShortCodeLenOffset
					bitMask = (1 << bitMask) - 1
					nextBits = uint16(bits & uint64(bitMask))
					nextSym = uint32(state.distTable.LongCodeLookup[uint16(nextSym&smallShortSymMask)+(nextBits>>distLookupBits)])
					bitCount := nextSym >> smallLongCodeLenOffset
					bits >>= bitCount
					bitsLen -= int32(bitCount)

					if bitCount == 0 {
						bitsLen -= int32(nextSym)
						nextSym = invalidSymbolValue
					}
					nextDist = uint8(nextSym & distSymMask)
				}

				lookBackDist := 0
				if bitsLen >= 0 {
					if nextDist >= distLen {
						err = errInvalidSymbol
						goto FINISH
					}
					bitCount := rfcLookupTable.DistExtraBitCount[nextDist]
					var extraBits uint64

					if bitsLen < 57 {
						if len(input) >= 8 {
							/* If there is enough space to load a 64 bits, load the data and use
							* that to fill readIn */
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

					extraBits = (bits) & ((1 << bitCount) - 1)
					bits = bits >> bitCount
					bitsLen -= int32(bitCount)
					lookBackDist = int(rfcLookupTable.DistStart[nextDist]) + int(extraBits)
				}

				if bitsLen < 0 {
					bits = bitsTemp
					bitsLen = bitsLenTemp
					input = inputTemp
					written = writtenTemp
					state.writeOverflowLits = 0
					state.writeOverflowLen = 0
					err = errEndInput
					goto FINISH
				}
				if written < int(lookBackDist) {
					err = errInvalidLookBack
					goto FINISH
				}
				availOut := len(output) - written
				if availOut < repeatLength {
					state.copyOverflowLength = int32(repeatLength - availOut)
					state.copyOverflowDistance = int32(lookBackDist)
					repeatLength = availOut
				}

				if lookBackDist >= repeatLength {
					copy(output[written:], output[written-lookBackDist:written-lookBackDist+repeatLength])
				} else {
					byteCopy(output, written, lookBackDist, repeatLength)
				}
				written += repeatLength

				if state.copyOverflowLength > 0 {
					err = errOutputOverflow
					goto FINISH

				}
			} else {
				err = errInvalidSymbol
				goto FINISH
			}

			nextLits >>= 8
			symCount--
		}
	}

FINISH:
	// save args
	if (64 - xbits.LeadingZeros64(bits)) > int(bitsLen) {
		bits &= ((1 << bitsLen) - 1)
	}
	{
		state.bits = bits
		state.input = input
		state.bitsLen = bitsLen
		finalWritten = written
	}

	return
}

func byteCopy(hist []byte, curr int, dist, length int) {
	end := curr + length
	start := curr - dist
	for curr < end {
		to := hist[curr:end]
		from := hist[start:curr]
		size := copy(to, from)
		curr += size
	}
}
