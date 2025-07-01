// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

// Package deflate provides Intel-optimized DEFLATE compression implementation.
// It contains the core compression logic with specialized optimizations for
// different compression levels and Intel CPU architectures.
package deflate

import (
	"compress/flate"
	"io"
)

// Writer implements Intel-optimized DEFLATE compression.
// It chooses between optimized implementations and standard library
// based on compression level and CPU capabilities.
type Writer struct {
	err error           // Last error encountered
	lc  LevelCompressor // Intel-optimized compressor for supported levels
	w   *flate.Writer   // Standard library writer for unsupported levels
}

// NewWriterwWith4KWindow creates a new compressor with a 4KB sliding window.
// This provides better performance for smaller data with reduced memory usage.
// For compression levels 1, 2, and HuffmanOnly, it uses Intel optimizations.
// Other levels fall back to standard library with custom window size.
func NewWriterwWith4KWindow(under io.Writer, level int) (w *Writer, err error) {
	w = &Writer{}
	if level == DefaultCompression {
		level = 2 // Default to level 2 for best speed/compression balance
	}
	switch level {
	case NoCompression:
		// No compression - use standard library
		w.w, err = flate.NewWriter(under, level)
		if err != nil {
			return nil, err
		}
	case HuffmanOnly:
		// Huffman-only compression - use Intel optimization
		w.lc = NewHuffmanOnly(under)
	case 1, 2:
		// Level 1 & 2 - use Intel optimization with 4KB window
		w.lc = NewDynCompressor(under, level, 4*1024)
	default:
		// Other levels - use Intel optimization or fallback
		w.lc = NewDynCompressor(under, level, 4*1024)
	}

	return w, nil
}

// NewWriterDict creates a new compressor with a preset dictionary.
// Dictionary support is limited in Intel optimizations, so this
// primarily uses the standard library implementation.
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

// NewWriter creates a new Intel-optimized DEFLATE compressor.
// It automatically selects the best implementation based on compression level:
// - HuffmanOnly: Intel-optimized Huffman compression
// - Level 1, 2: Intel-optimized with LZ77 + Huffman
// - Other levels: Falls back to standard library
func NewWriter(under io.Writer, level int) (w *Writer, err error) {
	w = &Writer{}
	if level == DefaultCompression {
		level = 2 // Default to level 2 for balanced performance
	}
	switch level {
	case HuffmanOnly:
		// Use Intel-optimized Huffman-only compression
		w.lc = NewHuffmanOnly(under)
	case 1, 2:
		// Use Intel-optimized compression with 32KB window
		w.lc = NewDynCompressor(under, level, 32*1024)
	default:
		// Fall back to standard library for unsupported levels
		w.w, err = flate.NewWriter(under, level)
		if err != nil {
			return nil, err
		}
	}
	return w, err
}

// Write compresses and writes data to the underlying writer.
// It routes data to either the Intel-optimized compressor or standard library
// based on the configuration determined during Writer creation.
func (w *Writer) Write(data []byte) (n int, err error) {
	if w.err != nil {
		return n, w.err
	}
	if w.w != nil {
		// Use standard library writer
		return w.w.Write(data)
	}
	// Use Intel-optimized compressor
	n = len(data)
	var num int
	for num < n {
		// Accumulate data until ready to compress
		accNum, ok := w.lc.Accumulate(data[num:])
		if ok {
			// Compress accumulated data
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

// Reset resets the writer to use a new underlying writer.
// This allows reusing the same Writer instance for multiple compression tasks.
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
