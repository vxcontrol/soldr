package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/judwhite/go-svc"
	"github.com/sirupsen/logrus"
	"github.com/takama/daemon"
	"go.opentelemetry.io/otel/attribute"
	"gopkg.in/natefinch/lumberjack.v2"

	"soldr/internal/app/agent/config"
	"soldr/internal/app/agent/mmodule"
	readinessChecker "soldr/internal/app/agent/readiness_checker"
	"soldr/internal/app/agent/service"
	"soldr/internal/app/agent/upgrader"
	"soldr/internal/observability"
	"soldr/internal/protoagent"
	"soldr/internal/system"
	"soldr/internal/utils"
)

const (
	serviceName = "vxagent"
	loggerKey   = "component"
)

// Agent implements daemon structure
type Agent struct {
	ctx       context.Context
	stopAgent context.CancelFunc
	rc        *readinessChecker.ReadinessChecker

	module *mmodule.MainModule
	cfg    *config.Config
	svc    daemon.Daemon
	wg     sync.WaitGroup
}

func newAgent(ctx context.Context, c *config.Config) (*Agent, error) {
	rc, err := readinessChecker.NewReadinessChecker(c.LogDir)
	if err != nil {
		return nil, err
	}
	actx, cancel := context.WithCancel(context.Background())
	a := &Agent{
		ctx:       actx,
		cfg:       c,
		rc:        rc,
		stopAgent: cancel,
	}
	a.svc, err = createService(ctx, a.rc)
	if err != nil {
		logrus.WithContext(ctx).WithError(err).Error("error on service creating")
		FinalizeReadinessCheckAndLogPotentialError(ctx, a.rc, protoagent.AgentReadinessReportStatus_FAILURE)
		return nil, err
	}
	return a, nil
}

func FinalizeReadinessCheckAndLogPotentialError(
	ctx context.Context,
	rc *readinessChecker.ReadinessChecker,
	status protoagent.AgentReadinessReportStatus,
) {
	const (
		successStatusMessage = "failed to indicate a successfull status of the agent start to the readiness checker"
		failureStatusMessage = "failed to indicate a failed status of the agent start to the readiness checker"
		unknownStatusMessage = "failed to indicate an unknown status (%d) of the agent start to the readiness checker"
	)
	finCtx, finSpan := observability.Observer.NewSpan(ctx, observability.SpanKindInternal, "rcheck_finalizer")
	defer finSpan.End()

	err := rc.Finalize(finCtx, protoagent.AgentReadinessReportStatus(status))
	if err == nil {
		return
	}
	logEntry := logrus.WithContext(ctx).WithError(err)
	switch status {
	case protoagent.AgentReadinessReportStatus_SUCCESS:
		logEntry.Error(successStatusMessage)
	case protoagent.AgentReadinessReportStatus_FAILURE:
		logEntry.Error(failureStatusMessage)
	default:
		logEntry.Error(fmt.Sprintf(unknownStatusMessage, status))
	}
}

func createService(_ context.Context, rc *readinessChecker.ReadinessChecker) (daemon.Daemon, error) {
	if err := rc.CheckOS(); err != nil {
		return nil, err
	}
	svc, err := service.Initialize()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a new service: %w", err)
	}
	return svc, nil
}

// Init for preparing agent main module struct
func (a *Agent) Init(env svc.Environment) (err error) {
	utils.RemoveUnusedTempDir()
	initCtx, initSpan := observability.Observer.NewSpan(a.Context(), observability.SpanKindInternal, "init_agent_service")
	defer initSpan.End()

	a.module, err = mmodule.New(
		a.cfg.Connect,
		a.cfg.AgentID,
		a.cfg.Version,
		a.cfg.Service,
		a.cfg.LogDir,
		a.stopAgent,
		a.svc,
		a.cfg.MeterConfigClient,
		a.cfg.TracerConfigClient,
	)
	if err != nil {
		err = fmt.Errorf("failed to create new main module: %w", err)
		logrus.WithContext(initCtx).WithError(err).Error("failed to initialize")
		FinalizeReadinessCheckAndLogPotentialError(initCtx, a.rc, protoagent.AgentReadinessReportStatus_FAILURE)
		return
	}

	return
}

