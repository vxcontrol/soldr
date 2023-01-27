package mmodule

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"github.com/takama/daemon"
	"google.golang.org/protobuf/proto"

	"soldr/pkg/app/agent"
	"soldr/pkg/app/agent/config"
	upgraderPack "soldr/pkg/app/agent/upgrader"
	upgraderErrors "soldr/pkg/app/agent/upgrader/errors"
	"soldr/pkg/app/agent/upgrader/starter"
	"soldr/pkg/app/agent/upgrader/starter/types"
	upgraderUtils "soldr/pkg/app/agent/upgrader/utils"
	"soldr/pkg/app/agent/utils"
	obs "soldr/pkg/observability"
	"soldr/pkg/vxproto"
)

type upgrader struct {
	mm                   *MainModule
	isService            bool
	svc                  daemon.Daemon
	upgraderFile         string
	upgraderFileMux      *sync.RWMutex
	stopAgent            context.CancelFunc
	upgraderStartChecker starter.StartChecker
	agentStarter         starter.Starter
	pathResolver         *utils.PathResolver
	cleaner              *upgraderFilesCleaner
}

func newUpgrader(
	mm *MainModule,
	isService bool,
	logDir string,
	stopAgent context.CancelFunc,
	svc daemon.Daemon,
) (*upgrader, error) {
	upgraderStartChecker, err := starter.NewStartChecker(logDir, types.UpgraderComponentName)
	if err != nil {
		return nil, err
	}
	agentStarter, err := starter.NewStarter(logDir, types.AgentComponentName)
	if err != nil {
		return nil, err
	}
	pathResolver, err := upgraderUtils.NewPathResolver(logDir)
	if err != nil {
		return nil, err
	}
	return &upgrader{
		mm:                   mm,
		isService:            isService,
		svc:                  svc,
		upgraderFileMux:      &sync.RWMutex{},
		stopAgent:            stopAgent,
		upgraderStartChecker: upgraderStartChecker,
		agentStarter:         agentStarter,
		pathResolver:         pathResolver,
		cleaner:              newUpgraderFilesCleaner(logDir),
	}, nil
}

const componentUpgrader = "upgrader_recv_file"

func (u *upgrader) recvFile(ctx context.Context, file *vxproto.File) error {
	u.upgraderFileMux.Lock()
	defer u.upgraderFileMux.Unlock()

	upgraderCtx, upgraderSpan := obs.Observer.NewSpan(ctx, obs.SpanKindConsumer, componentUpgrader)
	defer upgraderSpan.End()

	stopTimeoutCtx, cancelStopTimeoutCtx := context.WithTimeout(upgraderCtx, time.Second)
	defer cancelStopTimeoutCtx()
	if err := u.cleaner.Stop(stopTimeoutCtx); err != nil {
		logrus.WithContext(upgraderCtx).WithError(err).Warn("failed to stop the cleaner routine")
	}

	upgraderFilePath := u.pathResolver.Resolve(composeUpgraderFileName())
	err := utils.MoveFile(file.Path, upgraderFilePath, &utils.MoveFileOptions{Mode: 0o755})
	if err != nil {
		err = fmt.Errorf(
			"failed to copy the received file %s to the upgrader file location %s: %w",
			file.Path,
			upgraderFilePath,
			err,
		)
		logrus.WithContext(upgraderCtx).WithError(err).Error("failed to copy upgrader file")
		return err
	}
	u.upgraderFile = upgraderFilePath
	logrus.WithContext(upgraderCtx).Debugf("upgrader file stored to %s", upgraderFilePath)
	return nil
}

func composeUpgraderFileName() string {
	if runtime.GOOS == "windows" {
		return "upgrader.exe"
	}
	return "upgrader"
}

func (u *upgrader) startUpgrade(ctx context.Context, src string, msg *agent.AgentUpgradeExecPush) error {
	upgraderCtx, upgraderSpan := obs.Observer.NewSpan(ctx, obs.SpanKindConsumer, componentUpgrader)
	defer upgraderSpan.End()

	logger := logrus.WithContext(upgraderCtx).WithFields(logrus.Fields{
		"is_service":    u.isService,
		"upgrader_file": u.upgraderFile,
	})
	logger.Debug("starting the agent upgrade")

	u.upgraderFileMux.RLock()
	defer u.upgraderFileMux.RUnlock()

	if checkFileErr := u.checkUpgraderFile(msg); checkFileErr != nil {
		if err := u.sendUpgradeResponseToServer(upgraderCtx, src, checkFileErr); err != nil {
			return err
		}
		return nil
	}
	logger.Debug("upgrader integrity check has passed")
	if runUpgraderErr := u.runUpgraderProcess(); runUpgraderErr != nil {
		logger.WithError(runUpgraderErr).Error("failed to upgrade agent")
		if err := u.sendUpgradeResponseToServer(upgraderCtx, src, runUpgraderErr); err != nil {
			logger.WithError(err).Error("failed to notify server about upgrader error")
			return fmt.Errorf("failed to send an error (%v) to the server: %w", runUpgraderErr, err)
		}
		return runUpgraderErr
	}
	logger.Info("agent has started the upgrader")
	if err := u.sendUpgradeResponseToServer(upgraderCtx, src, nil); err != nil {
		logger.WithError(err).Error("failed to notify server about upgrader result")
		return err
	}
	if err := u.agentStarter.SignalStart(); err != nil {
		logger.WithError(err).Error("failed to notify that the agent is ready for the upgrade")
	}
	logger.Info("agent wants to stop upgrader")
	upgraderSpan.End()
	if err := u.stop(); err != nil {
		return err
	}
	return nil
}

