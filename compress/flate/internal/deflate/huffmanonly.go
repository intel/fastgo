// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package deflate

import (
	"io"

	"github.com/intel/fastgo/compress/flate/internal/huffman"
)

type huffmanOnly struct {
	w      io.Writer
	hist   histogram
	buffer []byte
	offset int
	max    int
	hdr    *dynamicHeader
	litGen *huffman.LenLimitedCode
	buf    BitBuf
}

func NewHuffmanOnly(w io.Writer) *huffmanOnly {
	h := &huffmanOnly{}
	h.buf.output = make([]byte, 8*1024)
	h.litGen = huffman.NewLenLimitedCode()
	h.hdr = newDynamicHeader()
	h.buffer = make([]byte, 64*1024)
	h.max = 64 * 1024
	h.w = w
	return h
}

func (h *huffmanOnly) Reset(w io.Writer) {
	h.w = w
	h.buf.reset()
	h.offset = 0
}

func (h *huffmanOnly) Accumulate(data []byte) (n int, trigger bool) {
	n = copy(h.buffer[h.offset:h.max], data)
	h.offset += n
	if h.offset == h.max {
		return n, true
	}
	return n, false
}

func (h *huffmanOnly) Compress() error {
	return h.encodeBlock(false, false)
}

func bytesFreq(hist *histogram, input []byte) {
	for x := range hist.literalCodes[:256] {
		hist.literalCodes[x] = 0
	}
	for j := 0; j < len(input); j++ {
		hist.literalCodes[input[j]]++
	}
}

var optimizedEncodeBytes func(hist *histogram, data []byte, buf *BitBuf) (num int)

func (h *huffmanOnly) encodeBlock(final bool, flush bool) error {
	if final && h.offset == 0 {
		h.buf.writeFinalEmptyBlock()
		_, err := h.w.Write(h.buf.output[:h.buf.idx])
		return err
	}

	bytesFreq(&h.hist, h.buffer[:h.offset])
	h.hist.reduceCounts()
	h.hist.literalCodes[256] = 1

	// generate length
	h.litGen.Generate(15, h.hist.literalCodes[:286], h.hist.literalCodes[:286])
	// generate code & length
	huffman.GenerateCode2(h.hist.literalCodes[:286])
	h.hdr.writeTo(&h.hist, final, &h.buf)
	num := 0
	for num < h.offset {
		h.buf.Sync()
		num += optimizedEncodeBytes(&h.hist, h.buffer[num:h.offset], &h.buf)
		if num == h.offset && flush {
			h.buf.flushLastByte()
		}
		_, err := h.w.Write(h.buf.output[:h.buf.idx])
		if err != nil {
			return err
		}
		h.buf.idx = 0
	}
	h.offset = 0
	return nil
}

func (h *huffmanOnly) Flush() (err error) {
	err = h.encodeBlock(false, true)
	if err != nil {
		return err
	}
	// write one zero length no compression block to align to bytes
	h.buf.writeEmptyBlock()
	_, err = h.w.Write(h.buf.output[:h.buf.idx])
	h.buf.idx = 0
	return err
}

func (h *huffmanOnly) Close() (err error) {
	err = h.encodeBlock(true, true)
	if err != nil {
		return err
	}
	h.w = nil
	return err
}
