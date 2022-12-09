package private

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/jinzhu/gorm"

	"soldr/internal/app/api/models"
	srverrors "soldr/internal/app/api/server/errors"
	"soldr/internal/app/api/storage/mem"
	"soldr/internal/app/api/utils"
)

type agentDetails struct {
	Hash          string                   `json:"hash"`
	ActiveModules int                      `json:"active_modules"`
	JoinedModules string                   `json:"joined_modules"`
	Consistency   bool                     `json:"consistency"`
	Dependencies  []models.AgentDependency `json:"dependencies"`
	Group         *models.Group            `json:"group,omitempty"`
	UpgradeTask   *models.AgentUpgradeTask `json:"upgrade_task,omitempty"`
	Policies      []models.Policy          `json:"policies,omitempty"`
	Modules       []models.ModuleAShort    `json:"modules,omitempty"`
}

type agents struct {
	Agents  []models.Agent `json:"agents"`
	Details []agentDetails `json:"details"`
	Total   uint64         `json:"total"`
}

type agent struct {
	Agent   models.Agent `json:"agent"`
	Details agentDetails `json:"details"`
}

type agentInfo struct {
	Name string `json:"name" binding:"max=255,required"`
	OS   string `json:"os" binding:"oneof=windows linux darwin,required" default:"linux" enums:"windows,linux,darwin"`
	Arch string `json:"arch" binding:"oneof=386 amd64,required" default:"amd64" enums:"386,amd64"`
}

type agentCount struct {
	All           int `json:"all"`
	Authorized    int `json:"authorized"`
	Blocked       int `json:"blocked"`
	Unauthorized  int `json:"unauthorized"`
	WithoutGroups int `json:"without_groups"`
}

type AgentsAction struct {
	Action  string              `form:"action" json:"action" binding:"oneof=authorize block delete unauthorize move,required" enums:"authorize,block,delete,unauthorize,move"`
	Filters []utils.TableFilter `form:"filters" json:"filters" binding:"omitempty"`
	To      uint64              `form:"to" json:"to" binding:"min=0,numeric,omitempty"`
}

type patchAgentAction struct {
	Action string       `form:"action" json:"action" binding:"oneof=authorize block delete unauthorize move edit,required" enums:"authorize,block,delete,unauthorize,move,edit"`
	Agent  models.Agent `form:"agent" json:"agent" binding:"required"`
}

type agentsActionResult struct {
	Total uint64 `json:"total"`
}

var agentsSQLMappers = map[string]interface{}{
	"id":             "`{{table}}`.id",
	"hash":           "`{{table}}`.hash",
	"group_id":       "`{{table}}`.group_id",
	"group_name":     "JSON_UNQUOTE(JSON_EXTRACT(`groups`.info, '$.name.{{lang}}'))",
	"policy_id":      "`gtp`.policy_id",
	"module_name":    "`modules`.name",
	"description":    "`{{table}}`.description",
	"version":        "`{{table}}`.version",
	"info":           "`{{table}}`.info",
	"status":         "`{{table}}`.status",
	"auth_status":    "`{{table}}`.auth_status",
	"ip":             "`{{table}}`.ip",
	"os":             "CONCAT(`{{table}}`.os_type,':',`{{table}}`.os_arch)",
	"os_arch":        "`{{table}}`.os_arch",
	"os_type":        "`{{table}}`.os_type",
	"os_name":        "`{{table}}`.os_name",
	"hostname":       "`{{table}}`.hostname",
	"connected_date": "`{{table}}`.connected_date",
	"created_date":   "`{{table}}`.created_date",
	"net_ips":        "JSON_EXTRACT(`{{table}}`.info, '$.net.ips')",
	"tags":           utils.TagsMapper,
	"users":          "JSON_EXTRACT(`{{table}}`.info, '$.users')",
	"data": "CONCAT(`{{table}}`.hash, ' | ', " +
		"`{{table}}`.description, ' | ', " +
		"`{{table}}`.status, ' | ', " +
		"`{{table}}`.auth_status, ' | ', " +
		"CASE" +
		"  WHEN `{{table}}`.status = 'connected' THEN 'подключен'" +
		"  WHEN `{{table}}`.status = 'disconnected' THEN 'отключен'" +
		"  ELSE ''" +
		"END, ' | ', " +
		"CASE" +
		"  WHEN `{{table}}`.auth_status = 'authorized' THEN 'авторизован'" +
		"  WHEN `{{table}}`.auth_status = 'unauthorized' THEN 'неавторизован'" +
		"  WHEN `{{table}}`.auth_status = 'blocked' THEN 'заблокирован'" +
		"  ELSE ''" +
		"END, ' | ', " +
		"JSON_EXTRACT(`{{table}}`.info, '$.net.ips'), ' | ', " +
		"JSON_EXTRACT(`{{table}}`.info, '$.tags'), ' | ', " +
		"`{{table}}`.os_type, ' | ', " +
		"`{{table}}`.os_arch, ' | ', " +
		"`{{table}}`.os_name, ' | ', " +
		"`{{table}}`.hostname, ' | ', " +
		"`{{table}}`.version)",
}

