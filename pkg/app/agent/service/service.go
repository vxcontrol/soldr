package service

import (
	"fmt"
	"os"
	"runtime"

	"github.com/takama/daemon"
)

const (
	svcName        = "vxagent"
	svcDescription = "VXAgent service to the OS control"
)

func Initialize() (daemon.Daemon, error) {
	svc, err := daemon.New(svcName, svcDescription, getServiceKind(), getServiceDependencies()...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a new service: %w", err)
	}
	return svc, nil
}

func getServiceKind() daemon.Kind {
	if runtime.GOOS != "darwin" {
		return daemon.SystemDaemon
	}
	if os.Getuid() == 0 {
		return daemon.GlobalDaemon
	}
	return daemon.UserAgent
}

func getServiceDependencies() []string {
	if runtime.GOOS == "linux" {
		return []string{"multi-user.target", "sockets.target"}
	}
	if runtime.GOOS == "windows" {
		return []string{"tcpip"}
	}
	return nil
}
