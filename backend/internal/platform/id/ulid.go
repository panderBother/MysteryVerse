package id

import (
	"fmt"

	"github.com/oklog/ulid/v2"
)

type Generator interface {
	New() string
}

type ULIDGenerator struct{}

func (ULIDGenerator) New() string {
	return ulid.Make().String()
}

func Validate(value string) error {
	if _, err := ulid.ParseStrict(value); err != nil {
		return fmt.Errorf("parse ULID: %w", err)
	}
	return nil
}
