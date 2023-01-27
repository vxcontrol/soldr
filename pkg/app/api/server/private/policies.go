package private

import (
	"errors"
	"net/http"
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
	"soldr/pkg/app/api/utils"
	"soldr/pkg/semvertooling"
)

type policyDetails struct {
	Hash          string                    `json:"hash"`
	Agents        int                       `json:"agents"`
	ActiveModules int                       `json:"active_modules"`
	JoinedModules string                    `json:"joined_modules"`
	UpdateModules bool                      `json:"update_modules"`
	Consistency   bool                      `json:"consistency"`
	Dependencies  []models.PolicyDependency `json:"dependencies"`
	Groups        []models.Group            `json:"groups,omitempty"`
	Modules       []models.ModuleAShort     `json:"modules,omitempty"`
}

type policies struct {
	Policies []models.Policy `json:"policies"`
	Details  []policyDetails `json:"details"`
	Total    uint64          `json:"total"`
}

type policy struct {
	Policy  models.Policy `json:"policy"`
	Details policyDetails `json:"details"`
}

type policyCount struct {
	All           int `json:"all"`
	WithoutGroups int `json:"without_groups"`
}

type policyInfo struct {
	Name string   `json:"name" binding:"max=255,required_without=From"`
	Tags []string `json:"tags" binding:"omitempty"`
	From uint64   `json:"from" binding:"min=0,numeric,omitempty"`
}

type policyGroupPatch struct {
	// Action on policy group must be one of activate, deactivate
	Action string       `form:"action" json:"action" binding:"oneof=activate deactivate,required" default:"activate" enums:"activate,deactivate"`
	Group  models.Group `form:"group" json:"group" binding:"required"`
}

var policiesSQLMappers = map[string]interface{}{
	"id":             "`{{table}}`.id",
	"hash":           "`{{table}}`.hash",
	"created_date":   "`{{table}}`.created_date",
	"module_name":    "`modules`.name",
	"module_os":      storage.ModulesOSMapper,
	"module_os_arch": storage.ModulesOSArchMapper,
	"module_os_type": storage.ModulesOSTypeMapper,
	"group_id":       "`gtp`.group_id",
	"group_name":     "JSON_UNQUOTE(JSON_EXTRACT(`groups`.info, '$.name.{{lang}}'))",
	"name":           "JSON_UNQUOTE(JSON_EXTRACT(`{{table}}`.info, '$.name.{{lang}}'))",
	"tags":           storage.TagsMapper,
	"ngroups":        "(SELECT count(*) FROM `groups_to_policies` WHERE `policy_id` = `{{table}}`.id)",
	"data": "CONCAT(`{{table}}`.hash, ' | ', " +
		"COALESCE(JSON_EXTRACT(`{{table}}`.info, '$.name.ru'), ''), ' | ', " +
		"COALESCE(JSON_EXTRACT(`{{table}}`.info, '$.name.en'), ''), ' | ', " +
		"COALESCE(JSON_EXTRACT(`{{table}}`.info, '$.tags'), ''))",
}

const sqlPolicyDetails = `
	SELECT p.hash,
		(SELECT COUNT(a.id) FROM agents a
			LEFT JOIN groups g ON a.group_id = g.id AND g.deleted_at IS NULL
			LEFT JOIN groups_to_policies AS gtp ON g.id = gtp.group_id
			WHERE p.id = gtp.policy_id AND a.deleted_at IS NULL) AS agents,
		(SELECT COUNT(m.id) FROM modules m
			WHERE m.policy_id = p.id AND m.status = 'joined' AND
				m.deleted_at IS NULL) AS active_modules,
		(SELECT GROUP_CONCAT(m.name SEPARATOR ',') FROM modules m
			WHERE m.policy_id = p.id AND m.status = 'joined' AND m.deleted_at IS NULL
			GROUP BY m.policy_id) AS joined_modules
	FROM policies AS p`

