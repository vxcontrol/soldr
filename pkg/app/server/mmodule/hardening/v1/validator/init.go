// nolint: goconst
package validator

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"soldr/pkg/app/agent"
	"soldr/pkg/app/api/models"
	"soldr/pkg/app/server/mmodule/hardening/v1/abher/types"
	utilsErrors "soldr/pkg/utils/errors"
	"soldr/pkg/vxproto"
)

func (v *ConnectionValidator) CheckInitConnectionTLS(s *tls.ConnectionState) error {
	if s == nil {
		return nil
	}
	if err := checkInitConnCertificates(s.VerifiedChains); err != nil {
		return fmt.Errorf("initial connection certificates check has failed: %w", err)
	}
	return nil
}

func (v *ConnectionValidator) OnInitConnect(
	ctx context.Context,
	tlsConnState *tls.ConnectionState,
	socket vxproto.SyncWS,
	info *vxproto.InitConnectionInfo,
) error {
	initConnectCtx, cancelInitConnectCtx := context.WithCancel(ctx)
	defer cancelInitConnectCtx()
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-socket.Done():
			cancelInitConnectCtx()
		}
	}()
	req, err := getInitConnectRequest(initConnectCtx, socket)
	if err != nil {
		return fmt.Errorf("failed to get the init connection request: %w", err)
	}

	logger := logrus.WithFields(logrus.Fields{
		"component": "init_conn_validator",
		"module":    "main",
	})
	logger.Infof("agent %s is performing its initial connection", string(req.AgentID))

	resp := &agent.InitConnectionResponse{}
	resp.Ltac, err = v.processInitConnectRequest(initConnectCtx, tlsConnState, req, info, logger)
	if err != nil {
		logger.WithError(err).Error("failed to process the init connection request")
		return fmt.Errorf("failed to process the init connection request: %w", err)
	}
	resp.Sbh, err = v.sbher.Get(initConnectCtx, tlsConnState.ServerName)
	if err != nil {
		logger.WithError(err).Error("failed to get the SBH value")
		return fmt.Errorf("failed to get the SBH value: %w", err)
	}
	resp.Ssa, err = v.ssaGenerator.GenerateSSAScript(req.AgentID)
	if err != nil {
		logger.WithError(err).Error("failed to generate the SSA script")
		return fmt.Errorf("failed to generate the SSA script: %w", err)
	}
	if err := sendInitConnectResponse(ctx, socket, resp); err != nil {
		logger.WithError(err).Error("failed to send the init connection response")
		return fmt.Errorf("failed to send the init connection response: %w", err)
	}
	return nil
}

const (
	initRequestTimeout            = time.Second * 60
	syncConnectedDateForInitAgent = time.Second * 60
)

func getInitConnectRequest(ctx context.Context, ws vxproto.SyncWS) (*agent.InitConnectionRequest, error) {
	ctx, cancelCtx := context.WithTimeout(ctx, initRequestTimeout)
	defer cancelCtx()
	msg, err := ws.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read the init connect request: %w", err)
	}
	var initConnectReq agent.InitConnectionRequest
	if err := agent.UnpackProtoMessage(&initConnectReq, msg, agent.Message_INIT_CONNECTION); err != nil {
		return nil, fmt.Errorf("failed to unpack the init connect message: %w", err)
	}
	return &initConnectReq, nil
}

