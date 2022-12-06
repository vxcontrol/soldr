package app

import (
	"github.com/oklog/run"
	"github.com/sirupsen/logrus"
)

type AppGroup struct {
	runGroup run.Group
}

func NewAppGroup() *AppGroup {
	return &AppGroup{}
}

func (a *AppGroup) Run() {
	err := a.runGroup.Run()
	if err != nil {
		logrus.Errorf("failed to run appGroup: %s", err)
	}
}

// Add (function) to the application group. Each actor must be pre-emptable by an
// interrupt function. That is, if interrupt is invoked, execute should return.
// Also, it must be safe to call interrupt even after execute has returned.
//
// The first actor (function) to return interrupts all running actors.
// The error is passed to the interrupt functions, and is returned by Run.
func (a *AppGroup) Add(execute func() error, interrupt func(error)) {
	a.runGroup.Add(execute, interrupt)
}
