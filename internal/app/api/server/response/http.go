package response

import "fmt"

type HttpError struct {
	message  string
	code     string
	httpCode int
}

func (h *HttpError) Code() string {
	return h.code
}

func (h *HttpError) HttpCode() int {
	return h.httpCode
}

func (h *HttpError) Msg() string {
	return h.message
}

func NewHttpError(httpCode int, code, message string) *HttpError {
	return &HttpError{httpCode: httpCode, message: message, code: code}
}

func (h *HttpError) Error() string {
	return fmt.Sprintf("%s: %s", h.code, h.message)
}