const sqlAgentDetails = `
	SELECT a.hash,
		(SELECT COUNT(m.id) FROM modules m
			LEFT JOIN policies p ON m.policy_id = p.id AND p.deleted_at IS NULL
			LEFT JOIN groups_to_policies AS gtp ON p.id = gtp.policy_id
			WHERE gtp.group_id = g.id AND m.status = 'joined' AND
				m.deleted_at IS NULL) AS active_modules,
		(SELECT GROUP_CONCAT(m.name SEPARATOR ',') FROM modules m
			LEFT JOIN policies p ON m.policy_id = p.id AND p.deleted_at IS NULL
			LEFT JOIN groups_to_policies AS gtp ON p.id = gtp.policy_id
			WHERE gtp.group_id = g.id AND m.status = 'joined' AND m.deleted_at IS NULL
			GROUP BY gtp.group_id) AS joined_modules
	FROM agents a
		LEFT JOIN groups g ON g.id = a.group_id AND g.deleted_at IS NULL`

func getAgentConsistency(modules []models.ModuleAShort, agent *models.Agent) (bool, []models.AgentDependency) {
	var (
		rdeps bool = true
		adeps []models.AgentDependency
		gdeps []models.GroupDependency
	)

	rdeps, gdeps = getGroupConsistency(modules)

	for _, gdep := range gdeps {
		adeps = append(adeps, models.AgentDependency{
			GroupDependency: gdep,
		})
	}

	for _, mod := range modules {
		var deps []models.DependencyItem
		deps = append(deps, mod.StaticDependencies...)
		deps = append(deps, mod.DynamicDependencies...)
		for _, dep := range deps {
			if dep.Type != "agent_version" {
				continue
			}

			var sdeps bool
			switch utils.CompareVersions(agent.Version, dep.MinAgentVersion) {
			case utils.TargetVersionEmpty, utils.VersionsEqual, utils.SourceVersionGreat:
				sdeps = true
			default:
				rdeps = false
			}
			adeps = append(adeps, models.AgentDependency{
				GroupDependency: models.GroupDependency{
					PolicyID: mod.PolicyID,
					PolicyDependency: models.PolicyDependency{
						SourceModuleName: mod.Info.Name,
						ModuleDependency: models.ModuleDependency{
							Status:         sdeps,
							DependencyItem: dep,
						},
					},
				},
			})
		}
	}

	return rdeps, adeps
}

func getActionCode(action string) string {
	var actionCode string
	switch action {
	case "authorize":
		actionCode = "authorization"
	case "block":
		actionCode = "blocking"
	case "delete":
		actionCode = "deletion"
	case "unauthorize":
		actionCode = "editing"
	case "move":
		actionCode = "group change"
	case "edit":
		actionCode = "editing"
	case "count":
		actionCode = "counting"
	default:
		actionCode = ""
	}
	return actionCode
}

func fillAgentUserActionFields(agents []models.Agent, actionCode string) []utils.UserActionFields {
	res := make([]utils.UserActionFields, len(agents))
	for i, v := range agents {
		res[i] = utils.UserActionFields{
			Domain:            "agent",
			ObjectType:        "agent",
			ObjectId:          v.Hash,
			ObjectDisplayName: v.Description,
			ActionCode:        actionCode,
		}
	}
	return res
}

type AgentService struct {
	db             *gorm.DB
	serviceDBConns *mem.ServiceDBConnectionStorage
	serviceS3Conns *mem.ServiceS3ConnectionStorage
}

func NewAgentService(
	db *gorm.DB,
	serviceDBConns *mem.ServiceDBConnectionStorage,
	serviceS3Conns *mem.ServiceS3ConnectionStorage,
) *AgentService {
	return &AgentService{
		db:             db,
		serviceDBConns: serviceDBConns,
		serviceS3Conns: serviceS3Conns,
	}
}

