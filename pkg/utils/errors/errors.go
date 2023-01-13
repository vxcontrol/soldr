package errors

import "errors"

var (
	ErrFailedResponseTunnelError      = errors.New("tunnel error")
	ErrFailedResponseUnauthorized     = errors.New("unauthorized")
	ErrFailedResponseCorrupted        = errors.New("corrupted")
	ErrFailedResponseAlreadyConnected = errors.New("already connected")
	ErrFailedResponseBlocked          = errors.New("blocked")
)
