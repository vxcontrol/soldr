package private

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	"soldr/pkg/app/api/client"
	"soldr/pkg/app/api/logger"
	"soldr/pkg/app/api/models"
	"soldr/pkg/app/api/modules"
	"soldr/pkg/app/api/server/response"
	"soldr/pkg/app/api/storage"
	"soldr/pkg/app/api/useraction"
	"soldr/pkg/filestorage"
	"soldr/pkg/filestorage/s3"
)

type upgradeAgentDetails struct {
	Agent models.Agent  `json:"agent,omitempty"`
	Group *models.Group `json:"group,omitempty"`
}

type upgradesAgentsAction struct {
	Filters []storage.TableFilter `form:"filters" json:"filters" binding:"omitempty"`
	Version string                `form:"version" json:"version" binding:"required"`
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
// @Param request query storage.TableQuery true "query table params"
// @Success 200 {object} response.successResp{data=upgradesAgents} "agents upgrades list received successful"
// @Failure 400 {object} response.errorResp "invalid query request data"
// @Failure 403 {object} response.errorResp "getting agents upgrades not permitted"
// @Failure 500 {object} response.errorResp "internal error on getting agents upgrades"
// @Router /upgrades/agents [get]
func (s *UpgradeService) GetAgentsUpgrades(c *gin.Context) {
	var (
		query storage.TableQuery
		resp  upgradesAgents
	)

	if err := c.ShouldBindQuery(&query); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error binding query")
		response.Error(c, response.ErrGetAgentsUpgradesInvalidRequest, err)
		return
	}

	serviceHash := c.GetString("svc")
	if serviceHash == "" {
		logger.FromContext(c).Errorf("could not get service hash")
		response.Error(c, response.ErrInternal, nil)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		logger.FromContext(c).WithError(err).Error()
		response.Error(c, response.ErrInternalDBNotFound, err)
		return
	}

	query.Init("upgrade_tasks", upgradesAgentsSQLMappers)

	if resp.Total, err = query.Query(iDB, &resp.Tasks); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding agents upgrades")
		response.Error(c, response.ErrInternal, err)
		return
	}

	for i := 0; i < len(resp.Tasks); i++ {
		if err = resp.Tasks[i].Valid(); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error validating agents upgrades data '%d'", resp.Tasks[i].ID)
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
// @Success 201 {object} response.successResp{data=upgradesAgentsActionResult} "agents upgrade requested succesful"
// @Failure 400 {object} response.errorResp "invalid agents upgrade request"
// @Failure 403 {object} response.errorResp "upgrading agents not permitted"
// @Failure 404 {object} response.errorResp "agent binary file not found"
// @Failure 500 {object} response.errorResp "internal error on requesting agents to upgrade"
// @Router /upgrades/agents [post]
func (s *UpgradeService) CreateAgentsUpgrades(c *gin.Context) {
	var (
		binary     models.Binary
		sv         *models.Service
		query      storage.TableQuery
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
		logger.FromContext(c).WithError(err).Errorf("error binding JSON")
		response.Error(c, response.ErrCreateAgentsUpgradesInvalidRequest, err)
		return
	}

	serviceHash := c.GetString("svc")
	if serviceHash == "" {
		logger.FromContext(c).Errorf("could not get service hash")
		response.Error(c, response.ErrInternal, nil)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		logger.FromContext(c).WithError(err).Error()
		response.Error(c, response.ErrInternalDBNotFound, err)
		return
	}

	if sv = modules.GetService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	tid := c.GetUint64("tid")
	batchRaw := make([]byte, 32)
	if _, err := rand.Read(batchRaw); err != nil {
		logger.FromContext(c).WithError(err).Errorf("failed to get random batch id")
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
		logger.FromContext(c).Errorf("error getting binary info by version '%s', record not found", upgradeReq.Version)
		response.Error(c, response.ErrCreateAgentsUpgradesAgentNotFound, err)
		return
	} else if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error getting binary info by version '%s'", upgradeReq.Version)
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
		logger.FromContext(c).WithError(err).Errorf("error collecting agents by filter")
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
		logger.FromContext(c).WithError(err).Errorf("error inserting new upgrade tasks into the table")
		response.Error(c, response.ErrCreateAgentsUpgradesCreateTaskFail, err)
		return
	}

	if sqlInsertResult.RowsAffected != 0 {
		s3Client, err := s3.New(sv.Info.S3.ToS3ConnParams())
		if err != nil {
			logger.FromContext(c).WithError(err).Errorf("error openning connection to RemoteStorage")
			response.Error(c, response.ErrInternal, err)
			return
		}

		if err = uploadAgentBinariesToInstBucket(binary, s3Client); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error uploading agent binaries to RemoteStorage instance bucket")
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
// @Success 200 {object} response.successResp{data=upgradeAgent} "last agent upgrade information received successful"
// @Failure 400 {object} response.errorResp "invalid query request data"
// @Failure 403 {object} response.errorResp "getting last agent upgrade information not permitted"
// @Failure 404 {object} response.errorResp "agent or group or task not found"
// @Failure 500 {object} response.errorResp "internal error on getting last agent upgrade information"
// @Router /upgrades/agents/{hash}/last [get]
func (s *UpgradeService) GetLastAgentUpgrade(c *gin.Context) {
	var (
		hash = c.Param("hash")
		resp upgradeAgent
	)

	serviceHash := c.GetString("svc")
	if serviceHash == "" {
		logger.FromContext(c).Errorf("could not get service hash")
		response.Error(c, response.ErrInternal, nil)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		logger.FromContext(c).WithError(err).Error()
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
		logger.FromContext(c).WithError(err).Errorf("error finding last agent upgrade information")
		response.Error(c, response.ErrGetLastAgentUpgradeLastUpgradeNotFound, err)
		return
	} else if err = resp.Task.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating last agent upgrade data '%d'", resp.Task.ID)
		response.Error(c, response.ErrGetLastAgentUpgradeInvalidLastData, err)
		return
	}

	if err = iDB.Take(&resp.Details.Agent, "id = ?", resp.Task.AgentID).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding agent")
		response.Error(c, response.ErrGetLastAgentUpgradeAgentNotFound, err)
		return
	} else if err = resp.Details.Agent.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating agent data '%s'", resp.Details.Agent.Hash)
		response.Error(c, response.ErrGetLastAgentUpgradeInvalidAgentData, err)
		return
	}

	if resp.Details.Agent.GroupID != 0 {
		resp.Details.Group = &models.Group{}
		if err = iDB.Take(resp.Details.Group, "id = ?", resp.Details.Agent.GroupID).Error; err != nil {
			logger.FromContext(c).WithError(err).Errorf("error finding group")
			response.Error(c, response.ErrGetLastAgentUpgradeGroupNotFound, err)
			return
		} else if err = resp.Details.Group.Valid(); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error validating group data '%s'", resp.Details.Group.Hash)
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
// @Success 200 {object} response.successResp{data=models.AgentUpgradeTask} "last agent upgrade information updated successful"
// @Failure 400 {object} response.errorResp "invalid last agent upgrade information"
// @Failure 403 {object} response.errorResp "updating last agent upgrade information not permitted"
// @Failure 404 {object} response.errorResp "agent or group or task not found"
// @Failure 500 {object} response.errorResp "internal error on updating last agent upgrade information"
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

	serviceHash := c.GetString("svc")
	if serviceHash == "" {
		logger.FromContext(c).Errorf("could not get service hash")
		response.Error(c, response.ErrInternal, nil)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		logger.FromContext(c).WithError(err).Error()
		response.Error(c, response.ErrInternalDBNotFound, err)
		return
	}

	if err := c.ShouldBindJSON(&task); err != nil || task.Valid() != nil {
		if err == nil {
			err = task.Valid()
		}
		name, nameErr := storage.GetAgentName(iDB, hash)
		if nameErr == nil {
			uaf.ObjectDisplayName = name
		}
		logger.FromContext(c).WithError(err).Errorf("error binding JSON")
		response.Error(c, response.ErrPatchLastAgentUpgradeInvalidAgentUpgradeInfo, err)
		return
	}
	if task.Status == "failed" && task.Reason == "Canceled.By.User" {
		uaf.ActionCode = "version update task cancellation"
	} else {
		uaf.ActionCode = "version update task creation"
	}

	if err = iDB.Take(&agent, "id = ?", task.AgentID).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding agent")
		response.Error(c, response.ErrPatchLastAgentUpgradeAgentNotFound, err)
		return
	} else if err = agent.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating agent data '%s'", agent.Hash)
		response.Error(c, response.ErrPatchLastAgentUpgradeInvalidAgentData, err)
		return
	} else if hash != agent.Hash {
		logger.FromContext(c).Errorf("mismatch agent hash to requested one")
		response.Error(c, response.ErrPatchLastAgentUpgradeInvalidAgentUpgradeInfo, err)
		return
	}
	uaf.ObjectDisplayName = agent.Description

	if sv = modules.GetService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	tid := c.GetUint64("tid")

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
			logger.FromContext(c).Errorf("error getting binary info by version '%s', record not found", task.Version)
			response.Error(c, response.ErrPatchLastAgentUpgradeAgentBinaryNotFound, err)
			return
		} else if err != nil {
			logger.FromContext(c).WithError(err).Errorf("error getting binary info by version '%s'", task.Version)
			response.Error(c, response.ErrInternal, err)
			return
		}

		s3Client, err := s3.New(sv.Info.S3.ToS3ConnParams())
		if err != nil {
			logger.FromContext(c).WithError(err).Errorf("error openning connection to RemoteStorage")
			response.Error(c, response.ErrInternal, err)
			return
		}

		if err = uploadAgentBinariesToInstBucket(binary, s3Client); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error uploading agent binaries to RemoteStorage instance bucket")
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
		logger.FromContext(c).Errorf("error updating last agent upgrade information by id '%d', task not found", task.ID)
		response.Error(c, response.ErrPatchLastAgentUpgradeLastUpgradeInfoNotFound, err)
		return
	} else if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error updating last agent upgrade information by id '%d'", task.ID)
		response.Error(c, response.ErrInternal, err)
		return
	}

	response.Success(c, http.StatusOK, task)
}

