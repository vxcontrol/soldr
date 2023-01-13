package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	obs "soldr/pkg/observability"
)

// PackageVer is semantic version of vxagent
var PackageVer string

// PackageRev is revision of vxagent build
var PackageRev string

type Config struct {
	Connect             string
	AgentID             string
	Command             string
	LogDir              string
	Debug               bool
	Service             bool
	PrintVersion        bool
	Version             string
	Mode                RunningMode
	PPID                int
	AgentExecutablePath string
	PreviousConfig      string

	MeterConfigClient  *obs.HookClientConfig
	TracerConfigClient *obs.HookClientConfig

	runningMode string
	ppidArg     string
}

type ArgName string

const (
	ArgNameMode      ArgName = "mode"
	argNameMode              = string(ArgNameMode)
	ArgNamePPID      ArgName = "ppid"
	argNamePPID              = string(ArgNamePPID)
	ArgNameAgentExec ArgName = "agent_exec"
	argNameAgentExec         = string(ArgNameAgentExec)
	argNameAgentID           = "agent"
)

func (n ArgName) String() string {
	return string(n)
}

func GetConfig() (*Config, error) {
	c := &Config{
		Version: getVersion(),
	}
	flag.StringVar(&c.Connect, "connect", "wss://localhost:8443", "Connection string")
	flag.StringVar(&c.AgentID, argNameAgentID, "", "Agent ID for connection to server")
	flag.StringVar(&c.Command, "command", "", `Command to service control (not required):
  install - install the service to the system
  uninstall - uninstall the service from the system
  start - start the service
  stop - stop the service
  status - status of the service`)
	flag.StringVar(&c.LogDir, "logdir", "", "System option to define log directory to vxagent")
	flag.BoolVar(&c.Debug, "debug", false, "System option to run vxagent in debug mode")
	flag.BoolVar(&c.Service, "service", false, "System option to run vxagent as a service")
	flag.BoolVar(&c.PrintVersion, "version", false, "Print current version of vxagent and exit")
	flag.StringVar(&c.runningMode, argNameMode, string(RunningModeAgent), `Running mode:
agent - normal agent mode
upgrader - upgrader mode, used for the agent upgrade`)
	flag.StringVar(&c.ppidArg, argNamePPID, "", "Upgrader parent process ID")
	flag.StringVar(&c.AgentExecutablePath, argNameAgentExec, "", "Path to the agent's executable file")
	flag.Parse()

	if unknownArgs := flag.Args(); len(unknownArgs) != 0 {
		return nil, fmt.Errorf(
			"unknown arguments have been passed: %v (use '-help' to get information on valid flags)",
			unknownArgs)
	}

	if connect, ok := os.LookupEnv("CONNECT"); ok {
		c.Connect = connect
	}
	if agentID, ok := os.LookupEnv("AGENT_ID"); ok {
		c.AgentID = agentID
	}
	if logDir, ok := os.LookupEnv("LOG_DIR"); ok {
		c.LogDir = logDir
	}

	if os.Getenv("DEBUG") != "" {
		c.Debug = true
	}
	if err := checkConfig(c); err != nil {
		return nil, err
	}
	return c, nil
}

func getVersion() string {
	version := "develop"
	if PackageVer != "" {
		version = PackageVer
	}
	if PackageRev != "" {
		version += "-" + PackageRev
	}
	return version
}

func checkConfig(c *Config) error {
	if err := checkConfigCommand(c.Command); err != nil {
		return err
	}
	if err := checkRunningMode(c); err != nil {
		return err
	}
	if err := checkOrSetLogDir(c); err != nil {
		return err
	}
	if c.Mode == RunningModeUpgrader {
		if err := checkArgsForUpgraderMode(c); err != nil {
			return err
		}
	}
	return nil
}

const startCmd = "start"

func checkConfigCommand(configCommand string) error {
	switch configCommand {
	case "install":
	case "uninstall":
	case startCmd:
	case "stop":
	case "status":
	case "":
	default:
		return fmt.Errorf("invalid value of 'command' argument: %s", configCommand)
	}
	return nil
}

type RunningMode string

func (m RunningMode) String() string {
	return string(m)
}

const (
	RunningModeAgent    RunningMode = "agent"
	RunningModeUpgrader RunningMode = "upgrader"
)

func checkRunningMode(c *Config) error {
	switch c.runningMode {
	case string(RunningModeAgent):
		c.Mode = RunningModeAgent
	case string(RunningModeUpgrader):
		c.Mode = RunningModeUpgrader
	default:
		return fmt.Errorf("invalid value of the 'mode' argument: %s", c.runningMode)
	}
	return nil
}

func checkOrSetLogDir(c *Config) error {
	if c.LogDir == "" {
		switch c.Mode {
		case RunningModeAgent:
			execPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("failed to get the current executable path: %w", err)
			}
			c.LogDir = filepath.Dir(execPath)
		case RunningModeUpgrader:
			c.LogDir = filepath.Dir(c.AgentExecutablePath)
		default:
			return fmt.Errorf("invalid running mode: %s", c.Mode)
		}
	}
	var err error
	c.LogDir, err = filepath.Abs(c.LogDir)
	if err != nil {
		return fmt.Errorf("invalid value of 'logdir' argument: %s", c.LogDir)
	}
	return nil
}

func GetDefaultLogDir(agentPath string) string {
	return filepath.Dir(agentPath)
}

func checkArgsForUpgraderMode(c *Config) error {
	if len(c.AgentExecutablePath) == 0 {
		return fmt.Errorf(genArgNotSetErrMsg(argNameAgentExec))
	}
	if err := checkPPID(c); err != nil {
		return err
	}
	return nil
}

func checkPPID(c *Config) error {
	if len(c.ppidArg) == 0 {
		return fmt.Errorf(genArgNotSetErrMsg(argNamePPID))
	}
	var err error
	c.PPID, err = strconv.Atoi(c.ppidArg)
	if err != nil {
		return fmt.Errorf("%s is not a valid PPID", c.ppidArg)
	}
	return nil
}

func genArgNotSetErrMsg(field string) string {
	return fmt.Sprintf("param %s must be specified", field)
}

func ChangeOrSetArg(args []string, argName ArgName, argValue string) []string {
	argChanged := false
	argFlag := fmt.Sprintf("-%s", argName)
	argDoubleMinusFlag := fmt.Sprintf("--%s", argName)
	argToAdd := fmt.Sprintf("%s=%s", argFlag, argValue)
	for i, a := range args {
		if a == argFlag || a == argDoubleMinusFlag {
			args[i+1] = argValue
			argChanged = true
			break
		}
		if strings.HasPrefix(a, argFlag) || strings.HasPrefix(a, argDoubleMinusFlag) {
			args[i] = argToAdd
			argChanged = true
			continue
		}
	}
	if argChanged {
		return args
	}
	return append(args, argToAdd)
}

func IfServiceAddStartFlag(args []string) []string {
	for i, a := range args {
		if a != "-service" {
			continue
		}
		args = append(append(args[:i], "-command", startCmd), args[i+1:]...)
		break
	}
	return args
}

func ChangeOrSetArgs(args []string, argsToSet map[ArgName]string) []string {
	for aName, aVal := range argsToSet {
		args = ChangeOrSetArg(args, aName, aVal)
	}
	return args
}
