package system

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"runtime"

	"soldr/pkg/protoagent"
	"soldr/pkg/utils"
)

type AgentInfoGetter interface {
	Get(ctx context.Context) (*protoagent.Information, error)
}

func GetAgentInfo(ctx context.Context) (*protoagent.Information, error) {
	agentInfo := make(chan *protoagent.Information, 1)
	go runFetcher(agentInfo)
	select {
	case info := <-agentInfo:
		return info, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func GetAgentInfoAsync() AgentInfoGetter {
	return newAgentInfoGetter()
}

type agentInfoGetter struct {
	agentInfo <-chan *protoagent.Information
}

func newAgentInfoGetter() *agentInfoGetter {
	agentInfo := make(chan *protoagent.Information, 1)
	go runFetcher(agentInfo)
	return &agentInfoGetter{
		agentInfo: agentInfo,
	}
}

func runFetcher(resp chan<- *protoagent.Information) {
	resp <- getAgentInformation()
	close(resp)
}

func (a *agentInfoGetter) Get(ctx context.Context) (*protoagent.Information, error) {
	select {
	case info, ok := <-a.agentInfo:
		if !ok {
			return nil, fmt.Errorf("cannot get the value twice")
		}
		return info, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("context closed: %w", ctx.Err())
	}
}

func MakeAgentID() string {
	salt := "a1e2d4af50a3c3fe0fd8abfd91f9fa7636622b667"
	id, err := getMachineID()
	if err != nil || id == "" {
		id = getHostname() + ":" + id
	}
	hash := md5.Sum([]byte(id + salt))
	return hex.EncodeToString(hash[:])
}

func getAgentInformation() *protoagent.Information {
	infoMessage := &protoagent.Information{
		Os: &protoagent.Information_OS{
			Type: utils.GetRef(runtime.GOOS),
			Name: utils.GetRef(getOSName() + " " + getOSVer()),
			Arch: utils.GetRef(runtime.GOARCH),
		},
		Net: &protoagent.Information_Net{
			Hostname: utils.GetRef(getHostname()),
			Ips:      getIPs(),
		},
		Users: getUsersInformation(),
	}

	return infoMessage
}
