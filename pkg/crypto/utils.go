package crypto

import (
	"fmt"
)

type ErrDecryptFailed struct {
	err error
}

func NewErrDecryptFailed(err error) error {
	return &ErrDecryptFailed{fmt.Errorf("failed to decrypt module secure config: %w", err)}
}

func (e *ErrDecryptFailed) Error() string {
	return e.err.Error()
}
