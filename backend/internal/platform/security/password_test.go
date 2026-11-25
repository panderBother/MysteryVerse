package security

import "testing"

func TestPasswordHasher(t *testing.T) {
	hasher := NewPasswordHasher(Argon2Params{
		Memory:      8 * 1024,
		Iterations:  1,
		Parallelism: 1,
	})

	encoded, err := hasher.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if encoded == "correct horse battery staple" {
		t.Fatal("Hash() returned the plaintext password")
	}

	valid, err := hasher.Verify(encoded, "correct horse battery staple")
	if err != nil || !valid {
		t.Fatalf("Verify() = %t, %v", valid, err)
	}

	valid, err = hasher.Verify(encoded, "wrong password")
	if err != nil {
		t.Fatalf("Verify() wrong password error = %v", err)
	}
	if valid {
		t.Fatal("Verify() accepted a wrong password")
	}
}
