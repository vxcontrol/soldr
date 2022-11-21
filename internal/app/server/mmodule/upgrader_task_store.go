package mmodule

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jinzhu/gorm"

	"soldr/internal/app/api/models"
	store2 "soldr/internal/app/server/mmodule/upgrader/store"
	"soldr/internal/app/server/mmodule/upgrader/watcher"
)

type fetchParams struct {
	batchSize int
	limit     int

	offset int
}

func newFetchParams(batchSize int, limit int) (*fetchParams, error) {
	if batchSize <= 0 {
		return nil, getFetchParamsValueErr("batch size", batchSize)
	}
	if limit <= 0 {
		return nil, getFetchParamsValueErr("limit", limit)
	}
	return &fetchParams{
		batchSize: batchSize,
		limit:     limit,
	}, nil
}

func getFetchParamsValueErr(paramName string, val int) error {
	return fmt.Errorf("%s configuration parameter of the tasks batch fetcher must be positive, got %d", paramName, val)
}

type TasksStore struct {
	store                map[string]*watcher.NotificationChan
	storeMux             *sync.RWMutex
	watchersWG           *sync.WaitGroup
	agents               *agentList
	db                   *gorm.DB
	validator            *validator.Validate
	currentTasksCount    int
	currentTasksCountMux *sync.Mutex
	fetchParams          *fetchParams

	isClosed    bool
	isClosedMux *sync.RWMutex
}

const (
	tasksStoreBatchSize  = 10
	tasksStoreTasksLimit = 100
)

func NewTasksStore(conn *gorm.DB, validator *validator.Validate, agents *agentList) (*TasksStore, error) {
	fetchParams, err := newFetchParams(tasksStoreBatchSize, tasksStoreTasksLimit)
	if err != nil {
		return nil, err
	}
	return &TasksStore{
		store:                make(map[string]*watcher.NotificationChan),
		watchersWG:           &sync.WaitGroup{},
		storeMux:             &sync.RWMutex{},
		agents:               agents,
		db:                   conn,
		validator:            validator,
		currentTasksCount:    0,
		currentTasksCountMux: &sync.Mutex{},
		fetchParams:          fetchParams,
		isClosed:             false,
		isClosedMux:          &sync.RWMutex{},
	}, nil
}

func (s *TasksStore) GetNewTasks(ctx context.Context) ([]*store2.Task, error) {
	s.storeMux.Lock()
	defer s.storeMux.Unlock()
	s.currentTasksCountMux.Lock()
	defer s.currentTasksCountMux.Unlock()

	tasks, err := s.fetchNewTasks(ctx)
	if err != nil {
		return nil, err
	}
	processedTasks, err := s.putTasksToStore(ctx, tasks)
	if err != nil {
		return nil, err
	}
	s.updateCurrentTasksCounter(len(processedTasks))
	return processedTasks, nil
}

func (s *TasksStore) DecreaseNumberOfUpgradeTasks() {
	s.currentTasksCountMux.Lock()
	defer s.currentTasksCountMux.Unlock()

	s.currentTasksCount--
}

func (s *TasksStore) calculateNewBatchSize() int {
	tasksToGetCount := s.fetchParams.limit - s.currentTasksCount
	if tasksToGetCount <= 0 {
		return 0
	}
	if tasksToGetCount > s.fetchParams.batchSize {
		tasksToGetCount = s.fetchParams.batchSize
	}
	return tasksToGetCount
}

