// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

// This example demonstrates basic usage of Intel FastGo's optimized flate compression.
// It shows how to compress and decompress data using the DEFLATE algorithm with
// Intel-specific optimizations for improved performance.
package main

import (
	"bytes"
	"io"
	"log"

	"github.com/intel/fastgo/compress/flate"
)

func main() {
	// Sample data to compress
	data := []byte(`simple text`)
	var buf bytes.Buffer

	// Create a Writer to compress data using Intel-optimized DEFLATE
	// BestSpeed (level 1) is one of the optimized compression levels
	w, err := flate.NewWriter(&buf, flate.BestSpeed)
	if err != nil {
		log.Fatal(err)
	}

	// Write and close to flush all data
	w.Write(data)
	w.Close()

	// Create a Reader to decompress data using Intel-optimized DEFLATE
	r := flate.NewReader(&buf)
	readData, err := io.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}

	// Verify that the decompressed data matches the original
	log.Println("Data matches:", bytes.Equal(data, readData))
}
