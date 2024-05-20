//go:build go1.18
// +build go1.18

package deflate

import (
	"bytes"
	"compress/flate"
	"io"
	"testing"
)

func FuzzDeflate(f *testing.F) {
	data := opticks(f)
	f.Add(data)
	f.Fuzz(func(t *testing.T, source []byte) {
		for lvl := range testLevels {
			buf := bytes.NewBuffer(nil)
			w, err := NewWriter(buf, lvl)
			if err != nil {
				t.Fatal(err)
			}
			_, err = io.Copy(w, bytes.NewBuffer(source))
			if err != nil {
				t.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(flate.NewReader(buf))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(data, source) {
				t.Fatal(err)
			}
			buf.Reset()
			w.Reset(buf)
			_, err = io.Copy(w, bytes.NewBuffer(source))
			if err != nil {
				t.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
	})
}
