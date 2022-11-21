package types

import (
	"fmt"
)

type AgentBinaryID struct {
	Version string `json:"version"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

func (id *AgentBinaryID) String() string {
	return fmt.Sprintf("%s/%s/%s", id.Version, id.OS, id.Arch)
}

type ABHList map[AgentBinaryID][]byte