func getPolicyConsistency(modules []models.ModuleAShort) (bool, []models.PolicyDependency) {
	var (
		rdeps bool = true
		pdeps []models.PolicyDependency
	)
	checkDependency := func(smod models.ModuleAShort, dep models.DependencyItem) bool {
		switch dep.Type {
		case "to_receive_data":
			fallthrough
		case "to_send_data":
			fallthrough
		case "to_make_action":
			if dep.ModuleName == "this" {
				return true
			}
			for _, mod := range modules {
				if dep.ModuleName == mod.Info.Name {
					for osType, osArchs := range smod.Info.OS {
						if _, ok := mod.Info.OS[osType]; !ok {
							return false
						}
						if !utils.StringsInSlice(osArchs, mod.Info.OS[osType]) {
							return false
						}
					}
					switch semvertooling.CompareVersions(mod.Info.Version.String(), dep.MinModuleVersion) {
					case semvertooling.TargetVersionEmpty, semvertooling.VersionsEqual, semvertooling.SourceVersionGreat:
						return true
					default:
						return false
					}
				}
			}
		}
		return false
	}

	for _, mod := range modules {
		var deps []models.DependencyItem
		deps = append(deps, mod.StaticDependencies...)
		deps = append(deps, mod.DynamicDependencies...)
		for _, dep := range deps {
			if dep.ModuleName == "this" || dep.Type == "agent_version" {
				continue
			}

			sdeps := checkDependency(mod, dep)
			if !sdeps {
				rdeps = false
			}
			pdeps = append(pdeps, models.PolicyDependency{
				SourceModuleName: mod.Info.Name,
				ModuleDependency: models.ModuleDependency{
					Status:         sdeps,
					DependencyItem: dep,
				},
			})
		}
	}

	return rdeps, pdeps
}

func getPolicyName(db *gorm.DB, hash string) (string, error) {
	var policy models.Policy
	if err := db.Take(&policy, "hash = ?", hash).Error; err != nil {
		return "", err
	}
	return policy.Info.Name.En, nil
}

type PolicyService struct {
	db               *gorm.DB
	serverConnector  *client.AgentServerClient
	userActionWriter useraction.Writer
}

func NewPolicyService(
	db *gorm.DB,
	serverConnector *client.AgentServerClient,
	userActionWriter useraction.Writer,
) *PolicyService {
	return &PolicyService{
		db:               db,
		serverConnector:  serverConnector,
		userActionWriter: userActionWriter,
	}
}

