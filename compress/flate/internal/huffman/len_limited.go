// Copyright (c) 2023, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package huffman

// LenLimitedCode implements a code length limited algorithm huffman tree generator.
type LenLimitedCode struct {
	counts    decLitCounts
	lenCounts []int
	w         []uint32
}

// NewLenLimitedCode creates a new LenLimitedCode instance
func NewLenLimitedCode() *LenLimitedCode {
	return &LenLimitedCode{}
}

// Generate huffman tree from histogram and write tree codes into codes.
func (l *LenLimitedCode) Generate(limitedLen int, histogram []uint32, codeLens []uint32) int {
	if l.counts == nil {
		l.counts = make(decLitCounts, 0, len(histogram)/2)
	} else {
		l.counts = l.counts[:0]
	}
	num := 0
	for i, v := range histogram {
		if v != 0 {
			num++
			l.counts = append(l.counts, litCount{
				lit:   uint16(i),
				count: uint16(v),
			})
		}
	}
	for i := range codeLens {
		codeLens[i] = 0
	}

	sortDecLitCounts(l.counts)

	if l.w == nil {
		l.w = make([]uint32, len(l.counts), len(histogram))
	} else {
		l.w = l.w[:len(l.counts)]
	}
	for i, v := range l.counts {
		l.w[i] = uint32(v.count)
	}

	maxLen := (&MoffatHuffmanCode{}).codeLens(l.w)
	if maxLen <= uint32(limitedLen) {
		for i, v := range l.w {
			codeLens[l.counts[i].lit] = v
		}
		return num
	}
	for i, v := range l.w {
		codeLens[l.counts[i].lit] = v
	}
	if l.lenCounts == nil {
		l.lenCounts = make([]int, maxLen+1)
	} else {
		if cap(l.lenCounts) < int(maxLen+1) {
			l.lenCounts = make([]int, maxLen+1)
		} else {
			l.lenCounts = l.lenCounts[:maxLen+1]
			for i := range l.lenCounts {
				l.lenCounts[i] = 0
			}
		}
	}

	for _, v := range l.w {
		l.lenCounts[v]++
	}

	l.enforceMaxLen(l.lenCounts, limitedLen)
	idx := 0
	for length := 1; length <= limitedLen; length++ {
		num := l.lenCounts[length]
		for j := 0; j < num; j++ {
			codeLens[l.counts[idx].lit] = uint32(length)
			idx++
		}
	}
	return num
}

func (l *LenLimitedCode) enforceMaxLen(lenCounts []int, maxLen int) {
	// move all oversize length to the maxLen
	for i := maxLen + 1; i < len(lenCounts); i++ {
		lenCounts[maxLen] += lenCounts[i]
		lenCounts[i] = 0
	}

	// Kraft-McMillan inequality
	// https://en.wikipedia.org/wiki/Kraft%E2%80%93McMillan_inequality
	// -> sum(2^-length[...]) == 1
	// -> sum(2 ^ - length[... ]) * (2 ^ maxLength) == 1 * 2^maxLength
	// -> sum(2 ^ (maxLength - length[0]) , ... 2 ^ (maxLength - length[n])) == 2 ^ maxLength
	// -> sum(lengthAnum * 2 ^ (maxLength - lengthA),...  maxLengthNum * 2 ^ (maxLength - maxLength)) == 2 ^ maxLength
	// -> sum(lengthAnum * 2 ^ (maxLength - lengthA),... ) + maxLengthNum == 2 ^ maxLength

	// try to make things meet the Kraft-McMillan inequality
	total := 0
	for i := 1; i <= maxLen; i++ {
		total += lenCounts[i] * 1 << (maxLen - i)
	}
	for total != 1<<maxLen {
		// move longest nodes
		lenCounts[maxLen]--
		for i := maxLen - 1; i > 0; i-- {
			if lenCounts[i] != 0 {
				lenCounts[i]--
				lenCounts[i+1] += 2
				break
			}
		}
		total-- // see comments above
	}
}
