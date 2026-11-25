package secret

import "testing"

func TestCipherRoundTrip(t *testing.T) {
	cipher, err := NewCipher("test-secret-key-with-enough-material", "test-key-v1")
	if err != nil {
		t.Fatalf("NewCipher() error = %v", err)
	}

	sealed, err := cipher.Seal([]byte(`{"private":"value"}`), "game:1")
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	plaintext, err := cipher.Open(sealed, "game:1")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if string(plaintext) != `{"private":"value"}` {
		t.Fatalf("plaintext = %q", plaintext)
	}
	if _, err := cipher.Open(sealed, "game:2"); err == nil {
		t.Fatal("Open() accepted a different AAD")
	}
}