// GetPolicies is a function to return policy list view on dashboard
// @Summary Retrieve policies list by filters
// @Tags Policies
// @Produce json
// @Param request query storage.TableQuery true "query table params"
// @Success 200 {object} response.successResp{data=policies} "policies list received successful"
// @Failure 400 {object} response.errorResp "invalid query request data"
// @Failure 403 {object} response.errorResp "getting policies not permitted"
// @Failure 404 {object} response.errorResp "policies not found"
// @Failure 500 {object} response.errorResp "internal error on getting policies"
// @Router /policies/ [get]
func (s *PolicyService) GetPolicies(c *gin.Context) {
	var (
		groups       []models.Group
		moduleList   []models.ModuleSShort
		modulesa     []models.ModuleAShort
		pgss         []models.GroupToPolicy
		pids         []uint64
		query        storage.TableQuery
		groupedResp  response.GroupedData
		resp         policies
		sv           *models.Service
		useGroup     bool
		useModule    bool
		useGroupName bool
	)

	if err := c.ShouldBindQuery(&query); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error binding query")
		response.Error(c, response.ErrPoliciesInvalidRequest, err)
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
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("tenant_id = ? AND service_type = ?", tid, sv.Type)
	}

	if err = s.db.Scopes(modules.LatestModulesQuery, scope).Find(&moduleList).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error loading system modules latest version")
		response.Error(c, response.ErrGetPoliciesSystemModulesNotFound, err)
		return
	}

	if err = query.Init("policies", policiesSQLMappers); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error binding query")
		response.Error(c, response.ErrPoliciesInvalidRequest, err)
		return
	}

	modulesFields := []string{"module_name", "module_os", "module_os_type", "module_os_arch"}
	setUsingTables := func(sfield string) {
		if sfield == "group_id" {
			useGroup = true
		}
		for _, field := range modulesFields {
			if sfield == field {
				useModule = true
				break
			}
		}
		if sfield == "group_name" {
			useGroup = true
			useGroupName = true
		}
	}
	setUsingTables(query.Sort.Prop)
	setUsingTables(query.Group)
	for _, filter := range query.Filters {
		setUsingTables(filter.Field)
	}
	query.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Where("policies.deleted_at IS NULL")
		},
	})
	funcs := []func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			if useGroup {
				db = db.Joins(`LEFT JOIN groups_to_policies gtp ON gtp.policy_id = policies.id`)
			}
			if useModule {
				db = db.Joins(`LEFT JOIN modules ON policies.id = modules.policy_id AND modules.status = 'joined' AND modules.deleted_at IS NULL`)
			}
			if useGroupName {
				db = db.Joins(`LEFT JOIN groups ON groups.id = gtp.group_id AND groups.deleted_at IS NULL`)
			}
			if useGroup || useModule || useGroupName {
				db = db.Group("policies.id")
			}
			return db
		},
	}

	if query.Group == "" {
		if resp.Total, err = query.Query(iDB, &resp.Policies, funcs...); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error finding policies")
			response.Error(c, response.ErrPoliciesInvalidQuery, err)
			return
		}
	} else {
		if groupedResp.Total, err = query.QueryGrouped(iDB, &groupedResp.Grouped, funcs...); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error finding grouped policies")
			response.Error(c, response.ErrPoliciesInvalidQuery, err)
			return
		}
		response.Success(c, http.StatusOK, groupedResp)
		return
	}

	for i := 0; i < len(resp.Policies); i++ {
		pids = append(pids, resp.Policies[i].ID)
		if err = resp.Policies[i].Valid(); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", resp.Policies[i].Hash)
			response.Error(c, response.ErrPoliciesInvalidData, err)
			return
		}
	}

	sqlQuery := sqlPolicyDetails + ` WHERE p.id IN (?) AND p.deleted_at IS NULL`
	if err = iDB.Raw(sqlQuery, pids).Scan(&resp.Details).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error loading policies details")
		response.Error(c, response.ErrGetPoliciesDetailsNotFound, err)
		return
	}

	if err = iDB.Find(&modulesa, "policy_id IN (?) AND status = 'joined'", pids).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy modules")
		response.Error(c, response.ErrGetPoliciesModulesNotFound, err)
		return
	} else {
		for i := 0; i < len(modulesa); i++ {
			id := modulesa[i].ID
			name := modulesa[i].Info.Name
			if err = modulesa[i].Valid(); err != nil {
				logger.FromContext(c).WithError(err).Errorf("error validating policy module data '%d' '%s'", id, name)
				response.Error(c, response.ErrGetPoliciesInvalidModuleData, err)
				return
			}
		}
	}

	if err = iDB.Find(&pgss, "policy_id IN (?)", pids).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding group to policies links")
		response.Error(c, response.ErrGroupPolicyPoliciesNotFound, err)
		return
	}

	err = iDB.Model(&models.Group{}).
		Group("groups.id").
		Joins(`LEFT JOIN groups_to_policies gtp ON gtp.group_id = groups.id`).
		Find(&groups, "gtp.policy_id IN (?)", pids).Error
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy groups")
		response.Error(c, response.ErrGroupPolicyGroupsNotFound, err)
		return
	} else {
		for i := 0; i < len(groups); i++ {
			id := groups[i].ID
			name := groups[i].Info.Name
			if err = groups[i].Valid(); err != nil {
				logger.FromContext(c).WithError(err).Errorf("error validating agent group data '%d' '%s'", id, name)
				response.Error(c, response.ErrGetPoliciesInvalidGroupData, err)
				return
			}
		}
	}

	for _, policy := range resp.Policies {
		var details *policyDetails
		for idx := range resp.Details {
			if resp.Details[idx].Hash == policy.Hash {
				details = &resp.Details[idx]
			}
		}
		if details == nil {
			continue
		}

		for i := 0; i < len(modulesa); i++ {
			name := modulesa[i].Info.Name
			if modulesa[i].PolicyID != policy.ID {
				continue
			}
			details.Modules = append(details.Modules, modulesa[i])
			for _, module := range moduleList {
				if !details.UpdateModules && module.Info.Name == name {
					isSameVersion := module.Info.Version.String() == modulesa[i].Info.Version.String()
					details.UpdateModules = !isSameVersion
				}
			}
		}
		details.Consistency, details.Dependencies = getPolicyConsistency(details.Modules)

		for idx := range pgss {
			if pgss[idx].PolicyID != policy.ID {
				continue
			}
			for jdx := range groups {
				if pgss[idx].GroupID != groups[jdx].ID {
					continue
				}
				details.Groups = append(details.Groups, groups[jdx])
			}
		}
	}

	response.Success(c, http.StatusOK, resp)
}

