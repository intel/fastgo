// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

// Package cpu provides CPU architecture detection functionality
// for Intel FastGo optimizations. It determines the architecture
// level to enable appropriate performance optimizations.
package cpu

// ArchLevel represents the detected CPU architecture level.
// Different levels enable different sets of optimizations:
// - 0: No Intel-specific optimizations available
// - 1+: Various levels of Intel CPU optimizations available
// The value is determined at package initialization time.
var (
	ArchLevel = cpuArchLevel()
)
