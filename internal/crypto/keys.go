package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
)

// Candidate is one scored decryption attempt.
type Candidate struct {
	Key     string  `json:"key"`
	Algo    string  `json:"algorithm"`
	Score   float64 `json:"score"` // lower = more likely plaintext
	Preview string  `json:"preview"`
}

// DeriveKeys builds a heuristic key list from profile constants and the
// device's serial/IMEI (common key sources in Tozed firmware).
func DeriveKeys(known []string, serial, imei string) []string {
	keys := append([]string(nil), known...)
	for _, id := range []string{serial, imei} {
		if id == "" {
			continue
		}
		keys = append(keys,
			id,
			fmt.Sprintf("%x", md5.Sum([]byte(id))),
			fmt.Sprintf("%x", sha256.Sum256([]byte(id))),
		)
	}

	// De-duplicate while preserving order.
	seen := make(map[string]bool, len(keys))
	out := keys[:0]
	for _, k := range keys {
		if k != "" && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	return out
}

func keyVariants(s string) [][]byte {
	out := [][]byte{[]byte(s)}
	if decoded, err := hex.DecodeString(s); err == nil && len(decoded) > 0 {
		out = append(out, decoded)
	}
	return out
}

// TryDecrypt attempts every key variant and algorithm, returning candidates
// sorted from most to least plausible.
func TryDecrypt(data []byte, keys []string, algos []string, limit int) []Candidate {
	out := make([]Candidate, 0)
	for _, k := range keys {
		for _, variant := range keyVariants(k) {
			for _, algo := range algos {
				plain, err := Decrypt(data, variant, algo)
				if err != nil {
					continue
				}
				out = append(out, Candidate{
					Key:     k,
					Algo:    algo,
					Score:   Score(plain),
					Preview: preview(plain),
				})
			}
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Score < out[j].Score })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// Decrypt decrypts data with the selected algorithm.
func Decrypt(data, key []byte, algo string) ([]byte, error) {
	switch strings.ToLower(algo) {
	case "xor":
		out := make([]byte, len(data))
		for i, b := range data {
			out[i] = b ^ key[i%len(key)]
		}
		return out, nil
	case "xor-add":
		out := make([]byte, len(data))
		for i, b := range data {
			out[i] = b + key[i%len(key)]
		}
		return out, nil
	case "aes-cbc":
		block, err := aes.NewCipher(fitKey(key, 16, 24, 32))
		if err != nil {
			return nil, err
		}
		return cbcDecrypt(block, data)
	case "des-cbc":
		block, err := des.NewCipher(fitKey(key, 8))
		if err != nil {
			return nil, err
		}
		return cbcDecrypt(block, data)
	default:
		return nil, fmt.Errorf("unknown algorithm %q", algo)
	}
}

// Encrypt is the symmetric counterpart of Decrypt, useful for building test
// vectors and for re-encrypting modified config blobs.
func Encrypt(data, key []byte, algo string) ([]byte, error) {
	switch strings.ToLower(algo) {
	case "xor":
		out := make([]byte, len(data))
		for i, b := range data {
			out[i] = b ^ key[i%len(key)]
		}
		return out, nil
	case "xor-add":
		out := make([]byte, len(data))
		for i, b := range data {
			out[i] = b - key[i%len(key)]
		}
		return out, nil
	case "aes-cbc":
		block, err := aes.NewCipher(fitKey(key, 16, 24, 32))
		if err != nil {
			return nil, err
		}
		return cbcEncrypt(block, data)
	case "des-cbc":
		block, err := des.NewCipher(fitKey(key, 8))
		if err != nil {
			return nil, err
		}
		return cbcEncrypt(block, data)
	default:
		return nil, fmt.Errorf("unknown algorithm %q", algo)
	}
}

func cbcEncrypt(block cipher.Block, data []byte) ([]byte, error) {
	bs := block.BlockSize()
	if len(data) == 0 || len(data)%bs != 0 {
		return nil, fmt.Errorf("plaintext length %d is not a multiple of block size %d", len(data), bs)
	}
	out := make([]byte, len(data))
	iv := make([]byte, bs)
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, data)
	return out, nil
}

func cbcDecrypt(block cipher.Block, data []byte) ([]byte, error) {
	bs := block.BlockSize()
	if len(data) == 0 || len(data)%bs != 0 {
		return nil, fmt.Errorf("ciphertext length %d is not a multiple of block size %d", len(data), bs)
	}
	out := make([]byte, len(data))
	iv := make([]byte, bs) // zero IV, used by many embedded firmwares
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, data)
	return out, nil
}

func fitKey(key []byte, sizes ...int) []byte {
	target := sizes[len(sizes)-1]
	for _, s := range sizes {
		if len(key) <= s {
			target = s
			break
		}
	}
	out := make([]byte, target)
	copy(out, key)
	return out
}

// Score ranks plaintext candidates. Lower values indicate more structure and
// a higher printable-character ratio.
func Score(data []byte) float64 {
	if len(data) == 0 {
		return math.Inf(1)
	}
	sample := data
	if len(sample) > 4096 {
		sample = sample[:4096]
	}
	printable := 0
	for _, b := range sample {
		if b == '\t' || b == '\n' || b == '\r' || (b >= 0x20 && b <= 0x7e) {
			printable++
		}
	}
	ratio := float64(printable) / float64(len(sample))
	return Shannon(sample) - ratio*3
}

// Preview returns a printable, truncated representation of data.
func Preview(data []byte) string { return preview(data) }

func preview(data []byte) string {
	const n = 96
	end := len(data)
	if end > n {
		end = n
	}
	var sb strings.Builder
	for _, b := range data[:end] {
		if b >= 0x20 && b <= 0x7e {
			sb.WriteByte(b)
		} else {
			sb.WriteByte('.')
		}
	}
	return sb.String()
}
