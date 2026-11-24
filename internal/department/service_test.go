package department

import (
	"errors"
	"testing"
)

func TestDuplicateNameMapsToValidation(t *testing.T) {
	err := duplicateName(errors.New(`ERROR: duplicate key value violates unique constraint "departments_name_unique" (SQLSTATE 23505)`))
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("duplicate name not reported as a validation error: %v", err)
	}
	if other := errors.New("connection reset"); duplicateName(other) != other {
		t.Fatal("unrelated errors must pass through unchanged")
	}
}
