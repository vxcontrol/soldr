package private

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	"soldr/pkg/app/api/client"
	"soldr/pkg/app/api/models"
	srvcontext "soldr/pkg/app/api/server/context"
	srverrors "soldr/pkg/app/api/server/response"
	"soldr/pkg/app/api/utils"
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
	"module_os":      utils.ModulesOSMapper,
	"module_os_arch": utils.ModulesOSArchMapper,
	"module_os_type": utils.ModulesOSTypeMapper,
	"group_id":       "`gtp`.group_id",
	"group_name":     "JSON_UNQUOTE(JSON_EXTRACT(`groups`.info, '$.name.{{lang}}'))",
	"name":           "JSON_UNQUOTE(JSON_EXTRACT(`{{table}}`.info, '$.name.{{lang}}'))",
	"tags":           utils.TagsMapper,
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
					switch utils.CompareVersions(mod.Info.Version.String(), dep.MinModuleVersion) {
					case utils.TargetVersionEmpty, utils.VersionsEqual, utils.SourceVersionGreat:
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
	db              *gorm.DB
	serverConnector *client.AgentServerClient
}

func NewPolicyService(db *gorm.DB, serverConnector *client.AgentServerClient) *PolicyService {
	return &PolicyService{
		db:              db,
		serverConnector: serverConnector,
	}
}