func (s *TasksStore) fetchNewTasks(ctx context.Context) ([]*store2.Task, error) {
	batchSize := s.calculateNewBatchSize()
	tasks, err := s.fetchTasks(
		ctx,
		store2.TaskStatusNew,
		&fetchTasksLimitOffset{
			Limit:  s.fetchParams.limit,
			Offset: s.fetchParams.offset,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get new tasks: %w", err)
	}
	s.fetchParams.offset = s.getNewOffset(len(tasks), batchSize)
	return tasks, nil
}

func (s *TasksStore) fetchRunningTasks(ctx context.Context) ([]*store2.Task, error) {
	tasks, err := s.fetchTasks(ctx, store2.TaskStatusRunning, nil)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

type fetchTasksLimitOffset struct {
	Limit  int
	Offset int
}

func (s *TasksStore) fetchTasks(
	_ context.Context,
	status store2.TaskStatus,
	limitOffset *fetchTasksLimitOffset,
) ([]*store2.Task, error) {
	var (
		tx    *gorm.DB
		tasks []*store2.Task
	)
	if s.db == nil {
		return tasks, nil
	}
	tx = s.db.
		Table("upgrade_tasks").
		Select("upgrade_tasks.id, agent_id, agents.hash as agent_hash, upgrade_tasks.version").
		Joins("JOIN agents ON agent_id = agents.id AND agents.deleted_at IS NULL").
		Where(fmt.Sprintf("upgrade_tasks.status = '%s'", string(status)))
	if limitOffset != nil {
		tx = tx.Limit(limitOffset.Limit).Offset(limitOffset.Offset)
	}
	if err := tx.Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to get tasks with the status %s: %w", string(status), err)
	}
	return tasks, nil
}

func (s *TasksStore) getNewOffset(resultLen int, batchSize int) int {
	if resultLen < batchSize {
		return 0
	}
	return s.fetchParams.offset + resultLen
}

func (s *TasksStore) putTasksToStore(ctx context.Context, tasks []*store2.Task) ([]*store2.Task, error) {
	filteredTasks := s.filterTasks(ctx, tasks)
	filteredTasksIDs := make([]string, len(filteredTasks))
	for i, ft := range filteredTasks {
		filteredTasksIDs[i] = ft.ID
	}
	if err := s.setTasksStatus(ctx, filteredTasksIDs, store2.TaskStatusRunning); err != nil {
		agentIDs := make([]string, len(filteredTasks))
		for i, ft := range filteredTasks {
			agentIDs[i] = ft.AgentID
		}
		s.removeNewTasks(ctx, agentIDs)
		return nil, err
	}
	for _, t := range filteredTasks {
		s.startTask(ctx, t)
	}
	return filteredTasks, nil
}

func (s *TasksStore) filterTasks(_ context.Context, tasks []*store2.Task) []*store2.Task {
	filteredTasks := make([]*store2.Task, 0, len(tasks))
	for _, t := range tasks {
		if len(s.agents.dumpID(t.AgentHash)) == 0 {
			// process upgrade tasks only for connected agents
			continue
		}
		if _, ok := s.store[t.AgentID]; ok {
			// upgrade agents that are not being upgraded
			continue
		}
		filteredTasks = append(filteredTasks, t)
	}
	return filteredTasks
}

func (s *TasksStore) setTasksStatus(_ context.Context, tasksIDs []string, status store2.TaskStatus) error {
	if s.db == nil {
		return nil
	}
	if len(tasksIDs) == 0 {
		return nil
	}
	updateMap, err := getUpdateMap(
		map[string]*struct {
			Value interface{}
			Rule  string
		}{
			"status": {
				Value: string(status),
				Rule:  "oneof=new running ready failed,required",
			},
			"last_upgrade": {
				Value: gorm.Expr("NOW()"),
			},
		},
		s.validator,
	)
	if err != nil {
		return fmt.Errorf("failed to validate the upgrade task status values: %w", err)
	}
	err = s.db.
		Model(&models.AgentUpgradeTask{}).
		Where("id IN (?)", tasksIDs).
		UpdateColumns(updateMap).
		Error
	if err != nil {
		return fmt.Errorf("failed to set the tasks %v status to %s: %w", tasksIDs, status, err)
	}
	return nil
}

func (s *TasksStore) setTaskFailed(_ context.Context, taskID string, reason store2.TaskFailureReason) error {
	if s.db == nil {
		return nil
	}
	updateMap, err := getUpdateMap(
		map[string]*struct {
			Value interface{}
			Rule  string
		}{
			"status": {
				Value: string(store2.TaskStatusFailed),
			},
			"reason": {
				Value: string(reason),
				Rule:  "max=150,lockey=omitempty,omitempty",
			},
			"last_upgrade": {
				Value: gorm.Expr("NOW()"),
			},
		},
		s.validator,
	)
	if err != nil {
		return fmt.Errorf("failed to validate the upgrade task status values: %w", err)
	}
	err = s.db.
		Model(&models.AgentUpgradeTask{}).
		Where("id = ?", taskID).
		UpdateColumns(updateMap).
		Error
	if err != nil {
		return fmt.Errorf(
			"failed to set the tasks %v status to %s with the failure reason %s: %w",
			taskID, store2.TaskStatusFailed, reason, err)
	}
	return nil
}

func (s *TasksStore) removeNewTasks(_ context.Context, agentIDs []string) {
	for _, aID := range agentIDs {
		if ch, ok := s.store[aID]; ok {
			ch.Close()
		}
	}
}

func (s *TasksStore) startTask(ctx context.Context, t *store2.Task) {
	ch := watcher.NewNotificationChan()
	s.store[t.AgentHash] = ch
	s.watchersWG.Add(1)
	go s.watchTask(ctx, t, ch.Chan)
}

const (
	watchTaskTimeout      = time.Second * 120
	phaseNameTaskWatching = "task-watching"
)

func (s *TasksStore) watchTask(ctx context.Context, task *store2.Task, ch <-chan interface{}) {
	defer s.watchersWG.Done()
	timeout, cancelTimeout := context.WithTimeout(context.Background(), watchTaskTimeout)
	defer cancelTimeout()

	var ev interface{}
	isTimeout, ok := false, true

	select {
	case ev, ok = <-ch:
	case <-timeout.Done():
		isTimeout = true
	}

	s.storeMux.Lock()
	defer s.storeMux.Unlock()

	s.currentTasksCountMux.Lock()
	defer s.currentTasksCountMux.Unlock()

	defer func() {
		s.currentTasksCount--
		if ok := s.removeTask(ctx, task.AgentHash); !ok {
			generateLoggerEventForUpgradeTaskWithTaskID(ctx, task.AgentID, phaseNameTaskWatching, task.ID).
				Warnf("the task could not be removed, as it is not found in the store")
		}
	}()

	if !ok {
		s.handleUnexpectedWatcherChanClosingEvent(ctx, task.AgentID, task.ID)
		return
	}
	if isTimeout {
		s.handleWatcherTimeoutEvent(ctx, task.AgentID, task.ID)
		return
	}
	if err := s.processUpgradeEvent(ctx, ev, task); err != nil {
		generateLoggerEventForUpgradeTaskWithTaskID(ctx, task.AgentID, phaseNameTaskWatching, task.ID).
			WithError(err).
			Warn("an error occurred during an upgrade task event processing")
	}
}

func (s *TasksStore) handleWatcherTimeoutEvent(ctx context.Context, agentID string, taskID string) {
	logEntry := generateLoggerEventForUpgradeTaskWithTaskID(ctx, agentID, phaseNameTaskWatching, taskID)
	if err := s.setTaskFailed(ctx, taskID, store2.TaskFailureReasonTimeout); err != nil {
		logEntry = logEntry.WithError(err)
	}
	logEntry.Warn("upgrade task timeout has exceeded")
}

func (s *TasksStore) handleUnexpectedWatcherChanClosingEvent(ctx context.Context, agentID string, taskID string) {
	logEntry := generateLoggerEventForUpgradeTaskWithTaskID(ctx, agentID, phaseNameTaskWatching, taskID)
	if err := s.setTaskFailed(ctx, taskID, store2.TaskFailureReasonChanClosedUnexpectedly); err != nil {
		logEntry = logEntry.WithError(err)
	}
	logEntry.Warn("the event channel was unexpectedly closed")
}

func (s *TasksStore) removeTask(_ context.Context, agentID string) bool {
	if _, ok := s.store[agentID]; !ok {
		return false
	}
	delete(s.store, agentID)
	return true
}

func (s *TasksStore) processUpgradeEvent(ctx context.Context, ev interface{}, task *store2.Task) error {
	switch typedEv := ev.(type) {
	case *AgentConnectionEvent:
		return s.processAgentConnectionEvent(ctx, typedEv, task)
	case *AgentUpgradeFailureEvent:
		return s.processAgentUpgradeFailureEvent(ctx, task.ID)
	case *AgentAlreadyUpgradedEvent:
		return s.processAgentAlreadyUpgradedEvent(ctx, task.ID)
	default:
		return fmt.Errorf("event %v: %w", ev, errUnknownUpgradeTaskEventType)
	}
}

func (s *TasksStore) processAgentConnectionEvent(
	ctx context.Context, ev *AgentConnectionEvent, task *store2.Task,
) error {
	if ev.AgentVersion != task.Version {
		generateLoggerEventForUpgradeTaskWithTaskID(ctx, task.AgentID, phaseNameTaskWatching, task.ID).
			Warnf("new agent version %s does not match the expected agent version %s", ev.AgentVersion, task.Version)
		if err := s.setTaskFailed(ctx, task.ID, store2.TaskFailureReasonUnexpectedVersion); err != nil {
			return err
		}
		return nil
	}
	if err := s.setTasksStatus(ctx, []string{task.ID}, store2.TaskStatusReady); err != nil {
		return err
	}
	generateLoggerEventForUpgradeTaskWithTaskID(ctx, task.AgentID, phaseNameTaskWatching, task.ID).
		Info("agent has been successfully upgraded")
	return nil
}

func (s *TasksStore) processAgentUpgradeFailureEvent(ctx context.Context, taskID string) error {
	if err := s.setTaskFailed(ctx, taskID, store2.TaskFailureReasonUpgraderFailed); err != nil {
		return err
	}
	return nil
}

func (s *TasksStore) processAgentAlreadyUpgradedEvent(ctx context.Context, taskID string) error {
	if err := s.setTasksStatus(ctx, []string{taskID}, store2.TaskStatusReady); err != nil {
		return err
	}
	return nil
}

func (s *TasksStore) updateCurrentTasksCounter(tasksCount int) {
	s.currentTasksCount += tasksCount
}

func (s *TasksStore) SignalTaskFailure(ctx context.Context, agentID string) {
	s.isClosedMux.RLock()
	defer s.isClosedMux.RUnlock()
	if s.isClosed {
		return
	}

	if err := s.sendTaskSignal(ctx, agentID, &AgentUpgradeFailureEvent{}); err != nil {
		logSignalError(ctx, err, agentID, "task-failure-signaling")
		return
	}
}

func (s *TasksStore) SignalAgentConnection(ctx context.Context, agentID, agentVersion string) {
	s.isClosedMux.RLock()
	defer s.isClosedMux.RUnlock()
	if s.isClosed {
		return
	}

	err := s.sendTaskSignal(ctx, agentID, &AgentConnectionEvent{AgentVersion: agentVersion})
	if err != nil && !errors.Is(err, errTaskNotFound) {
		logSignalError(ctx, err, agentID, "agent-connection-signaling")
	}
}

func (s *TasksStore) Close(ctx context.Context) {
	s.isClosedMux.Lock()
	defer s.isClosedMux.Unlock()
	if s.isClosed {
		return
	}

	s.storeMux.RLock()
	defer s.storeMux.RUnlock()

	for _, ch := range s.store {
		ch.Close()
	}

	s.watchersWG.Wait()
}

func (s *TasksStore) sendTaskSignal(_ context.Context, agentID string, ev interface{}) error {
	s.storeMux.RLock()
	defer s.storeMux.RUnlock()
	ch, ok := s.store[agentID]
	if !ok {
		return errTaskNotFound
	}

	if err := ch.SendAndClose(ev); err != nil {
		return err
	}
	return nil
}
