package errs

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrCorruptedData = errors.New("corrupted data")
	ErrNoOp          = errors.New("no operation")
)

func Wrap(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}