// GetPolicy is a function to return policy info and details view
// @Summary Retrieve policy info by policy hash
// @Tags Policies
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Success 200 {object} response.successResp{data=policy} "policy info received successful"
// @Failure 403 {object} response.errorResp "getting policy info not permitted"
// @Failure 404 {object} response.errorResp "policy not found"
// @Failure 500 {object} response.errorResp "internal error on getting policy"
// @Router /policies/{hash} [get]
func (s *PolicyService) GetPolicy(c *gin.Context) {
	var (
		hash        = c.Param("hash")
		modulesList []models.ModuleSShort
		resp        policy
		sv          *models.Service
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

	if sv = modules.GetService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	tid := c.GetUint64("tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("tenant_id = ? AND service_type = ?", tid, sv.Type)
	}

	if err = s.db.Scopes(modules.LatestModulesQuery, scope).Find(&modulesList).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error loading system modules latest version")
		response.Error(c, response.ErrGetPolicySystemModulesNotFound, err)
		return
	}

	if err = iDB.Take(&resp.Policy, "hash = ?", hash).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrPoliciesNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = resp.Policy.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", resp.Policy.Hash)
		response.Error(c, response.ErrPoliciesInvalidData, err)
		return
	}

	sqlQuery := sqlPolicyDetails + ` WHERE p.hash = ? AND p.deleted_at IS NULL`
	if err = iDB.Raw(sqlQuery, hash).Scan(&resp.Details).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error loading details by policy hash '%s'", hash)
		response.Error(c, response.ErrGetPolicyDetailsNotFound, err)
		return
	}

	if err = iDB.Find(&resp.Details.Modules, "policy_id = ? AND status = 'joined'", resp.Policy.ID).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy modules by policy ID '%d'", resp.Policy.ID)
		response.Error(c, response.ErrGetPolicyModulesNotFound, err)
		return
	} else {
		for i := 0; i < len(resp.Details.Modules); i++ {
			id := resp.Details.Modules[i].ID
			name := resp.Details.Modules[i].Info.Name
			if err = resp.Details.Modules[i].Valid(); err != nil {
				logger.FromContext(c).WithError(err).Errorf("error validating policy module data '%d' '%s'", id, name)
				response.Error(c, response.ErrGetPolicyInvalidModuleData, err)
				return
			}
			for _, module := range modulesList {
				if !resp.Details.UpdateModules && module.Info.Name == name {
					isSameVersion := module.Info.Version.String() == resp.Details.Modules[i].Info.Version.String()
					resp.Details.UpdateModules = !isSameVersion
				}
			}
		}
	}
	resp.Details.Consistency, resp.Details.Dependencies = getPolicyConsistency(resp.Details.Modules)

	pgs := models.PolicyGroups{
		Policy: resp.Policy,
	}
	if err = iDB.Model(pgs).Association("groups").Find(&pgs.Groups).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy groups by policy model")
		response.Error(c, response.ErrGetPolicyGroupsNotFound, err)
		return
	}
	resp.Details.Groups = pgs.Groups

	response.Success(c, http.StatusOK, resp)
}

