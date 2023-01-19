package private

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	"soldr/pkg/app/api/client"
	"soldr/pkg/app/api/models"
	srvcontext "soldr/pkg/app/api/server/context"
	"soldr/pkg/app/api/server/response"
	useraction "soldr/pkg/app/api/user_action"
	"soldr/pkg/app/api/utils"
	"soldr/pkg/storage"
)

type upgradeAgentDetails struct {
	Agent models.Agent  `json:"agent,omitempty"`
	Group *models.Group `json:"group,omitempty"`
}

type upgradesAgentsAction struct {
	Filters []utils.TableFilter `form:"filters" json:"filters" binding:"omitempty"`
	Version string              `form:"version" json:"version" binding:"required"`
}

type upgradesAgentsActionResult struct {
	Batch string `json:"batch"`
	Total int64  `json:"total"`
}

type upgradesAgents struct {
	Tasks []models.AgentUpgradeTask `json:"tasks"`
	Total uint64                    `json:"total"`
}

type upgradeAgent struct {
	Task    models.AgentUpgradeTask `json:"task"`
	Details upgradeAgentDetails     `json:"details"`
}

var upgradesAgentsSQLMappers = map[string]interface{}{
	"id":       "`{{table}}`.id",
	"agent_id": "`{{table}}`.agent_id",
	"batch":    "`{{table}}`.batch",
	"status":   "`{{table}}`.status",
	"version":  "`{{table}}`.version",
}

type UpgradeService struct {
	db               *gorm.DB
	serverConnector  *client.AgentServerClient
	userActionWriter useraction.Writer
}

func NewUpgradeService(
	db *gorm.DB,
	serverConnector *client.AgentServerClient,
	userActionWriter useraction.Writer,
) *UpgradeService {
	return &UpgradeService{
		db:               db,
		serverConnector:  serverConnector,
		userActionWriter: userActionWriter,
	}
}

// GetUpgradesAgents is a function to return agents upgrades list
// @Summary Retrieve agents upgrades list
// @Tags Upgrades,Agents
// @Produce json
// @Param request query utils.TableQuery true "query table params"
// @Success 200 {object} utils.successResp{data=upgradesAgents} "agents upgrades list received successful"
// @Failure 400 {object} utils.errorResp "invalid query request data"
// @Failure 403 {object} utils.errorResp "getting agents upgrades not permitted"
// @Failure 500 {object} utils.errorResp "internal error on getting agents upgrades"
// @Router /upgrades/agents [get]
func (s *UpgradeService) GetAgentsUpgrades(c *gin.Context) {
	var (
		query utils.TableQuery
		resp  upgradesAgents
	)

	if err := c.ShouldBindQuery(&query); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding query")
		response.Error(c, response.ErrGetAgentsUpgradesInvalidRequest, err)
		return
	}

	serviceHash, ok := srvcontext.GetString(c, "svc")
	if !ok {
		utils.FromContext(c).Errorf("could not get service hash")
		response.Error(c, response.ErrInternal, nil)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		utils.FromContext(c).WithError(err).Error()
		response.Error(c, response.ErrInternalDBNotFound, err)
		return
	}

	query.Init("upgrade_tasks", upgradesAgentsSQLMappers)

	if resp.Total, err = query.Query(iDB, &resp.Tasks); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding agents upgrades")
		response.Error(c, response.ErrInternal, err)
		return
	}

	for i := 0; i < len(resp.Tasks); i++ {
		if err = resp.Tasks[i].Valid(); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error validating agents upgrades data '%d'", resp.Tasks[i].ID)
			response.Error(c, response.ErrGetAgentsUpgradesInvalidData, err)
			return
		}
	}

	response.Success(c, http.StatusOK, resp)
}

