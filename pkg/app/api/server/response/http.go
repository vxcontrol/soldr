package response

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"soldr/pkg/app/api/logger"
	obs "soldr/pkg/observability"
	"soldr/pkg/version"
)

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

func Error(c *gin.Context, err *HttpError, original error) {
	body := gin.H{"status": "error", "code": err.Code()}

	if version.IsDevelop == "true" {
		body["msg"] = err.Msg()
		if original != nil {
			body["error"] = original.Error()
		}
	}

	traceID := obs.Observer.SpanContextFromContext(c.Request.Context()).TraceID()
	if traceID.IsValid() {
		body["trace_id"] = traceID.String()
	}

	logger.FromContext(c).
		Errorf("api error with status code: '%d'; message: '%s'; error: '%s'",
			err.HttpCode(), err.Msg(), original)
	c.AbortWithStatusJSON(err.HttpCode(), body)
}

func Success(c *gin.Context, code int, data interface{}) {
	logger.FromContext(c).Infof("api success with status code: '%d'", code)
	c.JSON(code, gin.H{"status": "success", "data": data})
}

type GroupedData struct {
	Grouped []string `json:"grouped"`
	Total   uint64   `json:"total"`
}

//lint:ignore U1000 successResp
type successResp struct {
	Status string      `json:"status" example:"success"`
	Data   interface{} `json:"data" swaggertype:"object"`
} // @name SuccessResponse

//lint:ignore U1000 errorResp
type errorResp struct {
	Status  string `json:"status" example:"error"`
	Code    string `json:"code" example:"Internal"`
	Msg     string `json:"msg,omitempty" example:"internal server error"`
	Error   string `json:"error,omitempty" example:"original server error message"`
	TraceID string `json:"trace_id,omitempty" example:"1234567890abcdef1234567890abcdef"`
} // @name ErrorResponse
