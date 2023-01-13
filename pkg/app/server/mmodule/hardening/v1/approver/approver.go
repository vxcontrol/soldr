package approver

import (
	"context"
)

type Approver interface {
	WaitForAuth(ctx context.Context, agentID string) error
}
