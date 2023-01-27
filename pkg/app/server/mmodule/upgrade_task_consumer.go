package mmodule

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	"soldr/pkg/app/agent"
	"soldr/pkg/app/server/mmodule/upgrader/cache"
	"soldr/pkg/app/server/mmodule/upgrader/store"
	"soldr/pkg/app/server/mmodule/upgrader/watcher"
	"soldr/pkg/vxproto"
)

type upgradeTaskConsumer struct {
	mm         *MainModule
	cache      *cache.Cache
	tasksStore *TasksStore

	agentsTasks    map[string]string
	agentsTasksMux *sync.Mutex
}

func newUpgradeTaskConsumer(ctx context.Context, mm *MainModule) (*upgradeTaskConsumer, error) {
	tasksStore, err := NewTasksStore(mm.gdbc, mm.validator, mm.agents)
	if err != nil {
		return nil, err
	}
	upgraderCache, err := cache.NewCache(mm.store)
	if err != nil {
		return nil, err
	}
	tc := &upgradeTaskConsumer{
		mm:         mm,
		cache:      upgraderCache,
		tasksStore: tasksStore,

		agentsTasks:    make(map[string]string),
		agentsTasksMux: &sync.Mutex{},
	}
	if err := registerRunningTasks(ctx, tc); err != nil {
		generateLoggerEventForUpgradeTaskWithPhase(ctx, "consume_new_tasks").
			WithError(err).Error("failed to fetch the running tasks")
	}
	return tc, nil
}

func registerRunningTasks(ctx context.Context, tc *upgradeTaskConsumer) error {
	tasks, err := tc.tasksStore.fetchRunningTasks(ctx)
	if err != nil {
		return err
	}
	for _, t := range tasks {
		tc.tasksStore.startTask(ctx, t)
		generateLoggerEventForUpgradeTaskWithTaskID(ctx, t.AgentHash, "register_running_tasks", t.ID).
			Info("watching an already running task")
	}
	return nil
}

const consumerPollingInterval = time.Second * 30

func (tc *upgradeTaskConsumer) consume(ctx context.Context) {
	defer tc.mm.wgControl.Done()
	var processRoutinesWG sync.WaitGroup
	for {
		select {
		case <-ctx.Done():
			processRoutinesWG.Wait()
			return
		case <-time.After(consumerPollingInterval):
		}
		tasks, err := tc.tasksStore.GetNewTasks(ctx)
		if err != nil {
			generateLoggerEventForUpgradeTaskWithPhase(ctx, "get_new_tasks").
				WithError(err).Error("failed to update the new tasks statuses")
			continue
		}
		if len(tasks) == 0 {
			continue
		}
		processRoutinesWG.Add(1)
		go func() {
			defer processRoutinesWG.Done()
			tc.processNewTasks(ctx, tasks)
		}()
	}
}

func (tc *upgradeTaskConsumer) processNewTasks(ctx context.Context, tasks []*store.Task) {
	failedTasks := make([]*store.Task, 0, len(tasks))
	for _, t := range tasks {
		if err := tc.registerUpgradeRequest(ctx, t); err != nil {
			failedTasks = append(failedTasks, t)
			continue
		}
		generateLoggerEventForUpgradeTaskWithTaskID(ctx, t.AgentHash, "process_new_tasks", t.ID).
			Info("new upgrade task is registered")
	}
	for _, ft := range failedTasks {
		tc.tasksStore.SignalTaskFailure(ctx, ft.AgentHash)
	}
}

func (tc *upgradeTaskConsumer) registerUpgradeRequest(ctx context.Context, t *store.Task) error {
	ainfo, err := tc.getAgentInfo(t.AgentHash)
	if err != nil {
		return err
	}
	if err := tc.checkUpgradeVersionCompatibility(ctx, ainfo.info.Ver, t.Version); err != nil {
		generateLoggerEventForUpgradeTaskWithTaskID(ctx, ainfo.info.ID, "upgrade-task-registration", t.ID).
			WithError(err).
			Warn("a problem with the agent and upgrader versions compatibility has been detected")
	}
	ainfo.upgrade <- t
	return nil
}