// GetAgents is a function to return agent list view on dashboard
// @Summary Retrieve agents list by filters
// @Tags Agents
// @Produce json
// @Param request query utils.TableQuery true "query table params"
// @Success 200 {object} utils.successResp{data=agents} "agents list received successful"
// @Failure 400 {object} utils.errorResp "invalid query request data"
// @Failure 403 {object} utils.errorResp "getting agents not permitted"
// @Failure 500 {object} utils.errorResp "internal error on getting agents"
// @Router /agents/ [get]
func (s *AgentService) GetAgents(c *gin.Context) {
	var (
		aids        []uint64
		err         error
		gids        []uint64
		gpss        []models.GroupToPolicy
		groups      = make(map[uint64]*models.Group)
		groupsa     []models.Group
		iDB         *gorm.DB
		modules     = make(map[uint64][]models.ModuleAShort)
		modulesa    []models.ModuleAShort
		policies    = make(map[uint64][]models.Policy)
		policiesa   []models.Policy
		query       utils.TableQuery
		resp        agents
		groupedResp utils.GroupedData
		tasks       []models.AgentUpgradeTask
		useGroup    bool
		useModule   bool
		usePolicy   bool
	)

	if err = c.ShouldBindQuery(&query); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding query")
		utils.HTTPError(c, srverrors.ErrGetAgentsInvalidRequest, err)
		return
	}

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	if err = query.Init("agents", agentsSQLMappers); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding query")
		utils.HTTPError(c, srverrors.ErrGetAgentsInvalidRequest, err)
		return
	}

	setUsingTables := func(sfield string) {
		if sfield == "policy_id" {
			usePolicy = true
		}
		if sfield == "module_name" {
			usePolicy = true
			useModule = true
		}
		if sfield == "group_name" {
			useGroup = true
		}
	}
	setUsingTables(query.Sort.Prop)
	setUsingTables(query.Group)
	for _, filter := range query.Filters {
		setUsingTables(filter.Field)
	}
	query.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Where("agents.deleted_at IS NULL")
		},
	})
	funcs := []func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			if usePolicy {
				db = db.Joins(`LEFT JOIN groups_to_policies gtp ON gtp.group_id = agents.group_id`)
			}
			if useModule {
				db = db.Joins(`LEFT JOIN modules ON gtp.policy_id = modules.policy_id AND modules.status = 'joined' AND modules.deleted_at IS NULL`)
			}
			if useGroup {
				db = db.Joins(`LEFT JOIN groups ON groups.id = agents.group_id AND groups.deleted_at IS NULL`)
			}
			if usePolicy || useModule || useGroup {
				db = db.Group("agents.id")
			}
			return db
		},
	}

	if query.Group == "" {
		if resp.Total, err = query.Query(iDB, &resp.Agents, funcs...); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error finding agents")
			utils.HTTPError(c, srverrors.ErrGetAgentsInvalidQuery, err)
			return
		}
	} else {
		if groupedResp.Total, err = query.QueryGrouped(iDB, &groupedResp.Grouped, funcs...); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error finding grouped agents")
			utils.HTTPError(c, srverrors.ErrGetAgentsInvalidQuery, err)
			return
		}
		utils.HTTPSuccess(c, http.StatusOK, groupedResp)
		return
	}

	for i := 0; i < len(resp.Agents); i++ {
		aids = append(aids, resp.Agents[i].ID)
		gids = append(gids, resp.Agents[i].GroupID)
		if err = resp.Agents[i].Valid(); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error validating agent data '%s'", resp.Agents[i].Hash)
			utils.HTTPError(c, srverrors.ErrAgentsInvalidData, err)
			return
		}
	}
	if err = iDB.Where("id IN (?)", gids).Find(&groupsa).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding linked groups")
		utils.HTTPError(c, srverrors.ErrGetAgentsInvalidQuery, err)
		return
	}
	for i := 0; i < len(groupsa); i++ {
		if err = groupsa[i].Valid(); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error validating group data '%s'", groupsa[i].Hash)
			utils.HTTPError(c, srverrors.ErrAgentsInvalidData, err)
			return
		}
		groups[groupsa[i].ID] = &groupsa[i]
	}

	sqlQuery := sqlAgentDetails + ` WHERE a.id IN (?) AND a.deleted_at IS NULL`
	if err = iDB.Raw(sqlQuery, aids).Scan(&resp.Details).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error loading details agents")
		utils.HTTPError(c, srverrors.ErrGetAgentsInvalidQuery, err)
		return
	}

	err = iDB.
		Joins("RIGHT JOIN (?) lt ON lt.last_id = upgrade_tasks.id", iDB.
			Model(&models.AgentUpgradeTask{}).
			Select("MAX(id) AS last_id").
			Where("status IN ('new', 'running') OR last_upgrade > NOW() - INTERVAL 1 DAY").
			Where("agent_id IN (?)", aids).
			Group("agent_id").
			QueryExpr()).
		Find(&tasks).Error
	if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding linked agent upgrade tasks")
		utils.HTTPError(c, srverrors.ErrGetAgentsInvalidQuery, err)
		return
	}

	modsToPolicies := make(map[uint64][]models.ModuleAShort)
	err = iDB.Model(&models.ModuleAShort{}).
		Group("modules.id").
		Joins(`LEFT JOIN groups_to_policies gtp ON gtp.policy_id = modules.policy_id`).
		Find(&modulesa, "gtp.group_id IN (?) AND status = 'joined'", gids).Error
	if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding group modules")
		utils.HTTPError(c, srverrors.ErrGetAgentsInvalidQuery, err)
		return
	} else {
		for i := 0; i < len(modulesa); i++ {
			id := modulesa[i].ID
			name := modulesa[i].Info.Name
			policy_id := modulesa[i].PolicyID
			if err = modulesa[i].Valid(); err != nil {
				utils.FromContext(c).WithError(err).Errorf("error validating group module data '%d' '%s'", id, name)
				utils.HTTPError(c, srverrors.ErrAgentsInvalidData, err)
				return
			}
			if mods, ok := modsToPolicies[policy_id]; ok {
				modsToPolicies[policy_id] = append(mods, modulesa[i])
			} else {
				modsToPolicies[policy_id] = []models.ModuleAShort{modulesa[i]}
			}
		}
	}

	if err = iDB.Find(&gpss, "group_id IN (?)", gids).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy to groups links")
		utils.HTTPError(c, srverrors.ErrGroupPolicyGroupsNotFound, err)
		return
	}

	err = iDB.
		Model(&models.Policy{}).
		Group("policies.id").
		Joins(`LEFT JOIN groups_to_policies gtp ON gtp.policy_id = policies.id AND gtp.group_id IN (?)`, gids).
		Find(&policiesa).Error
	if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding group policies")
		utils.HTTPError(c, srverrors.ErrGroupPolicyPoliciesNotFound, err)
		return
	} else {
		for i := 0; i < len(policiesa); i++ {
			id := policiesa[i].ID
			name := policiesa[i].Info.Name
			if err = policiesa[i].Valid(); err != nil {
				utils.FromContext(c).WithError(err).Errorf("error validating policy data '%d' '%s'", id, name)
				utils.HTTPError(c, srverrors.ErrGetAgentsInvalidAgentModuleData, err)
				return
			}
			for idx := range gpss {
				if gpss[idx].PolicyID != id {
					continue
				}
				group_id := gpss[idx].GroupID
				if pols, ok := policies[group_id]; ok {
					policies[group_id] = append(pols, policiesa[i])
				} else {
					policies[group_id] = []models.Policy{policiesa[i]}
				}
				if _, ok := modules[group_id]; !ok {
					modules[group_id] = []models.ModuleAShort{}
				}
				if mods, ok := modsToPolicies[id]; ok {
					modules[group_id] = append(modules[group_id], mods...)
				}
			}
		}
	}

	getAgentByHash := func(hash string) *models.Agent {
		for i := 0; i < len(resp.Agents); i++ {
			if resp.Agents[i].Hash == hash {
				return &resp.Agents[i]
			}
		}
		return nil
	}
	getTasksByAgentID := func(aid uint64) *models.AgentUpgradeTask {
		for i := 0; i < len(tasks); i++ {
			if tasks[i].AgentID == aid {
				return &tasks[i]
			}
		}
		return nil
	}
	for idx := range resp.Details {
		details := &resp.Details[idx]
		agent := getAgentByHash(details.Hash)
		if agent != nil {
			details.Group = groups[agent.GroupID]
			details.Modules = modules[agent.GroupID]
			details.Policies = policies[agent.GroupID]
			details.UpgradeTask = getTasksByAgentID(agent.ID)
		}
		details.Consistency, details.Dependencies = getAgentConsistency(details.Modules, agent)
	}

	utils.HTTPSuccess(c, http.StatusOK, resp)
}