// GetPolicies is a function to return policy list view on dashboard
// @Summary Retrieve policies list by filters
// @Tags Policies
// @Produce json
// @Param request query utils.TableQuery true "query table params"
// @Success 200 {object} utils.successResp{data=policies} "policies list received successful"
// @Failure 400 {object} utils.errorResp "invalid query request data"
// @Failure 403 {object} utils.errorResp "getting policies not permitted"
// @Failure 404 {object} utils.errorResp "policies not found"
// @Failure 500 {object} utils.errorResp "internal error on getting policies"
// @Router /policies/ [get]
func (s *PolicyService) GetPolicies(c *gin.Context) {
	var (
		groups       []models.Group
		modules      []models.ModuleSShort
		modulesa     []models.ModuleAShort
		pgss         []models.GroupToPolicy
		pids         []uint64
		query        utils.TableQuery
		groupedResp  utils.GroupedData
		resp         policies
		sv           *models.Service
		useGroup     bool
		useModule    bool
		useGroupName bool
	)

	if err := c.ShouldBindQuery(&query); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding query")
		utils.HTTPError(c, srverrors.ErrPoliciesInvalidRequest, err)
		return
	}

	serviceHash, ok := srvcontext.GetString(c, "svc")
	if !ok {
		utils.FromContext(c).Errorf("could not get service hash")
		utils.HTTPError(c, srverrors.ErrInternal, nil)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		utils.FromContext(c).WithError(err).Error()
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, err)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPError(c, srverrors.ErrInternalServiceNotFound, nil)
		return
	}

	tid, _ := srvcontext.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("tenant_id = ? AND service_type = ?", tid, sv.Type)
	}

	if err = s.db.Scopes(LatestModulesQuery, scope).Find(&modules).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error loading system modules latest version")
		utils.HTTPError(c, srverrors.ErrGetPoliciesSystemModulesNotFound, err)
		return
	}

	if err = query.Init("policies", policiesSQLMappers); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding query")
		utils.HTTPError(c, srverrors.ErrPoliciesInvalidRequest, err)
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
			utils.FromContext(c).WithError(err).Errorf("error finding policies")
			utils.HTTPError(c, srverrors.ErrPoliciesInvalidQuery, err)
			return
		}
	} else {
		if groupedResp.Total, err = query.QueryGrouped(iDB, &groupedResp.Grouped, funcs...); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error finding grouped policies")
			utils.HTTPError(c, srverrors.ErrPoliciesInvalidQuery, err)
			return
		}
		utils.HTTPSuccess(c, http.StatusOK, groupedResp)
		return
	}

	for i := 0; i < len(resp.Policies); i++ {
		pids = append(pids, resp.Policies[i].ID)
		if err = resp.Policies[i].Valid(); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", resp.Policies[i].Hash)
			utils.HTTPError(c, srverrors.ErrPoliciesInvalidData, err)
			return
		}
	}

	sqlQuery := sqlPolicyDetails + ` WHERE p.id IN (?) AND p.deleted_at IS NULL`
	if err = iDB.Raw(sqlQuery, pids).Scan(&resp.Details).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error loading policies details")
		utils.HTTPError(c, srverrors.ErrGetPoliciesDetailsNotFound, err)
		return
	}

	if err = iDB.Find(&modulesa, "policy_id IN (?) AND status = 'joined'", pids).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy modules")
		utils.HTTPError(c, srverrors.ErrGetPoliciesModulesNotFound, err)
		return
	} else {
		for i := 0; i < len(modulesa); i++ {
			id := modulesa[i].ID
			name := modulesa[i].Info.Name
			if err = modulesa[i].Valid(); err != nil {
				utils.FromContext(c).WithError(err).Errorf("error validating policy module data '%d' '%s'", id, name)
				utils.HTTPError(c, srverrors.ErrGetPoliciesInvalidModuleData, err)
				return
			}
		}
	}

	if err = iDB.Find(&pgss, "policy_id IN (?)", pids).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding group to policies links")
		utils.HTTPError(c, srverrors.ErrGroupPolicyPoliciesNotFound, err)
		return
	}

	err = iDB.Model(&models.Group{}).
		Group("groups.id").
		Joins(`LEFT JOIN groups_to_policies gtp ON gtp.group_id = groups.id`).
		Find(&groups, "gtp.policy_id IN (?)", pids).Error
	if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy groups")
		utils.HTTPError(c, srverrors.ErrGroupPolicyGroupsNotFound, err)
		return
	} else {
		for i := 0; i < len(groups); i++ {
			id := groups[i].ID
			name := groups[i].Info.Name
			if err = groups[i].Valid(); err != nil {
				utils.FromContext(c).WithError(err).Errorf("error validating agent group data '%d' '%s'", id, name)
				utils.HTTPError(c, srverrors.ErrGetPoliciesInvalidGroupData, err)
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
			for _, module := range modules {
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

	utils.HTTPSuccess(c, http.StatusOK, resp)
}

// GetPolicy is a function to return policy info and details view
// @Summary Retrieve policy info by policy hash
// @Tags Policies
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Success 200 {object} utils.successResp{data=policy} "policy info received successful"
// @Failure 403 {object} utils.errorResp "getting policy info not permitted"
// @Failure 404 {object} utils.errorResp "policy not found"
// @Failure 500 {object} utils.errorResp "internal error on getting policy"
// @Router /policies/{hash} [get]
func (s *PolicyService) GetPolicy(c *gin.Context) {
	var (
		hash    = c.Param("hash")
		modules []models.ModuleSShort
		resp    policy
		sv      *models.Service
	)

	serviceHash, ok := srvcontext.GetString(c, "svc")
	if !ok {
		utils.FromContext(c).Errorf("could not get service hash")
		utils.HTTPError(c, srverrors.ErrInternal, nil)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		utils.FromContext(c).WithError(err).Error()
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, err)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPError(c, srverrors.ErrInternalServiceNotFound, nil)
		return
	}

	tid, _ := srvcontext.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("tenant_id = ? AND service_type = ?", tid, sv.Type)
	}

	if err = s.db.Scopes(LatestModulesQuery, scope).Find(&modules).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error loading system modules latest version")
		utils.HTTPError(c, srverrors.ErrGetPolicySystemModulesNotFound, err)
		return
	}

	if err = iDB.Take(&resp.Policy, "hash = ?", hash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPError(c, srverrors.ErrPoliciesNotFound, err)
		} else {
			utils.HTTPError(c, srverrors.ErrInternal, err)
		}
		return
	} else if err = resp.Policy.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", resp.Policy.Hash)
		utils.HTTPError(c, srverrors.ErrPoliciesInvalidData, err)
		return
	}

	sqlQuery := sqlPolicyDetails + ` WHERE p.hash = ? AND p.deleted_at IS NULL`
	if err = iDB.Raw(sqlQuery, hash).Scan(&resp.Details).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error loading details by policy hash '%s'", hash)
		utils.HTTPError(c, srverrors.ErrGetPolicyDetailsNotFound, err)
		return
	}

	if err = iDB.Find(&resp.Details.Modules, "policy_id = ? AND status = 'joined'", resp.Policy.ID).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy modules by policy ID '%d'", resp.Policy.ID)
		utils.HTTPError(c, srverrors.ErrGetPolicyModulesNotFound, err)
		return
	} else {
		for i := 0; i < len(resp.Details.Modules); i++ {
			id := resp.Details.Modules[i].ID
			name := resp.Details.Modules[i].Info.Name
			if err = resp.Details.Modules[i].Valid(); err != nil {
				utils.FromContext(c).WithError(err).Errorf("error validating policy module data '%d' '%s'", id, name)
				utils.HTTPError(c, srverrors.ErrGetPolicyInvalidModuleData, err)
				return
			}
			for _, module := range modules {
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
		utils.FromContext(c).WithError(err).Errorf("error finding policy groups by policy model")
		utils.HTTPError(c, srverrors.ErrGetPolicyGroupsNotFound, err)
		return
	}
	resp.Details.Groups = pgs.Groups

	utils.HTTPSuccess(c, http.StatusOK, resp)
}

// PatchPolicy is a function to update policy public info only
// @Summary Update policy info by policy hash
// @Tags Policies
// @Accept json
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Param json body models.Policy true "policy info as JSON data"
// @Success 200 {object} utils.successResp{data=models.Policy} "policy info updated successful"
// @Failure 400 {object} utils.errorResp "invalid policy info"
// @Failure 403 {object} utils.errorResp "updating policy info not permitted"
// @Failure 404 {object} utils.errorResp "policy not found"
// @Failure 500 {object} utils.errorResp "internal error on updating policy"
// @Router /policies/{hash} [put]
func (s *PolicyService) PatchPolicy(c *gin.Context) {
	var (
		count  int64
		hash   = c.Param("hash")
		policy models.Policy
	)
	uaf := utils.UserActionFields{
		Domain:            "policy",
		ObjectType:        "policy",
		ActionCode:        "editing",
		ObjectId:          hash,
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	serviceHash, ok := srvcontext.GetString(c, "svc")
	if !ok {
		utils.FromContext(c).Errorf("could not get service hash")
		utils.HTTPError(c, srverrors.ErrInternal, nil)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		utils.FromContext(c).WithError(err).Error()
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, err)
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
		utils.FromContext(c).WithError(err).Errorf("error binding JSON")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPoliciesInvalidRequest, err, uaf)
		return
	}
	uaf.ObjectDisplayName = policy.Info.Name.En

	if hash != policy.Hash {
		utils.FromContext(c).WithError(nil).Errorf("mismatch policy hash to requested one")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPoliciesInvalidRequest, err, uaf)
		return
	}

	if err = iDB.Model(&policy).Count(&count).Error; err != nil || count == 0 {
		utils.FromContext(c).WithError(nil).Errorf("error updating policy by hash '%s', group not found", hash)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPoliciesNotFound, err, uaf)
		return
	}

	public_info := []interface{}{"info", "updated_at"}
	err = iDB.Select("", public_info...).Save(&policy).Error

	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		utils.FromContext(c).WithError(nil).Errorf("error updating policy by hash '%s', policy not found", hash)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPoliciesNotFound, err, uaf)
		return
	} else if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error updating policy by hash '%s'", hash)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		return
	}

	utils.HTTPSuccessWithUAFields(c, http.StatusOK, policy, uaf)
}

