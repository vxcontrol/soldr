package mmodule

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/vxcontrol/luar"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"

	"soldr/internal/app/api/models"
	"soldr/internal/app/server/certs"
	"soldr/internal/app/server/mmodule/hardening"
	hardeningConfig "soldr/internal/app/server/mmodule/hardening/config"
	hardeningUtils "soldr/internal/app/server/mmodule/hardening/utils"
	"soldr/internal/app/server/mmodule/hardening/v1/crypto"
	"soldr/internal/app/server/mmodule/upgrader/store"
	"soldr/internal/controller"
	"soldr/internal/loader"
	"soldr/internal/lua"
	obs "soldr/internal/observability"
	"soldr/internal/protoagent"
	"soldr/internal/storage"
	"soldr/internal/system"
	utilsErrors "soldr/internal/utils/errors"
	"soldr/internal/vxproto"
)

const (
	// publishEventsInterval is a time period to store received events into DB
	publishEventsInterval = 1 * time.Second
	// syncGroupsInterval is a time period to retrieve agents changes from DB
	syncAgentsInterval = 20 * time.Second
	// syncGroupsInterval is a time period to retrieve grpups from DB
	syncGroupsInterval = 5 * time.Second
	// syncAliveStatusBatchSize is an amount agents ID which will using in SQL update request
	syncAliveStatusBatchSize = 100
	// lostAgentDeltaTime is amount minutes which agent was not update its connected_date
	lostAgentDeltaTime = 10
	// publishEventsQueueLimit is maximum amount events which keeped in RAM
	publishEventsQueueLimit = 100
)

// MainModule is struct which contains full state for agent working
type MainModule struct {
	proto                     vxproto.IVXProto
	cnt                       controller.IController
	gdbc                      *gorm.DB
	validator                 *validator.Validate
	store                     storage.IStorage
	agents                    *agentList
	groups                    *groupList
	listen                    string
	version                   string
	eventsQueue               chan *models.Event
	msocket                   vxproto.IModuleSocket
	wgControl                 sync.WaitGroup
	wgReceiver                sync.WaitGroup
	wgExchAgent               sync.WaitGroup
	mutexAgent                *sync.Mutex
	isCloseState              bool
	tracerClient              otlptrace.Client
	meterClient               otlpmetric.Client
	quitSyncAgents            chan struct{}
	quitSyncGroups            chan struct{}
	upgradeTaskConsumer       *upgradeTaskConsumer
	cancelEventsPublisher     context.CancelFunc
	cancelUpgradeTaskConsumer context.CancelFunc
	certsProvider             certs.Provider
	authenticator             *Authenticator
	connValidatorFactory      *hardening.ConnectionValidatorFactory
	moduleConfigDecryptor     *crypto.ConfigDecryptor

	cancelContext context.CancelFunc
}

type AgentInfoDB struct {
	GID        string `gorm:"column:gid"`
	AuthStatus string `gorm:"column:auth_status"`
	DeletedAt  *time.Time
}

type GroupInfoDB struct {
	GID       string `gorm:"column:gid"`
	DeletedAt *time.Time
}

type AgentBinary struct {
	AgentHash string       `gorm:"column:agent_hash"`
	AgentInfo *AgentInfoDB `gorm:"embedded"`
}

type agentInfo struct {
	info    *vxproto.AgentInfo
	isfin   bool
	quit    chan struct{}
	update  chan struct{}
	upgrade chan *store.Task
	done    chan struct{}
	mxdone  sync.Mutex
}

type agentList struct {
	list  map[string]*agentInfo
	mutex *sync.Mutex
}

func (agents *agentList) dump() map[string]*agentInfo {
	agents.mutex.Lock()
	defer agents.mutex.Unlock()

	nlist := make(map[string]*agentInfo)
	for token, ainfo := range agents.list {
		nlist[token] = ainfo
	}
	return nlist
}

func (agents *agentList) dumpID(id string) map[string]*agentInfo {
	agents.mutex.Lock()
	defer agents.mutex.Unlock()

	nlist := make(map[string]*agentInfo)
	for token, ainfo := range agents.list {
		if ainfo.info.ID == id {
			nlist[token] = ainfo
		}
	}

	return nlist
}

func (agents *agentList) add(token string, ainfo *agentInfo) {
	agents.mutex.Lock()
	defer agents.mutex.Unlock()

	agents.list[token] = ainfo
}

func (agents *agentList) get(token string) *agentInfo {
	agents.mutex.Lock()
	defer agents.mutex.Unlock()

	if ainfo, ok := agents.list[token]; ok {
		return ainfo
	}

	return nil
}

func (agents *agentList) del(token string) {
	agents.mutex.Lock()
	defer agents.mutex.Unlock()

	if ainfo, ok := agents.list[token]; ok {
		ainfo.isfin = true
	}
	delete(agents.list, token)
}

type groupInfo struct {
	isfin  bool
	quit   chan bool
	update chan struct{}
	done   chan struct{}
	mxdone sync.Mutex
}

type groupList struct {
	list  map[string]*groupInfo
	mutex *sync.Mutex
}

func (groups *groupList) dump() map[string]*groupInfo {
	groups.mutex.Lock()
	defer groups.mutex.Unlock()

	nlist := make(map[string]*groupInfo)
	for token, ginfo := range groups.list {
		nlist[token] = ginfo
	}
	return nlist
}

func (groups *groupList) add(gid string, ginfo *groupInfo) {
	groups.mutex.Lock()
	defer groups.mutex.Unlock()

	groups.list[gid] = ginfo
}

func (groups *groupList) get(gid string) *groupInfo {
	groups.mutex.Lock()
	defer groups.mutex.Unlock()

	if ginfo, ok := groups.list[gid]; ok {
		return ginfo
	}

	return nil
}

func (groups *groupList) del(gid string) {
	groups.mutex.Lock()
	defer groups.mutex.Unlock()

	if ginfo, ok := groups.list[gid]; ok {
		ginfo.isfin = true
	}
	delete(groups.list, gid)
}

// GetVersion is function that return of server version
func (mm *MainModule) GetVersion() string {
	return mm.version
}

const sqlNowFunction = "NOW()"
const (
	dbAgentStatusAuthorized   = "authorized"
	dbAgentStatusUnauthorized = "unauthorized"
)