// PatchPolicy is a function to update policy public info only
// @Summary Update policy info by policy hash
// @Tags Policies
// @Accept json
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Param json body models.Policy true "policy info as JSON data"
// @Success 200 {object} response.successResp{data=models.Policy} "policy info updated successful"
// @Failure 400 {object} response.errorResp "invalid policy info"
// @Failure 403 {object} response.errorResp "updating policy info not permitted"
// @Failure 404 {object} response.errorResp "policy not found"
// @Failure 500 {object} response.errorResp "internal error on updating policy"
// @Router /policies/{hash} [put]
func (s *PolicyService) PatchPolicy(c *gin.Context) {
	var (
		count  int64
		hash   = c.Param("hash")
		policy models.Policy
	)
	uaf := useraction.Fields{
		Domain:            "policy",
		ObjectType:        "policy",
		ActionCode:        "editing",
		ObjectID:          hash,
		ObjectDisplayName: useraction.UnknownObjectDisplayName,
	}
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

	if err = c.ShouldBindJSON(&policy); err != nil || policy.Valid() != nil {
		if err == nil {
			err = policy.Valid()
		}
		name, nameErr := getPolicyName(iDB, hash)
		if nameErr == nil {
			uaf.ObjectDisplayName = name
		}
		logger.FromContext(c).WithError(err).Errorf("error binding JSON")
		response.Error(c, response.ErrPoliciesInvalidRequest, err)
		return
	}
	uaf.ObjectDisplayName = policy.Info.Name.En

	if hash != policy.Hash {
		logger.FromContext(c).Errorf("mismatch policy hash to requested one")
		response.Error(c, response.ErrPoliciesInvalidRequest, err)
		return
	}

	if err = iDB.Model(&policy).Count(&count).Error; err != nil || count == 0 {
		logger.FromContext(c).Errorf("error updating policy by hash '%s', group not found", hash)
		response.Error(c, response.ErrPoliciesNotFound, err)
		return
	}

	public_info := []interface{}{"info", "updated_at"}
	err = iDB.Select("", public_info...).Save(&policy).Error

	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		logger.FromContext(c).Errorf("error updating policy by hash '%s', policy not found", hash)
		response.Error(c, response.ErrPoliciesNotFound, err)
		return
	} else if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error updating policy by hash '%s'", hash)
		response.Error(c, response.ErrInternal, err)
		return
	}

	response.Success(c, http.StatusOK, policy)
}

// PatchPolicyGroup is a function to update policy group linking
// @Summary Update or patch policy group by policy hash and group object
// @Tags Policies,Groups
// @Accept json
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Param json body policyGroupPatch true "action on policy group as JSON data (activate, deactivate)"
// @Success 200 {object} response.successResp "policy group patched successful"
// @Failure 400 {object} response.errorResp "invalid patch request data"
// @Failure 403 {object} response.errorResp "updating policy group not permitted"
// @Failure 404 {object} response.errorResp "policy or group not found"
// @Failure 500 {object} response.errorResp "internal error on updating policy group"
// @Router /policies/{hash}/groups [put]
func (s *PolicyService) PatchPolicyGroup(c *gin.Context) {
	var (
		form   policyGroupPatch
		group  models.Group
		hash   = c.Param("hash")
		policy models.Policy
	)
	uaf := useraction.Fields{
		Domain:            "policy",
		ObjectType:        "policy",
		ObjectID:          hash,
		ActionCode:        "undefined action",
		ObjectDisplayName: useraction.UnknownObjectDisplayName,
	}
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

	if err = c.ShouldBindJSON(&form); err != nil {
		name, nameErr := getPolicyName(iDB, hash)
		if nameErr == nil {
			uaf.ObjectDisplayName = name
		}
		logger.FromContext(c).WithError(err).Errorf("error binding JSON")
		response.Error(c, response.ErrPoliciesInvalidRequest, err)
		return
	}
	if form.Action == "activate" {
		uaf.ActionCode = "creation of the connection with the group"
	} else {
		uaf.ActionCode = "deletion of the connection with the group"
	}

	if err = iDB.Take(&policy, "hash = ?", hash).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrPoliciesNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	}
	uaf.ObjectDisplayName = policy.Info.Name.En
	if err = policy.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policy.Hash)
		response.Error(c, response.ErrPoliciesInvalidData, err)
		return
	}

	if err = iDB.Take(&group, "hash = ?", form.Group.Hash).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding group by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrPoliciesNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = group.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating group data '%s'", group.Hash)
		response.Error(c, response.ErrPatchPolicyGroupInvalidGroupData, err)
		return
	}

	httpErr, err := makeGroupPolicyAction(form.Action, iDB, group, policy)
	if httpErr != nil {
		logger.FromContext(c).WithError(err).Errorf("error patching group policy by action: %s", httpErr.Error())
		response.Error(c, httpErr, err)
	}
	response.Success(c, http.StatusOK, struct{}{})
}