// PatchAgents is a function to update agents public info only by action
// @Summary Update agents public info by action
// @Tags Agents
// @Accept json
// @Produce json
// @Param json body AgentsAction true "action on agents as JSON data"
// @Success 200 {object} utils.successResp{data=agentsActionResult} "agents updated successful"
// @Failure 400 {object} utils.errorResp "invalid agents action"
// @Failure 403 {object} utils.errorResp "updating agents not permitted"
// @Failure 500 {object} utils.errorResp "internal error on updating agents"
// @Router /agents/ [put]
func (s *AgentService) PatchAgents(c *gin.Context) {
	var (
		action AgentsAction
		err    error
		iDB    *gorm.DB
		query  utils.TableQuery
		resp   agentsActionResult
		uafArr = []utils.UserActionFields{
			{
				Domain:            "agent",
				ObjectType:        "agent",
				ActionCode:        "undefined action",
				ObjectDisplayName: utils.UnknownObjectDisplayName,
			},
		}
	)

	if err = c.ShouldBindBodyWith(&action, binding.JSON); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding JSON")
		utils.HTTPErrorWithUAFieldsSlice(c, srverrors.ErrPatchAgentsInvalidAction, err, uafArr)
		return
	}
	uafArr[0].ActionCode = getActionCode(action.Action)

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		utils.HTTPErrorWithUAFieldsSlice(c, srverrors.ErrInternalDBNotFound, nil, uafArr)
		return
	}

	query.Init("agents", agentsSQLMappers)
	query.Filters = action.Filters

	scope := query.DataFilter()

	var agents []models.Agent
	if err = iDB.Scopes(scope).Model(&models.Agent{}).Count(&resp.Total).Find(&agents).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error collecting agents by filter")
		utils.HTTPErrorWithUAFieldsSlice(c, srverrors.ErrPatchAgentsInvalidQuery, err, uafArr)
		return
	}

	uafArr = fillAgentUserActionFields(agents, getActionCode(action.Action))

	updateAuthStatus := func(status string) bool {
		update_fields := map[string]interface{}{
			"auth_status": status,
		}
		if status == "blocked" {
			update_fields["group_id"] = 0
			err = iDB.
				Model(&models.AgentUpgradeTask{}).
				Where("status IN (?)", []string{"new", "running"}).
				Where("agent_id IN (?)", iDB.
					Scopes(scope).
					Model(&models.Agent{}).
					Select("id").
					QueryExpr()).
				UpdateColumns(map[string]interface{}{
					"status": "failed",
					"reason": "Canceled.By.User",
				}).Error
			if err != nil {
				utils.FromContext(c).WithError(err).Errorf("error updating tasks by filter")
				utils.HTTPErrorWithUAFieldsSlice(c, srverrors.ErrPatchAgentsUpdateTasksFail, err, uafArr)
				return false
			}
		}
		err = iDB.Scopes(scope).Model(&models.Agent{}).
			Count(&resp.Total).UpdateColumns(update_fields).Error
		if err != nil {
			utils.FromContext(c).WithError(err).Errorf("error updating agents by filter")
			utils.HTTPErrorWithUAFieldsSlice(c, srverrors.ErrPatchAgentsUpdateAgentsFail, err, uafArr)
			return false
		}
		return true
	}

	switch action.Action {
	case "delete":
		for _, agent := range agents {
			if err = iDB.Delete(&agent).Error; err != nil {
				utils.FromContext(c).WithError(err).Errorf("error deleting agents by filter")
				utils.HTTPErrorWithUAFieldsSlice(c, srverrors.ErrPatchAgentsDeleteAgentsFail, err, uafArr)
				return
			}
		}
	case "block":
		if !updateAuthStatus("blocked") {
			return
		}
	case "authorize":
		if !updateAuthStatus("authorized") {
			return
		}
	case "unauthorize":
		if !updateAuthStatus("unauthorized") {
			return
		}
	case "move":
		if action.To != 0 {
			var group models.Group
			err = iDB.Where("id = ?", action.To).Take(&group).Error
			if err != nil || group.ID == 0 {
				utils.FromContext(c).WithError(err).Errorf("error getting agents group")
				utils.HTTPErrorWithUAFieldsSlice(c, srverrors.ErrPatchAgentsMoveFail, err, uafArr)
				return
			}
		}
		agentIds := make([]uint64, len(agents))
		for i, v := range agents {
			agentIds[i] = v.ID
			if v.AuthStatus == "unauthorized" || v.AuthStatus == "blocked" {
				uafArr = append(uafArr, utils.UserActionFields{
					Domain:            "agent",
					ObjectType:        "agent",
					ObjectId:          v.Hash,
					ObjectDisplayName: v.Description,
					ActionCode:        "authorization",
				})
			}
		}
		err = iDB.Where("agent_id IN (?)", agentIds).
			Delete(&models.Event{}).Error
		if err != nil {
			utils.FromContext(c).WithError(err).Errorf("error deleting agents on moving")
			utils.HTTPErrorWithUAFieldsSlice(c, srverrors.ErrPatchAgentsMoveFail, err, uafArr)
			return
		}
		update_fields := map[string]interface{}{
			"group_id":    action.To,
			"auth_status": "authorized",
			"updated_at":  time.Now(),
		}
		err = iDB.Scopes(query.DataFilter()).Model(&models.Agent{}).Count(&resp.Total).
			UpdateColumns(update_fields).Error
		if err != nil {
			utils.FromContext(c).WithError(err).Errorf("error updating agents on moving")
			utils.HTTPErrorWithUAFieldsSlice(c, srverrors.ErrPatchAgentsMoveFail, err, uafArr)
			return
		}
	}

	utils.HTTPSuccessWithUAFieldsSlice(c, http.StatusOK, resp, uafArr)
}

