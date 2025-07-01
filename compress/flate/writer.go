// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package flate

import (
	"compress/flate"
	"io"

	"github.com/intel/fastgo/compress/flate/internal/deflate"
)

// Compression level constants compatible with standard library
const (
	NoCompression      = flate.NoCompression      // No compression, just store
	BestSpeed          = flate.BestSpeed          // Level 1: fastest compression
	BestCompression    = flate.BestCompression    // Level 9: best compression ratio
	DefaultCompression = flate.DefaultCompression // Default compression level
	HuffmanOnly        = flate.HuffmanOnly        // Huffman-only compression
)

// Writer provides Intel-optimized DEFLATE compression
type Writer = deflate.Writer

// NewWriter creates a new Intel-optimized DEFLATE compressor with the specified level.
// The compressor automatically selects between optimized and standard implementations
// based on the compression level and CPU capabilities.
// Supported optimized levels: 1 (BestSpeed), 2, and HuffmanOnly.
func NewWriter(under io.Writer, level int) (w *Writer, err error) {
	return deflate.NewWriter(under, level)
}

// NewWriterwWith4KWindow creates a new compressor with a 4KB sliding window.
// This can provide better performance for smaller data sets at the cost of
// reduced compression ratio compared to the standard 32KB window.
func NewWriterwWith4KWindow(under io.Writer, level int) (w *Writer, err error) {
	return deflate.NewWriterwWith4KWindow(under, level)
}

// NewWriterDict creates a new compressor with a preset dictionary.
// Note: Dictionary support is limited in the current Intel optimization.
// When optimizations are not available, it falls back to standard library.
func NewWriterDict(under io.Writer, level int, dict []byte) (w *Writer, err error) {
	return deflate.NewWriterDict(under, level, dict)
}