// Start logic of agent main module
func (a *Agent) Start() (err error) {
	startCtx, startSpan := observability.Observer.NewSpan(a.Context(), observability.SpanKindInternal, "start_agent_service")
	defer startSpan.End()

	logrus.WithContext(startCtx).Info("vxagent is starting...")
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		err = a.module.Start()
	}()

	// Wait a little time to catch error on start
	time.Sleep(time.Second)
	logrus.WithContext(startCtx).Info("vxagent started")
	FinalizeReadinessCheckAndLogPotentialError(startCtx, a.rc, protoagent.AgentReadinessReportStatus_SUCCESS)

	return
}

// Stop logic of agent main module
func (a *Agent) Stop() (err error) {
	stopCtx, stopSpan := observability.Observer.NewSpan(a.Context(), observability.SpanKindInternal, "stop_agent_service")
	logrus.WithContext(stopCtx).Info("vxagent is stopping...")
	stopSpan.End()

	if a.module != nil {
		if err = a.module.Stop("agent_stop"); err != nil {
			return
		}
	}
	logrus.WithContext(stopCtx).Info("vxagent is waiting of modules release...")
	a.wg.Wait()
	logrus.WithContext(stopCtx).Info("vxagent stopped")

	return
}

// Manage by daemon commands or run the daemon
func (a *Agent) Manage() (string, error) {
	switch a.cfg.Command {
	case "install":
		opts := []string{
			"-service",
			"-connect", a.cfg.Connect,
			"-logdir", a.cfg.LogDir,
		}
		if a.cfg.AgentID != "" {
			opts = append(opts, "-agent", a.cfg.AgentID)
		}
		if a.cfg.Debug {
			opts = append(opts, "-debug")
		}
		if err := configureServiceAutorestart(a.svc); err != nil {
			return "", err
		}
		return a.svc.Install(opts...)
	case "uninstall":
		return a.svc.Remove()
	case "start":
		return a.svc.Start()
	case "stop":
		return a.svc.Stop()
	case "status":
		return a.svc.Status()
	}
	if err := svc.Run(a); err != nil {
		logrus.WithContext(a.Context()).WithField(loggerKey, "runtime").
			WithError(err).Error("vxagent executing failed")
		return "vxagent running failed", err
	}

	return "vxagent exited normally", nil
}

func configureServiceAutorestart(svc daemon.Daemon) error {
	logrus.Info("in configure service autorestart")
	if runtime.GOOS != "linux" {
		return nil
	}
	const systemdDir = "/run/systemd/system"
	if _, err := os.Stat(systemdDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to check if the directory %s exists: %w", systemdDir, err)
	}
	logrus.Info("configuring autorestart")
	const tmpl = `[Unit]
Description={{.Description}}
Requires={{.Dependencies}}
After={{.Dependencies}}

[Service]
PIDFile=/var/run/{{.Name}}.pid
ExecStartPre=/bin/rm -f /var/run/{{.Name}}.pid
ExecStart={{.Path}} {{.Args}}
Restart=always

[Install]
WantedBy=multi-user.target
`
	if err := svc.SetTemplate(tmpl); err != nil {
		return fmt.Errorf("failed to configure the linux service autorestart")
	}
	return nil
}

func (a *Agent) Context() context.Context {
	return a.ctx
}

