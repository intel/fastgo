// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package deflate

import (
	"compress/flate"
	"io"
)

type Writer struct {
	err error
	lc  LevelCompressor
	w   *flate.Writer
}

func NewWriterwWith4KWindow(under io.Writer, level int) (w *Writer, err error) {
	w = &Writer{}
	if level == DefaultCompression {
		level = 2
	}
	switch level {
	case NoCompression:
		w.w, err = flate.NewWriter(under, level)
		if err != nil {
			return nil, err
		}
	case HuffmanOnly:
		w.lc = NewHuffmanOnly(under)
	case 1, 2:
		w.lc = NewDynCompressor(under, level, 4*1024)
	default:
		w.lc = NewDynCompressor(under, level, 4*1024)
	}

	return w, nil
}

func NewWriterDict(under io.Writer, level int, dict []byte) (w *Writer, err error) {
	if dict == nil {
		return NewWriter(under, level)
	}
	w = &Writer{}
	w.w, err = flate.NewWriterDict(under, level, dict)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func NewWriter(under io.Writer, level int) (w *Writer, err error) {
	w = &Writer{}
	if level == DefaultCompression {
		level = 2
	}
	switch level {
	case HuffmanOnly:
		w.lc = NewHuffmanOnly(under)
	case 1, 2:
		w.lc = NewDynCompressor(under, level, 32*1024)
	default:
		w.w, err = flate.NewWriter(under, level)
		if err != nil {
			return nil, err
		}
	}
	return w, err
}

func (w *Writer) Write(data []byte) (n int, err error) {
	if w.err != nil {
		return n, w.err
	}
	if w.w != nil {
		return w.w.Write(data)
	}
	n = len(data)
	var num int
	for num < n {
		accNum, ok := w.lc.Accumulate(data[num:])
		if ok {
			err = w.lc.Compress()
			if err != nil {
				w.err = err
				return num, err
			}
		}
		num += accNum
	}
	return num, nil
}

func (w *Writer) Reset(under io.Writer) {
	w.err = nil
	if w.w != nil {
		w.w.Reset(under)
		return
	}
	w.lc.Reset(under)
}

func (w *Writer) Flush() (err error) {
	if w.err != nil {
		return w.err
	}
	if w.w != nil {
		return w.w.Flush()
	}
	return w.lc.Flush()
}

func (w *Writer) Close() (err error) {
	if w.err != nil {
		return w.err
	}
	if w.w != nil {
		return w.w.Close()
	}
	return w.lc.Close()
}
