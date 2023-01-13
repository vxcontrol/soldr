package upgrader

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"soldr/pkg/app/agent/config"
	readinessChecker "soldr/pkg/app/agent/readiness_checker"
	"soldr/pkg/app/agent/service"
	upgraderErrors "soldr/pkg/app/agent/upgrader/errors"
	"soldr/pkg/app/agent/upgrader/starter"
	"soldr/pkg/app/agent/upgrader/starter/types"
	upgraderUtils "soldr/pkg/app/agent/upgrader/utils"
	"soldr/pkg/app/agent/utils"
	"soldr/pkg/app/agent/utils/pswatcher"
	"soldr/pkg/protoagent"
)

func RunAsUpgrader(ctx context.Context, conf *config.Config) (Status, error) {
	u, err := newUpgrader(conf)
	if err != nil {
		return StatusNotStarted, fmt.Errorf("failed to initialize an upgrader: %w", err)
	}
	return u.RunAsUpgrader(ctx)
}

type upgrader struct {
	pathResolver *utils.PathResolver
	conf         *config.Config
}

func newUpgrader(conf *config.Config) (*upgrader, error) {
	pr, err := upgraderUtils.NewPathResolver(conf.LogDir)
	if err != nil {
		return nil, err
	}
	return &upgrader{
		pathResolver: pr,
		conf:         conf,
	}, nil
}

func (u *upgrader) RunAsUpgrader(ctx context.Context) (Status, error) {
	if err := createUpgraderLock(u.pathResolver); err != nil {
		logrus.WithError(err).Warn("failed to create an upgrader lock")
	}
	agentBackupPath, err := u.backupAgentExec()
	if err != nil {
		logrus.WithError(err).Error("failed to backup the old agent file")
		return StatusNotStarted, err
	}
	logrus.WithField("path", agentBackupPath).Info("agent backup created")
	defer func() {
		if err := os.Remove(agentBackupPath); err != nil {
			logrus.WithError(err).Warn("failed to remove the agent back up file")
		}
	}()
	if err := u.signalUpgraderStart(); err != nil {
		logrus.WithError(err).Error("failed to signal the upgrader start")
		return StatusNotStarted, err
	}
	if u.conf.Service {
		if err := stopService(ctx); err != nil {
			return StatusNotStarted, err
		}
	}
	upgraderCtx, cancelUpgraderCtx := context.WithTimeout(ctx, upgraderTimeout)
	defer cancelUpgraderCtx()
	if err := pswatcher.WaitForProcessToFinish(upgraderCtx, u.conf.PPID, time.Second); err != nil {
		if errors.Is(err, upgraderErrors.ErrUpgraderTimeout) {
			logrus.WithError(err).Error("the old agent process has not finished in time")
			return StatusFailed, err
		}
		logrus.WithError(err).Error("failed to wait for the old agent process to finish")
		return StatusFailed, err
	}
	logrus.Info("the old agent process has finished")
	return u.tryUpgradeAgent(ctx, agentBackupPath)
}

func (u *upgrader) tryUpgradeAgent(ctx context.Context, agentBackupPath string) (Status, error) {
	if err := u.replaceAgentExecutableWithUpgrader(ctx); err != nil {
		logrus.WithError(err).
			Warn("failed to replace the old agent executable with a new one, " +
				"trying to rerun the previous version of the agent")
		if err := u.startAgent(ctx, upgraderTimeout); err != nil {
			logrus.WithError(err).Error("failed to restart the old version of the agent")
			return StatusFailed, err
		}
		logrus.Info("the old version of the agent has been successfully restarted")
		return StatusRestarted, nil
	}
	if err := u.startAgent(ctx, upgraderTimeout); err != nil {
		logrus.WithError(err).Warn("the upgrade process failed, trying to rerun the previous version of the agent")
		if err := u.restorePreviousAgent(ctx, agentBackupPath); err != nil {
			logrus.WithError(err).Error("failed to start the previous version of the agent")
			return StatusFailed, err
		}
		logrus.Info("the previous version of the agent has successfully started")
		return StatusRestarted, nil
	}
	return StatusSuccess, nil
}