// HasAgentInfoValid is function that validate Agent Information in Agent list
func (mm *MainModule) HasAgentInfoValid(ctx context.Context, asocket vxproto.IAgentSocket) error {
	if mm.isCloseState {
		return fmt.Errorf("server is stopping")
	}

	validCtx, validSpan := obs.Observer.NewSpan(ctx, obs.SpanKindInternal, "conn_validator")
	defer validSpan.End()

	info := asocket.GetPublicInfo()
	log := logrus.WithContext(validCtx).WithFields(logrus.Fields{
		"component": "conn_validator",
		"module":    "main",
		"id":        info.ID,
		"type":      info.Type.String(),
		"ver":       info.Ver,
		"info":      info.Info.String(),
	})
	if mm.gdbc == nil {
		if info.ID == "" {
			log.Error("can't connect to the DB")
			return utilsErrors.ErrFailedResponseCorrupted
		}
		// TODO: here need to check agent group and policy by config file
		// and set this property if config contains defined value to the agent
		return nil
	}

	if err := hardeningUtils.IsAgentIDValid(info.ID); err != nil {
		log.Error(err)
		return utilsErrors.ErrFailedResponseCorrupted
	}

	if info.Type == vxproto.VXAgent && !checkAgentInfo(info.Info) {
		log.Error("failed to validate agent Info")
		return utilsErrors.ErrFailedResponseCorrupted
	}

	if info.Type == vxproto.Aggregate {
		ginfodb := mm.getGroupInfoDB(validCtx, info.ID)
		if ginfodb != nil {
			asocket.SetGroupID(ginfodb.GID)
			return nil
		}
		log.Errorf("requested group '%s' is not exist in DB", info.ID)
		return utilsErrors.ErrFailedResponseCorrupted
	}

	ainfodb := mm.getAgentInfoDB(validCtx, info.ID)
	switch {
	case ainfodb == nil:
		if info.Type != vxproto.VXAgent {
			log.Errorf("agent '%s' not exist in DB", info.Type)
			return utilsErrors.ErrFailedResponseCorrupted
		}

		host, _, err := net.SplitHostPort(info.IP)
		if err != nil {
			log.WithError(err).Error("failed to split host and port")
			return utilsErrors.ErrFailedResponseCorrupted
		}

		descData := []string{
			host,
			info.Info.Os.GetType(),
			info.Info.Net.GetHostname(),
			info.ID[:6],
		}
		desc := strings.Join(descData, "_")

		if err := mm.createAgent(info.ID, host, desc, info.Ver, info.Info); err != nil {
			log.WithError(err).Error("failed to create agent into DB")
			return utilsErrors.ErrFailedResponseCorrupted
		}
		log.Debug("agent was created with unauthorized status")
		return utilsErrors.ErrFailedResponseUnauthorized
	case ainfodb.AuthStatus != dbAgentStatusAuthorized:
		log.Error("agent auth_status is not authorized")
		return fmt.Errorf(ainfodb.AuthStatus)
	default:
		asocket.SetGroupID(ainfodb.GID)
	}

	if info.Type == vxproto.VXAgent {
		// Deny second connection from VXAgent with the same agent ID
		for _, ainfo := range mm.agents.dumpID(info.ID) {
			if ainfo.info.Type == vxproto.VXAgent {
				log.Error("deny second connection from other agent")
				return utilsErrors.ErrFailedResponseAlreadyConnected
			}
		}

		jinfo, err := json.Marshal(info.Info)
		if err != nil {
			log.WithError(err).Error("failed to marshal agent Info to store")
			return utilsErrors.ErrFailedResponseCorrupted
		}
		agentUpdateMap, err := getUpdateMap(
			map[string]*struct {
				Value interface{}
				Rule  string
			}{
				"info": {
					Value: gorm.Expr("JSON_MERGE_PATCH(`info`, ?)", jinfo),
				},
				"version": {
					Value: info.Ver,
					Rule:  "max=20,required",
				},
				"updated_at": {
					Value: gorm.Expr(sqlNowFunction),
				},
			},
			mm.validator,
		)
		if err != nil {
			log.WithError(err).Error("failed to validate agent's update values")
			return utilsErrors.ErrFailedResponseCorrupted
		}
		err = mm.gdbc.
			Scopes(agentWithHash(info.ID)).
			UpdateColumns(agentUpdateMap).
			Error
		if err != nil {
			log.WithError(err).Error("failed to set agent Info into DB")
			return utilsErrors.ErrFailedResponseCorrupted
		}
	}

	return nil
}

func checkAgentInfo(info *protoagent.Information) bool {
	switch info.GetOs().GetType() {
	case "linux":
	case "windows":
	case "darwin":
	default:
		return false
	}

	switch info.GetOs().GetArch() {
	case "amd64":
	case "386":
	default:
		return false
	}

	return true
}

type updateMapWithRules map[string]*struct {
	Value interface{}
	Rule  string
}

func getUpdateMap(m updateMapWithRules, v *validator.Validate) (map[string]interface{}, error) {
	updateMap := make(map[string]interface{}, len(m))
	rules := make(map[string]interface{}, len(m))
	for key, entry := range m {
		updateMap[key] = entry.Value
		rules[key] = entry.Rule
	}
	if errs := v.ValidateMap(updateMap, rules); len(errs) != 0 {
		return nil, fmt.Errorf("update agent values validation failed: %w", generateErrFromValidationErrs(errs))
	}
	return updateMap, nil
}

func (mm *MainModule) createAgent(
	hash string,
	ip string,
	description string,
	version string,
	agentInfo *protoagent.Information,
) error {
	agentInfoUsers := agentInfo.GetUsers()
	agentUsers := make([]models.AgentUser, len(agentInfoUsers))
	for i, u := range agentInfoUsers {
		agentUsers[i] = models.AgentUser{
			Name:   u.GetName(),
			Groups: u.GetGroups(),
		}
	}
	a := models.Agent{
		Hash:        hash,
		GroupID:     0,
		IP:          ip,
		Description: description,
		Version:     version,
		Info: models.AgentInfo{
			OS: models.AgentOS{
				Type: agentInfo.Os.GetType(),
				Arch: agentInfo.Os.GetArch(),
				Name: agentInfo.Os.GetName(),
			},
			Net: models.AgentNet{
				Hostname: agentInfo.Net.GetHostname(),
				IPs:      agentInfo.Net.GetIps(),
			},
			Users: agentUsers,
			Tags:  []string{},
		},
		Status:     "disconnected",
		AuthStatus: dbAgentStatusUnauthorized,
	}
	if err := mm.gdbc.Create(&a).Error; err != nil {
		return fmt.Errorf("failed to create a new agent: %w", err)
	}
	return nil
}

// RegisterLuaAPI is function that registrate extra API function for each of type service
func (mm *MainModule) RegisterLuaAPI(state *lua.State, config *loader.ModuleConfig) error {
	gid := config.GroupID
	pid := config.PolicyID
	mname := config.Name

	luar.Register(state.L, "__api", luar.Map{
		"push_event": func(aid, info string) bool {
			ctx := context.Background()
			eventCtx, eventSpan := obs.Observer.NewSpan(ctx, obs.SpanKindInternal, "push_event")
			defer eventSpan.End()
			return mm.pushEventToQueue(eventCtx, aid, gid, pid, mname, info)
		},
	})
	luar.Register(state.L, "__sec", luar.Map{
		"get": func(key string) (string, bool) {
			cfg := config.IConfigItem.GetSecureCurrentConfig()
			b, err := mm.moduleConfigDecryptor.Decrypt(cfg)
			if err != nil {
				return "", false
			}

			var model models.ModuleSecureConfig
			err = json.Unmarshal([]byte(b), &model)
			if err != nil {
				return "", false
			}

			val, ok := model[key]
			if !ok {
				return "", false
			}

			result, err := json.Marshal(val.Value)
			if err != nil {
				return "", false
			}

			return string(result), true
		},
	})
	luar.GoToLua(state.L, mm.version)
	state.L.SetGlobal("__version")
	luar.GoToLua(state.L, gid)
	state.L.SetGlobal("__gid")
	luar.GoToLua(state.L, pid)
	state.L.SetGlobal("__pid")

	getBaseFields := func() logrus.Fields {
		return logrus.Fields{
			"group_id":    gid,
			"policy_id":   pid,
			"module_name": mname,
		}
	}
	if err := state.RegisterLogger(logrus.GetLevel(), getBaseFields()); err != nil {
		return fmt.Errorf("failed to register logging functions: %w", err)
	}
	fields := getBaseFields()
	fields["server_id"] = system.MakeAgentID()
	if err := state.RegisterMeter(fields); err != nil {
		return fmt.Errorf("failed to register metrics gathering functions: %w", err)
	}

	return nil
}

func (mm *MainModule) getAgentIDByHash(hash string, gid string) (uint64, error) {
	var agentIDs []uint64
	subQueryGroups := mm.gdbc.
		Model(&models.Group{}).
		Select("groups.id").
		Where("groups.hash = ?", gid).
		Limit(1).
		QueryExpr()
	result := mm.gdbc.
		Model(&models.Agent{}).
		Where("agents.hash = ? AND agents.group_id IN (0, (?))", hash, subQueryGroups).
		Limit(1).
		Pluck("agents.id", &agentIDs)
	if result.Error != nil {
		return 0, fmt.Errorf(
			"failed to get the agent ID by hash %s (group ID %s): %w",
			hash, gid,
			result.Error,
		)
	}
	if len(agentIDs) == 0 {
		return 0, fmt.Errorf(
			"failed to get the agent ID by hash %s (group ID %s): no agent ID found",
			hash, gid,
		)
	}
	return agentIDs[0], nil
}

func (mm *MainModule) getModuleIDByName(moduleName string, gid, pid string) (uint64, error) {
	var moduleIDs []uint64
	subQueryGroupPolicies := mm.gdbc.
		Model(&models.Group{}).
		Select("p.id").
		Joins("LEFT JOIN groups_to_policies gtp ON gtp.group_id = groups.id").
		Joins("LEFT JOIN policies p ON gtp.policy_id = p.id AND p.deleted_at IS NULL").
		Where("groups.hash LIKE ? AND p.hash LIKE ?", gid, pid).
		QueryExpr()
	subQueryUnionGroups := mm.gdbc.
		Raw("? UNION SELECT 0", subQueryGroupPolicies).
		QueryExpr()
	result := mm.gdbc.
		Model(&models.ModuleA{}).
		Where("modules.name LIKE ? AND modules.policy_id IN (?)", moduleName, subQueryUnionGroups).
		Order("modules.policy_id DESC").
		Limit(1).
		Pluck("id", &moduleIDs)
	if result.Error != nil {
		return 0, fmt.Errorf(
			"failed to get the module id by its name %s (group ID %s): %w",
			moduleName, gid, result.Error,
		)
	}
	if len(moduleIDs) == 0 {
		return 0, fmt.Errorf(
			"failed to get the module id by its name %s (group ID %s): no module name found",
			moduleName, gid,
		)
	}
	return moduleIDs[0], nil
}