// PatchPolicyGroup is a function to update policy group linking
// @Summary Update or patch policy group by policy hash and group object
// @Tags Policies,Groups
// @Accept json
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Param json body policyGroupPatch true "action on policy group as JSON data (activate, deactivate)"
// @Success 200 {object} utils.successResp "policy group patched successful"
// @Failure 400 {object} utils.errorResp "invalid patch request data"
// @Failure 403 {object} utils.errorResp "updating policy group not permitted"
// @Failure 404 {object} utils.errorResp "policy or group not found"
// @Failure 500 {object} utils.errorResp "internal error on updating policy group"
// @Router /policies/{hash}/groups [put]
func (s *PolicyService) PatchPolicyGroup(c *gin.Context) {
	var (
		form   policyGroupPatch
		group  models.Group
		hash   = c.Param("hash")
		policy models.Policy
	)
	uaf := utils.UserActionFields{
		Domain:            "policy",
		ObjectType:        "policy",
		ObjectId:          hash,
		ActionCode:        "undefined action",
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	serviceHash, ok := srvcontext.GetString(c, "svc")
	if !ok {
		utils.FromContext(c).Errorf("could not get service hash")
		utils.HTTPError(c, srverrors.ErrInternal, nil)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		utils.FromContext(c).WithError(err).Error()
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, err)
		return
	}

	if err = c.ShouldBindJSON(&form); err != nil {
		name, nameErr := getPolicyName(iDB, hash)
		if nameErr == nil {
			uaf.ObjectDisplayName = name
		}
		utils.FromContext(c).WithError(err).Errorf("error binding JSON")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPoliciesInvalidRequest, err, uaf)
		return
	}
	if form.Action == "activate" {
		uaf.ActionCode = "creation of the connection with the group"
	} else {
		uaf.ActionCode = "deletion of the connection with the group"
	}

	if err = iDB.Take(&policy, "hash = ?", hash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPoliciesNotFound, err, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		}
		return
	}
	uaf.ObjectDisplayName = policy.Info.Name.En
	if err = policy.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policy.Hash)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPoliciesInvalidData, err, uaf)
		return
	}

	if err = iDB.Take(&group, "hash = ?", form.Group.Hash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding group by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPoliciesNotFound, err, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		}
		return
	} else if err = group.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating group data '%s'", group.Hash)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchPolicyGroupInvalidGroupData, err, uaf)
		return
	}

	httpErr, err := makeGroupPolicyAction(form.Action, iDB, group, policy)
	if httpErr != nil {
		utils.FromContext(c).WithError(err).Errorf("error patching group policy by action: %s", httpErr.Error())
		utils.HTTPErrorWithUAFields(c, httpErr, err, uaf)
	}
	utils.HTTPSuccessWithUAFields(c, http.StatusOK, struct{}{}, uaf)
}