func (tc *upgradeTaskConsumer) requestAgentUpgrade(ctx context.Context, ainfo *agentInfo, req *store.Task) {
	var err error
	defer func() {
		defer tc.tasksStore.DecreaseNumberOfUpgradeTasks()
		if err == nil {
			return
		}
		agentInfo, err := tc.getAgentInfo(ainfo.info.ID)
		if err != nil {
			genAgentError(ctx, ainfo, err).Error("failed to get the agent ID")
			return
		}
		tc.tasksStore.SignalTaskFailure(ctx, agentInfo.info.ID)
	}()

	if err = tc.sendUpgradeToAgent(ctx, req.Version, ainfo); err != nil {
		genAgentError(ctx, ainfo, err).Error("vxserver: failed to send the upgrade file to the agent")
		return
	}
	if err = tc.requestAgentToUpgrade(ctx, req.Version, ainfo); err != nil {
		genAgentError(ctx, ainfo, err).Error("vxserver: failed to request the agent upgrade")
		return
	}
}

func (tc *upgradeTaskConsumer) checkUpgradeVersionCompatibility(
	_ context.Context,
	agentVersion string,
	upgraderVersion string,
) error {
	agentVersionSemver, err := semver.NewVersion(normalizeSemver(agentVersion))
	if err != nil {
		return fmt.Errorf("failed to parse the agent version %s: %w", agentVersion, err)
	}
	upgraderVersionSemver, err := semver.NewVersion(normalizeSemver(upgraderVersion))
	if err != nil {
		return fmt.Errorf("failed to parse the upgrader version %s: %w", upgraderVersion, err)
	}
	switch comparisonValue := upgraderVersionSemver.Compare(agentVersionSemver); comparisonValue {
	case -1:
		return fmt.Errorf(
			"upgrader version %s is lower than the current agent version %s",
			upgraderVersionSemver.String(),
			agentVersionSemver.String(),
		)
	case 0, 1:
		return nil
	default:
		return fmt.Errorf("an unexpected comparison value %d has been received", comparisonValue)
	}
}

// nolint: goconst
func normalizeSemver(version string) string {
	if preReleaseIdx := strings.Index(version, "-"); preReleaseIdx != -1 {
		version = version[:preReleaseIdx]
	}
	if cnt := strings.Count(version, "."); cnt >= 3 {
		version = strings.Join(
			strings.Split(version, ".")[0:3], // take only major, minor and patch parts
			".",
		)
	}
	return version
}

func (tc *upgradeTaskConsumer) sendUpgradeToAgent(ctx context.Context, version string, ainfo *agentInfo) error {
	upgrader, err := tc.getUpgrader(version, ainfo.info)
	if err != nil {
		return err
	}
	f := &vxproto.File{
		Data: upgrader.Upgrader,
		Name: vxproto.UpgraderFileName,
	}
	if err := tc.mm.msocket.SendFileTo(ctx, ainfo.info.Dst, f); err != nil {
		return fmt.Errorf("failed to send the file %s to the agent %s: %w", vxproto.UpgraderFileName, ainfo.info.ID, err)
	}
	return nil
}

func (tc *upgradeTaskConsumer) requestAgentToUpgrade(ctx context.Context, version string, ainfo *agentInfo) error {
	upgrader, err := tc.getUpgrader(version, ainfo.info)
	if err != nil {
		return err
	}
	msg, err := proto.Marshal(&agent.AgentUpgradeExecPush{
		Thumbprint: upgrader.UpgraderThumbprint,
	})
	if err != nil {
		return err
	}
	var resp agent.AgentUpgradeExecPushResult
	err = tc.mm.requestAgentWithDestStruct(
		ctx, ainfo.info.Dst, agent.Message_AGENT_UPGRADE_EXEC_PUSH,
		msg, agent.Message_AGENT_UPGRADE_EXEC_PUSH_RESULT, &resp)
	if err != nil {
		return err
	}
	return tc.processAgentUpgradeResponse(ctx, &resp, ainfo)
}