func (mm *MainModule) createEvents(ctx context.Context, events []*models.Event) error {
	valueStrings := []string{}
	valueArgs := []interface{}{}

	for _, event := range events {
		valueStrings = append(valueStrings, "(?, ?, ?, NOW())")
		valueArgs = append(valueArgs, event.ModuleID, event.AgentID, event.Info)
	}

	dupl := `ON DUPLICATE KEY UPDATE info=VALUES(info), date=NOW()`
	smt := `INSERT INTO events (module_id, agent_id, info, date) VALUES %s %s`
	smt = fmt.Sprintf(smt, strings.Join(valueStrings, ","), dupl)

	tx := mm.gdbc.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelDefault,
	})
	if err := tx.Exec(smt, valueArgs...).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to insert batch events %d to local DB: %w", len(events), err)
	}

	tx.Commit()
	return nil
}

// UnregisterLuaAPI is function that unregistrate extra API function for each of type service
func (mm *MainModule) UnregisterLuaAPI(state *lua.State, config *loader.ModuleConfig) error {
	luar.Register(state.L, "__api", luar.Map{})

	return nil
}

// DefaultRecvPacket is function that operate packets as default receiver
func (mm *MainModule) DefaultRecvPacket(ctx context.Context, packet *vxproto.Packet) error {
	log := logrus.WithContext(ctx).WithFields(logrus.Fields{
		"module": packet.Module,
		"type":   packet.PType.String(),
		"src":    packet.Src,
		"dst":    packet.Dst,
	})

	if packet.Module != "main" {
		log.Debug("default receiver got new packet")
	}

	return nil
}

func (mm *MainModule) recvData(ctx context.Context, src string, data *vxproto.Data) {
	logrus.WithContext(ctx).WithFields(logrus.Fields{
		"module": "main",
		"type":   "data",
		"src":    src,
		"len":    len(data.Data),
	}).Debug("received data")
}

func (mm *MainModule) recvFile(ctx context.Context, src string, file *vxproto.File) {
	logrus.WithContext(ctx).WithFields(logrus.Fields{
		"module": "main",
		"type":   "file",
		"name":   file.Name,
		"path":   file.Path,
		"uniq":   file.Uniq,
		"src":    src,
	}).Debug("received file")
}

func (mm *MainModule) recvText(ctx context.Context, src string, text *vxproto.Text) {
	logrus.WithContext(ctx).WithFields(logrus.Fields{
		"module": "main",
		"type":   "text",
		"name":   text.Name,
		"len":    len(text.Data),
		"src":    src,
	}).Debug("received text")
}

func (mm *MainModule) recvMsg(ctx context.Context, src string, msg *vxproto.Msg) {
	logrus.WithContext(ctx).WithFields(logrus.Fields{
		"module": "main",
		"type":   "msg",
		"msg":    msg.MType.String(),
		"len":    len(msg.Data),
		"src":    src,
	}).Debug("received message")
}

func (mm *MainModule) recvAction(ctx context.Context, src string, act *vxproto.Action) error {
	agnt := mm.agents.get(src)

	actionCtx, actionSpan := obs.Observer.NewSpan(ctx, obs.SpanKindInternal, "recv_action")
	defer actionSpan.End()

	log := logrus.WithContext(actionCtx).WithFields(logrus.Fields{
		"module": "main",
		"type":   "action",
		"name":   act.Name,
		"len":    len(act.Data),
		"src":    src,
	})

	if agnt == nil {
		err := fmt.Errorf("failed to get agent ID from token")
		log.WithError(err).Error("action push_event")
		return err
	}

	aid := agnt.info.ID
	gid := agnt.info.GID
	log = log.WithFields(logrus.Fields{
		"agent_id": aid,
		"group_id": gid,
	})

	switch act.Name {
	case "push_event":
		var ape protoagent.ActionPushEvent
		const msgBadActionData = "bad action data: push_event"
		if err := proto.Unmarshal(act.Data, &ape); err != nil {
			err := fmt.Errorf("failed to unmarshal action data")
			log.WithError(err).Error(msgBadActionData)
			return err
		}

		log = log.WithFields(logrus.Fields{
			"policy_id":   ape.GetGroupId(),
			"module_name": ape.GetModuleName(),
		})
		log.Debugf("received action:\n%s", ape.GetEventInfo())

		if gid != ape.GetGroupId() {
			err := fmt.Errorf("mismatch group ID in received event")
			log.WithError(err).Error(msgBadActionData)
			return err
		}

		pid := ape.GetPolicyId()
		mname := ape.GetModuleName()
		info := ape.GetEventInfo()
		if !mm.pushEventToQueue(actionCtx, aid, gid, pid, mname, info) {
			err := fmt.Errorf("failed to process action")
			log.WithError(err).Error(msgBadActionData)
			return err
		}

	case "push_obs_packet":
		var obsPacket protoagent.ObsPacket
		if err := proto.Unmarshal(act.Data, &obsPacket); err != nil {
			log.WithError(err).Error("bad action data: push_obs_packet")
			return fmt.Errorf("failed to unmarshal action data")
		}

		if obsPacket.Traces != nil {
			protoSpans := make([]*tracepb.ResourceSpans, 0, len(obsPacket.Traces))
			for _, trace := range obsPacket.Traces {
				spans := tracepb.ResourceSpans{}
				err := proto.Unmarshal(trace, &spans)
				if err != nil {
					continue
				}
				protoSpans = append(protoSpans, &spans)
			}
			if len(protoSpans) != 0 {
				if err := mm.tracerClient.UploadTraces(actionCtx, protoSpans); err != nil {
					logrus.WithError(err).Warn("tracer error")
				}
			}
		}

		if obsPacket.Metrics != nil {
			protoMetrics := make([]*metricspb.ResourceMetrics, 0, len(obsPacket.Metrics))
			for _, metric := range obsPacket.Metrics {
				metrics := metricspb.ResourceMetrics{}
				err := proto.Unmarshal(metric, &metrics)
				if err != nil {
					continue
				}
				protoMetrics = append(protoMetrics, &metrics)
			}
			if len(protoMetrics) != 0 {
				if err := mm.meterClient.UploadMetrics(actionCtx, protoMetrics); err != nil {
					log.WithError(err).Error("failed to upload metrics")
				}
			}
		}

	default:
		err := fmt.Errorf("failed to process unknown action")
		log.WithError(err).Error("recieved unknown action: " + act.Name)
		return err
	}

	return nil
}

func (mm *MainModule) getGroupsList(ctx context.Context) []string {
	if mm.gdbc == nil {
		// TODO: here need to all groups according to config file
		return []string{""}
	}

	var groupHashes []string
	err := mm.gdbc.
		Model(&models.Group{}).
		Pluck("hash", &groupHashes).
		Error
	if err != nil {
		logrus.WithContext(ctx).WithError(err).Error("failed to get groups list from DB")
		return []string{}
	}
	return groupHashes
}

func (mm *MainModule) getAgentsList(ctx context.Context) (map[string]*AgentInfoDB, error) {
	if mm.gdbc == nil {
		// TODO: here need to all agents according to config file
		agentsDB := make(map[string]*AgentInfoDB)
		for _, ainfo := range mm.agents.dump() {
			agentsDB[ainfo.info.ID] = &AgentInfoDB{
				GID:        ainfo.info.GID,
				AuthStatus: dbAgentStatusAuthorized,
			}
		}
		return agentsDB, nil
	}

	agents, err := mm.fetchAgentsList(ctx)
	if err != nil {
		return nil, err
	}
	return agents, nil
}

