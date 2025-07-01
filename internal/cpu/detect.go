// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

//go:build !amd64 || noasmtest
// +build !amd64 noasmtest

package cpu

// cpuArchLevel returns 0 for non-AMD64 architectures or when assembly tests are disabled.
// This ensures that Intel-specific optimizations are only available on supported platforms.
func cpuArchLevel() int {
	return 0
}