// CreatePolicy is a function to create new policy
// @Summary Create new policy in service
// @Tags Policies
// @Accept json
// @Produce json
// @Param json body policyInfo true "policy info to create one"
// @Success 201 {object} utils.successResp{data=models.Policy} "policy created successful"
// @Failure 400 {object} utils.errorResp "invalid policy info"
// @Failure 403 {object} utils.errorResp "creating policy not permitted"
// @Failure 500 {object} utils.errorResp "internal error on creating policy"
// @Router /policies/ [post]
func (s *PolicyService) CreatePolicy(c *gin.Context) {
	var (
		info       policyInfo
		policyFrom models.Policy
	)
	uaf := utils.UserActionFields{
		Domain:            "policy",
		ObjectType:        "policy",
		ActionCode:        "creation",
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	if err := c.ShouldBindJSON(&info); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding JSON")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPoliciesInvalidRequest, err, uaf)
		return
	}
	uaf.ObjectDisplayName = info.Name

	serviceHash, ok := srvcontext.GetString(c, "svc")
	if !ok {
		utils.FromContext(c).Errorf("could not get service hash")
		utils.HTTPError(c, srverrors.ErrInternal, nil)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		utils.FromContext(c).WithError(err).Error()
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, err)
		return
	}

	policy := models.Policy{
		Hash: utils.MakePolicyHash(info.Name),
		Info: models.PolicyInfo{
			Name: models.PolicyItemLocale{
				Ru: info.Name,
				En: info.Name,
			},
			Tags:   info.Tags,
			System: false,
		},
	}
	uaf.ObjectId = policy.Hash

	if info.From != 0 {
		if err = iDB.Take(&policyFrom, "id = ?", info.From).Error; err != nil {
			utils.FromContext(c).WithError(err).Errorf("error finding source policy by ID")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrCreatePolicySourceNotFound, err, uaf)
			return
		} else if err = policyFrom.Valid(); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policyFrom.Hash)
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPoliciesInvalidData, err, uaf)
			return
		}

		policy = policyFrom
		policy.ID = 0
		policy.Info.System = false
		policy.CreatedDate = time.Time{}
		policy.Hash = utils.MakePolicyHash(policy.Hash)
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
		utils.FromContext(c).WithError(err).Errorf("error creating policy")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrCreatePolicyCreateFail, err, uaf)
		return
	}

	if policyFrom.ID != 0 {
		var modules []models.ModuleA
		err = iDB.Where("policy_id = ? AND status = 'joined'", policyFrom.ID).Find(&modules).Error
		if err != nil {
			utils.FromContext(c).WithError(err).Errorf("error finding policy modules by policy ID '%d'", policyFrom.ID)
			utils.HTTPErrorWithUAFields(c, srverrors.ErrCreatePolicyModulesNotFound, err, uaf)
			return
		}
		for _, module := range modules {
			module.ID = 0
			module.PolicyID = policy.ID
			if err = iDB.Create(&module).Error; err != nil {
				utils.FromContext(c).WithError(err).Errorf("error creating policy module")
				utils.HTTPErrorWithUAFields(c, srverrors.ErrCreatePolicyCreateModulesFail, err, uaf)
				return
			}
		}
	}

	utils.HTTPSuccessWithUAFields(c, http.StatusCreated, policy, uaf)
}