func (mm *MainModule) fetchAgentsList(_ context.Context) (map[string]*AgentInfoDB, error) {
	var agentsList []*AgentBinary
	err := mm.gdbc.
		Table("agents").
		Select("agents.hash as agent_hash, IFNULL(g.hash, '') AS gid, agents.auth_status").
		Joins("LEFT JOIN groups g ON group_id LIKE g.id AND g.deleted_at IS NULL").
		Find(&agentsList).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the agents info from the DB: %w", err)
	}
	agents := make(map[string]*AgentInfoDB, len(agentsList))
	for _, a := range agentsList {
		agents[a.AgentHash] = a.AgentInfo
	}
	return agents, nil
}

func (mm *MainModule) publishEvents(ctx context.Context) {
	defer mm.wgControl.Done()

	queue := make([]*models.Event, 0)
	publish := func() {
		if len(queue) == 0 {
			return
		}
		publisherCtx, publisherSpan := obs.Observer.NewSpan(
			context.TODO(),
			obs.SpanKindInternal, "events_publisher",
		)
		if err := mm.createEvents(publisherCtx, queue); err != nil {
			logrus.WithContext(publisherCtx).WithError(err).Warn("failed to publish events")
			if len(queue) > publishEventsQueueLimit {
				queue = queue[:0]
			}
		} else {
			queue = queue[:0]
		}
		publisherSpan.End()
	}

	timer := time.NewTicker(publishEventsInterval)
	defer timer.Stop()
	for {
		select {
		case event := <-mm.eventsQueue:
			queue = append(queue, event)
		case <-timer.C:
			publish()
		case <-ctx.Done():
			publish()
			return
		}
	}
}

func (mm *MainModule) getAgentInfoDB(ctx context.Context, aid string) *AgentInfoDB {
	if mm.gdbc == nil {
		// TODO: here need to get agent info according to config file
		return &AgentInfoDB{
			GID:        "",
			AuthStatus: dbAgentStatusAuthorized,
		}
	}

	var agentInfo AgentInfoDB
	err := mm.gdbc.
		Table("agents").
		Select("IFNULL(g.hash, '') as gid, agents.auth_status").
		Joins("LEFT JOIN groups g ON group_id LIKE g.id AND g.deleted_at IS NULL").
		Where("agents.hash = ?", aid).
		First(&agentInfo).Error
	if err != nil {
		logrus.WithContext(ctx).WithError(err).WithField("agent_id", aid).
			Error("failed to get agent info by ID")
		return nil
	}
	return &agentInfo
}

func (mm *MainModule) getGroupInfoDB(ctx context.Context, gid string) *GroupInfoDB {
	if mm.gdbc == nil {
		// TODO: here need to get agent info according to config file
		return &GroupInfoDB{
			GID: "",
		}
	}

	var groupInfo GroupInfoDB
	err := mm.gdbc.
		Table("groups").
		Select("IFNULL(hash, '') as gid").
		Where("deleted_at IS NULL AND hash = ?", gid).
		First(&groupInfo).Error
	if err != nil {
		logrus.WithContext(ctx).WithError(err).WithField("group_id", gid).
			Error("failed to get group info by ID")
		return nil
	}
	return &groupInfo
}

func (mm *MainModule) pushEventToQueue(ctx context.Context, aid, gid, pid, mname, info string) bool {
	log := logrus.WithContext(ctx).WithFields(logrus.Fields{
		"agent_id":    aid,
		"group_id":    gid,
		"policy_id":   pid,
		"module_name": mname,
		"event_info":  info,
	})

	// TODO: here need look up agent ID in the group to validate event
	if aid == "" {
		log.Warn("failed to get agent ID from arguments")
		return false
	}

	if mm.gdbc == nil {
		return false
	}

	var evInfo models.EventInfo
	if err := json.Unmarshal([]byte(info), &evInfo); err != nil {
		log.WithError(err).Error("failed to parse event info")
		return false
	}
	if evInfo.Uniq == "" {
		uniq := make([]byte, 16)
		if _, err := io.ReadFull(rand.Reader, uniq); err != nil {
			log.WithError(err).Error("failed to make event info unique id")
			return false
		}
		evInfo.Uniq = hex.EncodeToString(uniq)
	}
	if err := evInfo.Valid(); err != nil {
		log.WithError(err).Error("failed to validate event info")
		return false
	}

	var agentID uint64
	if aid != "" {
		var err error
		agentID, err = mm.getAgentIDByHash(aid, gid)
		if err != nil {
			log.WithError(err).Warn("failed to get agent ID from local DB")
			return false
		}
	}
	moduleID, err := mm.getModuleIDByName(mname, gid, pid)
	if err != nil {
		log.WithError(err).Warn("failed to get module ID from local DB")
		return false
	}

	event := &models.Event{
		AgentID:  agentID,
		ModuleID: moduleID,
		Info:     evInfo,
	}
	select {
	case mm.eventsQueue <- event:
	case <-time.NewTimer(5 * time.Second).C:
		log.WithError(err).Error("failed to insert new event to the queue: timeout exceeded")
		return false
	case <-ctx.Done():
		log.WithError(err).Error("failed to insert new event to the queue: interrupted")
		return false
	}

	return true
}

func (mm *MainModule) disconnectAllAgents(ctx context.Context) {
	if mm.gdbc == nil {
		return
	}

	err := mm.gdbc.
		Model(&models.Agent{}).
		Where("status LIKE ?", "connected").
		UpdateColumns(map[string]interface{}{
			"status":     "disconnected",
			"updated_at": gorm.Expr(sqlNowFunction),
		}).Error
	if err != nil {
		logrus.WithContext(ctx).WithError(err).Warn("failed to disconnect all agents")
	}
}

func (mm *MainModule) syncAliveStatusAgents(ctx context.Context, agents map[string]*agentInfo) {
	if mm.gdbc == nil {
		return
	}

	aids := make([]interface{}, 0, syncAliveStatusBatchSize)
	ainfos := make([]*agentInfo, 0, syncAliveStatusBatchSize)
	updateAgents := func() {
		agentUpdateMap := map[string]interface{}{
			"status":         "connected",
			"connected_date": gorm.Expr(sqlNowFunction),
		}
		err := mm.gdbc.
			Model(&models.Agent{}).
			Where("hash IN (?)", aids).
			UpdateColumns(agentUpdateMap).Error
		if err != nil {
			logrus.WithContext(ctx).WithError(err).WithFields(logrus.Fields{
				"aids": aids,
			}).Warn("failed to update alive agents status")
		}
		aids = make([]interface{}, 0, syncAliveStatusBatchSize)
		for _, ainfo := range ainfos {
			ainfo.mxdone.Unlock()
		}
		ainfos = make([]*agentInfo, 0, syncAliveStatusBatchSize)
	}
	disconnectLostAgents := func() {
		timeExpr := gorm.Expr("NOW() - INTERVAL ? MINUTE", lostAgentDeltaTime)
		err := mm.gdbc.
			Model(&models.Agent{}).
			Where("connected_date < ? AND status LIKE ?", timeExpr, "connected").
			UpdateColumns(map[string]interface{}{
				"status":     "disconnected",
				"updated_at": gorm.Expr(sqlNowFunction),
			}).Error
		if err != nil {
			logrus.WithContext(ctx).WithError(err).Warn("failed to update lost agents status")
		}
	}

	for _, ainfo := range agents {
		if ainfo == nil {
			continue
		}
		ainfo.mxdone.Lock()
		if !ainfo.isfin && ainfo.info.Type == vxproto.VXAgent {
			aids = append(aids, ainfo.info.ID)
			ainfos = append(ainfos, ainfo)
		} else {
			ainfo.mxdone.Unlock()
		}
		if len(aids) >= syncAliveStatusBatchSize {
			updateAgents()
		}
	}
	if len(aids) != 0 {
		updateAgents()
	}
	disconnectLostAgents()
}

