package events

import (
	"sync"
	"sync/atomic"
)

type EventChannelName EventName

const (
	CreateAgentsChannel = EventChannelName(CreateAgentEvent)
	UpdateAgentsChannel = EventChannelName(UpdateAgentEvent)
	DeleteAgentsChannel = EventChannelName(DeleteAgentEvent)

	CreateGroupsChannel = EventChannelName(CreateGroupEvent)
	UpdateGroupsChannel = EventChannelName(UpdateGroupEvent)
	DeleteGroupsChannel = EventChannelName(DeleteGroupEvent)

	CreatePoliciesChannel = EventChannelName(CreatePolicyEvent)
	UpdatePoliciesChannel = EventChannelName(UpdatePolicyEvent)
	DeletePoliciesChannel = EventChannelName(DeletePolicyEvent)

	CreateModulesChannel = EventChannelName(CreateModuleEvent)
	UpdateModulesChannel = EventChannelName(UpdateModuleEvent)
	DeleteModulesChannel = EventChannelName(DeleteModuleEvent)

	CreateGroupToPolicyChannel = EventChannelName(CreateGroupToPolicyEvent)
	DeleteGroupToPolicyChannel = EventChannelName(DeleteGroupToPolicyEvent)

	AllEventsChannel = EventChannelName("all")
)

func AllEventChannels() []EventChannelName {
	return []EventChannelName{
		CreateAgentsChannel, UpdateAgentsChannel, DeleteAgentsChannel,
		CreateGroupsChannel, UpdateGroupsChannel, DeleteGroupsChannel,
		CreatePoliciesChannel, UpdatePoliciesChannel, DeletePoliciesChannel,
		CreateModulesChannel, UpdateModulesChannel, DeleteModulesChannel,
		CreateGroupToPolicyChannel, DeleteGroupToPolicyChannel,
		AllEventsChannel,
	}
}

func ValidChannelName(candidate EventChannelName) bool {
	for _, en := range AllEventChannels() {
		if candidate == en {
			return true
		}
	}
	return false
}

type (
	//Subscription object. Use channel Subscription.C for receive events.
	Subscription struct {
		C         chan []Event
		id        subscribeID
		channels  []EventChannelName
		serviceID uint64
	}
	subscribeID uint64
)

type subscriptionContainer struct {
	data map[EventChannelName]map[uint64]map[subscribeID]*Subscription
	lock sync.RWMutex
}

func newSubscriptionContainer() *subscriptionContainer {
	sc := &subscriptionContainer{
		data: make(map[EventChannelName]map[uint64]map[subscribeID]*Subscription),
	}

	for _, channel := range AllEventChannels() {
		sc.data[channel] = make(map[uint64]map[subscribeID]*Subscription)
	}

	return sc
}

func (sc *subscriptionContainer) find(serviceID uint64, channel EventChannelName) map[subscribeID]*Subscription {
	sc.lock.RLock()
	defer sc.lock.RUnlock()

	if _, exists := sc.data[channel]; !exists {
		return nil
	}

	return sc.data[channel][serviceID]
}

func (sc *subscriptionContainer) add(s *Subscription) {
	sc.lock.Lock()
	defer sc.lock.Unlock()

	for _, channel := range s.channels {
		if _, exists := sc.data[channel]; !exists {
			sc.data[channel] = make(map[uint64]map[subscribeID]*Subscription)
		}

		if _, exists := sc.data[channel][s.serviceID]; !exists {
			sc.data[channel][s.serviceID] = make(map[subscribeID]*Subscription)
		}

		sc.data[channel][s.serviceID][s.id] = s
	}
}

func (sc *subscriptionContainer) delete(s *Subscription) {
	sc.lock.Lock()
	defer sc.lock.Unlock()

	for _, channel := range s.channels {
		if _, exists := sc.data[channel]; !exists {
			continue
		}

		if _, exists := sc.data[channel][s.serviceID]; !exists {
			continue
		}

		delete(sc.data[channel][s.serviceID], s.id)
	}
}

// Exchanger component for routing events to interested subscribers.
type Exchanger struct {
	lastID subscribeID

	subscriptions *subscriptionContainer
}

// NewExchanger create new Exchanger.
func NewExchanger() *Exchanger {
	return &Exchanger{
		subscriptions: newSubscriptionContainer(),
	}
}

func (e *Exchanger) nextID() subscribeID {
	return subscribeID(atomic.AddUint64((*uint64)(&e.lastID), 1))
}

// UnSubscribe drop subscription.
func (e *Exchanger) UnSubscribe(sub *Subscription) {
	e.subscriptions.delete(sub)

	close(sub.C)
}

func (e *Exchanger) fireEvents(serviceID uint64, channel EventChannelName, events ...Event) {
	if len(events) == 0 {
		return
	}

	for _, subscription := range e.subscriptions.find(serviceID, channel) {
		subscription.C <- events
	}
}

// Subscribe create subscription on events, filtered by serviceID and eventNames.
func (e *Exchanger) Subscribe(serviceID uint64, channels ...EventChannelName) *Subscription {
	subscription := &Subscription{
		C:         make(chan []Event),
		id:        e.nextID(),
		channels:  channels,
		serviceID: serviceID,
	}

	e.subscriptions.add(subscription)

	return subscription
}