// uploadAgentBinariesToInstBucket is function to check and upload agent binaries to RemoteStorage instance bucket
func uploadAgentBinariesToInstBucket(binary models.Binary, iS3 filestorage.Storage) error {
	joinPath := func(args ...string) string {
		tpath := filepath.Join(args...)
		return strings.Replace(tpath, "\\", "/", -1)
	}

	gS3, err := s3.New(nil)
	if err != nil {
		return fmt.Errorf("failed to initialize global RemoteStorage driver: %w", err)
	}

	prefix := joinPath("vxagent", binary.Version)
	ifiles, err := gS3.ListDirRec(prefix)
	if err != nil {
		return fmt.Errorf("failed to read info about agent binaries files: %w", err)
	}

	for _, fpath := range binary.Info.Files {
		if _, ok := ifiles[strings.TrimPrefix(fpath, prefix)]; !ok {
			return fmt.Errorf("failed to get agent binary file from global RemoteStorage '%s'", fpath)
		}
	}

	for fpath, finfo := range ifiles {
		ifpath := joinPath(prefix, fpath)
		if finfo.IsDir() || iS3.IsExist(ifpath) {
			continue
		}

		ifdata, err := gS3.ReadFile(ifpath)
		if err != nil {
			return fmt.Errorf("failed to read agent binary file '%s': %w", ifpath, err)
		}

		chksums, ok := binary.Info.Chksums[ifpath]
		if !ok {
			return fmt.Errorf("failed to get check sums of agent binary file '%s'", ifpath)
		}
		if err := validateBinaryFileByChksums(ifdata, chksums); err != nil {
			return fmt.Errorf("failed to check agent binary file '%s': %w", ifpath, err)
		}
		if err := iS3.WriteFile(ifpath, ifdata); err != nil {
			return fmt.Errorf("failed to write agent binary file to RemoteStorage '%s': %w", ifpath, err)
		}

		tpdata, err := json.Marshal(chksums)
		if err != nil {
			return fmt.Errorf("failed to make agent binary thumbprint for '%s': %w", ifpath, err)
		}
		tpfpath := ifpath + ".thumbprint"
		if err := iS3.WriteFile(tpfpath, tpdata); err != nil {
			return fmt.Errorf("failed to write agent binary thumbprint to RemoteStorage '%s': %w", tpfpath, err)
		}
	}

	return nil
}

func validateBinaryFileByChksums(data []byte, chksums models.BinaryChksum) error {
	md5Hash := md5.Sum(data)
	if chksums.MD5 != "" && chksums.MD5 != hex.EncodeToString(md5Hash[:]) {
		return fmt.Errorf("failed to match binary file MD5 hash sum: %s", chksums.MD5)
	}

	sha256Hash := sha256.Sum256(data)
	if chksums.SHA256 != "" && chksums.SHA256 != hex.EncodeToString(sha256Hash[:]) {
		return fmt.Errorf("failed to match binary file SHA256 hash sum: %s", chksums.SHA256)
	}

	return nil
}
