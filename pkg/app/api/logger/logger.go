package logger

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// FromContext is function to get logrus Entry with context
func FromContext(c *gin.Context) *logrus.Entry {
	return logrus.WithContext(c.Request.Context())
}