// GetAgent is a function to return agent info and details view
// @Summary Retrieve agent info by agent hash
// @Tags Agents
// @Produce json
// @Param hash path string true "agent hash in hex format (md5)" minlength(32) maxlength(32)
// @Success 200 {object} utils.successResp{data=agent} "agent info received successful"
// @Failure 403 {object} utils.errorResp "getting agent info not permitted"
// @Failure 404 {object} utils.errorResp "agent not found"
// @Failure 500 {object} utils.errorResp "internal error on getting agent"
// @Router /agents/{hash} [get]
func (s *AgentService) GetAgent(c *gin.Context) {
	var (
		err   error
		group models.Group
		hash  = c.Param("hash")
		iDB   *gorm.DB
		resp  agent
		task  models.AgentUpgradeTask
	)

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	if err = iDB.Take(&resp.Agent, "hash = ?", hash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding agent by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPError(c, srverrors.ErrAgentsNotFound, err)
		} else {
			utils.HTTPError(c, srverrors.ErrInternal, err)
		}
		return
	} else if err = resp.Agent.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating agent data '%s'", resp.Agent.Hash)
		utils.HTTPError(c, srverrors.ErrAgentsInvalidData, err)
		return
	}

	sqlQuery := sqlAgentDetails + ` WHERE a.hash = ? AND a.deleted_at IS NULL`
	if err = iDB.Raw(sqlQuery, hash).Scan(&resp.Details).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error loading details by agent hash '%s'", hash)
		utils.HTTPError(c, srverrors.ErrGetAgentDetailsNotFound, err)
		return
	}

	scopeUpgrade := func(db *gorm.DB) *gorm.DB {
		return db.
			Where("status NOT LIKE 'failed' OR version NOT LIKE ?").
			Where("status IN ('new', 'running') OR last_upgrade > NOW() - INTERVAL 1 DAY").
			Where("agent_id = ?", resp.Agent.Version, resp.Agent.ID)
	}
	if err = iDB.Scopes(scopeUpgrade).Order("id DESC").Take(&task).Error; err == nil {
		resp.Details.UpgradeTask = &task
	}

	if resp.Agent.GroupID != 0 {
		if err = iDB.Take(&group, "id = ?", resp.Agent.GroupID).Error; err != nil {
			utils.FromContext(c).WithError(err).Errorf("error finding group by id")
			if errors.Is(err, gorm.ErrRecordNotFound) {
				utils.HTTPError(c, srverrors.ErrGetAgentGroupNotFound, err)
			} else {
				utils.HTTPError(c, srverrors.ErrInternal, err)
			}
			return
		} else if err = group.Valid(); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error validating group data '%s'", group.Hash)
			utils.HTTPError(c, srverrors.ErrGetAgentInvalidGroupData, err)
			return
		}
		resp.Details.Group = &group

		resp.Details.Modules = make([]models.ModuleAShort, 0)
		err = iDB.Model(&models.ModuleAShort{}).
			Group("modules.id").
			Joins(`LEFT JOIN groups_to_policies gtp ON gtp.policy_id = modules.policy_id`).
			Find(&resp.Details.Modules, "gtp.group_id = ? AND status = 'joined'", resp.Agent.GroupID).Error
		if err != nil {
			utils.FromContext(c).WithError(err).Errorf("error finding group modules by group ID '%d'", resp.Agent.GroupID)
			utils.HTTPError(c, srverrors.ErrGetAgentGroupModulesNotFound, err)
			return
		} else {
			for i := 0; i < len(resp.Details.Modules); i++ {
				if err = resp.Details.Modules[i].Valid(); err != nil {
					id := resp.Details.Modules[i].ID
					name := resp.Details.Modules[i].Info.Name
					utils.FromContext(c).WithError(err).Errorf("error validating group module data '%d' '%s'", id, name)
					utils.HTTPError(c, srverrors.ErrGetAgentInvalidAgentModuleData, err)
					return
				}
			}
		}
		resp.Details.Consistency, resp.Details.Dependencies = getAgentConsistency(resp.Details.Modules, &resp.Agent)

		gps := models.GroupPolicies{
			Group: *resp.Details.Group,
		}
		if err = iDB.Model(gps).Association("policies").Find(&gps.Policies).Error; err != nil {
			utils.FromContext(c).WithError(err).Errorf("error finding group policies by group model")
			utils.HTTPError(c, srverrors.ErrGetAgentPoliciesNotFound, err)
			return
		}
		resp.Details.Policies = gps.Policies
	}

	utils.HTTPSuccess(c, http.StatusOK, resp)
}