// filterModulesForAgent is a helper function to make relevant modules list
// to the agent from whole group modules list (mismatching OS type and arch)
func (mm *MainModule) filterModulesForAgent(ctx context.Context, mIDs []string, ainfo *agentInfo) []string {
	var nmIDs []string

	// Check agent exist or moving to another group then need to remove all modules
	ainfodb := mm.getAgentInfoDB(ctx, ainfo.info.ID)
	if ainfodb == nil || ainfodb.AuthStatus != dbAgentStatusAuthorized || ainfodb.GID != ainfo.info.GID {
		return nmIDs
	}

	stringInSlice := func(a string, list []string) bool {
		for _, b := range list {
			if b == a {
				return true
			}
		}
		return false
	}

	moduleInSlice := func(a string, list []string) bool {
		var m string
		if p := strings.Split(a, ":"); len(p) < 3 {
			return false
		} else {
			m = ":" + p[2]
		}
		for _, b := range list {
			if strings.HasSuffix(b, m) {
				return true
			}
		}
		return false
	}

	for _, mID := range mIDs {
		mConfig := mm.cnt.GetModule(mID).GetConfig()
		agentOS := ainfo.info.Info.GetOs()
		if archList, ok := mConfig.OS[agentOS.GetType()]; ok && stringInSlice(agentOS.GetArch(), archList) {
			if !moduleInSlice(mID, nmIDs) {
				nmIDs = append(nmIDs, mID)
			}
		}
	}

	return nmIDs
}

// syncModules is a function to sync agent modules state to group modules state
func (mm *MainModule) syncModules(ctx context.Context, gid, dst string, ainfo *agentInfo,
	mStatusList *protoagent.ModuleStatusList,
) (*protoagent.ModuleStatusList, error) {
	var err error
	var (
		wantStartModuleIDs  []string
		wantStopModuleIDs   []string
		wantUpdateModuleIDs []string
		wantUpdateConfigIDs []string
	)
	mIDs := append(mm.cnt.GetModuleIdsForGroup(gid), mm.cnt.GetSharedModuleIds()...)
	mIDs = mm.filterModulesForAgent(ctx, mIDs, ainfo)
	mStates := mm.cnt.GetModuleStates(mIDs)
	mObjs := mm.cnt.GetModules(mIDs)

	isEqualModuleConfig := func(mc1 *loader.ModuleConfig, mc2 *protoagent.Config) bool {
		isSameState := mc1.State == mc2.GetState()
		isSameTemplate := mc1.Template == mc2.GetTemplate()
		isSameVersion := mc1.Version.String() == mc2.GetVersion()
		isSameLastModuleUpdate := mc1.LastModuleUpdate == mc2.GetLastModuleUpdate()
		if !isSameState || !isSameTemplate || !isSameVersion || !isSameLastModuleUpdate {
			return false
		}
		if mc1.PolicyID != mc2.GetPolicyId() || mc1.GroupID != mc2.GetGroupId() {
			return false
		}
		return true
	}

	isEqualModuleConfigItem := func(mc1 *loader.ModuleConfig, mc2 *protoagent.Config) bool {
		return mc1.LastUpdate == mc2.GetLastUpdate()
	}

	for _, mStatusItem := range mStatusList.GetList() {
		mConfig := mStatusItem.GetConfig()
		mID := mConfig.GetGroupId() + ":" + mConfig.GetPolicyId() + ":" + mConfig.GetName()
		if state, ok := mStates[mID]; ok && state != nil && state.GetStatus() == protoagent.ModuleStatus_RUNNING {
			if obj, ok := mObjs[mID]; ok && obj != nil {
				if !isEqualModuleConfig(obj.GetConfig(), mConfig) {
					wantUpdateModuleIDs = append(wantUpdateModuleIDs, mID)
				} else if !isEqualModuleConfigItem(obj.GetConfig(), mConfig) {
					wantUpdateConfigIDs = append(wantUpdateConfigIDs, mID)
				}
			}
			// TODO: here should be exception if module object not found
		} else {
			wantStopModuleIDs = append(wantStopModuleIDs, mID)
		}
	}

	for _, mID := range mIDs {
		var mStatus *protoagent.ModuleStatus
		for _, mStatusItem := range mStatusList.GetList() {
			mConfig := mStatusItem.GetConfig()
			if mID == mConfig.GetGroupId()+":"+mConfig.GetPolicyId()+":"+mConfig.GetName() {
				mStatus = mStatusItem
				break
			}
		}
		if mStatus == nil {
			wantStartModuleIDs = append(wantStartModuleIDs, mID)
		}
	}

	if len(wantStopModuleIDs) != 0 {
		if mStatusList, err = mm.stopModules(ctx, dst, ainfo, wantStopModuleIDs); err != nil {
			return nil, err
		}
	}
	if len(wantStartModuleIDs) != 0 {
		if mStatusList, err = mm.startModules(ctx, dst, ainfo, wantStartModuleIDs); err != nil {
			return nil, err
		}
	}
	if len(wantUpdateModuleIDs) != 0 {
		if mStatusList, err = mm.updateModules(ctx, dst, ainfo, wantUpdateModuleIDs); err != nil {
			return nil, err
		}
	}
	if len(wantUpdateConfigIDs) != 0 {
		if mStatusList, err = mm.updateModulesConfig(ctx, dst, ainfo, wantUpdateConfigIDs); err != nil {
			return nil, err
		}
	}

	return mStatusList, nil
}

func (mm *MainModule) syncAgents() {
	defer mm.wgControl.Done()

	ctx := context.Background()
	syncCtx, syncSpan := obs.Observer.NewSpan(ctx, obs.SpanKindInternal, "sync_agents")
	mm.disconnectAllAgents(syncCtx)
	syncSpan.End()

	for {
		ctx = context.Background()
		syncCtx, syncSpan = obs.Observer.NewSpan(ctx, obs.SpanKindInternal, "sync_agents")
		agents := mm.agents.dump()
		mm.syncAliveStatusAgents(syncCtx, agents)
		mm.updateAgentsStatusWithDBInfo(syncCtx, agents)
		syncSpan.End()

		select {
		case <-time.NewTimer(syncAgentsInterval).C:
			continue
		case <-mm.quitSyncAgents:
		}
		break
	}
}

func (mm *MainModule) updateAgentsStatusWithDBInfo(ctx context.Context, agentsList map[string]*agentInfo) {
	gids := make(map[string]struct{})
	for _, gid := range mm.getGroupsList(ctx) {
		gids[gid] = struct{}{}
	}
	ids, err := mm.getAgentsList(ctx)
	if err != nil {
		logrus.WithContext(ctx).WithError(err).Error("failed to get agents changes from DB")
		return
	}
	agentsToDrop := make(map[string]string)
	agentsToMove := make(map[string]string)
	connectedAgents := make(map[string]struct{}, len(agentsList))
	for _, ainfo := range agentsList {
		if ainfo == nil || ainfo.isfin {
			continue
		}
		if ainfo.info.Type == vxproto.Aggregate {
			if _, ok := gids[ainfo.info.GID]; !ok {
				agentsToDrop[ainfo.info.ID] = ""
				ainfo.update <- struct{}{}
			} else {
				connectedAgents[ainfo.info.ID] = struct{}{}
			}
			continue
		}
		if ainfoDB, ok := ids[ainfo.info.ID]; !ok || ainfoDB.AuthStatus != dbAgentStatusAuthorized {
			agentsToDrop[ainfo.info.ID] = ""
			ainfo.update <- struct{}{}
		} else {
			connectedAgents[ainfo.info.ID] = struct{}{}
			if ainfo.info.GID != ainfoDB.GID {
				agentsToMove[ainfo.info.ID] = ainfoDB.GID
			}
		}
	}
	mm.authenticator.registerAuth(connectedAgents, ids)
	for aid := range agentsToDrop {
		mm.proto.DropAgent(ctx, aid)
	}
	for aid, gid := range agentsToMove {
		mm.proto.MoveAgent(ctx, aid, gid)
	}
}

func (mm *MainModule) syncGroups() {
	defer mm.wgControl.Done()

	isInSlice := func(list []string, el string) bool {
		for _, v := range list {
			if el == v {
				return true
			}
		}
		return false
	}
	stopGroupState := func(_ context.Context, _ string, ginfo *groupInfo) {
		ginfo.mxdone.Lock()
		if !ginfo.isfin {
			ginfo.quit <- false
			<-ginfo.done
		}
		ginfo.mxdone.Unlock()
	}
	startGroupState := func(ctx context.Context, gid string) {
		stateControlSync := make(chan struct{})
		mm.groups.add(gid, &groupInfo{
			quit:   make(chan bool),
			update: make(chan struct{}),
			done:   make(chan struct{}),
		})
		mm.wgControl.Add(1)
		go mm.controlGroupState(ctx, gid, stateControlSync)
		<-stateControlSync
	}

	for {
		ctx := context.Background()
		syncCtx, syncSpan := obs.Observer.NewSpan(ctx, obs.SpanKindInternal, "sync_groups")
		ids := mm.getGroupsList(syncCtx)
		groups := mm.groups.dump()
		for gid, ginfo := range groups {
			if ginfo == nil || ginfo.isfin {
				continue
			}
			if !isInSlice(ids, gid) {
				stopGroupState(syncCtx, gid, ginfo)
			}
		}
		for _, gid := range ids {
			if ginfo, ok := groups[gid]; !ok || ginfo.isfin {
				startGroupState(syncCtx, gid)
			}
		}
		syncSpan.End()

		select {
		case <-time.NewTimer(syncGroupsInterval).C:
			continue
		case <-mm.quitSyncGroups:
		}

		break
	}
}

