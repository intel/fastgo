// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

//go:build amd64 && !noasmtest
// +build amd64,!noasmtest

// This file contains Intel AMD64-specific optimizations for DEFLATE decompression.
// It provides assembly-accelerated Huffman decoding that significantly improves
// decompression performance on supported Intel architectures.
package flate

import "github.com/intel/fastgo/internal/cpu"

// decodeHuffmanAsmArchV3 is the assembly implementation of optimized Huffman decoding
// for Intel ArchLevel 3 and above. This function is implemented in assembly language
// for maximum performance.
func decodeHuffmanAsmArchV3(state *inflate, output []byte, offset int) (written int, errno int)

// decodeHuffman performs Intel-optimized Huffman decoding.
// It automatically selects between assembly-optimized and fallback implementations
// based on CPU capabilities and buffer sizes.
func decodeHuffman(state *inflate, output []byte, written int) (w int, err error) {
	// Check if CPU supports Level 3+ optimizations
	if cpu.ArchLevel < 3 {
		return decodeHuffmanLargeLoop(state, output, written)
	}
	// Ensure sufficient buffer space for safe assembly operation
	if len(output)-written > outBufferSlop && len(state.input) > inBufferSlop {
		var errno int
		// Use assembly-optimized implementation with safety margins
		written, errno = decodeHuffmanAsmArchV3(state, output[:len(output)-outBufferSlop], written)
		if errno != 0 && errno != errorNoEndInput {
			switch errno {
			case errorNoInvalidBlock:
				err = errInvalidBlock
			case errorNoInvalidSymbol:
				err = errInvalidSymbol
			case errorNoInvalidLookback:
				err = errInvalidLookBack
			case errorNoOutOverflow:
				err = errOutputOverflow
			}
			return written, err
		}
		if errno == 0 {
			state.phase = phaseNewBlock
			if state.bfinal == 1 {
				state.phase = phaseStreamEnd
			}
			return written, nil
		}
	}

	// fallback to slow path
	return decodeHuffmanLargeLoop(state, output, written)
}