func (u *upgrader) sendUpgradeResponseToServer(ctx context.Context, src string, e error) error {
	var respHint string
	respSuccess := true
	if e != nil {
		respHint = e.Error()
		respSuccess = false
	}
	resp := agent.AgentUpgradeExecPushResult{
		Hint:    &respHint,
		Success: &respSuccess,
	}
	respData, err := proto.Marshal(&resp)
	if err != nil {
		return fmt.Errorf("failed to marshal the upgrade exec push result message: %w", err)
	}
	if err := u.mm.responseAgent(ctx, src, agent.Message_AGENT_UPGRADE_EXEC_PUSH_RESULT, respData); err != nil {
		return fmt.Errorf("failed to send the upgrade exec push result: %w", err)
	}
	return nil
}

func (u *upgrader) checkUpgraderFile(msg *agent.AgentUpgradeExecPush) error {
	upgraderFileData, err := ioutil.ReadFile(u.upgraderFile)
	if err != nil {
		return fmt.Errorf("failed to read the upgrader file: %w", err)
	}
	if err := checkUpgraderFileIntegrity(upgraderFileData, msg.GetThumbprint()); err != nil {
		return err
	}
	return nil
}

func (u *upgrader) runUpgraderProcess() error {
	if err := u.startUpgraderProcess(u.isService); err != nil {
		return err
	}
	if err := u.upgraderStartChecker.WaitForStart(); err != nil {
		return fmt.Errorf("upgrader failed to start: %w", err)
	}
	return nil
}

func (u *upgrader) stop() error {
	if u.isService {
		return nil
	}
	u.stopAgent()
	return nil
}

func getUpgraderProcessArgs(procArgs []string, pid int, agentExecPath string) []string {
	argsToSet := map[config.ArgName]string{
		config.ArgNamePPID:      fmt.Sprintf("%d", pid),
		config.ArgNameAgentExec: agentExecPath,
		config.ArgNameMode:      config.RunningModeUpgrader.String(),
	}
	newProcArgs := make([]string, 0, len(procArgs)+len(argsToSet))
	newProcArgs = append(newProcArgs, procArgs...)
	return config.ChangeOrSetArgs(newProcArgs, argsToSet)
}

func checkUpgraderFileIntegrity(upgraderFileData []byte, checksumsData []byte) error {
	ic, err := newIntegrityChecker(checksumsData)
	if err != nil {
		return fmt.Errorf("failed to initialize an integrity checker")
	}
	if err := ic.Check(upgraderFileData); err != nil {
		return err
	}
	return nil
}

type integrityChecker struct {
	MD5    string `json:"md5" validate:"required,len=32,hexadecimal,lowercase"`
	SHA256 string `json:"sha256" validate:"required,len=64,hexadecimal,lowercase"`
}

func newIntegrityChecker(checksumsFile []byte) (*integrityChecker, error) {
	var checksums integrityChecker
	if err := json.Unmarshal(checksumsFile, &checksums); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal the received checksums file into models.BinaryChksum object: %w",
			err,
		)
	}
	validate := validator.New()
	if err := validate.Struct(checksums); err != nil {
		return nil, fmt.Errorf("integrity checker checksums are empty: %w", err)
	}
	return &checksums, nil
}

func (c *integrityChecker) Check(data []byte) error {
	if err := c.checkMD5(data); err != nil {
		return fmt.Errorf("MD5 integrity check has failed: %w", err)
	}
	if err := c.checkSHA256(data); err != nil {
		return fmt.Errorf("SHA256 integrity check has failed: %w", err)
	}
	return nil
}

func (c *integrityChecker) checkMD5(data []byte) error {
	dataHash := md5.Sum(data)
	hexDataHash := hex.EncodeToString(dataHash[:])
	if c.MD5 != hexDataHash {
		return fmt.Errorf(
			"MD5 integrity check failed: expected hash (%s) and actual hash (%s) are not equal",
			c.MD5, hexDataHash,
		)
	}
	return nil
}

