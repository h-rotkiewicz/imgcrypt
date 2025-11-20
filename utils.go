package main

import (
	"fmt"
	"hash/fnv"
	"image"
	"math/rand"
	"time"
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

func GenerateRandomPassword() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}

func GeneratePointsInRange(width, height int, seed int64, count int, startIdx, endIdx int) ([]image.Point, error) {
	windowSize := endIdx - startIdx
	
	if windowSize <= 0 {
		return nil, fmt.Errorf("invalid window range")
	}
	if count > windowSize {
		return nil, fmt.Errorf("not enough pixels in window for requested count")
	}

	r := rand.New(rand.NewSource(seed))
	
	perm := r.Perm(windowSize)

	var points []image.Point
	
	for i := 0; i < count && i < len(perm); i++ {
		globalIndex := startIdx + perm[i]

		x := globalIndex % width
		y := globalIndex / width
		points = append(points, image.Point{X: x, Y: y})
	}
	return points, nil
}
