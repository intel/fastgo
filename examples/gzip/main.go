// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

// This example demonstrates Intel FastGo's optimized gzip compression and decompression.
// It provides a command-line tool to compare performance between Intel FastGo
// and standard library implementations for both compression and decompression operations.
package main

import (
	"compress/gzip"
	"flag"
	"io"
	"log"
	"os"

	fastgo "github.com/intel/fastgo/compress/gzip"
)

// Command-line flags for configuration
var (
	f = flag.String("f", "", "input file (use stdin if empty)")
	o = flag.String("o", "", "output file (use stdout if empty)")
	a = flag.String("a", "c", "action: 'c' for compression, 'd' for decompression")
	e = flag.String("e", "fastgo", "engine: 'fastgo' for Intel optimization, 'std' for standard library")
)

func main() {
	flag.Parse()
	err := initInputOutput()
	if err != nil {
		log.Fatalln(err)
		return
	}
	action := *e + "/" + *a
	switch action {
	case "fastgo/c":
		err = fastgoGzip()
	case "fastgo/d":
		err = fastgoGunzip()
	case "std/c":
		err = stdGzip()
	case "std/d":
		err = stdGunzip()
	}
	if err != nil {
		log.Fatalln(err)
		return
	}
}

var (
	input  io.Reader // Input data source
	output io.Writer // Output data destination
)

// initInputOutput sets up input and output streams based on command-line flags.
// Uses stdin/stdout if no files are specified.
func initInputOutput() (err error) {
	if *f == "" {
		input = os.Stdin
	} else {
		input, err = os.Open(*f)
		if err != nil {
			return err
		}
	}
	if *o == "" {
		output = os.Stdout
	} else {
		output, err = os.Create(*o)
		if err != nil {
			return err
		}
	}
	return nil
}

// fastgoGzip compresses data using Intel FastGo's optimized gzip implementation.
func fastgoGzip() (err error) {
	w := fastgo.NewWriter(output)
	_, err = io.Copy(w, input)
	w.Close()
	return
}

// stdGzip compresses data using the standard library gzip implementation.
func stdGzip() (err error) {
	w := gzip.NewWriter(output)
	_, err = io.Copy(w, input)
	w.Close()
	return
}

// fastgoGunzip decompresses data using Intel FastGo's optimized gzip implementation.
func fastgoGunzip() (err error) {
	r, err := fastgo.NewReader(input)
	if err != nil {
		return err
	}
	_, err = io.Copy(output, r)
	r.Close()
	return
}

// stdGunzip decompresses data using the standard library gzip implementation.
func stdGunzip() (err error) {
	r, err := gzip.NewReader(input)
	if err != nil {
		return err
	}
	_, err = io.Copy(output, r)
	r.Close()
	return
}
