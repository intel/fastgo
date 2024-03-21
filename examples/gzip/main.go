// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"compress/gzip"
	"flag"
	"io"
	"log"
	"os"

	fastgo "github.com/intel/fastgo/compress/gzip"
)

var (
	f = flag.String("f", "", "input file")
	o = flag.String("o", "", "output file")
	a = flag.String("a", "c", "action: c for compression , d for decompression")
	e = flag.String("e", "fastgo", "engine: fastgo / std")
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
	input  io.Reader
	output io.Writer
)

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

func fastgoGzip() (err error) {
	w := fastgo.NewWriter(output)
	_, err = io.Copy(w, input)
	w.Close()
	return
}

func stdGzip() (err error) {
	w := gzip.NewWriter(output)
	_, err = io.Copy(w, input)
	w.Close()
	return
}

func fastgoGunzip() (err error) {
	r, err := fastgo.NewReader(input)
	if err != nil {
		return err
	}
	_, err = io.Copy(output, r)
	r.Close()
	return
}

func stdGunzip() (err error) {
	r, err := gzip.NewReader(input)
	if err != nil {
		return err
	}
	_, err = io.Copy(output, r)
	r.Close()
	return
}