// controlAggregate is a dummy function to enumerate all aggregate/group connections
// and linking those to the group modules state to make update notifications
func (mm *MainModule) controlAggregate(dst string, syncRun chan struct{}, ainfo *agentInfo) {
	mm.controlProxy(dst, syncRun, ainfo)
}

// controlBrowser is a dummy function to enumerate all browser connections
// and linking those to the group modules state to make update notifications
func (mm *MainModule) controlBrowser(dst string, syncRun chan struct{}, ainfo *agentInfo) {
	mm.controlProxy(dst, syncRun, ainfo)
}

// controlExternal is a dummy function to enumerate all external connections
// and linking those to the group modules state to make update notifications
func (mm *MainModule) controlExternal(dst string, syncRun chan struct{}, ainfo *agentInfo) {
	mm.controlProxy(dst, syncRun, ainfo)
}

func (mm *MainModule) controlProxy(dst string, syncRun chan struct{}, ainfo *agentInfo) {
	defer mm.wgControl.Done()

	// finalize agent state and remove from connected agents list
	defer func() { ainfo.done <- struct{}{} }()
	defer mm.agents.del(dst)

	mm.cnt.SetUpdateChanForGroup(ainfo.info.GID, ainfo.update)
	defer mm.cnt.UnsetUpdateChanForGroup(ainfo.info.GID, ainfo.update)

	syncRun <- struct{}{}
	for {
		select {
		case <-ainfo.update:
			continue
		case <-ainfo.quit:
		}
		break
	}
}

// controlAgent is a function to check modules state per group and
// provide notifications to agent side about needs to module update
func (mm *MainModule) controlAgent(dst string, syncRun chan struct{}, ainfo *agentInfo) {
	defer mm.wgControl.Done()

	ctx := context.Background()
	controlCtx, controlSpan := obs.Observer.NewSpan(ctx, obs.SpanKindInternal, "control_agent")
	log := logrus.WithContext(controlCtx).WithFields(logrus.Fields{
		"agent_id": ainfo.info.ID,
		"group_id": ainfo.info.GID,
	})

	host, _, err := net.SplitHostPort(ainfo.info.IP)
	if err != nil {
		log.Warn("failed to get agent ID address from handshake")
		host = "0.0.0.0"
	}

	// finalize agent state and remove from connected agents list
	defer func() { ainfo.done <- struct{}{} }()
	defer mm.agents.del(dst)

	defer func() {
		controlCtx, controlSpan := obs.Observer.NewSpan(ctx, obs.SpanKindInternal, "control_agent")
		defer controlSpan.End()
		if err := mm.updateAgentOnDisconnection(controlCtx, ainfo.info.ID); err != nil {
			log.WithContext(controlCtx).WithError(err).Warn("failed to update agent status to disconnected")
		}
	}()

	mm.cnt.SetUpdateChanForGroup(ainfo.info.GID, ainfo.update)
	defer mm.cnt.UnsetUpdateChanForGroup(ainfo.info.GID, ainfo.update)

	if err := mm.updateAgentOnConnection(controlCtx, ainfo.info.ID, host); err != nil {
		log.WithError(err).Warn("failed to update agent status to connected")
	}

	// end of startup section (there must not return before)
	syncRun <- struct{}{}
	controlSpan.End()

	mxSync := &sync.Mutex{}
	for {
		if !ainfo.info.IsOnlyForUpgrade {
			mm.runSyncAgentModules(ctx, mxSync, dst, ainfo)
		}
		select {
		case <-ainfo.update:
			continue
		// agent upgrade signal
		case req := <-ainfo.upgrade:
			mm.wgExchAgent.Add(1)
			go func() {
				upgradeCtx, upgradeSpan := obs.Observer.NewSpan(ctx, obs.SpanKindInternal, "upgrade_agent")
				defer upgradeSpan.End()
				defer mm.wgExchAgent.Done()
				mm.upgradeTaskConsumer.requestAgentUpgrade(upgradeCtx, ainfo, req)
			}()
		case <-ainfo.quit:
			return
		}
	}
}

func (mm *MainModule) runSyncAgentModules(ctx context.Context, mxSync *sync.Mutex, dst string, ainfo *agentInfo) {
	// TODO: here need use Context to avoid hanging on receive on closed socket
	mm.wgExchAgent.Add(1)
	go func() {
		syncCtx, syncSpan := obs.Observer.NewSpan(ctx, obs.SpanKindInternal, "sync_agent_modules")
		defer syncSpan.End()
		defer mm.wgExchAgent.Done()
		defer mxSync.Unlock()
		mxSync.Lock()

		commitError := func(err error, msg string) {
			logrus.WithContext(syncCtx).WithError(err).WithFields(logrus.Fields{
				"agent_id": ainfo.info.ID,
				"group_id": ainfo.info.GID,
			}).Error(msg)
			// do some a little bit delay before retry sync modules to prevent loop fast call
			time.Sleep(500 * time.Millisecond)
			// try to rerun sync modules and here uses nonblocking
			// because agent can get quit message in concurrent time
			select {
			case ainfo.update <- struct{}{}:
			default:
			}
		}

		mStatusList, err := mm.getStatusModules(syncCtx, dst, ainfo)
		if err != nil {
			commitError(err, "failed to get status of modules from agent side")
			return
		}

		if _, err = mm.syncModules(syncCtx, ainfo.info.GID, dst, ainfo, mStatusList); err != nil {
			commitError(err, "failed to update modules list on agent side")
			return
		}
	}()
}

func (mm *MainModule) updateAgentOnConnection(_ context.Context, hash string, ip string) error {
	if mm.gdbc == nil {
		return nil
	}
	agentUpdateMap, err := getUpdateMap(
		map[string]*struct {
			Value interface{}
			Rule  string
		}{
			"status": {
				Value: "connected",
			},
			"ip": {
				Value: ip,
				Rule:  "max=50,ip,required",
			},
			"connected_date": {
				Value: gorm.Expr(sqlNowFunction),
			},
			"updated_at": {
				Value: gorm.Expr(sqlNowFunction),
			},
		},
		mm.validator,
	)
	if err != nil {
		return err
	}
	err = mm.gdbc.
		Scopes(agentWithHash(hash)).
		UpdateColumns(agentUpdateMap).Error
	if err != nil {
		return fmt.Errorf("failed to update the agent information on its connection: %w", err)
	}
	return nil
}

func generateErrFromValidationErrs(errs map[string]interface{}) error {
	return fmt.Errorf("validation failed due to the following errors: %v", errs)
}

func (mm *MainModule) updateAgentOnDisconnection(_ context.Context, hash string) error {
	if mm.gdbc == nil {
		return nil
	}
	err := mm.gdbc.
		Scopes(agentWithHash(hash)).
		UpdateColumns(map[string]interface{}{
			"status":     "disconnected",
			"updated_at": gorm.Expr(sqlNowFunction),
		}).Error
	if err != nil {
		return fmt.Errorf("failed to update the agent information on its disconnection: %w", err)
	}
	return err
}

func agentWithHash(hash string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Model(&models.Agent{}).Where("hash = ?", hash)
	}
}

func genAgentMessage(ctx context.Context, ainfo *agentInfo) *logrus.Entry {
	return logrus.WithContext(ctx).WithFields(logrus.Fields{
		"agent_id": ainfo.info.ID,
		"group_id": ainfo.info.GID,
	})
}

func genAgentError(ctx context.Context, ainfo *agentInfo, e error) *logrus.Entry {
	return genAgentMessage(ctx, ainfo).WithError(e)
}