// PatchAgent is a function to update agent public info only
// @Summary Update agent info by agent hash
// @Tags Agents
// @Accept json
// @Produce json
// @Param hash path string true "agent hash in hex format (md5)" minlength(32) maxlength(32)
// @Param json body patchAgentAction true "agent info as JSON data"
// @Success 200 {object} utils.successResp{data=models.Agent} "agent info updated successful"
// @Failure 400 {object} utils.errorResp "invalid agent info"
// @Failure 403 {object} utils.errorResp "updating agent info not permitted"
// @Failure 404 {object} utils.errorResp "agent not found"
// @Failure 500 {object} utils.errorResp "internal error on updating agent"
// @Router /agents/{hash} [put]
func (s *AgentService) PatchAgent(c *gin.Context) {
	var (
		action patchAgentAction
		count  int64
		err    error
		hash   = c.Param("hash")
		iDB    *gorm.DB
	)
	uaf := utils.UserActionFields{
		Domain:            "agent",
		ObjectType:        "agent",
		ActionCode:        "undefined action",
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	if err = c.ShouldBindJSON(&action); err != nil || action.Agent.Valid() != nil {
		if err == nil {
			err = action.Agent.Valid()
		}
		name, nameErr := utils.GetAgentName(c, hash)
		if nameErr == nil {
			uaf.ObjectDisplayName = name
		}
		utils.FromContext(c).WithError(err).Errorf("error binding JSON")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchAgentValidationError, err, uaf)
		return
	}
	uaf.ActionCode = getActionCode(action.Action)
	uaf.ObjectId = hash
	uaf.ObjectDisplayName = action.Agent.Description

	if hash != action.Agent.Hash {
		utils.FromContext(c).WithError(nil).Errorf("mismatch agent hash to requested one")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchAgentValidationError, nil, uaf)
		return
	}

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBNotFound, nil, uaf)
		return
	}

	if err = iDB.Model(&action.Agent).Count(&count).Error; err != nil || count == 0 {
		utils.FromContext(c).WithError(nil).Errorf("error updating agent by hash '%s', agent not found", hash)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrAgentsNotFound, err, uaf)
		return
	}

	if action.Agent.AuthStatus == "blocked" {
		err = iDB.
			Model(&models.AgentUpgradeTask{}).
			Where("agent_id = ?", action.Agent.ID).
			Where("status IN (?)", []string{"new", "running"}).
			UpdateColumns(map[string]interface{}{
				"status": "failed",
				"reason": "Canceled.By.User",
			}).Error
		if err != nil {
			utils.FromContext(c).WithError(err).Errorf("error updating tasks by agent")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchAgentTaskUpdateFail, err, uaf)
			return
		}
	}

	public_info := []interface{}{"auth_status", "description", "info", "group_id", "updated_at"}
	err = iDB.Select("", public_info...).Save(&action.Agent).Error

	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		utils.FromContext(c).WithError(nil).Errorf("error updating agent by hash '%s', agent not found", hash)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrAgentsNotFound, err, uaf)
		return
	} else if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error updating agent by hash '%s'", hash)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		return
	}

	utils.HTTPSuccessWithUAFields(c, http.StatusOK, action.Agent, uaf)
}

