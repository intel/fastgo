// Code generated by command: go run main.go. DO NOT EDIT.
//go:build amd64 && !noasmtest

#include "textflag.h"

// func cpuArchLevel() int
// Requires: CPUID
TEXT ·cpuArchLevel(SB), NOSPLIT, $0-8
	XORQ AX, AX
	XORQ CX, CX
	CPUID
	CMPL BX, $0x756e6547
	JNE  level_0
	CMPL DX, $0x49656e69
	JNE  level_0
	CMPL CX, $0x6c65746e
	JNE  level_0
	CMPL AX, $0x00000001
	JL   level_0
	MOVQ $0x00000001, AX
	MOVQ $0x00000000, CX
	CPUID
	XORQ SI, SI
	MOVL DX, SI
	SHLQ $0x20, SI
	ORQ  CX, SI
	MOVQ $0x80000000, AX
	MOVQ $0x00000000, CX
	CPUID
	MOVQ $0x80000001, AX
	MOVQ $0x00000000, CX
	CPUID
	XORQ DI, DI
	MOVL DX, DI
	SHLQ $0x20, DI
	ORQ  CX, DI
	MOVQ $0x00000007, AX
	MOVQ $0x00000000, CX
	CPUID
	XORQ AX, AX
	MOVL CX, AX
	SHLQ $0x20, AX
	ORQ  BX, AX
	MOVQ $0x00982201, CX
	MOVQ SI, DX
	ANDQ CX, DX
	CMPQ DX, CX
	JNE  level_1
	MOVQ $0x00000001, CX
	MOVQ DI, DX
	ANDQ CX, DX
	CMPQ DX, CX
	JNE  level_1
	MOVQ $0x00000128, CX
	MOVQ AX, DX
	ANDQ CX, DX
	CMPQ DX, CX
	JNE  level_2
	MOVQ $0x00000020, CX
	MOVQ DI, DX
	ANDQ CX, DX
	CMPQ DX, CX
	JNE  level_2
	MOVQ $0x38401000, CX
	MOVQ SI, DX
	ANDQ CX, DX
	CMPQ DX, CX
	JNE  level_2
	MOVQ $0xd0030000, CX
	ANDQ CX, AX
	CMPQ AX, CX
	JNE  level_3
	MOVQ $0x00000004, AX
	MOVQ AX, ret+0(FP)
	RET

level_1:
	MOVQ $0x00000001, AX
	MOVQ AX, ret+0(FP)
	RET

level_2:
	MOVQ $0x00000002, AX
	MOVQ AX, ret+0(FP)
	RET

level_3:
	MOVQ $0x00000003, AX
	MOVQ AX, ret+0(FP)
	RET

level_0:
	MOVQ $0x00000000, AX
	MOVQ AX, ret+0(FP)
	RET
