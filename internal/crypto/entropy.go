// Package crypto contains config.bin container parsing, entropy analysis and
// heuristic key discovery for Tozed S20 configuration backups.
package crypto

import (
	"math"
)

// Shannon returns the Shannon entropy of data in bits per byte.
func Shannon(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}
	freq := make(map[byte]int, 256)
	for _, b := range data {
		freq[b]++
	}
	n := float64(len(data))
	var h float64
	for _, count := range freq {
		p := float64(count) / n
		h -= p * math.Log2(p)
	}
	return h
}

// Block describes one block of an entropy report.
type Block struct {
	Offset  int     `json:"offset"`
	Length  int     `json:"length"`
	Entropy float64 `json:"entropy"`
	Class   string  `json:"class"`
}

// Classify maps an entropy value to a rough content class.
func Classify(e float64) string {
	switch {
	case e >= 7.5:
		return "encrypted/compressed"
	case e >= 6.0:
		return "mixed"
	default:
		return "structured"
	}
}

// BlockReport splits data into fixed-size blocks and scores each one.
func BlockReport(data []byte, blockSize int) []Block {
	if blockSize < 16 {
		blockSize = 4096
	}
	blocks := make([]Block, 0, (len(data)+blockSize-1)/blockSize)
	for off := 0; off < len(data); off += blockSize {
		end := off + blockSize
		if end > len(data) {
			end = len(data)
		}
		e := Shannon(data[off:end])
		blocks = append(blocks, Block{
			Offset:  off,
			Length:  end - off,
			Entropy: e,
			Class:   Classify(e),
		})
	}
	return blocks
}
