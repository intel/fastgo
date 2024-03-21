// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package flate

import (
	"bufio"
	"compress/flate"
	"encoding/binary"
	"errors"
	"io"
)

type (
	Reader            = flate.Reader
	Resetter          = flate.Resetter
	CorruptInputError = flate.CorruptInputError
)

var (
	errInvalidBlock    = errors.New("invalid block")
	errInvalidSymbol   = errors.New("invalid symbol")
	errInvalidLookBack = errors.New("invalid look back")
)

var (
	errEndInput       = errors.New("end of input")
	errOutputOverflow = errors.New("output overflow")
)

func isError(err error) bool {
	if err == errInvalidBlock || err == errInvalidSymbol || err == errInvalidLookBack {
		return true
	}
	return false
}

var NewReaderDict = flate.NewReaderDict

func NewReader(r io.Reader) io.ReadCloser {
	rr := &decompressor{}
	rr.r = r
	rr.rBuf = bufio.NewReader(r)
	return rr
}

type decompressor struct {
	state         inflate
	writePos      int
	readPos       int
	historyBuffer [2*historySize + lookAhead]uint8
	r             io.Reader
	rBuf          *bufio.Reader
	err           error
	peekSize      int
	eof           bool
}

func (r *decompressor) Reset(under io.Reader, _ []byte) error {
	r.r = under
	if ur, ok := under.(*bufio.Reader); ok {
		r.rBuf = ur
	} else {
		if r.rBuf != nil {
			r.rBuf.Reset(under)
		} else {
			r.rBuf = bufio.NewReader(under)
		}
	}

	r.peekSize = 0
	r.eof = false
	r.err = nil
	r.state.reset()
	return nil
}

func (r *decompressor) Close() error {
	return nil
}

func (f *decompressor) Read(b []byte) (n int, err error) {
	for {
		if f.writePos-f.readPos > 0 {
			num := copy(b, f.historyBuffer[f.readPos:f.writePos])
			f.readPos += num
			n += num
			if f.writePos == f.readPos {
				return n, f.err
			}
			return n, nil
		}
		if f.err != nil {
			return 0, f.err
		}
		f.err = f.step()
		if f.err != nil && f.writePos-f.readPos == 0 {
			return n, f.err
		}
	}
}

func (f *decompressor) step() (err error) {
	state := &f.state

	if state.phase == phaseFinish {
		return io.EOF
	}

	if state.input == nil {
		state.input, err = f.rBuf.Peek(f.rBuf.Size())
		f.peekSize = len(state.input)
		if err != nil && err != bufio.ErrBufferFull && err != io.EOF {
			return err
		}
		f.eof = err == io.EOF
		state.input = state.input[f.state.bitsLen/8:]
	}
	f.readPos = f.writePos

	if f.readPos >= historySize*2 {
		// make sure we are doing non-overlapping copy
		copy(f.historyBuffer[:historySize], f.historyBuffer[f.readPos-historySize:f.readPos])
		f.readPos = historySize
		f.writePos = historySize
	}

	startInputSize, startBitsLen := len(f.state.input), int(f.state.bitsLen)
	err = f.decomperss()
	f.state.rOffset(startInputSize, startBitsLen)

	if isError(err) || (err == errEndInput && f.eof) {
		discardSize := f.peekSize - len(f.state.input) - int(state.bitsLen/8)
		if discardSize > 0 {
			_, err := f.rBuf.Discard(discardSize)
			if err != nil {
				return err
			}
		}
		f.state.input = nil
		if err == errEndInput {
			return io.ErrUnexpectedEOF
		}
		err = CorruptInputError(f.state.roffset)
		return
	}

	err = nil
	if state.phase == phaseStreamEnd && f.writePos == f.readPos {
		state.phase = phaseFinish
		err = io.EOF
	}
	if len(f.state.input) == 0 || state.phase == phaseFinish {
		discardSize := f.peekSize - len(f.state.input) - int(state.bitsLen/8)
		if discardSize > 0 {
			_, err := f.rBuf.Discard(discardSize)
			if err != nil {
				return err
			}
		}
		f.state.input = nil
	}
	return
}

func (f *decompressor) decomperss() (err error) {
	state := &f.state
	limitedBoundary := len(f.historyBuffer) - lookAhead
	output := f.historyBuffer[:limitedBoundary]
	idx := int(f.writePos)
	/* Decode into internal buffer until exit */
	for state.phase != phaseStreamEnd {
		if state.phase == phaseNewBlock || state.phase == phaseDecodingHeader {
			err = state.readHeader()
			if err != nil {
				break
			}
		}

		if state.phase == phaseLitBlock {
			idx, err = state.decodeLiteralBlock(output, idx)
		} else {
			idx, err = decodeHuffman(state, output, idx)
		}

		if err != nil {
			break
		}
	}

	/* Copy valid data from internal buffer into outBuffer */
	if state.writeOverflowLen != 0 {
		binary.LittleEndian.PutUint32(f.historyBuffer[idx:], uint32(state.writeOverflowLits))
		idx += int(state.writeOverflowLen)
		state.writeOverflowLits = 0
		state.writeOverflowLen = 0
	}

	if state.copyOverflowLength != 0 {
		byteCopy(f.historyBuffer[:], idx, int(state.copyOverflowDistance), int(state.copyOverflowLength))
		idx += int(state.copyOverflowLength)
		state.copyOverflowDistance = 0
		state.copyOverflowLength = 0
	}

	f.writePos = (idx)
	return
}
