package ainfo

import (
	"context"
	"fmt"

	"github.com/jinzhu/gorm"

	"soldr/internal/app/api/models"
	"soldr/internal/app/server/mmodule/hardening/cache"
	"soldr/internal/errors"
	"soldr/internal/vxproto"
)

type AgentInfoFetcher struct {
	storeConn *cache.Connector
}

func NewAgentInfoFetcher(store interface{}) (*AgentInfoFetcher, error) {
	storeConn, err := cache.NewConnector(store, &cache.ConnectorParams{
		DBFetcher:   fetchAgentInfoFromDB,
		FileFetcher: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a store connector: %w", err)
	}
	return &AgentInfoFetcher{
		storeConn: storeConn,
	}, nil
}

func (f *AgentInfoFetcher) GetAgentConnectionInfo(
	ctx context.Context,
	info *vxproto.AgentInfoForIDFetcher,
) (*vxproto.AgentConnectionInfo, error) {
	if info == nil {
		return nil, fmt.Errorf("passed info object is nil")
	}
	ctx = context.WithValue(ctx, ctxKeyAgentID, info.ID)
	connInfoIface, err := f.storeConn.Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the agent connection info: %w", err)
	}
	connInfo, ok := connInfoIface.(*vxproto.AgentConnectionInfo)
	if !ok {
		return nil, fmt.Errorf("fetched object is not of the type *vxproto.AgentConnectionInfo")
	}
	return connInfo, nil
}

type ctxKey int

const (
	ctxKeyAgentID ctxKey = iota + 1
)

func fetchAgentInfoFromDB(ctx context.Context, store *gorm.DB) (interface{}, error) {
	agentID, ok := ctx.Value(ctxKeyAgentID).(string)
	if !ok {
		return nil, fmt.Errorf("not agent ID found in the passed context")
	}
	agent := &models.Agent{}

	// TODO(SSH): we should use the context passed to the function in this operation.
	// We can either:
	// 1. user gorm v2 (as it supports passing context to operations), or
	// 2. use a transaction with BeginTx (we used it before and it caused connection lags,
	// 		we should use a non-restrictive isolation level if we decide to bring it back)
	err := store.
		Select("group_id, auth_status").
		Where("hash = ?", agentID).
		First(&agent).
		Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, fmt.Errorf("agent %s not found in the DB: %w", agentID, errors.ErrRecordNotFound)
		}
		return nil, fmt.Errorf("failed to fetch the agent %s from the DB: %w", agentID, err)
	}
	info := &vxproto.AgentConnectionInfo{
		ID:         agentID,
		GroupID:    agent.GroupID,
		AuthStatus: agent.AuthStatus,
	}
	return info, nil
}
