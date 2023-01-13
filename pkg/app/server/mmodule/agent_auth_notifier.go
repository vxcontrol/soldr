package mmodule

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"

	utilsErrors "soldr/pkg/utils/errors"
)

type Authenticator struct {
	authEvents    map[string][]chan error
	authEventsMux *sync.Mutex
}

func newAuthenticator() *Authenticator {
	return &Authenticator{
		authEvents:    make(map[string][]chan error),
		authEventsMux: &sync.Mutex{},
	}
}

func (a *Authenticator) WaitForAuth(ctx context.Context, agentID string) error {
	authNotifier := a.getAuthNotifier(agentID)
	select {
	case err := <-authNotifier:
		return err
	case <-ctx.Done():
		a.unregisterAgentNotifications(agentID)
		return ctx.Err()
	}
}

func (a *Authenticator) getAuthNotifier(agentID string) <-chan error {
	a.authEventsMux.Lock()
	defer a.authEventsMux.Unlock()

	ch := make(chan error, 1)
	if _, ok := a.authEvents[agentID]; !ok {
		a.authEvents[agentID] = make([]chan error, 0, 1)
	}
	a.authEvents[agentID] = append(a.authEvents[agentID], ch)
	return ch
}

func (a *Authenticator) registerAuth(connectedAgents map[string]struct{}, dbIDs map[string]*AgentInfoDB) {
	a.authEventsMux.Lock()
	defer a.authEventsMux.Unlock()

	shouldNotifyFn := func(id string) (bool, error) {
		if _, ok := connectedAgents[id]; ok {
			logrus.Debugf("agent %s is already connected, authentication passed", id)
			return true, nil
		}
		dbInfo, ok := dbIDs[id]
		if !ok {
			logrus.Debugf("no agent %s entry found in the DB, authentication failed", id)
			return true, utilsErrors.ErrFailedResponseCorrupted
		}
		switch dbInfo.AuthStatus {
		case "authorized":
			logrus.Debugf("agent %s status is \"%s\", authentication passed", id, dbInfo.AuthStatus)
			return true, nil
		case "blocked":
			logrus.Debugf("agent %s status is \"%s\", authentication failed", id, dbInfo.AuthStatus)
			return true, utilsErrors.ErrFailedResponseBlocked
		default:
			return false, nil
		}
	}

	for id, chans := range a.authEvents {
		shouldNotify, notification := shouldNotifyFn(id)
		if !shouldNotify {
			continue
		}
		for _, c := range chans {
			c <- notification
			close(c)
		}
		delete(a.authEvents, id)
	}
}

func (a *Authenticator) unregisterAgentNotifications(id string) {
	a.authEventsMux.Lock()
	defer a.authEventsMux.Unlock()

	agentChans, ok := a.authEvents[id]
	if !ok {
		return
	}
	for _, c := range agentChans {
		close(c)
	}
	delete(a.authEvents, id)
	logrus.Debugf("notification channels for authentication have been closed for the agent %s", id)
}