// controlGroupState is function to run and control all modules
// with GroupID equals gid that modules for all agents in this group
func (mm *MainModule) controlGroupState(ctx context.Context, gid string, syncRun chan struct{}) {
	var shutdown bool
	ginfo := mm.groups.get(gid)
	controlCtx, controlSpan := obs.Observer.NewSpan(ctx, obs.SpanKindInternal, "control_group_state")
	defer mm.wgControl.Done()
	if ginfo == nil {
		logrus.WithContext(controlCtx).WithFields(logrus.Fields{
			"group_id": gid,
		}).Error("failed to start control group state, group info not found")
		return
	}
	defer func() { ginfo.done <- struct{}{} }()
	defer mm.groups.del(gid)
	defer func(sd *bool) {
		if _, err := mm.cnt.StopModulesForGroup(gid, *sd); err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{
				"group_id": gid,
			}).WithError(err).Error("failed to stop modules for the group")
		}
	}(&shutdown)
	defer mm.cnt.UnsetUpdateChanForGroup(gid, ginfo.update)

	if _, err := mm.cnt.StartModulesForGroup(gid); err != nil {
		logrus.WithContext(controlCtx).WithError(err).WithFields(logrus.Fields{
			"group_id": gid,
		}).Error("can't start modules for group")
	}

	mm.cnt.SetUpdateChanForGroup(gid, ginfo.update)
	syncRun <- struct{}{}
	controlSpan.End()
	for {
		select {
		case <-ginfo.update:
			continue
		case shutdown = <-ginfo.quit:
		}

		break
	}
}

// controlSharedState is function to run and control all modules
// with GroupID equals 0 that modules for all connected agents
func (mm *MainModule) controlSharedState(ctx context.Context, syncRun chan struct{}) {
	var shutdown bool
	ginfo := mm.groups.get("")
	controlCtx, controlSpan := obs.Observer.NewSpan(ctx, obs.SpanKindInternal, "control_shared_state")
	defer mm.wgControl.Done()
	if ginfo == nil {
		logrus.WithContext(controlCtx).WithFields(logrus.Fields{
			"group_id": "",
		}).Error("failed to start control shared group state, group info not found")
		return
	}
	defer func() { ginfo.done <- struct{}{} }()
	defer mm.groups.del("")
	defer func(sd *bool) {
		if _, err := mm.cnt.StopSharedModules(*sd); err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{
				"group_id": "",
			}).WithError(err).Error("failed to stop shared modules")
		}
	}(&shutdown)
	defer mm.cnt.UnsetUpdateChan(ginfo.update)

	if _, err := mm.cnt.StartSharedModules(); err != nil {
		logrus.WithContext(controlCtx).WithError(err).Error("can't start shared modules")
	}

	mm.cnt.SetUpdateChan(ginfo.update)
	syncRun <- struct{}{}
	controlSpan.End()
	for {
		select {
		case <-ginfo.update:
			continue
		case shutdown = <-ginfo.quit:
		}

		break
	}
}

func (mm *MainModule) handlerAgentConnected(ctx context.Context, info *vxproto.AgentInfo) {
	mm.mutexAgent.Lock()
	defer mm.mutexAgent.Unlock()

	agentControlSync := make(chan struct{})
	ainfo := &agentInfo{
		info:    info,
		quit:    make(chan struct{}),
		update:  make(chan struct{}),
		upgrade: make(chan *store.Task),
		done:    make(chan struct{}),
	}
	mm.agents.add(info.Dst, ainfo)

	mm.wgControl.Add(1)
	switch info.Type {
	case vxproto.Aggregate:
		go mm.controlAggregate(info.Dst, agentControlSync, ainfo)
		defer func() { <-agentControlSync }()
	case vxproto.Browser:
		go mm.controlBrowser(info.Dst, agentControlSync, ainfo)
		defer func() { <-agentControlSync }()
	case vxproto.External:
		go mm.controlExternal(info.Dst, agentControlSync, ainfo)
		defer func() { <-agentControlSync }()
	case vxproto.VXAgent:
		go mm.controlAgent(info.Dst, agentControlSync, ainfo)
		defer func() { <-agentControlSync }()
	default:
		mm.wgControl.Done()
	}

	mm.upgradeTaskConsumer.tasksStore.SignalAgentConnection(ctx, info.ID, info.Ver)
}

func (mm *MainModule) handlerAgentDisconnected(ctx context.Context, info *vxproto.AgentInfo) {
	mm.mutexAgent.Lock()
	defer mm.mutexAgent.Unlock()

	ainfo := mm.agents.get(info.Dst)
	if ainfo == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{
			"agent_id": info.ID,
			"group_id": info.GID,
		}).Error("failed to disconnect agent, it's already removed")
		return
	}
	ainfo.mxdone.Lock()
	if !ainfo.isfin {
		ainfo.quit <- struct{}{}
		<-ainfo.done
	}
	ainfo.mxdone.Unlock()
}

func (mm *MainModule) handlerStopMainModule(_ context.Context) {
	// Stop synchers routines
	mm.quitSyncAgents <- struct{}{}
	mm.quitSyncGroups <- struct{}{}

	for _, ginfo := range mm.groups.dump() {
		if ginfo == nil {
			continue
		}

		ginfo.mxdone.Lock()
		if !ginfo.isfin {
			ginfo.quit <- true
			<-ginfo.done
		}
		ginfo.mxdone.Unlock()
	}
	for _, ainfo := range mm.agents.dump() {
		if ainfo == nil {
			continue
		}

		ainfo.mxdone.Lock()
		if !ainfo.isfin {
			ainfo.quit <- struct{}{}
			<-ainfo.done
		}
		ainfo.mxdone.Unlock()
	}

	mm.cancelUpgradeTaskConsumer()
	mm.cancelEventsPublisher()

	mm.wgControl.Wait()
	mm.wgExchAgent.Wait()
}

func (mm *MainModule) recvPacket() error {
	defer mm.wgReceiver.Done()

	component := "main_packet_receiver"
	receiver := mm.msocket.GetReceiver()
	if receiver == nil {
		logrus.Error("failed to initialize packet receiver")
		return fmt.Errorf("failed to initialize packet receiver")
	}
	getAgentEntry := func(ctx context.Context, agentInfo *vxproto.AgentInfo) *logrus.Entry {
		return logrus.WithContext(ctx).WithFields(logrus.Fields{
			"id":   agentInfo.ID,
			"gid":  agentInfo.GID,
			"type": agentInfo.Type.String(),
			"ip":   agentInfo.IP,
			"src":  agentInfo.Src,
			"dst":  agentInfo.Dst,
		})
	}
	for {
		packet := <-receiver
		if packet == nil {
			return fmt.Errorf("failed receive packet")
		}
		packetCtx, packetSpan := obs.Observer.NewSpan(packet.Context(), obs.SpanKindConsumer, component)

		switch pType := packet.PType; pType {
		case vxproto.PTData:
			mm.recvData(packetCtx, packet.Src, packet.GetData())
		case vxproto.PTFile:
			mm.recvFile(packetCtx, packet.Src, packet.GetFile())
		case vxproto.PTText:
			mm.recvText(packetCtx, packet.Src, packet.GetText())
		case vxproto.PTMsg:
			mm.recvMsg(packetCtx, packet.Src, packet.GetMsg())
		case vxproto.PTAction:
			if err := mm.recvAction(packetCtx, packet.Src, packet.GetAction()); err != nil {
				logrus.
					WithContext(packetCtx).
					WithField("packet_type", pType).
					WithError(err).
					Error("failed to receive an action")
			}
		case vxproto.PTControl:
			msg := packet.GetControlMsg()
			switch msg.MsgType {
			case vxproto.AgentConnected:
				getAgentEntry(packetCtx, msg.AgentInfo).Debug("agent connected")
				mm.handlerAgentConnected(packetCtx, msg.AgentInfo)
				getAgentEntry(packetCtx, msg.AgentInfo).Debug("agent connected done")
			case vxproto.AgentDisconnected:
				getAgentEntry(packetCtx, msg.AgentInfo).Debug("agent disconnected")
				mm.handlerAgentDisconnected(packetCtx, msg.AgentInfo)
				getAgentEntry(packetCtx, msg.AgentInfo).Debug("agent disconnected done")
			case vxproto.StopModule:
				mm.handlerStopMainModule(packetCtx)
				logrus.WithContext(packetCtx).Info("main module stopped")
				packetSpan.End()
				packet.SetAck()
				return nil
			}

		default:
			logrus.WithContext(packetCtx).Error("got packet has unexpected packet type")
			packetSpan.End()
			packet.SetAck()
			return fmt.Errorf("unexpected packet type")
		}
		// use default acknowledgement callback
		packetSpan.End()
		packet.SetAck()
	}
}

