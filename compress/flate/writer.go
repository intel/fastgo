// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package flate

import (
	"compress/flate"
	"io"

	"github.com/intel/fastgo/compress/flate/internal/deflate"
)

const (
	NoCompression      = flate.NoCompression
	BestSpeed          = flate.BestSpeed
	BestCompression    = flate.BestCompression
	DefaultCompression = flate.DefaultCompression
	HuffmanOnly        = flate.HuffmanOnly
)

type Writer = deflate.Writer

func NewWriter(under io.Writer, level int) (w *Writer, err error) {
	return deflate.NewWriter(under, level)
}

func NewWriterwWith4KWindow(under io.Writer, level int) (w *Writer, err error) {
	return deflate.NewWriterwWith4KWindow(under, level)
}

func NewWriterDict(under io.Writer, level int, dict []byte) (w *Writer, err error) {
	return deflate.NewWriterDict(under, level, dict)
}
