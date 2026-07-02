package security

import "testing"

func TestEncryptorRoundTrip(t *testing.T) {
	enc := NewEncryptor("secret")
	ciphertext, err := enc.Encrypt("gaia-password")
	if err != nil {
		t.Fatal(err)
	}
	plain, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "gaia-password" {
		t.Fatalf("expected round trip, got %q", plain)
	}
}

func TestHashTokenStable(t *testing.T) {
	if HashToken("abc") != HashToken("abc") {
		t.Fatal("hash must be stable")
	}
	if HashToken("abc") == HashToken("abcd") {
		t.Fatal("hash must differ for different input")
	}
}
