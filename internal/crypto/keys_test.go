package crypto

import (
	"bytes"
	"testing"
)

func TestCBCRoundTrip(t *testing.T) {
	plain := bytes.Repeat([]byte("abc"), 16) // 48 bytes, block aligned
	for _, tc := range []struct {
		algo string
		key  []byte
	}{
		{"aes-cbc", []byte("0123456789abcdef")},
		{"des-cbc", []byte("12345678")},
	} {
		enc, err := Encrypt(plain, tc.key, tc.algo)
		if err != nil {
			t.Fatalf("%s encrypt: %v", tc.algo, err)
		}
		dec, err := Decrypt(enc, tc.key, tc.algo)
		if err != nil {
			t.Fatalf("%s decrypt: %v", tc.algo, err)
		}
		if !bytes.Equal(dec, plain) {
			t.Fatalf("%s round trip failed: got %q", tc.algo, dec)
		}
	}
}

func TestDeriveKeys(t *testing.T) {
	keys := DeriveKeys([]string{"known"}, "12345", "")
	if len(keys) < 2 {
		t.Fatalf("keys = %v", keys)
	}
	if keys[0] != "known" || keys[1] != "12345" {
		t.Fatalf("keys = %v", keys)
	}
}

func TestScorePrefersStructured(t *testing.T) {
	structured := Score([]byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	random := Score([]byte{1, 99, 42, 177, 23, 88, 200, 7, 144, 33, 71, 129})
	if structured >= random {
		t.Fatalf("structured score %f should be lower than random %f", structured, random)
	}
}

func TestTryDecryptFindsXORKey(t *testing.T) {
	key := []byte("secretkey")
	plain := []byte("this is a plaintext config blob")
	enc, err := Encrypt(plain, key, "xor")
	if err != nil {
		t.Fatal(err)
	}
	cands := TryDecrypt(enc, []string{"secretkey", "wrongkey"}, []string{"xor"}, 5)
	if len(cands) == 0 {
		t.Fatal("no candidates")
	}
	if cands[0].Key != "secretkey" {
		t.Fatalf("best key = %q, want secretkey", cands[0].Key)
	}
}
