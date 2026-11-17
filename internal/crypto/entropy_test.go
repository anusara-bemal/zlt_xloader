package crypto

import (
	"math"
	"testing"
)

func TestShannon(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want float64
	}{
		{"empty", nil, 0},
		{"single repeated byte", []byte("aaaaaaaa"), 0},
		{"two symbols", []byte("abababab"), 1},
		{"four symbols", []byte("abcdabcd"), 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Shannon(tt.data)
			if math.Abs(got-tt.want) > 0.0001 {
				t.Fatalf("Shannon(%q) = %.4f, want %.4f", tt.data, got, tt.want)
			}
		})
	}
}

func TestClassify(t *testing.T) {
	if got := Classify(7.9); got != "encrypted/compressed" {
		t.Fatalf("Classify(7.9) = %q", got)
	}
	if got := Classify(6.5); got != "mixed" {
		t.Fatalf("Classify(6.5) = %q", got)
	}
	if got := Classify(3.2); got != "structured" {
		t.Fatalf("Classify(3.2) = %q", got)
	}
}

func TestBlockReport(t *testing.T) {
	data := make([]byte, 10000) // all zeros -> entropy 0
	blocks := BlockReport(data, 4096)
	if len(blocks) != 3 {
		t.Fatalf("got %d blocks, want 3", len(blocks))
	}
	if blocks[0].Offset != 0 || blocks[0].Length != 4096 {
		t.Fatalf("unexpected first block: %+v", blocks[0])
	}
	if blocks[2].Offset != 8192 || blocks[2].Length != 1808 {
		t.Fatalf("unexpected last block: %+v", blocks[2])
	}
	for _, b := range blocks {
		if b.Entropy != 0 {
			t.Fatalf("block entropy = %f, want 0", b.Entropy)
		}
	}
}

func TestXORRoundTrip(t *testing.T) {
	plain := []byte("hello zlt s20")
	key := []byte("secret")
	enc, err := Decrypt(plain, key, "xor")
	if err != nil {
		t.Fatal(err)
	}
	dec, err := Decrypt(enc, key, "xor")
	if err != nil {
		t.Fatal(err)
	}
	if string(dec) != string(plain) {
		t.Fatalf("round trip failed: got %q", dec)
	}
}
