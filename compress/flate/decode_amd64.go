// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

//go:build amd64 && !noasmtest
// +build amd64,!noasmtest

package flate

import "github.com/intel/fastgo/internal/cpu"

func decodeHuffmanAsmArchV3(state *inflate, output []byte, offset int) (written int, errno int)

func decodeHuffman(state *inflate, output []byte, written int) (w int, err error) {
	if cpu.ArchLevel < 3 {
		return decodeHuffmanLargeLoop(state, output, written)
	}
	if len(output)-written > outBufferSlop && len(state.input) > inBufferSlop {
		var errno int
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