func (v *ConnectionValidator) processInitConnectRequest(
	ctx context.Context,
	tlsConnState *tls.ConnectionState,
	req *agent.InitConnectionRequest,
	info *vxproto.InitConnectionInfo,
	logger *logrus.Entry,
) ([]byte, error) {
	if err := v.validateInitConnection(ctx, &initConnectionValidationPack{
		AgentID: &types.AgentBinaryID{
			OS:      req.GetAgentBinaryID().GetOs(),
			Arch:    req.GetAgentBinaryID().GetArch(),
			Version: req.GetAgentBinaryID().GetVersion(),
		},
		ABH:       req.Abh,
		AgentType: vxproto.VXAgent,
	}); err != nil {
		return nil, fmt.Errorf("initial connection validation failed: %w", err)
	}
	agentID := string(req.GetAgentID())
	if err := v.upsertAgent(ctx, agentID, req.GetAgentBinaryID().GetVersion(), req.GetInfo(), info); err != nil {
		return nil, fmt.Errorf("upsert failed: %w", err)
	}
	defer func() {
		logger.Debug("disconnecting agent")
		if err := v.disconnectAgent(ctx, agentID); err != nil {
			logger.WithError(err).Error("failed to set the agent status to 'disconnected'")
		}
	}()
	go v.updateAgentConnectDate(ctx, agentID, logger)

	logger.Debug("waiting for agent authentication")
	if err := v.approver.WaitForAuth(ctx, string(req.AgentID)); err != nil {
		return nil, fmt.Errorf("approver returned an error: %w", err)
	}
	ltac, err := v.certsProvider.CreateLTACFromCSR(tlsConnState, req.Csr)
	if err != nil {
		return nil, fmt.Errorf("failed to generate an LTAC cert using the passed CSR: %w", err)
	}
	return ltac, nil
}

func gormExprNow() *gorm.SqlExpr {
	return gorm.Expr("NOW()")
}

