// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package deflate

import (
	"bytes"
	"compress/flate"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"text/tabwriter"
)

var testLevels = []int{
	HuffmanOnly, BestSpeed, DefaultCompression,
}

func opticks(t testing.TB) (data []byte) {
	data, _ = os.ReadFile(filepath.Join(runtime.GOROOT(), "src", "testdata", "Isaac.Newton-Opticks.txt"))
	if data == nil {
		t.Skip("skip for no test data file")
	}
	return data
}

func TestWrite(t *testing.T) {
	testdata := opticks(t)

	for size := 1; size < 128*1024; size *= 2 {
		for offset := range []int{0, 1, 3, 5, 7, 9, 17} {
			offsetSize := size + offset
			if len(testdata) < offsetSize {
				break
			}
			source := testdata[:offsetSize]
			for _, lvl := range testLevels {

				buf := bytes.NewBuffer(nil)
				w, _ := NewWriter(buf, lvl)
				w.Write(source)
				w.Close()
				r := flate.NewReader(bytes.NewReader(buf.Bytes()))
				data, err := io.ReadAll(r)
				if err != nil {
					t.Fatal(lvl, err, len(source), len(data), diff(data, source))
				}
				if !bytes.Equal(data, source) {
					t.Fatalf("excepted data is the same, but failed.  level: %d,data_len:%d,source_len:%d,diff:%d \n", lvl, len(data), len(source), diff(data, source))
				}
			}
		}
	}
}

func diff(d, s []byte) (pos int) {
	pos = -1
	for i := 0; i < len(d); i++ {
		if d[i] != s[i] {
			pos = i
			break
		}
	}
	return
}

func TestCompressionRatio(t *testing.T) {
	cw := tabwriter.NewWriter(os.Stderr, 0, 15, 1, ' ', tabwriter.AlignRight)
	fmt.Fprintln(cw, "lvl1\tlvl2\tstd_lvl1\tstd_lvl6\tstd_lvl9\t")
	data := opticks(t)

	xdata := data[:]
	buf := bytes.NewBuffer(nil)
	var records []string
	for i := 1; i <= 2; i++ {
		buf.Reset()
		w, _ := NewWriter(buf, i)
		w.Write(xdata)
		w.Close()
		records = append(records, fmt.Sprintf("%.2f", float64(buf.Len())/float64(len(data[:]))))
	}

	for _, level := range []int{
		flate.BestSpeed, flate.DefaultCompression, flate.BestCompression,
	} {
		buf.Reset()
		sw, _ := flate.NewWriter(buf, level)
		sw.Write(xdata)
		sw.Close()
		records = append(records, fmt.Sprintf("%.2f", float64(buf.Len())/float64(len(data[:]))))
	}
	fmt.Fprintln(cw, strings.Join(records, "\t")+"\t")
	cw.Flush()
}

func BenchmarkDynamicCompress(b *testing.B) {
	data := opticks(b)
	for _, lvl := range testLevels {
		for i := 4; i <= 64; i *= 2 {
			input := data[:i*1024]
			subfix := "@size=" + strconv.Itoa(i) + "KB,level=" + strconv.Itoa(lvl)
			b.Run("fastgo"+subfix, func(b *testing.B) {
				w, _ := NewWriter(io.Discard, lvl)
				for i := 0; i < b.N; i++ {
					b.SetBytes(int64(len(input)))
					w.Write(input)
					w.Close()
					w.Reset(io.Discard)
				}
			})

			sw, _ := flate.NewWriter(io.Discard, lvl)
			b.Run("std"+subfix, func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					b.SetBytes(int64(len(input)))
					sw.Write(input)
					sw.Close()
					sw.Reset(io.Discard)
				}
			})
		}
	}
}
