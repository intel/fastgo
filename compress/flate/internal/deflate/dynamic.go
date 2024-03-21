// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package deflate

import (
	_ "embed"
	"io"
	"math/bits"

	"github.com/intel/fastgo/compress/flate/internal/huffman"
)

var _ LevelCompressor = &dynCompressor{}

type dynCompressor struct {
	windowSize int
	w          io.Writer
	buffer     []byte
	processed  int
	idx        int
	end        int
	tokens     []token
	hdr        *dynamicHeader
	buf        BitBuf
	litGen     *huffman.LenLimitedCode
	distGen    *huffman.LenLimitedCode
	hist       *histogram
	lz77       lz77compressor
}

const (
	tokensCap    = 32 * 1024
	maxTokenSize = tokensCap - 1
)

func NewDynCompressor(w io.Writer, level int, windowSize int) *dynCompressor {
	c := &dynCompressor{}
	c.w = w
	c.hdr = newDynamicHeader()
	c.windowSize = windowSize
	c.buffer = make([]byte, c.windowSize*2+maxMatchLength+minMatchLength)
	c.tokens = make([]token, 0, tokensCap)
	c.buf = BitBuf{output: make([]byte, 8*1024)}

	c.litGen = huffman.NewLenLimitedCode()
	c.distGen = huffman.NewLenLimitedCode()

	c.lz77 = buildLZ77(level, windowSize)
	c.hist = c.lz77.histogram()
	return c
}

func buildLZ77(level, windowSize int) lz77compressor {
	switch level {
	case 1:
		return &level1context{windowLevel: bits.TrailingZeros(uint(windowSize))}
	case 2:
		return &level2context{windowLevel: bits.TrailingZeros(uint(windowSize))}
	default:
		return &level2context{windowLevel: bits.TrailingZeros(uint(windowSize))}
	}
}

func (c *dynCompressor) Accumulate(data []byte) (n int, trigger bool) {
	// input: [history][resolved data][unresolved data]
	if c.idx >= 2*c.windowSize {
		offset := (c.idx - c.windowSize)
		copy(c.buffer, c.buffer[offset:c.end])
		c.idx -= offset
		c.end -= offset
	}
	n = copy(c.buffer[c.end:2*c.windowSize+maxMatchLength], data)
	c.end += n
	if c.end < 2*c.windowSize+maxMatchLength {
		return
	}
	return n, true
}

func (w *dynCompressor) Compress() (err error) {
	return w.compressBlock(false, false)
}

func (w *dynCompressor) compressBlock(flush bool, finalBlock bool) (err error) {
	if finalBlock && w.end == 0 {
		w.buf.writeFinalEmptyBlock()
		_, err = w.w.Write(w.buf.output[:w.buf.idx])
		return err
	}
	var nIdx int
again:
	nIdx, w.tokens = w.lz77.generate(flush, w.buffer[:w.end], w.processed, w.idx, w.tokens, maxTokenSize)
	w.processed += nIdx - w.idx
	w.idx = nIdx
	if len(w.tokens) < maxTokenSize && !flush {
		return
	}

	err = w.encodeBlock(finalBlock && w.idx == w.end)
	if err != nil {
		return
	}

	if w.idx == w.end {
		return
	}

	goto again
}

func (c *dynCompressor) genHuffCodes() {
	c.hist.reduceCounts()

	c.hist.literalCodes[256] = 1
	// generate length
	c.distGen.Generate(15, c.hist.distanceCodes[:30], c.hist.distanceCodes[:30])
	// generate code & length
	huffman.GenerateCode2(c.hist.distanceCodes[:30])
	// generate length
	c.litGen.Generate(15, c.hist.literalCodes[:286], c.hist.literalCodes[:286])
	// generate code & length
	huffman.GenerateCode2(c.hist.literalCodes[:286])
}

var endOfBlock = newToken(256, InvalidDist, 0)

func (c *dynCompressor) encodeBlock(last bool) error {
	c.buf.idx = 0
	c.tokens = append(c.tokens, endOfBlock)
	c.genHuffCodes()
	c.hdr.writeTo(c.hist, last, &c.buf)

	c.hist.expandCodes()
	idx := 0
	for idx < len(c.tokens) {
		bufSize := len(c.buf.output)
		c.buf.output = c.buf.output[:bufSize-8]
		idx += optimizedEncodeTokens(c.hist, c.tokens[idx:], &c.buf)
		c.buf.output = c.buf.output[:bufSize]
		if last && idx == len(c.tokens) {
			c.buf.flushLastByte()
		}
		_, err := c.w.Write(c.buf.output[:c.buf.idx])
		if err != nil {
			return err
		}
		c.buf.idx = 0
	}
	// reset tokens
	c.tokens = c.tokens[:0]
	c.hist.reset()
	return nil
}

func (w *dynCompressor) Flush() (err error) {
	err = w.compressBlock(true, false)
	if err != nil {
		return err
	}
	// write one zero length no compression block to align to bytes
	w.buf.writeEmptyBlock()
	_, err = w.w.Write(w.buf.output[:w.buf.idx])
	w.buf.idx = 0
	return err
}

func (c *dynCompressor) Close() error {
	err := c.compressBlock(true, true)
	if err != nil {
		return err
	}
	return nil
}

func (w *dynCompressor) Reset(under io.Writer) {
	w.w = under
	w.processed = 0

	w.idx = 0
	w.end = 0

	w.buf.reset()
	w.lz77.reset()
}
