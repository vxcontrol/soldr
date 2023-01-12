package loader

import (
	"fmt"
	"sync"
)

// ILoader is interface control modules
type ILoader interface {
	Add(id string, ms *ModuleState) bool
	Del(id, stopReason string) bool
	Get(id string) *ModuleState
	List() []string
	Start(id string) error
	StartAll() error
	Stop(id, stopReason string) error
	StopAll(stopReason string) error
}

// sLoader is container for modules loader
type sLoader struct {
	states map[string]*ModuleState
	mutex  *sync.Mutex
}

// New is function for construct loader object
func New() ILoader {
	return &sLoader{
		states: make(map[string]*ModuleState),
		mutex:  &sync.Mutex{},
	}
}

// get is internal function that get module state
func (l *sLoader) get(id string) *ModuleState {
	if ms, ok := l.states[id]; ok {
		return ms
	}

	return nil
}

// Add is function that add module state to loader
func (l *sLoader) Add(id string, ms *ModuleState) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.get(id) != nil {
		return false
	}
	l.states[id] = ms

	return true
}

// Del deletes module state from a loader
func (l *sLoader) Del(id, stopReason string) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if ms := l.get(id); ms == nil {
		return false
	} else {
		if ms.Close(stopReason) != nil {
			return false
		}
		delete(l.states, id)
	}

	return true
}

// Get is function that get module state from loader
func (l *sLoader) Get(id string) *ModuleState {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	return l.get(id)
}

// List is function that return list of modules id from loader
func (l *sLoader) List() []string {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	var list []string
	for id := range l.states {
		list = append(list, id)
	}

	return list
}

// Start is function that start module state which was added to loader
func (l *sLoader) Start(id string) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if ms := l.get(id); ms != nil {
		return ms.Start()
	}

	return genModuleStateNotFoundErr(id)
}

// StartAll is function that start all modules state from loader
func (l *sLoader) StartAll() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	for _, ms := range l.states {
		if err := ms.Start(); err != nil {
			return err
		}
	}

	return nil
}

// Stop stops module state in a loader
func (l *sLoader) Stop(id, stopReason string) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if ms := l.get(id); ms != nil {
		return ms.Stop(stopReason)
	}

	return genModuleStateNotFoundErr(id)
}

// StopAll stops all modules states in a loader
func (l *sLoader) StopAll(stopReason string) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	for _, ms := range l.states {
		if err := ms.Stop(stopReason); err != nil {
			return err
		}
	}

	return nil
}

func genModuleStateNotFoundErr(moduleID string) error {
	return fmt.Errorf("state for the module '%s' not found", moduleID)
}