func (tc *upgradeTaskConsumer) processAgentUpgradeResponse(
	ctx context.Context, resp *agent.AgentUpgradeExecPushResult, ainfo *agentInfo,
) error {
	if resp.Success == nil {
		return fmt.Errorf("the AGENT_UPGRADE_EXEC_PUSH_RESULT message does not contain an indicator of the upgrade status")
	}
	respHint := resp.GetHint()
	if *resp.Success {
		logMsg := genAgentMessage(ctx, ainfo)
		if len(respHint) != 0 {
			logMsg.WithField("hint", respHint)
		}
		logMsg.Info("agent is ready for the upgrade")
		return nil
	}
	if len(respHint) != 0 {
		return fmt.Errorf("the agent upgrade request could not be fulfilled by the agent: %s", respHint)
	}
	return fmt.Errorf("the upgrade request could not be fulfilled by the agent, but no hint is returned")
}

func (tc *upgradeTaskConsumer) getUpgrader(version string, agentInfo *vxproto.AgentInfo) (*cache.Item, error) {
	cacheKey := cache.NewKey(version, agentInfo)
	item, err := tc.cache.Get(cacheKey)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (tc *upgradeTaskConsumer) getAgentInfo(agentID string) (*agentInfo, error) {
	agentTokens := tc.mm.agents.dumpID(agentID)
	for _, ainfo := range agentTokens {
		if ainfo.info.Type == vxproto.VXAgent {
			return ainfo, nil
		}
	}
	return nil, fmt.Errorf("no token found for the agent ID %s", agentID)
}

func (tc *upgradeTaskConsumer) Close(ctx context.Context) {
	closeWG := &sync.WaitGroup{}

	closeWG.Add(1)
	go func() {
		defer closeWG.Done()
		tc.tasksStore.Close(ctx)
	}()
	closeWG.Add(1)
	go func() {
		defer closeWG.Done()
		tc.cache.Close()
	}()

	closeWG.Wait()
}

type AgentConnectionEvent struct {
	AgentVersion string
}

type AgentUpgradeFailureEvent struct{}

type AgentAlreadyUpgradedEvent struct{}

func logSignalError(ctx context.Context, err error, agentID, phase string) {
	logger := generateLoggerEventForUpgradeTask(ctx, agentID, phase)
	if errors.Is(err, errTaskNotFound) {
		logger.WithError(err).Warn("task for the given agent ID is not found")
		return
	}
	if errors.Is(err, watcher.ErrNotificationChanClosed) {
		logger.WithError(err).Warn("task notification channel has already been closed")
		return
	}
	logger.WithError(err).Error("failed to pass an agent connection signal")
}

var (
	errTaskNotFound                = errors.New("task for the given agent ID was not found")
	errUnknownUpgradeTaskEventType = errors.New("the type of the received upgrade task event is unknown")
)

func generateLoggerForUpgradeTasks(ctx context.Context) *logrus.Entry {
	return logrus.WithContext(ctx).WithFields(logrus.Fields{
		"component": "upgrader",
		"module":    "main",
	})
}

func generateLoggerEventForUpgradeTask(ctx context.Context, agentID, phase string) *logrus.Entry {
	return generateLoggerForUpgradeTasks(ctx).WithFields(logrus.Fields{
		"agent_id": agentID,
		"phase":    phase,
	})
}

func generateLoggerEventForUpgradeTaskWithPhase(ctx context.Context, phase string) *logrus.Entry {
	return generateLoggerForUpgradeTasks(ctx).WithField("phase", phase)
}

func generateLoggerEventForUpgradeTaskWithTaskID(ctx context.Context, agentID, phase, taskID string) *logrus.Entry {
	return generateLoggerEventForUpgradeTask(ctx, agentID, phase).WithField("task_id", taskID)
}
