package watcher

import (
	"errors"
	"sync"
)

var ErrNotificationChanClosed = errors.New("watcher notification channel is already closed")

type NotificationChan struct {
	Chan        chan interface{}
	isClosed    bool
	isClosedMux *sync.Mutex
}

func NewNotificationChan() *NotificationChan {
	return &NotificationChan{
		Chan:        make(chan interface{}, 1),
		isClosed:    false,
		isClosedMux: &sync.Mutex{},
	}
}

func (ch *NotificationChan) SendAndClose(ev interface{}) error {
	ch.isClosedMux.Lock()
	defer ch.isClosedMux.Unlock()
	if ch.isClosed {
		return ErrNotificationChanClosed
	}

	ch.Chan <- ev
	ch.close()
	return nil
}

func (ch *NotificationChan) Close() {
	ch.isClosedMux.Lock()
	defer ch.isClosedMux.Unlock()
	if ch.isClosed {
		return
	}
	ch.close()
}

func (ch *NotificationChan) close() {
	close(ch.Chan)
	ch.isClosed = true
}