func (u *upgrader) signalUpgraderStart() error {
	upgraderStarter, err := starter.NewStarter(u.conf.LogDir, types.UpgraderComponentName)
	if err != nil {
		return fmt.Errorf("failed to initialize a new upgrader starter: %w", err)
	}
	if err := upgraderStarter.SignalStart(); err != nil {
		return fmt.Errorf("failed to signal the upgrader start: %w", err)
	}
	agentWatcher, err := starter.NewStartChecker(u.conf.LogDir, types.AgentComponentName)
	if err != nil {
		return fmt.Errorf("failed to initialize a new upgrader start checker: %w", err)
	}
	if err := agentWatcher.WaitForStart(); err != nil {
		return fmt.Errorf("agent did not signal its readiness to be upgraded")
	}
	return nil
}

const failedToReplaceAgentFileMsg = "failed to replace the old agent file with the current executable: %w"

func (u *upgrader) restorePreviousAgent(ctx context.Context, agentBackupPath string) error {
	if err := utils.MoveFile(agentBackupPath, u.conf.AgentExecutablePath); err != nil {
		return fmt.Errorf(failedToReplaceAgentFileMsg, err)
	}
	return u.startAgent(ctx, upgraderTimeout)
}

func (u *upgrader) startAgent(ctx context.Context, timeout time.Duration) error {
	if err := readinessChecker.BackupAndDeleteReport(ctx, u.conf.LogDir); err != nil {
		logrus.WithError(err).Warn("failed to backup and delete the old readiness report")
	}
	if u.conf.Service {
		if err := u.startAgentService(ctx); err != nil {
			return err
		}
		return nil
	}
	agentPID, err := u.startAgentProcess(ctx)
	if err != nil {
		return err
	}
	startAgentCtx, cancelStartAgentCtx := context.WithTimeout(ctx, timeout)
	defer cancelStartAgentCtx()
	if err := u.checkAgentReport(startAgentCtx, agentPID); err != nil {
		return err
	}
	return nil
}

const agentBackupFile = "agent.bck"

func (u *upgrader) backupAgentExec() (string, error) {
	backupDst := u.pathResolver.Resolve(agentBackupFile)
	if err := utils.CopyFile(u.conf.AgentExecutablePath, backupDst); err != nil {
		return "", fmt.Errorf("failed to backup the agent executable file: %w", err)
	}
	return backupDst, nil
}

func (u *upgrader) replaceAgentExecutableWithUpgrader(_ context.Context) error {
	upgraderFilePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get the current executable file path: %w", err)
	}
	logrus.Infof("current file: %s, agent exec path: %s", upgraderFilePath, u.conf.AgentExecutablePath)
	err = utils.MoveFile(
		upgraderFilePath,
		u.conf.AgentExecutablePath,
		&utils.MoveFileOptions{
			Mode: 0o755,
		},
	)
	if err != nil {
		return fmt.Errorf(failedToReplaceAgentFileMsg, err)
	}
	return nil
}

func (u *upgrader) startAgentProcess(_ context.Context) (int, error) {
	args := getAgentProcessFlags()
	cmd := exec.Command(u.conf.AgentExecutablePath, args...)
	if err := cmd.Start(); err != nil {
		return -1, fmt.Errorf("failed to start the new agent process: %w", err)
	}
	return cmd.Process.Pid, nil
}

func (u *upgrader) startAgentService(_ context.Context) error {
	svc, err := service.Initialize()
	if err != nil {
		return fmt.Errorf("failed to initialize a service object: %w", err)
	}
	status, err := svc.Start()
	if err != nil {
		return fmt.Errorf("failed to start the vxagent service: %w", err)
	}
	logrus.Debugf("service status: %s", status)
	return nil
}

