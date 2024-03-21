// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package deflate

import (
	"compress/flate"
	"io"
)

const (
	NoCompression      = flate.NoCompression
	BestSpeed          = flate.BestSpeed
	BestCompression    = flate.BestCompression
	DefaultCompression = flate.DefaultCompression
	HuffmanOnly        = flate.HuffmanOnly
)

type LevelCompressor interface {
	Reset(w io.Writer)
	Accumulate(data []byte) (n int, trigger bool)
	Compress() error
	Flush() error
	Close() error
}

type lz77compressor interface {
	generate(flush bool, input []byte, processed int, offset int, tokens []token, maxToken int) (nOffset int, ntokens []token)
	reset()
	histogram() *histogram
}

type level1context struct {
	table       [1 << 12]uint16
	hist        histogram
	windowLevel int
}

func (c *level1context) reset() {
	for i := range c.table {
		c.table[i] = 0
	}
	c.hist.reset()
}

func (c *level1context) histogram() *histogram {
	return &c.hist
}

type level2context struct {
	table       [1 << 15]uint16
	hist        histogram
	windowLevel int
}

func (c *level2context) reset() {
	for i := range c.table {
		c.table[i] = 0
	}
	c.hist.reset()
}

func (c *level2context) histogram() *histogram {
	return &c.hist
}
