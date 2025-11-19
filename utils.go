package main

import (
	"hash/fnv"
)


func passwordToSeed(password string) int64 {
	h := fnv.New64a()
	h.Write([]byte(password))
	return int64(h.Sum64())
}

func bitsToText(bits []int) string {
	var bytes []byte
	for i := 0; i < len(bits); i += 8 {
		var b byte
		for j := 0; j < 8; j++ {
			b = (b << 1) | byte(bits[i+j])
		}
		bytes = append(bytes, b)
	}
	return string(bytes)
}

func IntTo32Bits(n int) []int {
	bits := make([]int, 32)
	for i := range 32 {
		bits[31-i] = (n >> i) & 1 
	}
	return bits
}

func BitsToInt(bits []int) int {
	result := 0
	for _, bit := range bits {
		result = (result << 1) | bit
	}
	return result
}
func TextToBits(s string) []int {
	var bits []int
	for _, b := range []byte(s) {
		for i := 7; i >= 0; i-- {
			bits = append(bits, int((b>>i)&1))
		}
	}
	return bits
}

// BitsToText: [0,1,0,0,0,0,0,1] -> "A"
func BitsToText(bits []int) string {
	var bytes []byte
	// We process 8 bits at a time
	for i := 0; i < len(bits); i += 8 {
		var b byte
		// Reconstruct the byte
		for j := range 8 {
			if i+j < len(bits) {
				b = (b << 1) | byte(bits[i+j])
			}
		}
		bytes = append(bytes, b)
	}
	return string(bytes)
}