// DeletePolicy is a function to cascade delete policy
// @Summary Delete policy from instance DB
// @Tags Policies
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Success 200 {object} utils.successResp "policy deleted successful"
// @Failure 403 {object} utils.errorResp "deleting policy not permitted"
// @Failure 404 {object} utils.errorResp "policy not found"
// @Failure 500 {object} utils.errorResp "internal error on deleting policy"
// @Router /policies/{hash} [delete]
func (s *PolicyService) DeletePolicy(c *gin.Context) {
	var (
		hash    = c.Param("hash")
		modules []models.ModuleAShort
		policy  models.Policy
		sv      *models.Service
	)
	uaf := utils.UserActionFields{
		Domain:            "policy",
		ObjectType:        "policy",
		ActionCode:        "deletion",
		ObjectId:          hash,
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	serviceHash, ok := srvcontext.GetString(c, "svc")
	if !ok {
		utils.FromContext(c).Errorf("could not get service hash")
		utils.HTTPError(c, srverrors.ErrInternal, nil)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		utils.FromContext(c).WithError(err).Error()
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, err)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalServiceNotFound, nil, uaf)
		return
	}

	if err = iDB.Take(&policy, "hash = ?", hash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPoliciesNotFound, err, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		}
		return
	} else if err = policy.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policy.Hash)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPoliciesInvalidData, err, uaf)
		return
	}
	uaf.ObjectDisplayName = policy.Info.Name.En

	if policy.Info.System {
		utils.FromContext(c).WithError(nil).Errorf("error removing system policy")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrDeletePolicySystemPolicy, err, uaf)
		return
	}

	pgs := models.PolicyGroups{
		Policy: policy,
	}
	if err = iDB.Model(pgs).Association("groups").Find(&pgs.Groups).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy groups by policy model")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrDeletePolicyGroupsNotFound, err, uaf)
		return
	}
	if len(pgs.Groups) != 0 {
		utils.FromContext(c).WithError(nil).Errorf("error removing policy which linked to groups")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrDeletePolicyPolicyLinkedToGroups, err, uaf)
		return
	}

	if err = iDB.Find(&modules, "policy_id = ?", policy.ID).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy modules by policy ID '%d'", policy.ID)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrDeletePolicyModulesNotFound, err, uaf)
		return
	}

	if err = iDB.Delete(&policy).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error deleting policy by hash '%s'", hash)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		return
	}

	for _, module := range modules {
		moduleName := module.Info.Name
		moduleVersion := module.Info.Version.String()
		if err = removeUnusedModuleVersion(c, iDB, moduleName, moduleVersion, sv); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error removing unused module data")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
			return
		}
	}

	utils.HTTPSuccessWithUAFields(c, http.StatusOK, struct{}{}, uaf)
}

// GetPoliciesCount is a function to return groups of counted policies
// @Summary Retrieve groups of counted policies
// @Tags Policies
// @Produce json
// @Success 200 {object} utils.successResp{data=policyCount} "groups of counted agents policies successfully"
// @Failure 500 {object} utils.errorResp "internal error"
// @Router /policies/count [get]
func (s *PolicyService) GetPoliciesCount(c *gin.Context) {
	var (
		resp policyCount
	)
	logger := utils.FromContext(c)
	uaf := utils.UserActionFields{
		Domain:            "policy",
		ObjectType:        "policy",
		ActionCode:        "counting",
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	serviceHash, ok := srvcontext.GetString(c, "svc")
	if !ok {
		utils.FromContext(c).Errorf("could not get service hash")
		utils.HTTPError(c, srverrors.ErrInternal, nil)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		utils.FromContext(c).WithError(err).Error()
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, err)
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
	err = iDB.Raw(q).
		Scan(&resp).
		Error
	if err != nil {
		logger.WithError(err).Errorf("could not count policies")
		utils.HTTPError(c, srverrors.ErrInternal, err)
		return
	}

	utils.HTTPSuccessWithUAFields(c, http.StatusOK, resp, uaf)
}
