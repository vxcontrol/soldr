package log

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/sirupsen/logrus"
)

// Logger generalizes the Logger types
type Logger interface {
	logrus.FieldLogger
}

type Config struct {
	Level  logrus.Level
	Format Format
}

func New(cfg Config, output io.Writer) Logger {
	logger := logrus.New()
	logger.Out = output
	switch cfg.Format {
	case FormatJSON:
		logger.Formatter = &logrus.JSONFormatter{}
	case FormatText:
		logger.Formatter = &logrus.TextFormatter{ForceColors: true, FullTimestamp: true}
	default:
		logger.Warnf("log: invalid formatter: %v, continue with default", cfg.Format)
	}
	logger.Level = cfg.Level
	return logger
}

const (
	FormatInvalid Format = iota
	FormatText
	FormatJSON
)

const (
	formatTextStr = "text"
	formatJSONStr = "json"
)

type Format int

func ParseFormat(s string) (Format, error) {
	s = strings.ToLower(s)
	switch s {
	case formatTextStr:
		return FormatText, nil
	case formatJSONStr:
		return FormatJSON, nil
	}
	return FormatInvalid, errors.New("invalid format string")
}

func (f Format) String() string {
	switch f {
	case FormatText:
		return formatTextStr
	case FormatJSON:
		return formatJSONStr
	}
	return fmt.Sprintf("invalid (%d)", f)
}

func ParseLevel(s string) (logrus.Level, error) {
	return logrus.ParseLevel(s)
}
