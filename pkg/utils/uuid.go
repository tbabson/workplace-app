package utils

import (
	"errors"

	"github.com/google/uuid"
)

func NewUUID() string {
	return uuid.New().String()
}

func ValidateUUID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return errors.New("invalid id format: must be a valid UUID")
	}
	return nil
}

func IsValidUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}
