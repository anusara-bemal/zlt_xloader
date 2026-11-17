package crypto

import (
	"encoding/binary"
	"fmt"
	"os"
)

// Container is a parsed config.bin with its header, payload and signature
// regions identified.
type Container struct {
	Path            string  `json:"path"`
	Size            int     `json:"size"`
	Magic           string  `json:"magic"`
	HeaderLength    int     `json:"header_length"`
	PayloadOffset   int     `json:"payload_offset"`
	PayloadLength   int     `json:"payload_length"`
	SignatureLength int     `json:"signature_length"`
	Blocks          []Block `json:"blocks"`
}

// Parse reads and structurally analyses a config.bin. headerLen and
// signatureLen of 0 trigger automatic detection; -1 disables that region.
func Parse(path string, headerLen, signatureLen, blockSize int) (*Container, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%q is empty", path)
	}

	c := &Container{Path: path, Size: len(data), Magic: detectMagic(data)}

	if headerLen < 0 {
		headerLen = 0
	}
	if signatureLen < 0 {
		signatureLen = 0
	}
	if headerLen == 0 {
		headerLen = detectHeader(data)
	}
	if signatureLen == 0 {
		signatureLen = detectSignature(data)
	}
	if headerLen+signatureLen > len(data) {
		return nil, fmt.Errorf("header (%d) + signature (%d) exceed file size (%d)", headerLen, signatureLen, len(data))
	}

	c.HeaderLength = headerLen
	c.PayloadOffset = headerLen
	c.PayloadLength = len(data) - headerLen - signatureLen
	c.SignatureLength = signatureLen
	c.Blocks = BlockReport(data[c.PayloadOffset:c.PayloadOffset+c.PayloadLength], blockSize)
	return c, nil
}

// Payload returns the encrypted/config payload region of the file.
func (c *Container) Payload(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	start := c.PayloadOffset
	end := start + c.PayloadLength
	if end > len(data) {
		return nil, fmt.Errorf("payload range out of bounds")
	}
	return data[start:end], nil
}

func detectMagic(data []byte) string {
	limit := len(data)
	if limit > 64 {
		limit = 64
	}
	window := data[:limit]

	// ZIP-based config containers.
	if len(data) >= 4 && data[0] == 'P' && data[1] == 'K' && data[2] == 3 && data[3] == 4 {
		return "PK\\x03\\x04"
	}

	// Common ASCII magic tags.
	for _, tag := range []string{"TCTZ", "ZLT", "CFG", "WRG0"} {
		if indexOf(window, []byte(tag)) >= 0 {
			return tag
		}
	}
	return ""
}

func detectHeader(data []byte) int {
	// Many Tozed config files begin with a short type/length header:
	//   2 bytes type + 2 bytes big-endian length + reserved bytes.
	if len(data) >= 8 {
		declared := int(binary.BigEndian.Uint16(data[2:4]))
		if declared > 0 && declared <= len(data) && declared%4 == 0 {
			return 8
		}
	}
	return 8
}

func detectSignature(data []byte) int {
	// A trailing signature is commonly 16 or 32 bytes. Choose the size that
	// makes the preceding payload's entropy look least like random noise.
	for _, size := range []int{32, 16, 8, 0} {
		if size >= len(data) {
			continue
		}
		payload := data[:len(data)-size]
		if len(payload) == 0 {
			continue
		}
		if Shannon(payload) < 7.9 {
			return size
		}
	}
	return 0
}

func indexOf(haystack, needle []byte) int {
	if len(needle) == 0 {
		return 0
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j, b := range needle {
			if haystack[i+j] != b {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