func getAgentProcessFlags() []string {
	var args []string
	if len(os.Args) > 1 {
		args = os.Args[1:]
	}
	args = config.ChangeOrSetArg(args, config.ArgNameMode, config.RunningModeAgent.String())
	args = config.IfServiceAddStartFlag(args)
	return args
}

func (u *upgrader) checkAgentReport(ctx context.Context, agentPID int) error {
	logrus.Info("waiting for the agent readiness report")
	report, err := u.readAgentReport(ctx)
	if err != nil {
		logrus.WithError(err).Error("error reading the agent report")
		return err
	}
	logrus.Info("got the agent readiness report")
	if err := u.verifyAgentReport(report, agentPID); err != nil {
		logrus.WithError(err).Error("error verifying the agent report")
		return err
	}
	logrus.Info("the agent has reported a successful start")
	return nil
}

func (u *upgrader) readAgentReport(ctx context.Context) (*protoagent.AgentReadinessReport, error) {
	for {
		time.Sleep(time.Second * 1)
		select {
		case <-ctx.Done():
			return nil, upgraderErrors.ErrUpgraderTimeout
		default:
		}

		report, err := readinessChecker.ReadReport(ctx, u.conf.LogDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("failed to read the agent readiness report: %w", err)
		}
		return report, nil
	}
}

func (u *upgrader) verifyAgentReport(report *protoagent.AgentReadinessReport, agentPID int) error {
	if !u.conf.Service {
		if err := checkAgentReportHeader(report.GetHeader(), agentPID); err != nil {
			return err
		}
	}
	if err := checkAgentReportChecks(report.GetChecks()); err != nil {
		return err
	}
	if err := checkAgentReportStatus(report.GetStatus()); err != nil {
		return err
	}
	return nil
}

func checkAgentReportHeader(header *protoagent.AgentReadinessReportHeader, agentPID int) error {
	if header == nil {
		return fmt.Errorf("the agent readiness report does not contain a header")
	}
	if header.Pid == nil {
		return fmt.Errorf("the agent readiness report header does not contain the agent PID")
	}
	if headerPID := int(*header.Pid); headerPID != agentPID {
		return fmt.Errorf("the readiness report points to a process with PID %d, the agent PID is: %d", headerPID, agentPID)
	}
	return nil
}

func checkAgentReportChecks(checks []*protoagent.AgentReadinessReportCheck) error {
	failedChecks := make([]string, 0, len(checks))
	for _, c := range checks {
		if *c.Passed {
			continue
		}
		failedChecks = append(failedChecks, c.GetType())
	}
	if len(failedChecks) == 0 {
		return nil
	}
	return fmt.Errorf("the following checks have failed during the agent start: %s", strings.Join(failedChecks, ", "))
}

func checkAgentReportStatus(status protoagent.AgentReadinessReportStatus) error {
	if status == protoagent.AgentReadinessReportStatus_SUCCESS {
		return nil
	}
	return fmt.Errorf("the agent has returned a non-successful status: %s", status.String())
}

func stopService(_ context.Context) error {
	svc, err := service.Initialize()
	if err != nil {
		return fmt.Errorf("failed to initialize the service handler object: %w", err)
	}
	status, err := svc.Stop()
	if err != nil {
		// Let's hope for the best!
		logrus.WithError(err).Warn("an error occurred during the agent service stopping")
		return nil
	}
	logrus.WithField("status", status).Info("the agent service has been stopped")
	return nil
}

type Status string

const (
	upgraderTimeout = time.Second * 60

	StatusSuccess    Status = "upgrader exited successfully"
	StatusFailed     Status = "upgrader failed"
	StatusRestarted  Status = "upgrader has restarted a previous agent version"
	StatusNotStarted Status = "upgrader did not start properly"
)

func (u Status) String() string {
	return string(u)
}