func (v *ConnectionValidator) updateAgentConnectDate(ctx context.Context, agentID string, logger *logrus.Entry) {
	if v.gdbc == nil {
		return
	}

	timer := time.NewTicker(syncConnectedDateForInitAgent)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			err := v.gdbc.
				Model(&models.Agent{}).
				Where("hash = ?", agentID).
				UpdateColumns(map[string]interface{}{
					"connected_date": gormExprNow(),
				}).Error
			if err != nil {
				logger.WithError(err).WithFields(logrus.Fields{
					"agent_id": agentID,
				}).Error("failed to update the agent connected date")
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

type initConnectionValidationPack struct {
	AgentID   *types.AgentBinaryID
	ABH       []byte
	AgentType vxproto.AgentType
}

func (v *ConnectionValidator) validateInitConnection(ctx context.Context, p *initConnectionValidationPack) error {
	if err := v.validateABH(ctx, p.AgentType, p.AgentID, p.ABH); err != nil {
		return fmt.Errorf("ABH validation failed: %w", err)
	}
	return nil
}

func (v *ConnectionValidator) validateABH(
	_ context.Context,
	agentType vxproto.AgentType,
	agentID *types.AgentBinaryID,
	actualABH []byte,
) error {
	expectedABH, err := v.abher.GetABH(agentType, agentID)
	if err != nil {
		return fmt.Errorf("failed to get the ABH for the agent's binary %s: %w", agentID.String(), err)
	}
	if !bytes.Equal(expectedABH, actualABH) {
		return fmt.Errorf(
			"passed ABH (%s) and expected ABH (%s) are different",
			base64.StdEncoding.EncodeToString(actualABH),
			base64.StdEncoding.EncodeToString(expectedABH),
		)
	}
	return nil
}

func sendInitConnectResponse(ctx context.Context, ws vxproto.SyncWS, resp *agent.InitConnectionResponse) error {
	respMsg, err := agent.PackProtoMessage(resp, agent.Message_INIT_CONNECTION)
	if err != nil {
		return fmt.Errorf("failed to pack the init connection response: %w", err)
	}
	if err := ws.Write(ctx, respMsg); err != nil {
		return fmt.Errorf("failed to send the init connection response: %w", err)
	}
	return nil
}

func checkInitConnCertificates(verifiedChains [][]*x509.Certificate) error {
	if len(verifiedChains) != 1 && len(verifiedChains[0]) != 1 {
		return fmt.Errorf("expected a verified chain of length 1, got %d", len(verifiedChains))
	}
	iac := verifiedChains[0][0]
	const iacCN = "IAC Cert"
	if iac.Subject.CommonName != iacCN {
		return fmt.Errorf("expected the IAC certificate CN to be \"%s\", got \"%s\"", iacCN, iac.Subject.CommonName)
	}
	return nil
}

func (v *ConnectionValidator) upsertAgent(
	ctx context.Context,
	agentID string,
	version string,
	info *agent.Information,
	connInfo *vxproto.InitConnectionInfo,
) error {
	if v.gdbc == nil {
		return nil
	}
	host, _, err := net.SplitHostPort(connInfo.IP)
	if err != nil {
		return fmt.Errorf("failed to split the passed IP address into host and port: %w", err)
	}
	descData := []string{
		host,
		info.Os.GetType(),
		info.Net.GetHostname(),
		agentID[:6],
	}
	agentUpdateMap := map[string]interface{}{
		"status":         "connected",
		"connected_date": gormExprNow(),
		"updated_at":     gormExprNow(),
	}
	desc := strings.Join(descData, "_")
	a := models.Agent{
		Hash:        agentID,
		GroupID:     0,
		IP:          host,
		Description: desc,
		Version:     version,
		Info: models.AgentInfo{
			OS: models.AgentOS{
				Type: info.Os.GetType(),
				Arch: info.Os.GetArch(),
				Name: info.Os.GetName(),
			},
			Net: models.AgentNet{
				Hostname: info.Net.GetHostname(),
				IPs:      info.Net.GetIps(),
			},
			Tags:  []string{},
			Users: getAgentUsers(info),
		},
		Status:        "connected",
		AuthStatus:    "unauthorized",
		ConnectedDate: time.Now(),
	}
	if err := a.Valid(); err != nil {
		return fmt.Errorf("created agent structure is not valid: %w", err)
	}

	agentScope := func(db *gorm.DB) *gorm.DB {
		return db.Where("hash = ?", a.Hash)
	}
	db := v.gdbc.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	}).Scopes(agentScope)
	defer func() {
		if err != nil {
			if rollbackErr := db.Rollback().Error; rollbackErr != nil {
				err = fmt.Errorf("failed to rollback transaction (%v), rollback reason: %w", rollbackErr, err)
			}
			return
		}
		if commitErr := db.Commit().Error; commitErr != nil {
			err = fmt.Errorf("failed to commit transaction: %w", err)
		}
	}()
	var existingAgent models.Agent
	if err := db.First(&existingAgent).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			if err := db.Create(&a).Error; err != nil {
				return fmt.Errorf("failed to create a new agent entry: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get the agent %s: %w", a.Hash, err)
	}

	if existingAgent.AuthStatus == "blocked" {
		return fmt.Errorf("upserting agent %s failed: %w", a.Hash, utilsErrors.ErrFailedResponseBlocked)
	}
	if err := db.Model(&a).UpdateColumns(agentUpdateMap).Error; err != nil {
		return fmt.Errorf("failed to update the agent status to connected: %w", err)
	}
	return nil
}

func getAgentUsers(info *agent.Information) []models.AgentUser {
	agentInfoUsers := info.GetUsers()
	agentUsers := make([]models.AgentUser, len(agentInfoUsers))
	for _, u := range agentInfoUsers {
		agentUsers = append(agentUsers, models.AgentUser{
			Name:   u.GetName(),
			Groups: u.GetGroups(),
		})
	}
	return agentUsers
}

func (v *ConnectionValidator) disconnectAgent(_ context.Context, agentID string) error {
	err := v.gdbc.
		Model(&models.Agent{}).
		Where("hash = ?", agentID).
		UpdateColumns(map[string]interface{}{
			"status":     "disconnected",
			"updated_at": gormExprNow(),
		}).Error
	if err != nil {
		return fmt.Errorf("failed to set the agent status to disconnected: %w", err)
	}
	return nil
}
