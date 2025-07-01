// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

//go:build amd64 && !noasmtest
// +build amd64,!noasmtest

package cpu

// cpuArchLevel detects the Intel CPU architecture level and returns
// an integer representing the optimization capabilities available.
// The actual implementation is in assembly for precise CPU feature detection.
// Returns:
// - 0: No Intel optimizations available
// - 1-4: Different levels of Intel CPU optimizations
func cpuArchLevel() int