func (c *integrityChecker) checkSHA256(data []byte) error {
	dataHash := sha256.Sum256(data)
	hexDataHash := hex.EncodeToString(dataHash[:])
	if c.SHA256 != hexDataHash {
		return fmt.Errorf(
			"SHA256 integrity check failed: expected hash (%s) and actual hash (%s) are not equal",
			c.SHA256, hexDataHash,
		)
	}
	return nil
}

const lockFileName = "upgrader.lock"

func (u *upgrader) CreateUpgraderLockFile() error {
	lockFileDst := u.pathResolver.Resolve(lockFileName)
	if err := ioutil.WriteFile(lockFileDst, nil, 0o600); err != nil {
		return fmt.Errorf("failed to write the lock file %s: %w", lockFileDst, err)
	}
	return nil
}

func (u *upgrader) DeleteUpgraderLockFile() error {
	lockFileDst := u.pathResolver.Resolve(lockFileName)
	if err := os.Remove(lockFileDst); err != nil {
		return fmt.Errorf("failed to remove the lock file %s: %w", lockFileDst, err)
	}
	return nil
}

type upgraderFilesCleaner struct {
	stopCh        <-chan struct{}
	ctrlCtx       context.Context
	cancelCtrlCtx context.CancelFunc
}

func newUpgraderFilesCleaner(logDir string) *upgraderFilesCleaner {
	stopCh := make(chan struct{})
	ctrlCtx, cancelCtrlCtx := context.WithCancel(context.Background())
	go tryCleanUpgraderDir(ctrlCtx, cancelCtrlCtx, stopCh, logDir)
	return &upgraderFilesCleaner{
		stopCh:        stopCh,
		ctrlCtx:       ctrlCtx,
		cancelCtrlCtx: cancelCtrlCtx,
	}
}

func (c *upgraderFilesCleaner) Stop(ctx context.Context) error {
	c.cancelCtrlCtx()
	select {
	case <-c.stopCh:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("the cleaner routine did not stop properly")
	}
}

func tryCleanUpgraderDir(ctx context.Context, cancelCtx context.CancelFunc, stopCh chan<- struct{}, logDir string) {
	defer close(stopCh)
	defer cancelCtx()
	for {
		isCleaned, err := waitToCleanUpgraderDir(ctx, logDir)
		if err != nil {
			logrus.WithError(err).Warn("failed to clean the upgrader dir")
			return
		}
		if isCleaned {
			logrus.Debug("the upgrader directory has been successfully removed")
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func waitToCleanUpgraderDir(ctx context.Context, logDir string) (bool, error) {
	timeoutCtx, cancelTimeoutCtx := context.WithTimeout(ctx, time.Millisecond*500)
	defer cancelTimeoutCtx()
	if err := upgraderPack.CleanUpgradeDir(timeoutCtx, logDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		if errors.Is(err, upgraderErrors.ErrUpgraderTimeout) {
			return false, nil
		}
		return false, fmt.Errorf("failed to clean the upgrade directory: %w", err)
	}
	return true, nil
}

const failedToStartTheUpgraderMsg = "failed to start the upgrader: %w"

func (u *upgrader) startUpgraderProcess(isService bool) error {
	upgraderProcArgs, err := getUpgraderArgs()
	if err != nil {
		return fmt.Errorf("failed to get the upgrader process args: %w", err)
	}
	if isService && runtime.GOOS == "linux" {
		if _, err := os.Stat("/run/systemd/system"); err == nil {
			return u.startSystemdUpgraderProcess(upgraderProcArgs)
		}
	}
	cmd := exec.Command(u.upgraderFile, upgraderProcArgs...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf(failedToStartTheUpgraderMsg, err)
	}
	return nil
}

func (u *upgrader) startSystemdUpgraderProcess(args []string) error {
	logrus.Debug("starting system upgrader process")
	cmdArgs := append([]string{
		"--unit=vxagent_upgrade",
		"--scope",
		"--slice=vxagent_upgrade_slice",
		u.upgraderFile,
	},
		args...,
	)
	cmd := exec.Command("systemd-run", cmdArgs...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf(failedToStartTheUpgraderMsg, err)
	}
	return nil
}

func getUpgraderArgs() ([]string, error) {
	var procArgs []string
	if len(os.Args) > 1 {
		// append old agent args, so the upgrader has the old agent config parameters
		procArgs = os.Args[1:]
	}
	agentExecPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get the path to the agent executable: %w", err)
	}
	upgraderProcArgs := getUpgraderProcessArgs(procArgs, os.Getpid(), agentExecPath)
	return upgraderProcArgs, nil
}
