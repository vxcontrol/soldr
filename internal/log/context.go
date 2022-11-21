package log

import (
	"context"
	"io"

	"github.com/sirupsen/logrus"
)

// Helper methods used by the application to get the request-scoped
// logger entry and set additional fields between handlers.
//
// This is a useful pattern to use to set state on the entry as it
// passes through the handler chain, which at any point can be logged
// with a call to .Print(), .Info(), etc.

type loggerKey struct{}

func FromContext(ctx context.Context) Logger {
	if l, ok := ctx.Value(loggerKey{}).(Logger); ok {
		return l
	}
	// Discard log in case if logger is not attached.
	l := logrus.New()
	l.Level = logrus.PanicLevel
	l.Out = io.Discard
	return l
}

func AttachToContext(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}
