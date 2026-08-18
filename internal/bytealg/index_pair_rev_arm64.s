// Copyright 2026 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

#include "textflag.h"

// func LastIndexPairFold(s string, pair uint32) int
//
// LastIndexPairFold returns the index of the last position i in s where
// s[i] matches c0 and s[i+1] matches c1 where the four bytes packed
// into pair are the lower and upper case forms of c0 and c1 (see
// IndexPairFold and PackPair).
//
// input:
//   R0: s ptr
//   R1: s len
//   R2: packed pair
// return:
//   R0: result
TEXT ·LastIndexPairFold(SB), NOSPLIT, $0-32
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
	CMP  $2, R1
	BLT  fail                // len(s) < 2

	// Skip the SIMD loop for short strings.
	CMP  $34, R1
	BLT  scalar_start

	VMOV R3, V16.B16         // dup c0l
	VMOV R4, V17.B16         // dup c0u
	VMOV R5, V18.B16         // dup c1l
	VMOV R6, V19.B16         // dup c1u

	// Magic constant 0x40100401 allows us to identify which lane matches
	// the requested byte (see indexbyte_arm64.s for details).
	MOVD $0x40100401, R7
	VMOV R7, V21.S4

	// Process 32 byte chunks from the end. The chunk at p covers pair
	// starts p..p+31 (the highest valid pair start is end-2 so the first
	// chunk begins at end-33).
	SUB  $33, R12, R10       // p = end - 33

loop:
	CMP  R11, R10
	BLT  scalar_head         // p < start of s

	VLD1 (R10), [V0.B16, V1.B16]
	ADD  $1, R10, R7
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

	SUB  $32, R10
	B    loop

found:
	// Compute the syndrome value (2 bits per byte) and locate the last
	// matching lane (see indexbyte_arm64.s for details).
	VAND  V21.B16, V4.B16, V4.B16
	VAND  V21.B16, V5.B16, V5.B16
	VADDP V5.B16, V4.B16, V6.B16 // 256->128
	VADDP V6.B16, V6.B16, V6.B16 // 128->64
	VMOV  V6.D[0], R7
	CLZ   R7, R7
	MOVD  $63, R9
	SUB   R7, R9, R7             // index of the highest set bit
	ADD   R7>>1, R10, R0         // R7 is twice the offset into the chunk
	SUB   R11, R0, R0
	MOVD  R0, (R8)
	RET

scalar_head:
	// The chunks processed so far cover pair starts p+32..end-2 (where
	// p+32 = R10+32). Scan the remaining starts start..R10+31 backwards.
	ADD  $31, R10, R9            // highest remaining pair start
	B    scalar_loop

scalar_start:
	SUB  $2, R12, R9             // last valid pair start (inclusive)

scalar_loop:
	CMP   R11, R9
	BLT   fail
	MOVBU (R9), R7
	MOVBU 1(R9), R10
	CMP   R3, R7
	BEQ   scalar_c1
	CMP   R4, R7
	BNE   scalar_next

scalar_c1:
	CMP   R5, R10
	BEQ   scalar_found
	CMP   R6, R10
	BNE   scalar_next

scalar_found:
	SUB  R11, R9, R0
	MOVD R0, (R8)
	RET

scalar_next:
	SUB  $1, R9
	B    scalar_loop

fail:
	MOVD $-1, R0
	MOVD R0, (R8)
	RET