// CreatePolicy is a function to create new policy
// @Summary Create new policy in service
// @Tags Policies
// @Accept json
// @Produce json
// @Param json body policyInfo true "policy info to create one"
// @Success 201 {object} response.successResp{data=models.Policy} "policy created successful"
// @Failure 400 {object} response.errorResp "invalid policy info"
// @Failure 403 {object} response.errorResp "creating policy not permitted"
// @Failure 500 {object} response.errorResp "internal error on creating policy"
// @Router /policies/ [post]
func (s *PolicyService) CreatePolicy(c *gin.Context) {
	var (
		info       policyInfo
		policyFrom models.Policy
	)
	uaf := useraction.Fields{
		Domain:            "policy",
		ObjectType:        "policy",
		ActionCode:        "creation",
		ObjectDisplayName: useraction.UnknownObjectDisplayName,
	}
	defer s.userActionWriter.WriteUserAction(c, uaf)

	if err := c.ShouldBindJSON(&info); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error binding JSON")
		response.Error(c, response.ErrPoliciesInvalidRequest, err)
		return
	}
	uaf.ObjectDisplayName = info.Name

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

	policy := models.Policy{
		Hash: storage.MakePolicyHash(info.Name),
		Info: models.PolicyInfo{
			Name: models.PolicyItemLocale{
				Ru: info.Name,
				En: info.Name,
			},
			Tags:   info.Tags,
			System: false,
		},
	}
	uaf.ObjectID = policy.Hash

	if info.From != 0 {
		if err = iDB.Take(&policyFrom, "id = ?", info.From).Error; err != nil {
			logger.FromContext(c).WithError(err).Errorf("error finding source policy by ID")
			response.Error(c, response.ErrCreatePolicySourceNotFound, err)
			return
		} else if err = policyFrom.Valid(); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policyFrom.Hash)
			response.Error(c, response.ErrPoliciesInvalidData, err)
			return
		}

		policy = policyFrom
		policy.ID = 0
		policy.Info.System = false
		policy.CreatedDate = time.Time{}
		policy.Hash = storage.MakePolicyHash(policy.Hash)
		if info.Name != "" {
			policy.Info.Name = models.PolicyItemLocale{
				Ru: info.Name,
				En: info.Name,
			}
		} else {
			policy.Info.Name.Ru += " (копия)"
			policy.Info.Name.En += " (copy)"
		}
		if len(info.Tags) != 0 {
			policy.Info.Tags = info.Tags
		}
	}

	if err = iDB.Create(&policy).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error creating policy")
		response.Error(c, response.ErrCreatePolicyCreateFail, err)
		return
	}

	if policyFrom.ID != 0 {
		var modules []models.ModuleA
		err = iDB.Where("policy_id = ? AND status = 'joined'", policyFrom.ID).Find(&modules).Error
		if err != nil {
			logger.FromContext(c).WithError(err).Errorf("error finding policy modules by policy ID '%d'", policyFrom.ID)
			response.Error(c, response.ErrCreatePolicyModulesNotFound, err)
			return
		}
		for _, module := range modules {
			module.ID = 0
			module.PolicyID = policy.ID
			if err = iDB.Create(&module).Error; err != nil {
				logger.FromContext(c).WithError(err).Errorf("error creating policy module")
				response.Error(c, response.ErrCreatePolicyCreateModulesFail, err)
				return
			}
		}
	}

	response.Success(c, http.StatusCreated, policy)
}

