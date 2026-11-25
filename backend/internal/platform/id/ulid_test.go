package id

import "testing"

func TestULIDGenerator(t *testing.T) {
	generator := ULIDGenerator{}
	first := generator.New()
	second := generator.New()

	if len(first) != 26 || len(second) != 26 {
		t.Fatalf("ULID lengths = %d and %d, want 26", len(first), len(second))
	}
	if first == second {
		t.Fatal("ULIDGenerator returned duplicate values")
	}
	if err := Validate(first); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if err := Validate("not-a-ulid"); err == nil {
		t.Fatal("Validate() accepted an invalid ULID")
	}
}