func configureLogging(ctx context.Context, c *config.Config) (func(), error) {
	logLevels := []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	}
	if c.Debug {
		logrus.SetLevel(logrus.DebugLevel)
		logLevels = append(logLevels, logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	logFileName := "agent.log"
	if c.Mode == config.RunningModeUpgrader {
		logFileName = "upgrader.log"
	}

	logFile := &lumberjack.Logger{
		Filename:   filepath.Join(c.LogDir, logFileName),
		MaxSize:    100,
		MaxBackups: 7,
		MaxAge:     14,
		Compress:   true,
	}
	if c.Service {
		logrus.SetOutput(logFile)
	} else {
		logrus.SetOutput(io.MultiWriter(os.Stdout, logFile))
	}

	agentID := c.AgentID
	if agentID == "" {
		agentID = system.MakeAgentID()
	}
	attr := attribute.String("agent_id", agentID)

	if c.TracerConfigClient == nil {
		c.TracerConfigClient = &observability.HookClientConfig{
			ResendTimeout:   observability.DefaultResendTimeout,
			QueueSizeLimit:  observability.DefaultQueueSizeLimit,
			PacketSizeLimit: observability.DefaultPacketSizeLimit,
		}
	}
	tracerClient := observability.NewHookTracerClient(c.TracerConfigClient)
	tracerProvider, err := observability.NewTracerProvider(ctx, tracerClient, serviceName, c.Version, attr)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a tracer provider for logging: %w", err)
	}

	if c.MeterConfigClient == nil {
		c.MeterConfigClient = &observability.HookClientConfig{
			ResendTimeout:   observability.DefaultResendTimeout,
			QueueSizeLimit:  observability.DefaultQueueSizeLimit,
			PacketSizeLimit: observability.DefaultPacketSizeLimit,
		}
	}
	meterClient := observability.NewHookMeterClient(c.MeterConfigClient)
	meterProvider, err := observability.NewMeterProvider(ctx, meterClient, serviceName, c.Version, attr)
	if err != nil {
		return nil, fmt.Errorf("failed to initialized a metrics provider for logging")
	}

	observability.InitObserver(ctx, tracerProvider, meterProvider, tracerClient, meterClient, serviceName, logLevels)
	return func() {
		observability.Observer.Close()
	}, nil
}

func runAsAgent(ctx context.Context, c *config.Config) (string, error) {
	runCtx, runSpan := observability.Observer.NewSpan(ctx, observability.SpanKindInternal, "run_agent")
	logrus.WithContext(runCtx).WithField("version", c.Version).Info("ready to start")
	a, err := newAgent(runCtx, c)
	if err != nil {
		runSpan.End()
		return "", fmt.Errorf("failed to initialize an agent: %w", err)
	}
	runSpan.End()
	return a.Manage()
}

func genExitMessage(ctx context.Context, status string, e error) *logrus.Entry {
	entry := logrus.WithFields(logrus.Fields{
		"component":   "shutdown",
		"exit-status": status,
	})
	if e != nil {
		entry = entry.WithError(e)
	}
	return entry.WithContext(ctx)
}

func main() {
	conf, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}
	if conf.PrintVersion {
		fmt.Printf("vxagent version is %s\n", conf.Version)
		return
	}

	rootCtx := context.Background()
	loggingTeardown, err := configureLogging(rootCtx, conf)
	if err != nil {
		log.Fatal(err)
	}
	defer loggingTeardown()

	if conf.Mode == config.RunningModeUpgrader {
		upgradeCtx, upgradeSpan := observability.Observer.NewSpan(rootCtx, observability.SpanKindInternal, "upgrade_agent")
		status, err := upgrader.RunAsUpgrader(upgradeCtx, conf)
		if err != nil {
			genExitMessage(upgradeCtx, status.String(), err).Error("upgrader failed")
			upgradeSpan.End()
			return
		}
		logrus.WithContext(upgradeCtx).Debug("the upgrader is exiting")
		upgradeSpan.End()
		return
	}

	status, err := runAsAgent(rootCtx, conf)
	exitCtx, exitSpan := observability.Observer.NewSpan(rootCtx, observability.SpanKindInternal, "exit_agent")
	if err != nil {
		genExitMessage(exitCtx, status, err).Error("agent failed")
		exitSpan.End()
		return
	}
	genExitMessage(exitCtx, status, nil).Info("agent exited normally")
	exitSpan.End()
}