// CreateUpgradesAgents is a function to request agents upgrades to a specific verison
// @Summary Upgrade agents to a specific version
// @Tags Upgrades,Agents
// @Accept json
// @Produce json
// @Param json body upgradesAgentsAction true "action on agents as JSON data"
// @Success 201 {object} utils.successResp{data=upgradesAgentsActionResult} "agents upgrade requested succesful"
// @Failure 400 {object} utils.errorResp "invalid agents upgrade request"
// @Failure 403 {object} utils.errorResp "upgrading agents not permitted"
// @Failure 404 {object} utils.errorResp "agent binary file not found"
// @Failure 500 {object} utils.errorResp "internal error on requesting agents to upgrade"
// @Router /upgrades/agents [post]
func (s *UpgradeService) CreateAgentsUpgrades(c *gin.Context) {
	var (
		binary     models.Binary
		sv         *models.Service
		query      utils.TableQuery
		upgradeReq upgradesAgentsAction
		upgradeRes upgradesAgentsActionResult
	)

	tStart := time.Now()

	uaf := useraction.NewFields(c, "agent", "agent", "version update task creation", "", useraction.UnknownObjectDisplayName)
	uafArr := []useraction.Fields{uaf}
	defer func() {
		for i := range uafArr {
			s.userActionWriter.WriteUserAction(c, uafArr[i])
		}
	}()

	if err := c.ShouldBindJSON(&upgradeReq); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding JSON")
		response.Error(c, response.ErrCreateAgentsUpgradesInvalidRequest, err)
		return
	}

	serviceHash, ok := srvcontext.GetString(c, "svc")
	if !ok {
		utils.FromContext(c).Errorf("could not get service hash")
		response.Error(c, response.ErrInternal, nil)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		utils.FromContext(c).WithError(err).Error()
		response.Error(c, response.ErrInternalDBNotFound, err)
		return
	}

	if sv = getService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	tid, _ := srvcontext.GetUint64(c, "tid")
	batchRaw := make([]byte, 32)
	if _, err := rand.Read(batchRaw); err != nil {
		utils.FromContext(c).WithError(err).Errorf("failed to get random batch id")
		response.Error(c, response.ErrInternal, err)
		return
	}
	batchHash := md5.Sum(batchRaw)
	upgradeRes.Batch = hex.EncodeToString(batchHash[:])

	scope := func(db *gorm.DB) *gorm.DB {
		db = db.Where("tenant_id IN (?)", []uint64{0, tid}).Where("type LIKE ?", "vxagent")
		if upgradeReq.Version == "latest" {
			return db.Order("ver_major DESC, ver_minor DESC, ver_patch DESC, ver_build DESC")
		} else {
			return db.Where("version LIKE ?", upgradeReq.Version)
		}
	}
	err = s.db.Scopes(scope).Model(&binary).Take(&binary).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		utils.FromContext(c).Errorf("error getting binary info by version '%s', record not found", upgradeReq.Version)
		response.Error(c, response.ErrCreateAgentsUpgradesAgentNotFound, err)
		return
	} else if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error getting binary info by version '%s'", upgradeReq.Version)
		response.Error(c, response.ErrInternal, err)
		return
	}

	query.Init("agents", agentsSQLMappers)
	query.Filters = upgradeReq.Filters
	agentsScope := query.DataFilter()
	var agents []models.Agent
	err = iDB.Scopes(agentsScope).Model(&models.Agent{}).
		Where("version NOT LIKE ? AND id NOT IN (?)", upgradeReq.Version, iDB.
			Model(&models.AgentUpgradeTask{}).
			Select("agent_id").
			Where("status IN (?)", []string{"new", "running"}).
			SubQuery()).Find(&agents).Error
	if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error collecting agents by filter")
		response.Error(c, response.ErrPatchAgentsInvalidQuery, err)
	}

	uafArr = fillAgentUserActionFields(c, agents, "version update task creation", tStart)

	sqlInsertResult := iDB.Exec("INSERT INTO upgrade_tasks(agent_id, version, batch) ?", iDB.
		Scopes(agentsScope).
		Model(&models.Agent{}).
		Select("id AS agent_id, ? AS version, ? AS batch", upgradeReq.Version, upgradeRes.Batch).
		Where("version NOT LIKE ? AND id NOT IN (?)", upgradeReq.Version, iDB.
			Model(&models.AgentUpgradeTask{}).
			Select("agent_id").
			Where("status IN (?)", []string{"new", "running"}).
			SubQuery()).
		QueryExpr())

	if err = sqlInsertResult.Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error inserting new upgrade tasks into the table")
		response.Error(c, response.ErrCreateAgentsUpgradesCreateTaskFail, err)
		return
	}

	if sqlInsertResult.RowsAffected != 0 {
		s3, err := storage.NewS3(sv.Info.S3.ToS3ConnParams())
		if err != nil {
			utils.FromContext(c).WithError(err).Errorf("error openning connection to S3")
			response.Error(c, response.ErrInternal, err)
			return
		}

		if err = utils.UploadAgentBinariesToInstBucket(binary, s3); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error uploading agent binaries to S3 instance bucket")
			response.Error(c, response.ErrCreateAgentsUpgradesUpdateAgentBinariesFail, err)
			return
		}
	}

	upgradeRes.Total = sqlInsertResult.RowsAffected
	response.Success(c, http.StatusCreated, upgradeRes)
}

