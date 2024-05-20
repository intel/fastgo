//go:build go1.18
// +build go1.18

package flate

import (
	"bytes"
	"io"
	"testing"
)

func FuzzInflate(f *testing.F) {
	data := opticks(f)
	f.Add(data)
	f.Fuzz(func(t *testing.T, source []byte) {
		input := compress(source)
		r := NewReader(bytes.NewReader(input))
		var err error
		data, err := io.ReadAll(r)
		n := len(data)
		if err != nil && err != io.EOF {
			t.Fatal(err, n, bytes.Equal(data[:n], source[:n]))
		}
		if !bytes.Equal(data[:n], source) {
			t.Fatal()
		}
	})
}
