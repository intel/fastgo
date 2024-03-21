// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package deflate

import (
	"github.com/intel/fastgo/compress/flate/internal/huffman"
)

// type codeLength [286 + 30 + 1]uint8

const (
	numRepeat3_6     = 16
	zeroRepeat3_10   = 17
	zeroRepeat11_138 = 18
)

var hclenOrder = []uint32{16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15}

type dynamicHeader struct {
	generator   huffman.TreeGenerator
	litNum      int
	distanceNum int
	source      []uint8  // lit/len lengths && offset lengths
	data        []uint8  // combined lengths
	histogram   []uint32 // length symbol frequency or lengths
	rcodes      []uint16
	countCache  []uint32
}

func newDynamicHeader() *dynamicHeader {
	return &dynamicHeader{
		generator:  huffman.NewLenLimitedCode(),
		histogram:  make([]uint32, 19),
		rcodes:     make([]uint16, 19),
		source:     make([]uint8, 286+30+1),
		countCache: make([]uint32, (7+1)*2),
	}
}

func (c *dynamicHeader) writeTo(histogram *histogram, eos bool, b *BitBuf) {
	c.prepareAlphabets(histogram)
	codeSize := c.codeSize()
	c.generator.Generate(7, c.histogram, c.histogram)
	for i := range c.rcodes {
		c.rcodes[i] = 0
	}
	huffman.GenerateCode(c.countCache, 7, c.histogram, c.rcodes)
	if eos {
		b.WriteBit(5, 3)
	} else {
		b.WriteBit(4, 3)
	}
	// HLIT
	b.WriteBit(uint16(c.litNum)-257, 5)
	// HDIST
	b.WriteBit(uint16(c.distanceNum)-1, 5)
	// HCLEN
	b.WriteBit(uint16(codeSize)-4, 4)
	//  (HCLEN + 4) x 3 bits: code lengths for the code length
	// alphabet given just above, in the order: 16, 17, 18,
	// 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15
	for i := 0; i < codeSize; i++ {
		b.WriteBit(uint16(c.histogram[hclenOrder[i]]), 3)
	}

	for i := 0; i < len(c.data); i++ {
		value := c.data[i]
		b.WriteBit(c.rcodes[value], uint8(c.histogram[value]))
		switch value {
		case numRepeat3_6:
			i++
			extra := c.data[i]
			b.WriteBit(uint16(extra), 2)
		case zeroRepeat3_10:
			i++
			extra := c.data[i]
			b.WriteBit(uint16(extra), 3)
		case zeroRepeat11_138:
			i++
			extra := c.data[i]
			b.WriteBit(uint16(extra), 7)
		}
	}
}

// HCLEN,  of Code Length codes
func (c *dynamicHeader) codeSize() (num int) {
	num = len(c.histogram)
	for num > 4 && c.histogram[hclenOrder[num-1]] == 0 {
		num--
	}
	return num
}

// prepareAlphabets
// 1. reduce codes by conbine the repeated continuously codes.
// 2. caculate the frequency of the codes
func (c *dynamicHeader) prepareAlphabets(histogram *histogram) {
	litNum, distanceNum := prepare(histogram, c.source)
	if distanceNum == 0 {
		c.source[litNum] = 1
		distanceNum = 1
	}

	c.litNum = int(litNum)
	c.distanceNum = int(distanceNum)

	// prepare to combine repeat numbers
	// source := c.source[:litNum+distanceNum+1]

	c.data = c.data[:0]
	// reset histogram
	for i := range c.histogram {
		c.histogram[i] = 0
	}
	c.alphabet(c.source[:litNum+1])
	c.alphabet(c.source[litNum : litNum+distanceNum+1])
}

func (c *dynamicHeader) alphabet(source []byte) {
	temp := source[len(source)-1]
	source[len(source)-1] = 255

	prev := uint8(0)
	start := 0
	for i, current := range source {
		if i == 0 {
			prev = current
			continue
		}
		if current == prev {
			continue
		}
		repeated := i - start
		if prev == 0 {
			c.zeroRepeat(repeated)
		} else {
			c.numRepeat(prev, repeated)
		}
		start = i
		prev = current
	}
	source[len(source)-1] = temp
}

func (c *dynamicHeader) numRepeat(num byte, repeated int) {
	for repeated != 0 {
		switch {
		case repeated <= 3:
			if repeated == 1 {
				c.data = append(c.data, num)
			} else if repeated == 2 {
				c.data = append(c.data, num, num)
			} else if repeated == 3 {
				c.data = append(c.data, num, num, num)
			}
			c.histogram[num] += uint32(repeated)
			repeated = 0
		case repeated <= 7:
			c.histogram[num]++
			c.data = append(c.data, num)

			c.histogram[numRepeat3_6]++

			c.data = append(c.data, numRepeat3_6, uint8(repeated-4))

			repeated = 0
		default:
			c.histogram[num]++
			c.histogram[numRepeat3_6]++
			c.data = append(c.data, num)
			c.data = append(c.data, numRepeat3_6, uint8(3))
			repeated -= 7
		}
	}
}

func (c *dynamicHeader) zeroRepeat(repeated int) {
	for repeated != 0 {
		switch {
		case repeated < 3:
			if repeated == 1 {
				c.data = append(c.data, 0)
			} else {
				c.data = append(c.data, 0, 0)
			}
			c.histogram[0] += uint32(repeated)
			// consume all repeated 0
			repeated = 0
		case repeated < 11:
			c.histogram[zeroRepeat3_10]++
			c.data = append(c.data, zeroRepeat3_10, byte(repeated-3))
			// consume all repeated 0
			repeated = 0
		case repeated < 139:
			c.histogram[zeroRepeat11_138]++
			c.data = append(c.data, zeroRepeat11_138, byte(repeated-11))
			// consume all repeated 0
			repeated = 0
		default:
			c.histogram[zeroRepeat11_138]++
			c.data = append(c.data, zeroRepeat11_138, byte(138-11))
			// consume max 138 repeated 0
			repeated -= 138
		}
	}
}

func prepare(histogram *histogram, source []byte) (litNum uint16, distanceNum uint16) {
	litNum = 0
	for i := 285; i >= 0; i-- {
		if histogram.literalCodes[i] != 0 {
			litNum = uint16(i) + 1
			break
		}
	}
	for i := 29; i >= 0; i-- {
		if histogram.distanceCodes[i] != 0 {
			distanceNum = uint16(i) + 1
			break
		}
	}
	insertOneDistance := false
	if distanceNum == 0 {
		distanceNum = 1
		insertOneDistance = true
	}

	// prepare to combine repeat numbers
	source = source[:litNum+distanceNum+1]
	for i := uint16(0); i < litNum; i++ {
		source[i] = uint8(histogram.literalCodes[i] >> 24)
	}
	for i := uint16(0); i < distanceNum; i++ {
		source[litNum+i] = uint8(histogram.distanceCodes[i] >> 24)
	}
	if insertOneDistance {
		source[litNum] = 1
	}
	return litNum, distanceNum
}
