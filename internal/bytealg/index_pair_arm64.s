// Copyright 2026 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

#include "textflag.h"

// func IndexPairFold(s string, pair uint32) int
//
// IndexPairFold returns the index of the first position i in s where
// s[i] matches c0 and s[i+1] matches c1 where the four bytes packed
// into pair are:
//
//	c0l = byte(pair)       // lower case form of c0
//	c0u = byte(pair >> 8)  // upper case form of c0 (equal to c0l if c0 is not a letter)
//	c1l = byte(pair >> 16) // lower case form of c1
//	c1u = byte(pair >> 24) // upper case form of c1 (equal to c1l if c1 is not a letter)
//
// input:
//   R0: s ptr
//   R1: s len
//   R2: packed pair
// return:
//   R0: result
TEXT ·IndexPairFold(SB), NOSPLIT, $0-32
	MOVD  s_base+0(FP), R0
	MOVD  s_len+8(FP), R1
	MOVWU pair+16(FP), R2
	MOVD  $ret+24(FP), R8

	// Unpack the two case forms of each byte.
	AND  $0xFF, R2, R3       // c0l
	UBFX $8, R2, $8, R4      // c0u
	UBFX $16, R2, $8, R5     // c1l
	LSR  $24, R2, R6         // c1u

	MOVD R0, R11             // start of s (for index calculation)
	ADD  R0, R1, R12         // end of s
	SUB  $2, R12, R13        // last valid pair start (inclusive)
	CMP  $2, R1
	BLT  fail                // len(s) < 2

	// Skip the SIMD loop for short strings.
	CMP  $34, R1
	BLT  tail

	VMOV R3, V16.B16         // dup c0l
	VMOV R4, V17.B16         // dup c0u
	VMOV R5, V18.B16         // dup c1l
	VMOV R6, V19.B16         // dup c1u

	// Magic constant 0x40100401 allows us to identify which lane matches
	// the requested byte (see indexbyte_arm64.s for details).
	MOVD $0x40100401, R7
	VMOV R7, V21.S4

	SUB  $33, R12, R14       // last SIMD chunk start: need s[p:p+33] in bounds

loop:
	CMP  R14, R0
	BHI  tail

	// Load 32 bytes at p and 32 bytes at p+1. A pair starts at byte i of
	// the chunk iff chunkA[i] matches c0 and chunkB[i] matches c1.
	VLD1 (R0), [V0.B16, V1.B16]
	ADD  $1, R0, R7
	VLD1 (R7), [V2.B16, V3.B16]

	// V4, V5 = chunkA matches c0 (either case)
	VCMEQ V16.B16, V0.B16, V4.B16
	VCMEQ V17.B16, V0.B16, V6.B16
	VORR  V6.B16, V4.B16, V4.B16
	VCMEQ V16.B16, V1.B16, V5.B16
	VCMEQ V17.B16, V1.B16, V6.B16
	VORR  V6.B16, V5.B16, V5.B16

	// V4, V5 &= chunkB matches c1 (either case)
	VCMEQ V18.B16, V2.B16, V6.B16
	VCMEQ V19.B16, V2.B16, V7.B16
	VORR  V7.B16, V6.B16, V6.B16
	VAND  V6.B16, V4.B16, V4.B16
	VCMEQ V18.B16, V3.B16, V6.B16
	VCMEQ V19.B16, V3.B16, V7.B16
	VORR  V7.B16, V6.B16, V6.B16
	VAND  V6.B16, V5.B16, V5.B16

	// Fast check for any match in this chunk.
	VORR  V5.B16, V4.B16, V6.B16
	VADDP V6.D2, V6.D2, V6.D2
	VMOV  V6.D[0], R7
	CBNZ  R7, found

	ADD  $32, R0
	B    loop

found:
	// Compute the syndrome value (2 bits per byte) to locate the first
	// matching lane (see indexbyte_arm64.s for details).
	VAND  V21.B16, V4.B16, V4.B16
	VAND  V21.B16, V5.B16, V5.B16
	VADDP V5.B16, V4.B16, V6.B16 // 256->128
	VADDP V6.B16, V6.B16, V6.B16 // 128->64
	VMOV  V6.D[0], R7
	RBIT  R7, R7
	CLZ   R7, R7
	ADD   R7>>1, R0, R0          // R7 is twice the offset into the chunk
	SUB   R11, R0, R0
	MOVD  R0, (R8)
	RET

tail:
	// Scalar loop for the remaining (at most 33) bytes.
	CMP   R13, R0
	BHI   fail
	MOVBU (R0), R7
	MOVBU 1(R0), R9
	CMP   R3, R7
	BEQ   tail_c1
	CMP   R4, R7
	BNE   tail_next

tail_c1:
	CMP   R5, R9
	BEQ   tail_found
	CMP   R6, R9
	BNE   tail_next

tail_found:
	SUB  R11, R0, R0
	MOVD R0, (R8)
	RET

tail_next:
	ADD  $1, R0
	B    tail

fail:
	MOVD $-1, R0
	MOVD R0, (R8)
	RET
