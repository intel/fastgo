// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package deflate

import (
	"fmt"
)

type token uint32

const InvalidDist = 30

// litLen : maxLitLen = 513 => 10 bit
// dist: maxDistSymbol = 29 => 5 bit => 9bit
// max dist Extra = 32768 - 24577 = 13 bit
// 10 + 9 + 13 = 32
// We can also combine two literal tokens into one token.
// |lit1|lit2+31|...|
// when we encode this token,
// we can check if dist >= 31 to decide how to handle encoding.
// Due to the structure of the histogram, we don't even need to check that.
func (c *token) Extract() (litLen, dist, distExtra uint32) {
	litLen = uint32(*c) & ((1 << 10) - 1)
	dist = (uint32(*c) >> 10) & ((1 << 9) - 1)
	distExtra = uint32(*c) >> 19
	return
}

func (c *token) ExtractLz77() (litLen, dist uint32, isLit bool) {
	litLen = uint32(*c) & ((1 << 10) - 1)
	dist = (uint32(*c) >> 10) & ((1 << 9) - 1)
	distExtra := uint32(*c) >> 19
	if dist == InvalidDist {
		return litLen, 256, true
	}
	if dist > 30 {
		return litLen, dist - 31, true
	}
	return litLen - 254, disttable[dist] + distExtra, false
}

func newToken(litLen, dist, extra uint32) token {
	return token(litLen | (dist << 10) | (extra << (10 + 9)))
}

func (t token) String() string {
	lit, dist, isLit := t.ExtractLz77()
	if isLit {
		str := string(rune(lit))
		if dist < 256 {
			str += string(rune(dist))
			return str
		}
		return str
	}
	return fmt.Sprintf("<LEN/DIST %d / %d >", lit, dist)
}

var disttable = []uint32{
	0x0001, 0x0002, 0x0003, 0x0004, 0x0005, 0x0007, 0x0009, 0x000d,
	0x0011, 0x0019, 0x0021, 0x0031, 0x0041, 0x0061, 0x0081, 0x00c1,
	0x0101, 0x0181, 0x0201, 0x0301, 0x0401, 0x0601, 0x0801, 0x0c01,
	0x1001, 0x1801, 0x2001, 0x3001, 0x4001, 0x6001, 0x0000, 0x0000,
}
