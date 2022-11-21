package vxproto

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	defQueueSize         int           = 100
	defRecvPacketTimeout time.Duration = 2 * time.Second
)

type recvBlocker struct {
	retChan chan *Packet
	pType   PacketType
	src     string
	closed  bool
}

type recvRouter struct {
	apiMutex *sync.Mutex
	receiver chan *Packet
	control  chan struct{}
	queue    []*recvBlocker
}

func newRouter() *recvRouter {
	return &recvRouter{
		apiMutex: &sync.Mutex{},
		receiver: make(chan *Packet, defQueueSize),
		control:  make(chan struct{}),
	}
}

func (r *recvRouter) routePacket(ctx context.Context, packet *Packet) (err error) {
	if packet.ackChan != nil {
		defer func() {
			select {
			case <-r.control:
				err = fmt.Errorf("router already closed")
			case <-packet.ackChan:
			case <-time.NewTimer(defRecvPacketTimeout).C:
				err = fmt.Errorf("timeout exceeded")
			}
		}()
	}
	r.apiMutex.Lock()
	defer r.apiMutex.Unlock()
	for idx, b := range r.queue {
		if (b.src == packet.Src || b.src == "") && b.pType == packet.PType && !b.closed {
			b.retChan <- packet
			// It's possible because there used return in the bottom
			r.queue = append(r.queue[:idx], r.queue[idx+1:]...)
			return
		}
	}
	select {
	case <-r.control:
		err = fmt.Errorf("router already closed")
	default:
		r.receiver <- packet
	}
	return
}

func (r *recvRouter) close() {
	r.unlockAll()
	close(r.control)
	queueLen := len(r.receiver)
	if queueLen == 0 {
		return
	}
	for i := 0; i < queueLen; i++ {
		<-r.receiver
	}
}

func (r *recvRouter) unlock(src string) {
	r.apiMutex.Lock()
	defer r.apiMutex.Unlock()
	for _, b := range r.queue {
		if b.src == src {
			b.closed = true
			close(b.retChan)
		}
	}
}

func (r *recvRouter) unlockAll() {
	r.apiMutex.Lock()
	defer r.apiMutex.Unlock()
	for _, b := range r.queue {
		b.closed = true
		close(b.retChan)
	}
}

func (r *recvRouter) addBlocker(blocker *recvBlocker) *Packet {
	r.apiMutex.Lock()
	defer r.apiMutex.Unlock()
	if packet, err := r.lookupPacket(blocker.src, blocker.pType); err == nil {
		return packet
	}
	r.queue = append(r.queue, blocker)
	return nil
}

func (r *recvRouter) delBlocker(blocker *recvBlocker) {
	r.apiMutex.Lock()
	defer r.apiMutex.Unlock()
	for idx, b := range r.queue {
		if b == blocker {
			// It's possible because there used return in the bottom
			r.queue = append(r.queue[:idx], r.queue[idx+1:]...)
			return
		}
	}
}

func (r *recvRouter) lookupPacket(src string, pType PacketType) (*Packet, error) {
	queueLen := len(r.receiver)
	if queueLen == 0 {
		return nil, fmt.Errorf("packets queue is empty")
	}

	queue := make([]*Packet, 0, queueLen)
	for idx := 0; idx < queueLen; idx++ {
		queue = append(queue, <-r.receiver)
	}
	defer func() {
		for _, p := range queue {
			r.receiver <- p
		}
	}()
	for idx, packet := range queue {
		if (src == packet.Src || src == "") && pType == packet.PType {
			// It's possible because there used return in the bottom
			queue = append(queue[:idx], queue[idx+1:]...)
			return packet, nil
		}
	}

	return nil, fmt.Errorf("packet not found")
}

func (r *recvRouter) recvPacket(ctx context.Context, src string, pType PacketType, timeout int64) (*Packet, error) {
	var packet *Packet
	blocker := &recvBlocker{
		retChan: make(chan *Packet),
		src:     src,
		pType:   pType,
	}
	if packet = r.addBlocker(blocker); packet != nil {
		return packet, nil
	}
	defer r.delBlocker(blocker)

	if timeout > 0 {
		select {
		case <-time.NewTimer(time.Millisecond * time.Duration(timeout)).C:
			return nil, fmt.Errorf("timeout exceeded")
		case packet = <-blocker.retChan:
		}
	} else if timeout == 0 {
		select {
		case packet = <-blocker.retChan:
		default:
			return nil, fmt.Errorf("nothing in queue")
		}
	} else {
		packet = <-blocker.retChan
	}
	if packet == nil {
		return nil, fmt.Errorf("connection reset")
	}

	return packet, nil
}
