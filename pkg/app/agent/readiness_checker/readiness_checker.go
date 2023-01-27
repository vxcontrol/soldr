package readiness_checker

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	"soldr/pkg/app/agent"
	"soldr/pkg/app/agent/utils"
)

type ReadinessChecker struct {
	report      *agent.AgentReadinessReport
	reportMux   *sync.Mutex
	logFilePath string
	isClosed    bool
	isClosedMux *sync.RWMutex
}

func NewReadinessChecker(logDir string) (*ReadinessChecker, error) {
	logFilePath, err := composeLogFilePath(logDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a readiness checker: %w", err)
	}
	return &ReadinessChecker{
		report: &agent.AgentReadinessReport{
			Header: initReportHeader(),
		},
		reportMux:   &sync.Mutex{},
		logFilePath: logFilePath,
		isClosed:    false,
		isClosedMux: &sync.RWMutex{},
	}, nil
}

func composeLogFilePath(logDir string) (string, error) {
	const logFileName = "agent-start.log"
	pathResolver, err := utils.NewPathResolver(logDir)
	if err != nil {
		return "", fmt.Errorf("failed to compose the readiness log file path: %w", err)
	}
	return pathResolver.Resolve(logFileName), nil
}

func initReportHeader() *agent.AgentReadinessReportHeader {
	pid := int32(os.Getpid())
	return &agent.AgentReadinessReportHeader{
		Pid: &pid,
	}
}

type checkType string

const (
	osCheck checkType = "CheckOS"
)

func (rc *ReadinessChecker) logCheck(ct checkType, hasPassed bool) {
	rc.isClosedMux.RLock()
	defer rc.isClosedMux.RUnlock()
	if rc.isClosed {
		return
	}

	rc.reportMux.Lock()
	defer rc.reportMux.Unlock()

	checkTypeStr := string(ct)
	rc.report.Checks = append(rc.report.Checks, &agent.AgentReadinessReportCheck{
		Type:   &checkTypeStr,
		Passed: &hasPassed,
	})
}

func (rc *ReadinessChecker) CheckOS() error {
	err := rc.checkOS()
	rc.logCheck(osCheck, err == nil)
	return err
}

func (rc *ReadinessChecker) checkOS() error {
	switch runtime.GOOS {
	case "linux":
	case "windows":
	case "darwin":
	default:
		return fmt.Errorf("unsupported OS type %s", runtime.GOOS)
	}
	return nil
}

func (rc *ReadinessChecker) Finalize(ctx context.Context, status agent.AgentReadinessReportStatus) error {
	rc.isClosedMux.Lock()
	defer rc.isClosedMux.Unlock()
	if rc.isClosed {
		return nil
	}

	rc.writeAgentFinalStatus(status)
	if err := rc.dumpReport(); err != nil {
		return fmt.Errorf("failed to dump the readiness checker report: %w", err)
	}
	return nil
}

func (rc *ReadinessChecker) dumpReport() (err error) {
	report, err := serializeReport(rc.report)
	if err != nil {
		return err
	}
	fp, err := openLogFile(rc.logFilePath)
	defer func() {
		derr := fp.Close()
		if derr == nil {
			return
		}
		if err != nil {
			err = fmt.Errorf("failed to close the readiness checker report (%v)"+
				"after getting the following error: %w",
				derr,
				err,
			)
			return
		}
		err = fmt.Errorf("failed to close the readiness checker report: %w", err)
	}()
	if err != nil {
		return fmt.Errorf("failed to open the readiness checker log file: %w", err)
	}
	if _, err = fp.Write(report); err != nil {
		err = fmt.Errorf("failed to write the report to the readiness checker log file: %w", err)
		return
	}
	return nil
}

func (rc *ReadinessChecker) writeAgentFinalStatus(status agent.AgentReadinessReportStatus) {
	rc.report.Status = &status
}

func serializeReport(r *agent.AgentReadinessReport) ([]byte, error) {
	serializedReport, err := proto.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize the readiness checker report: %w", err)
	}
	return serializedReport, nil
}

func openLogFile(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open the file \"%s\": %w", path, err)
	}
	return f, nil
}

func ReadReport(ctx context.Context, logDir string) (*agent.AgentReadinessReport, error) {
	logFilePath, err := composeLogFilePath(logDir)
	if err != nil {
		return nil, err
	}
	fp, err := os.Open(logFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to read the readiness checker report file: %w", err)
	}
	defer func() {
		_ = fp.Close()
	}()
	reportData, err := ioutil.ReadAll(fp)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve the readiness checker report data: %w", err)
	}
	var report agent.AgentReadinessReport
	if err := proto.Unmarshal(reportData, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal the readiness checker report: %w", err)
	}
	return &report, nil
}

func BackupAndDeleteReport(ctx context.Context, logDir string) error {
	logFilePath, err := composeLogFilePath(logDir)
	if err != nil {
		return err
	}
	if _, err := os.Stat(logFilePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to get the info on the old readiness report file %s: %w", logFilePath, err)
	}
	if err := utils.CopyFile(logFilePath, logFilePath+".old"); err != nil {
		logrus.WithContext(ctx).WithError(err).Warn("failed to backup the old agent readiness report")
	}
	if err := os.RemoveAll(logFilePath); err != nil {
		return fmt.Errorf("an error occurred while removing an old readiness report file: %w", err)
	}
	return nil
}