// CreateAgent is a function to create new agent
// @Summary Create new agent in service
// @Tags Agents
// @Accept json
// @Produce json
// @Param json body agentInfo true "agent info to create one"
// @Success 201 {object} utils.successResp{data=models.Agent} "agent created successful"
// @Failure 400 {object} utils.errorResp "invalid agent info"
// @Failure 403 {object} utils.errorResp "creating agent not permitted"
// @Failure 500 {object} utils.errorResp "internal error on creating agent"
// @Router /agents/ [post]
func (s *AgentService) CreateAgent(c *gin.Context) {
	logger := utils.FromContext(c)
	uaf := utils.UserActionFields{
		Domain:            "agent",
		ObjectType:        "agent",
		ActionCode:        "creation",
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	var info agentInfo
	if err := c.ShouldBindJSON(&info); err != nil {
		logger.WithError(err).Errorf("error binding JSON")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrCreateAgentValidationError, err, uaf)
		return
	}
	uaf.ObjectDisplayName = info.Name

	agentServerDB := utils.GetGormDB(c, "iDB")
	if agentServerDB == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBNotFound, nil, uaf)
		return
	}

	user := models.AgentUser{
		Name:   "unknown",
		Groups: []string{"unknown"},
	}
	newAgent := models.Agent{
		Hash:        utils.MakeAgentHash(info.Name),
		IP:          "0.0.0.0",
		Description: info.Name,
		Version:     "v1.0.0",
		Status:      "disconnected",
		AuthStatus:  "authorized",
		Info: models.AgentInfo{
			OS: models.AgentOS{
				Type: info.OS,
				Arch: info.Arch,
				Name: "unknown",
			},
			Net: models.AgentNet{
				Hostname: "unknown",
				IPs:      []string{"127.0.0.1/8", "::1/128"},
			},
			Users: []models.AgentUser{
				user,
			},
			Tags: []string{
				"manual_created",
			},
		},
	}
	uaf.ObjectId = newAgent.Hash

	if err := agentServerDB.Create(&newAgent).Error; err != nil {
		logger.WithError(err).Errorf("error creating agent")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrCreateAgentCreateError, err, uaf)
		return
	}

	utils.HTTPSuccessWithUAFields(c, http.StatusCreated, newAgent, uaf)
}