// DeletePolicy is a function to cascade delete policy
// @Summary Delete policy from instance DB
// @Tags Policies
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Success 200 {object} response.successResp "policy deleted successful"
// @Failure 403 {object} response.errorResp "deleting policy not permitted"
// @Failure 404 {object} response.errorResp "policy not found"
// @Failure 500 {object} response.errorResp "internal error on deleting policy"
// @Router /policies/{hash} [delete]
func (s *PolicyService) DeletePolicy(c *gin.Context) {
	var (
		hash       = c.Param("hash")
		moduleList []models.ModuleAShort
		policy     models.Policy
		sv         *models.Service
	)
	uaf := useraction.Fields{
		Domain:            "policy",
		ObjectType:        "policy",
		ActionCode:        "deletion",
		ObjectID:          hash,
		ObjectDisplayName: useraction.UnknownObjectDisplayName,
	}
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

	if sv = modules.GetService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	if err = iDB.Take(&policy, "hash = ?", hash).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrPoliciesNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = policy.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policy.Hash)
		response.Error(c, response.ErrPoliciesInvalidData, err)
		return
	}
	uaf.ObjectDisplayName = policy.Info.Name.En

	if policy.Info.System {
		logger.FromContext(c).Errorf("error removing system policy")
		response.Error(c, response.ErrDeletePolicySystemPolicy, err)
		return
	}

	pgs := models.PolicyGroups{
		Policy: policy,
	}
	if err = iDB.Model(pgs).Association("groups").Find(&pgs.Groups).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy groups by policy model")
		response.Error(c, response.ErrDeletePolicyGroupsNotFound, err)
		return
	}
	if len(pgs.Groups) != 0 {
		logger.FromContext(c).Errorf("error removing policy which linked to groups")
		response.Error(c, response.ErrDeletePolicyPolicyLinkedToGroups, err)
		return
	}

	if err = iDB.Find(&moduleList, "policy_id = ?", policy.ID).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy modules by policy ID '%d'", policy.ID)
		response.Error(c, response.ErrDeletePolicyModulesNotFound, err)
		return
	}

	if err = iDB.Delete(&policy).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error deleting policy by hash '%s'", hash)
		response.Error(c, response.ErrInternal, err)
		return
	}

	for _, module := range moduleList {
		moduleName := module.Info.Name
		moduleVersion := module.Info.Version.String()
		if err = modules.RemoveUnusedModuleVersion(c, iDB, moduleName, moduleVersion, sv); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error removing unused module data")
			response.Error(c, response.ErrInternal, err)
			return
		}
	}

	response.Success(c, http.StatusOK, struct{}{})
}

// GetPoliciesCount is a function to return groups of counted policies
// @Summary Retrieve groups of counted policies
// @Tags Policies
// @Produce json
// @Success 200 {object} response.successResp{data=policyCount} "groups of counted agents policies successfully"
// @Failure 500 {object} response.errorResp "internal error"
// @Router /policies/count [get]
func (s *PolicyService) GetPoliciesCount(c *gin.Context) {
	uaf := useraction.Fields{
		Domain:            "policy",
		ObjectType:        "policy",
		ActionCode:        "counting",
		ObjectDisplayName: useraction.UnknownObjectDisplayName,
	}
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

	// language=MySQL
	const q = `SELECT
    	COUNT(*) AS 'all',
    	(SELECT COUNT(*) FROM policies
		LEFT JOIN groups_to_policies gtp on policies.id = gtp.policy_id
		WHERE gtp.group_id IS NULL AND deleted_at IS NULL) AS 'without_groups'
	FROM policies
	WHERE deleted_at IS NULL`
	var resp policyCount
	err = iDB.Raw(q).
		Scan(&resp).
		Error
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("could not count policies")
		response.Error(c, response.ErrInternal, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}