type ConnectionValidatorConfig struct {
	Store    interface{}
	BasePath string
}

// New is function which constructed MainModule object
func New(
	listen string,
	cl controller.IConfigLoader,
	fl controller.IFilesLoader,
	gdb *gorm.DB,
	store storage.IStorage,
	certsProvider certs.Provider,
	version string,
	connectionValidatorConf *hardeningConfig.Validator,
	tracerClient otlptrace.Client,
	metricsClient otlpmetric.Client,
	logger *logrus.Entry,
) (mm *MainModule, err error) {
	mm = &MainModule{
		gdbc:      gdb,
		validator: models.GetValidator(),
		store:     store,
		listen:    listen,
		version:   version,
		agents: &agentList{
			list:  make(map[string]*agentInfo),
			mutex: &sync.Mutex{},
		},
		groups: &groupList{
			list:  make(map[string]*groupInfo),
			mutex: &sync.Mutex{},
		},
		mutexAgent:     &sync.Mutex{},
		eventsQueue:    make(chan *models.Event, 100),
		quitSyncAgents: make(chan struct{}),
		quitSyncGroups: make(chan struct{}),
		certsProvider:  certsProvider,
		authenticator:  newAuthenticator(),
		tracerClient:   tracerClient,
		meterClient:    metricsClient,
	}
	var ctx context.Context
	ctx, mm.cancelContext = context.WithCancel(context.Background())
	defer func() {
		if err != nil {
			mm.cancelContext()
		}
	}()

	mm.proto, err = vxproto.New(mm)
	if err != nil {
		return mm, fmt.Errorf("failed initialize VXProto object: %w", err)
	}

	mm.cnt = controller.NewController(mm, cl, fl, mm.proto)
	if err = mm.cnt.Load(); err != nil {
		return mm, err
	}

	mm.msocket = mm.proto.NewModule("main", "")
	if mm.msocket == nil {
		return mm, fmt.Errorf("failed initialize main module into VXProto")
	}

	mm.upgradeTaskConsumer, err = newUpgradeTaskConsumer(ctx, mm)
	if err != nil {
		return mm, fmt.Errorf("failed to initialize the update task consumer submodule: %w", err)
	}
	var validatorStore interface{}
	fsStore, err := storage.NewFS()
	if err != nil {
		return mm, fmt.Errorf("failed to initialize an FS store: %w", err)
	}
	switch connectionValidatorConf.Type {
	case "fs":
		validatorStore = fsStore
	case "s3":
		validatorStore = store
	case "db":
		validatorStore = gdb
	default:
		return mm, fmt.Errorf("validator does not support store of type %s", connectionValidatorConf.Type)
	}
	mm.connValidatorFactory = hardening.NewConnectionValidatorFactory(
		ctx,
		gdb,
		fsStore,
		validatorStore,
		connectionValidatorConf.Base,
		certsProvider,
		mm.authenticator,
		logger,
	)

	mm.moduleConfigDecryptor = crypto.NewConfigDecryptor()

	return mm, nil
}

// Start is function which execute main logic of MainModule
func (mm *MainModule) Start(serverConfig *vxproto.ServerConfig) error {
	ctx := context.Background()
	startCtx, startSpan := obs.Observer.NewSpan(ctx, obs.SpanKindConsumer, "start_server")
	defer startSpan.End()

	// Flush close state from previously stopping
	mm.isCloseState = false

	if mm.proto == nil {
		return fmt.Errorf("VXProto didn't initialized")
	}
	if mm.msocket == nil {
		return fmt.Errorf("module socket didn't initialized")
	}
	if mm.cnt == nil {
		return fmt.Errorf("controller didn't initialized")
	}

	// initialize system metric collection in current observer instance
	attr := attribute.String("server_id", system.MakeAgentID())
	const vxserverServiceName = "vxserver"
	if err := obs.Observer.StartProcessMetricCollect(vxserverServiceName, mm.version, attr); err != nil {
		logrus.WithError(err).Warn("failed to start process metrics collect")
	}
	if err := obs.Observer.StartGoRuntimeMetricCollect(vxserverServiceName, mm.version, attr); err != nil {
		logrus.WithError(err).Warn("failed to start go runtime metrics collect")
	}
	if err := obs.Observer.StartDumperMetricCollect(mm.proto, vxserverServiceName, mm.version, attr); err != nil {
		logrus.WithError(err).Warn("failed to start dumper metrics collect")
	}

	if !mm.proto.AddModule(mm.msocket) {
		return fmt.Errorf("failed module socket register")
	}

	// Run main handler of packets
	mm.wgReceiver.Add(1)
	go func() {
		// TODO(SSH): we should treat the error here
		_ = mm.recvPacket()
	}()

	var wgRun sync.WaitGroup
	stateSharedControlSync := make(chan struct{})
	mm.groups.add("", &groupInfo{
		quit:   make(chan bool),
		update: make(chan struct{}),
		done:   make(chan struct{}),
	})

	mm.wgControl.Add(1)
	go mm.controlSharedState(startCtx, stateSharedControlSync)
	wgRun.Add(1)
	go func() {
		<-stateSharedControlSync
		wgRun.Done()
	}()

	groups_list := mm.getGroupsList(startCtx)
	for _, gid := range groups_list {
		stateControlSync := make(chan struct{})
		mm.groups.add(gid, &groupInfo{
			quit:   make(chan bool),
			update: make(chan struct{}),
			done:   make(chan struct{}),
		})
		mm.wgControl.Add(1)
		go mm.controlGroupState(startCtx, gid, stateControlSync)
		wgRun.Add(1)
		go func() {
			<-stateControlSync
			wgRun.Done()
		}()
	}

	// Wait of running status for all group states (lua modules)
	wgRun.Wait()

	// Start synchers routines
	mm.wgControl.Add(1)
	go mm.syncAgents()
	mm.wgControl.Add(1)
	go mm.syncGroups()

	mm.wgControl.Add(1)
	var taskConsumerCtx context.Context
	taskConsumerCtx, mm.cancelUpgradeTaskConsumer = context.WithCancel(ctx)
	go mm.upgradeTaskConsumer.consume(taskConsumerCtx)

	mm.wgControl.Add(1)
	var eventsPublisherCtx context.Context
	eventsPublisherCtx, mm.cancelEventsPublisher = context.WithCancel(ctx)
	go mm.publishEvents(eventsPublisherCtx)

	listenLogger := logrus.WithContext(startCtx).WithField("type", "listen-logger")
	startSpan.End()
	return mm.proto.Listen(ctx, serverConfig, mm.connValidatorFactory, listenLogger)
}

// Stop is function which stop main logic of MainModule
func (mm *MainModule) Stop() error {
	ctx := context.Background()
	stopCtx, stopSpan := obs.Observer.NewSpan(ctx, obs.SpanKindConsumer, "stop_server")
	defer stopSpan.End()

	if mm.isCloseState {
		return fmt.Errorf("main module has already stopped")
	}

	// Set close state to decline any new connections and to except double stop main module
	mm.isCloseState = true

	if mm.proto == nil {
		return fmt.Errorf("VXProto didn't initialized")
	}
	if mm.msocket == nil {
		return fmt.Errorf("module socket didn't initialize")
	}
	if mm.cnt == nil {
		return fmt.Errorf("controller didn't initialized")
	}

	receiver := mm.msocket.GetReceiver()
	if receiver != nil {
		receiver <- &vxproto.Packet{
			PType: vxproto.PTControl,
			Payload: &vxproto.ControlMessage{
				MsgType: vxproto.StopModule,
			},
		}
	}
	mm.wgReceiver.Wait()

	if err := mm.cnt.Close(); err != nil {
		return fmt.Errorf("modules didn't stop. Error: %w", err)
	}

	if !mm.proto.DelModule(mm.msocket) {
		return fmt.Errorf("failed to delete module socket")
	}

	mm.upgradeTaskConsumer.Close(stopCtx)

	stopSpan.End()
	obs.Observer.Flush(context.Background())

	return mm.proto.Close(ctx)
}