// DeleteAgent is a function to cascade delete agent
// @Summary Delete agent from instance DB
// @Tags Agents
// @Produce json
// @Param hash path string true "agent hash in hex format (md5)" minlength(32) maxlength(32)
// @Success 200 {object} utils.successResp "agent deleted successful"
// @Failure 403 {object} utils.errorResp "deleting agent not permitted"
// @Failure 404 {object} utils.errorResp "agent not found"
// @Failure 500 {object} utils.errorResp "internal error on deleting agent"
// @Router /agents/{hash} [delete]
func (s *AgentService) DeleteAgent(c *gin.Context) {
	var (
		agent models.Agent
		err   error
		hash  = c.Param("hash")
		iDB   *gorm.DB
	)
	uaf := utils.UserActionFields{
		Domain:            "agent",
		ObjectType:        "agent",
		ActionCode:        "deletion",
		ObjectId:          hash,
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBNotFound, nil, uaf)
		return
	}

	if err = iDB.Take(&agent, "hash = ?", hash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding agent by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrAgentsNotFound, err, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		}
		return
	} else if err = agent.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating agent data '%s'", agent.Hash)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrAgentsInvalidData, err, uaf)
		return
	}
	uaf.ObjectDisplayName = agent.Description

	if err = iDB.Delete(&agent).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error deleting agent by hash '%s'", hash)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		return
	}

	utils.HTTPSuccessWithUAFields(c, http.StatusOK, struct{}{}, uaf)
}

// GetAgentsCount is a function to return groups of counted agents
// @Summary Retrieve groups of counted agents
// @Tags Agents
// @Produce json
// @Success 200 {object} utils.successResp{data=agentCount} "groups of counted agents retrieved successfully"
// @Failure 500 {object} utils.errorResp "internal error"
// @Router /agents/count [get]
func (s *AgentService) GetAgentsCount(c *gin.Context) {
	var (
		err  error
		iDB  *gorm.DB
		resp agentCount
	)
	logger := utils.FromContext(c)
	uaf := utils.UserActionFields{
		Domain:            "agent",
		ObjectType:        "agent",
		ActionCode:        getActionCode("count"),
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBNotFound, nil, uaf)
		return
	}

	// language=MySQL
	const q = `SELECT
		COUNT(*) AS 'all',
		SUM(auth_status = 'authorized') AS 'authorized',
		SUM(auth_status = 'blocked') AS 'blocked',
		SUM(auth_status = 'unauthorized') AS 'unauthorized',
		SUM(group_id = 0 AND auth_status = 'authorized') AS 'without_groups'
		FROM agents
		WHERE deleted_at IS NULL`
	err = iDB.Raw(q).
		Scan(&resp).
		Error
	if err != nil {
		logger.WithError(err).Errorf("could not count agents")
		utils.HTTPError(c, srverrors.ErrInternal, err)
		return
	}

	utils.HTTPSuccessWithUAFields(c, http.StatusOK, resp, uaf)
}
