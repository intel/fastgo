package main

import (
	"bytes"
	"io"
	"log"

	"github.com/intel/fastgo/compress/flate"
)

func main() {
	data := []byte(`simple text`)
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.BestSpeed)
	if err != nil {
		log.Fatal(err)
	}
	w.Write(data)
	w.Close()

	r := flate.NewReader(&buf)
	readData, err := io.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(bytes.Equal(data, readData))
}
