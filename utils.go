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

func GenerateRandomPassword() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
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

// Converts a byte slice (e.g., encrypted data) into a slice of bits (0s and 1s)
func BytesToBits(data []byte) []int {
	var bits []int
	for _, b := range data {
		for i := 7; i >= 0; i-- {
			bits = append(bits, int((b>>i)&1))
		}
	}
	return bits
}

// Converts a slice of bits back into a byte slice
func BitsToBytes(bits []int) []byte {
	var bytes []byte
	for i := 0; i < len(bits); i += 8 {
		var b byte
		for j := 0; j < 8; j++ {
			if i+j < len(bits) && bits[i+j] == 1 {
				b |= 1 << (7 - j)
			}
		}
		bytes = append(bytes, b)
	}
	return bytes
}
