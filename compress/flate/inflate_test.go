// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package flate

import (
	"bufio"
	"bytes"
	"compress/flate"
	"crypto/rand"
	_ "embed"
	"io"
	"os"
	"runtime"
	"testing"
)

//go:embed testdata/sparse_data_sample
var sparsedata []byte

func compress(data []byte) []byte {
	buf := bytes.NewBuffer(nil)
	w, _ := flate.NewWriter(buf, flate.DefaultCompression)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

func TestReader(t *testing.T) {
	textfile := opticks(t)
	input := compress(textfile)
	r := NewReader(bytes.NewReader(input))
	var err error
	data, err := io.ReadAll(r)
	n := len(data)
	if err != nil && err != io.EOF {
		t.Fatal(err, n, bytes.Equal(data[:n], textfile[:n]))
	}
	if !bytes.Equal(data[:n], textfile) {
		t.Fatal()
	}
}

func TestReaderLastBytes(t *testing.T) {
	restSizes := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	testdata := opticks(t)
	for i := 0; i < len(restSizes); i++ {
		input := compress(testdata[:256*i])
		rdsize := restSizes[i]

		rddata := make([]byte, rdsize)
		rand.Read(rddata)
		input = append(input, rddata...)
		br := bufio.NewReader(bytes.NewReader(input))
		r := NewReader(br)
		var err error
		data, err := io.ReadAll(r)
		n := len(data)
		if err != nil && err != io.EOF {
			t.Fatal(err, n, bytes.Equal(data[:n], testdata[:n]))
		}
		if !bytes.Equal(data[:n], testdata[:n]) {
			t.Fatal()
		}
		rddataTest := make([]byte, rdsize)
		n, err = br.Read(rddataTest)
		if !bytes.Equal(rddataTest, rddata) {
			t.Fatal("rest bytes wrong", err, n)
		}
	}
}

func benchmarkDecomp(decompressor io.Reader, compressed []byte) func(b *testing.B) {
	input := bytes.NewReader(compressed)
	output := bytes.NewBuffer(make([]byte, len(compressed)*5))
	output.Reset()
	input.Reset(compressed)

	return func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			decompressor.(Resetter).Reset(input, nil)
			io.Copy(output, decompressor)
			b.SetBytes(int64(output.Len()))
			output.Reset()
			input.Reset(compressed)
		}
	}
}

func BenchmarkInflate(b *testing.B) {
	b.ResetTimer()
	raw := opticks(b)
	input := compress(raw)
	b.Log(float64(len(raw)) / float64(len(input)))
	b.Run("method=fastgo", benchmarkDecomp(NewReader(nil), input))
	b.Run("method=flate", benchmarkDecomp(flate.NewReader(nil), input))
}

func BenchmarkInflateSparseData(b *testing.B) {
	temp := make([]byte, len(sparsedata)*16)
	for i := 0; i < 16; i++ {
		copy(temp[i*len(sparsedata):], sparsedata)
	}
	sparsedata := temp

	input := compress(sparsedata)
	b.Run("gostandard", benchmarkDecomp(flate.NewReader(nil), input))
	b.Run("fastgo", benchmarkDecomp(NewReader(nil), input))
}

func opticks(t testing.TB) (data []byte) {
	data, _ = os.ReadFile(runtime.GOROOT() + "/src/testdata/Isaac.Newton-Opticks.txt")
	if data == nil {
		t.Skip("skip for no test data file")
	}
	return data
}