// GetLastUpgradeAgent is a function to return last agent upgrade information
// @Summary Retrieve last agent upgrade information
// @Tags Upgrades,Agents
// @Produce json
// @Param hash path string true "agent hash in hex format (md5)" minlength(32) maxlength(32)
// @Success 200 {object} utils.successResp{data=upgradeAgent} "last agent upgrade information received successful"
// @Failure 400 {object} utils.errorResp "invalid query request data"
// @Failure 403 {object} utils.errorResp "getting last agent upgrade information not permitted"
// @Failure 404 {object} utils.errorResp "agent or group or task not found"
// @Failure 500 {object} utils.errorResp "internal error on getting last agent upgrade information"
// @Router /upgrades/agents/{hash}/last [get]
func (s *UpgradeService) GetLastAgentUpgrade(c *gin.Context) {
	var (
		hash = c.Param("hash")
		resp upgradeAgent
	)

	serviceHash, ok := srvcontext.GetString(c, "svc")
	if !ok {
		utils.FromContext(c).Errorf("could not get service hash")
		response.Error(c, response.ErrInternal, nil)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		utils.FromContext(c).WithError(err).Error()
		response.Error(c, response.ErrInternalDBNotFound, err)
		return
	}

	timeExpr := gorm.Expr("`upgrade_tasks`.last_upgrade > NOW() - INTERVAL 1 DAY")
	statusExpr := gorm.Expr("`upgrade_tasks`.status IN ('new', 'running')")
	versionExpr := gorm.Expr("`upgrade_tasks`.status LIKE 'failed' AND `upgrade_tasks`.version LIKE `agents`.version")
	sqlQueryResult := iDB.
		Joins("JOIN agents ON agents.id = agent_id").
		Order("id DESC").
		Take(&resp.Task, "`agents`.hash = ? AND NOT (?) AND (? OR ?)", hash, versionExpr, statusExpr, timeExpr)

	if err = sqlQueryResult.Error; err != nil || sqlQueryResult.RowsAffected == 0 {
		utils.FromContext(c).WithError(err).Errorf("error finding last agent upgrade information")
		response.Error(c, response.ErrGetLastAgentUpgradeLastUpgradeNotFound, err)
		return
	} else if err = resp.Task.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating last agent upgrade data '%d'", resp.Task.ID)
		response.Error(c, response.ErrGetLastAgentUpgradeInvalidLastData, err)
		return
	}

	if err = iDB.Take(&resp.Details.Agent, "id = ?", resp.Task.AgentID).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding agent")
		response.Error(c, response.ErrGetLastAgentUpgradeAgentNotFound, err)
		return
	} else if err = resp.Details.Agent.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating agent data '%s'", resp.Details.Agent.Hash)
		response.Error(c, response.ErrGetLastAgentUpgradeInvalidAgentData, err)
		return
	}

	if resp.Details.Agent.GroupID != 0 {
		resp.Details.Group = &models.Group{}
		if err = iDB.Take(resp.Details.Group, "id = ?", resp.Details.Agent.GroupID).Error; err != nil {
			utils.FromContext(c).WithError(err).Errorf("error finding group")
			response.Error(c, response.ErrGetLastAgentUpgradeGroupNotFound, err)
			return
		} else if err = resp.Details.Group.Valid(); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error validating group data '%s'", resp.Details.Group.Hash)
			response.Error(c, response.ErrGetLastAgentUpgradeInvalidGroupData, err)
			return
		}
	}

	response.Success(c, http.StatusOK, resp)
}

