package errors

import "errors"

var (
	ErrConnectionInitializationRequired = errors.New("failed to connect to the server: a connection initialization required")
	ErrUnexpectedUnpackType             = errors.New("unexpected agent message type")
	ErrRecordNotFound                   = errors.New("record not found")
)
