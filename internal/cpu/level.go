// Copyright (c) 2024, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

//go:build amd64 && !noasmtest

package cpu

// check arch level while loading package
var (
	ArchLevel = cpuArchLevel()
)