// PatchLastAgentUpgrade is a function to update last agent upgrade information
// @Summary Update last agent upgrade information
// @Tags Upgrades,Agents
// @Accept json
// @Produce json
// @Param hash path string true "agent hash in hex format (md5)" minlength(32) maxlength(32)
// @Param json body models.AgentUpgradeTask true "agent info as JSON data"
// @Success 200 {object} utils.successResp{data=models.AgentUpgradeTask} "last agent upgrade information updated successful"
// @Failure 400 {object} utils.errorResp "invalid last agent upgrade information"
// @Failure 403 {object} utils.errorResp "updating last agent upgrade information not permitted"
// @Failure 404 {object} utils.errorResp "agent or group or task not found"
// @Failure 500 {object} utils.errorResp "internal error on updating last agent upgrade information"
// @Router /upgrades/agents/{hash}/last [put]
func (s *UpgradeService) PatchLastAgentUpgrade(c *gin.Context) {
	var (
		agent  models.Agent
		binary models.Binary
		hash   = c.Param("hash")
		sv     *models.Service
		task   models.AgentUpgradeTask
	)

	uaf := useraction.NewFields(c, "agent", "agent", "undefined action", hash, useraction.UnknownObjectDisplayName)
	defer s.userActionWriter.WriteUserAction(c, uaf)

	serviceHash, ok := srvcontext.GetString(c, "svc")
	if !ok {
		utils.FromContext(c).Errorf("could not get service hash")
		response.Error(c, response.ErrInternal, nil)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		utils.FromContext(c).WithError(err).Error()
		response.Error(c, response.ErrInternalDBNotFound, err)
		return
	}

	if err := c.ShouldBindJSON(&task); err != nil || task.Valid() != nil {
		if err == nil {
			err = task.Valid()
		}
		name, nameErr := utils.GetAgentName(iDB, hash)
		if nameErr == nil {
			uaf.ObjectDisplayName = name
		}
		utils.FromContext(c).WithError(err).Errorf("error binding JSON")
		response.Error(c, response.ErrPatchLastAgentUpgradeInvalidAgentUpgradeInfo, err)
		return
	}
	if task.Status == "failed" && task.Reason == "Canceled.By.User" {
		uaf.ActionCode = "version update task cancellation"
	} else {
		uaf.ActionCode = "version update task creation"
	}

	if err = iDB.Take(&agent, "id = ?", task.AgentID).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding agent")
		response.Error(c, response.ErrPatchLastAgentUpgradeAgentNotFound, err)
		return
	} else if err = agent.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating agent data '%s'", agent.Hash)
		response.Error(c, response.ErrPatchLastAgentUpgradeInvalidAgentData, err)
		return
	} else if hash != agent.Hash {
		utils.FromContext(c).Errorf("mismatch agent hash to requested one")
		response.Error(c, response.ErrPatchLastAgentUpgradeInvalidAgentUpgradeInfo, err)
		return
	}
	uaf.ObjectDisplayName = agent.Description

	if sv = getService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	tid, _ := srvcontext.GetUint64(c, "tid")

	if task.Status == "new" {
		scope := func(db *gorm.DB) *gorm.DB {
			db = db.Where("tenant_id IN (?)", []uint64{0, tid}).Where("type LIKE ?", "vxagent")
			if task.Version == "latest" {
				return db.Order("ver_major DESC, ver_minor DESC, ver_patch DESC, ver_build DESC")
			} else {
				return db.Where("version LIKE ?", task.Version)
			}
		}
		err = s.db.Scopes(scope).Model(&binary).Take(&binary).Error
		if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
			utils.FromContext(c).Errorf("error getting binary info by version '%s', record not found", task.Version)
			response.Error(c, response.ErrPatchLastAgentUpgradeAgentBinaryNotFound, err)
			return
		} else if err != nil {
			utils.FromContext(c).WithError(err).Errorf("error getting binary info by version '%s'", task.Version)
			response.Error(c, response.ErrInternal, err)
			return
		}

		s3, err := storage.NewS3(sv.Info.S3.ToS3ConnParams())
		if err != nil {
			utils.FromContext(c).WithError(err).Errorf("error openning connection to S3")
			response.Error(c, response.ErrInternal, err)
			return
		}

		if err = utils.UploadAgentBinariesToInstBucket(binary, s3); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error uploading agent binaries to S3 instance bucket")
			response.Error(c, response.ErrPatchLastAgentUpgradeUpdateAgentBinariesFail, err)
			return
		}
	}

	var reason interface{} = gorm.Expr("NULL")
	if task.Reason != "" {
		reason = task.Reason
	}
	update_info := map[string]interface{}{
		"status": task.Status,
		"reason": reason,
	}
	err = iDB.Model(&task).UpdateColumns(update_info).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		utils.FromContext(c).Errorf("error updating last agent upgrade information by id '%d', task not found", task.ID)
		response.Error(c, response.ErrPatchLastAgentUpgradeLastUpgradeInfoNotFound, err)
		return
	} else if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error updating last agent upgrade information by id '%d'", task.ID)
		response.Error(c, response.ErrInternal, err)
		return
	}

	response.Success(c, http.StatusOK, task)
}
