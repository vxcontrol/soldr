package logger

import (
	"strings"

	"github.com/sirupsen/logrus"
)

// Logger generalizes the Logger types
type Logger interface {
	logrus.FieldLogger
}

func ParseFormat(s string) logrus.Formatter {
	s = strings.ToLower(s)
	if s == "text" {
		return &logrus.TextFormatter{ForceColors: true, FullTimestamp: true}
	}
	return &logrus.JSONFormatter{}
}

func ParseLevel(s string) logrus.Level {
	l, err := logrus.ParseLevel(s)
	if err != nil {
		return logrus.InfoLevel
	}
	return l
}
