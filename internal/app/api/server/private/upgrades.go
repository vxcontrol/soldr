package private

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	"soldr/internal/app/api/models"
	srverrors "soldr/internal/app/api/server/errors"
	"soldr/internal/app/api/utils"
	"soldr/internal/storage"
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
	db *gorm.DB
}

func NewUpgradeService(db *gorm.DB) *UpgradeService {
	return &UpgradeService{
		db: db,
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
		err   error
		iDB   *gorm.DB
		query utils.TableQuery
		resp  upgradesAgents
	)

	if err = c.ShouldBindQuery(&query); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding query")
		utils.HTTPError(c, srverrors.ErrGetAgentsUpgradesInvalidRequest, err)
		return
	}

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	query.Init("upgrade_tasks", upgradesAgentsSQLMappers)

	if resp.Total, err = query.Query(iDB, &resp.Tasks); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding agents upgrades")
		utils.HTTPError(c, srverrors.ErrInternal, err)
		return
	}

	for i := 0; i < len(resp.Tasks); i++ {
		if err = resp.Tasks[i].Valid(); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error validating agents upgrades data '%d'", resp.Tasks[i].ID)
			utils.HTTPError(c, srverrors.ErrGetAgentsUpgradesInvalidData, err)
			return
		}
	}

	utils.HTTPSuccess(c, http.StatusOK, resp)
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
		err        error
		iDB        *gorm.DB
		s3         storage.IStorage
		sv         *models.Service
		query      utils.TableQuery
		upgradeReq upgradesAgentsAction
		upgradeRes upgradesAgentsActionResult
		uafArr     = []utils.UserActionFields{
			{
				Domain:            "agent",
				ObjectType:        "agent",
				ActionCode:        "version update task creation",
				ObjectDisplayName: utils.UnknownObjectDisplayName,
			},
		}
	)

	if err = c.ShouldBindJSON(&upgradeReq); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding JSON")
		utils.HTTPErrorWithUAFieldsSlice(c, srverrors.ErrCreateAgentsUpgradesInvalidRequest, err, uafArr)
		return
	}

	iDB = utils.GetGormDB(c, "iDB")
	if iDB == nil {
		utils.HTTPErrorWithUAFieldsSlice(c, srverrors.ErrInternalDBNotFound, nil, uafArr)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPErrorWithUAFieldsSlice(c, srverrors.ErrInternalServiceNotFound, nil, uafArr)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")
	batchRaw := make([]byte, 32)
	if _, err := rand.Read(batchRaw); err != nil {
		utils.FromContext(c).WithError(err).Errorf("failed to get random batch id")
		utils.HTTPErrorWithUAFieldsSlice(c, srverrors.ErrInternal, err, uafArr)
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
		utils.FromContext(c).WithError(nil).Errorf("error getting binary info by version '%s', record not found", upgradeReq.Version)
		utils.HTTPErrorWithUAFieldsSlice(c, srverrors.ErrCreateAgentsUpgradesAgentNotFound, err, uafArr)
		return
	} else if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error getting binary info by version '%s'", upgradeReq.Version)
		utils.HTTPErrorWithUAFieldsSlice(c, srverrors.ErrInternal, err, uafArr)
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
		utils.HTTPErrorWithUAFieldsSlice(c, srverrors.ErrPatchAgentsInvalidQuery, err, uafArr)
	}
	uafArr = fillAgentUserActionFields(agents, "version update task creation")

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
		utils.HTTPErrorWithUAFieldsSlice(c, srverrors.ErrCreateAgentsUpgradesCreateTaskFail, err, uafArr)
		return
	}

	if sqlInsertResult.RowsAffected != 0 {
		s3, err = storage.NewS3(sv.Info.S3.ToS3ConnParams())
		if err != nil {
			utils.FromContext(c).WithError(err).Errorf("error openning connection to S3")
			utils.HTTPErrorWithUAFieldsSlice(c, srverrors.ErrInternal, err, uafArr)
			return
		}

		if err = utils.UploadAgentBinariesToInstBucket(binary, s3); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error uploading agent binaries to S3 instance bucket")
			utils.HTTPErrorWithUAFieldsSlice(c, srverrors.ErrCreateAgentsUpgradesUpdateAgentBinariesFail, err, uafArr)
			return
		}
	}

	upgradeRes.Total = sqlInsertResult.RowsAffected
	utils.HTTPSuccessWithUAFieldsSlice(c, http.StatusCreated, upgradeRes, uafArr)
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
		err  error
		iDB  *gorm.DB
		hash = c.Param("hash")
		resp upgradeAgent
	)

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
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
		utils.HTTPError(c, srverrors.ErrGetLastAgentUpgradeLastUpgradeNotFound, err)
		return
	} else if err = resp.Task.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating last agent upgrade data '%d'", resp.Task.ID)
		utils.HTTPError(c, srverrors.ErrGetLastAgentUpgradeInvalidLastData, err)
		return
	}

	if err = iDB.Take(&resp.Details.Agent, "id = ?", resp.Task.AgentID).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding agent")
		utils.HTTPError(c, srverrors.ErrGetLastAgentUpgradeAgentNotFound, err)
		return
	} else if err = resp.Details.Agent.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating agent data '%s'", resp.Details.Agent.Hash)
		utils.HTTPError(c, srverrors.ErrGetLastAgentUpgradeInvalidAgentData, err)
		return
	}

	if resp.Details.Agent.GroupID != 0 {
		resp.Details.Group = &models.Group{}
		if err = iDB.Take(resp.Details.Group, "id = ?", resp.Details.Agent.GroupID).Error; err != nil {
			utils.FromContext(c).WithError(err).Errorf("error finding group")
			utils.HTTPError(c, srverrors.ErrGetLastAgentUpgradeGroupNotFound, err)
			return
		} else if err = resp.Details.Group.Valid(); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error validating group data '%s'", resp.Details.Group.Hash)
			utils.HTTPError(c, srverrors.ErrGetLastAgentUpgradeInvalidGroupData, err)
			return
		}
	}

	utils.HTTPSuccess(c, http.StatusOK, resp)
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
		err    error
		iDB    *gorm.DB
		hash   = c.Param("hash")
		s3     storage.IStorage
		sv     *models.Service
		task   models.AgentUpgradeTask
	)
	uaf := utils.UserActionFields{
		Domain:            "agent",
		ObjectType:        "agent",
		ActionCode:        "undefined action",
		ObjectId:          hash,
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	if err = c.ShouldBindJSON(&task); err != nil || task.Valid() != nil {
		if err == nil {
			err = task.Valid()
		}
		name, nameErr := utils.GetAgentName(c, hash)
		if nameErr == nil {
			uaf.ObjectDisplayName = name
		}
		utils.FromContext(c).WithError(err).Errorf("error binding JSON")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchLastAgentUpgradeInvalidAgentUpgradeInfo, err, uaf)
		return
	}
	if task.Status == "failed" && task.Reason == "Canceled.By.User" {
		uaf.ActionCode = "version update task cancellation"
	} else {
		uaf.ActionCode = "version update task creation"
	}

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBNotFound, nil, uaf)
		return
	}

	if err = iDB.Take(&agent, "id = ?", task.AgentID).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding agent")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchLastAgentUpgradeAgentNotFound, err, uaf)
		return
	} else if err = agent.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating agent data '%s'", agent.Hash)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchLastAgentUpgradeInvalidAgentData, err, uaf)
		return
	} else if hash != agent.Hash {
		utils.FromContext(c).WithError(nil).Errorf("mismatch agent hash to requested one")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchLastAgentUpgradeInvalidAgentUpgradeInfo, err, uaf)
		return
	}
	uaf.ObjectDisplayName = agent.Description

	if sv = getService(c); sv == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalServiceNotFound, nil, uaf)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")

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
			utils.FromContext(c).WithError(nil).Errorf("error getting binary info by version '%s', record not found", task.Version)
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchLastAgentUpgradeAgentBinaryNotFound, err, uaf)
			return
		} else if err != nil {
			utils.FromContext(c).WithError(err).Errorf("error getting binary info by version '%s'", task.Version)
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
			return
		}

		s3, err = storage.NewS3(sv.Info.S3.ToS3ConnParams())
		if err != nil {
			utils.FromContext(c).WithError(err).Errorf("error openning connection to S3")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
			return
		}

		if err = utils.UploadAgentBinariesToInstBucket(binary, s3); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error uploading agent binaries to S3 instance bucket")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchLastAgentUpgradeUpdateAgentBinariesFail, err, uaf)
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
		utils.FromContext(c).WithError(nil).Errorf("error updating last agent upgrade information by id '%d', task not found", task.ID)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchLastAgentUpgradeLastUpgradeInfoNotFound, err, uaf)
		return
	} else if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error updating last agent upgrade information by id '%d'", task.ID)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		return
	}

	utils.HTTPSuccessWithUAFields(c, http.StatusOK, task, uaf)
}
