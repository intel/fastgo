# Intel FastGo 

[![GoDoc](https://godoc.org/github.com/intel/fastgo?status.svg)](https://pkg.go.dev/github.com/intel/fastgo)
[![Go Report Card](https://goreportcard.com/badge/github.com/intel/fastgo)](https://goreportcard.com/report/github.com/intel/fastgo)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/8690/badge)](https://www.bestpractices.dev/projects/8690)
[![License](https://img.shields.io/github/license/intel/fastgo)](./LICENSE)


## Introduction

Intel FastGo provides optimized Go packages, such as compress/flate and compress/gzip, for Go applications running on the Xeon platform. It is trying to provide an interface as close as possible to the standard libraries in order to provide a good user experience.

## Installation

### Requirements

- Go >= 1.20


### Steps

```sh
go get -u github.com/intel/fastgo
```

## Quick Start

```go
// examples/flate/main.go
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
	// Create a Writer to compress data
	w, err := flate.NewWriter(&buf, flate.BestSpeed)
	if err != nil {
		log.Fatal(err)
	}
	w.Write(data)
	w.Close()

	// Create a Reader to decompress data
	r := flate.NewReader(&buf)
	readData, err := io.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(bytes.Equal(data, readData))
}

```

### Example 

please check [gzip example](examples/gzip/main.go)

## Features and Functionality

Intel FastGo primarily offers optimizations related to compression and decompression, with a special focus on support for the [Deflate](https://www.rfc-editor.org/rfc/rfc1951) compression format. This includes enhancements for the compress/flate, compress/gzip, and compress/zlib packages. The APIs provided by Intel FastGo are designed to be consistent with the corresponding standard library interfaces, facilitating a seamless Drop-In Replacement for users.

### Limitations

Currently, Intel FastGo's compression capabilities are optimized only for levels 1, 2, and Huffman-only. At present, both compression and decompression do not support custom dictionaries. In cases where acceleration is not supported, we fallback to processing with the standard library.

### Features
- Deflate
    - Comression Acceleration
        - Level1 
        - Level2
        - Huffmanonly
    - Decompression Acceleration
- Gzip Format
- Zlib Format

### Future Developments
- Custom Dictionary Support: Enabling compression and decompression with custom dictionaries.
- Broader Compression Level Support: Extending optimization to cover more compression levels.
- More Packages Support: Including more packages, e.g. hash/crc32 
- Hardware Accelerator Support: Introducing support for hardware accelerators such as IAA (Intel® In-Memory Accelerator), DSA (Data Streaming Accelerator), and QAT (QuickAssist Technology) in upcoming releases. This enhancement aims to automatically detect specific scenarios and utilize the corresponding hardware accelerators to boost performance.
 
## Documentation

[Full Documentation](https://pkg.go.dev/github.com/intel/fastgo)


## Platform Compatibility
Please note that the optimizations included in this project have been exclusively tested and verified on Intel platforms. While the project may function on other platforms, we cannot guarantee optimal performance or compatibility outside of Intel environments.

## Contributing

If you would like to contribute to the project, please read the [Contribution Guidelines](./CONTRIBUTING.md).


## License

[BSD 3-Clause License](./LICENSE)


## Acknowledgments

Thanks to all contributors.

Special thanks to the Go Authors for their invaluable work on the Go language. Parts of this project were inspired by or directly derived from Go's source code. We appreciate their contributions to the open source community.

## Contact
If you have any questions, you can create an issue.
