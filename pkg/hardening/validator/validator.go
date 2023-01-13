package hardening

import (
	"errors"

	"soldr/pkg/hardening/luavm/vm"
	utilsErrors "soldr/pkg/utils/errors"
)

type Validator struct {
	vm                  vm.VM
	agentID             string
	version             string
	connProtocolVersion string
}

func NewValidator(agentID string, version string, luaVM vm.VM) *Validator {
	return &Validator{
		vm:      luaVM,
		agentID: agentID,
		version: version,
	}
}

func IsAgentUnauthorizedErr(err error) bool {
	return errors.Is(err, utilsErrors.ErrFailedResponseUnauthorized) ||
		errors.Is(err, utilsErrors.ErrFailedResponseBlocked) ||
		errors.Is(err, utilsErrors.ErrFailedResponseCorrupted) ||
		errors.Is(err, utilsErrors.ErrFailedResponseTunnelError)
}
