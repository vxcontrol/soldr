package private

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	"soldr/pkg/app/api/client"
	"soldr/pkg/app/api/logger"
	"soldr/pkg/app/api/models"
	"soldr/pkg/app/api/modules"
	"soldr/pkg/app/api/server/response"
	"soldr/pkg/app/api/storage"
	"soldr/pkg/app/api/useraction"
	"soldr/pkg/crypto"
	"soldr/pkg/filestorage/s3"
	"soldr/pkg/semvertooling"
)

type agentModuleDetails struct {
	Name   string        `json:"name"`
	Update bool          `json:"update"`
	Policy models.Policy `json:"policy"`
}

type groupModuleDetails struct {
	Name   string        `json:"name"`
	Update bool          `json:"update"`
	Policy models.Policy `json:"policy"`
}

type policyModuleDetails struct {
	Name      string `json:"name"`
	Active    bool   `json:"active"`
	Exists    bool   `json:"exists"`
	Update    bool   `json:"update"`
	Duplicate bool   `json:"duplicate"`
}

type agentModules struct {
	Modules []models.ModuleA     `json:"modules"`
	Details []agentModuleDetails `json:"details"`
	Total   uint64               `json:"total"`
}

type groupModules struct {
	Modules []models.ModuleA     `json:"modules"`
	Details []groupModuleDetails `json:"details"`
	Total   uint64               `json:"total"`
}

type policyModules struct {
	Modules []models.ModuleA      `json:"modules"`
	Details []policyModuleDetails `json:"details"`
	Total   uint64                `json:"total"`
}

type policyModulesUpdates struct {
	Modules  []models.ModuleA `json:"modules"`
	Policies []models.Policy  `json:"policies"`
}

type systemModules struct {
	Modules []models.ModuleS `json:"modules"`
	Total   uint64           `json:"total"`
}

type systemShortModules struct {
	Modules []models.ModuleSShort `json:"modules"`
	Total   uint64                `json:"total"`
}

type policyModulePatch struct {
	// Action on group module must be one of activate, deactivate, update, store
	Action  string         `form:"action" json:"action" binding:"oneof=activate deactivate store update,required" default:"update" enums:"activate,deactivate,store,update"`
	Version string         `form:"version,omitempty" json:"version,omitempty" binding:"required_if=Action update,omitempty"`
	Module  models.ModuleA `form:"module,omitempty" json:"module,omitempty" binding:"required_if=Action store,omitempty"`
}

type moduleVersionPatch struct {
	// Action on group module must be one of store, release
	Action string         `form:"action" json:"action" binding:"oneof=store release,required" default:"store" enums:"store,release"`
	Module models.ModuleS `form:"module" json:"module" binding:"required"`
}

type systemModuleFile struct {
	Path string `form:"path" json:"path" binding:"required"`
	Data string `form:"data" json:"data" binding:"required" default:"base64"`
}

type systemModuleFilePatch struct {
	Action  string `form:"action" json:"action" binding:"oneof=move remove save,required" default:"save" enums:"move,remove,save"`
	Path    string `form:"path" json:"path" binding:"required"`
	Data    string `form:"data,omitempty" json:"data,omitempty" default:"base64" binding:"omitempty,required_if=Action save"`
	NewPath string `form:"newpath,omitempty" json:"newpath,omitempty" binding:"omitempty,required_if=Action move"`
}

const sqlAgentModuleDetails = `
	SELECT
		m.name
	FROM modules AS m
		LEFT JOIN policies p ON m.policy_id = p.id AND p.deleted_at IS NULL
		LEFT JOIN groups_to_policies AS gtp ON p.id = gtp.policy_id
		LEFT JOIN groups g ON gtp.group_id = g.id AND g.deleted_at IS NULL
		LEFT JOIN agents AS a ON g.id = a.group_id AND a.deleted_at IS NULL
	WHERE a.id = ? AND m.deleted_at IS NULL`

const sqlGroupModuleDetails = `
	SELECT
		m.name
	FROM modules AS m
		LEFT JOIN policies p ON m.policy_id = p.id AND p.deleted_at IS NULL
		LEFT JOIN groups_to_policies AS gtp ON p.id = gtp.policy_id
		LEFT JOIN groups g ON gtp.group_id = g.id AND g.deleted_at IS NULL
		LEFT JOIN agents AS a ON g.id = a.group_id AND a.deleted_at IS NULL
	WHERE g.id = ? AND m.deleted_at IS NULL
	GROUP BY m.name`

const sqlPolicyModuleDetails = `
	SELECT
		m.name,
		(1 = 1) AS 'exists',
		(m.status = "joined") AS active
	FROM modules AS m
		LEFT JOIN policies p ON m.policy_id = p.id AND p.deleted_at IS NULL
	WHERE p.id = ? AND m.deleted_at IS NULL`

var modulesSQLMappers = map[string]interface{}{
	"status":               "`{{table}}`.status",
	"system":               "`{{table}}`.system",
	"actions":              storage.ActionsMapper,
	"events":               storage.EventsMapper,
	"fields":               storage.FieldsMapper,
	"tags":                 storage.TagsMapper,
	"os":                   storage.ModulesOSMapper,
	"os_arch":              storage.ModulesOSArchMapper,
	"os_type":              storage.ModulesOSTypeMapper,
	"version":              "`{{table}}`.version",
	"ver_major":            "`{{table}}`.ver_major",
	"ver_minor":            "`{{table}}`.ver_minor",
	"ver_patch":            "`{{table}}`.ver_patch",
	"state":                "`{{table}}`.state",
	"localizedName":        "JSON_UNQUOTE(JSON_EXTRACT(`{{table}}`.locale, '$.module.{{lang}}.title'))",
	"localizedDescription": "JSON_UNQUOTE(JSON_EXTRACT(`{{table}}`.locale, '$.module.{{lang}}.description'))",
	"localizedTagsName":    "JSON_UNQUOTE(JSON_EXTRACT(`{{table}}`.locale, '$.tags.*.{{lang}}.title'))",
	"localizedEventsName":  "JSON_UNQUOTE(JSON_EXTRACT(`{{table}}`.locale, '$.events.*.{{lang}}.title'))",
	"data": "CONCAT(`{{table}}`.name, ' | ', `{{table}}`.version, ' | ', " +
		"COALESCE(JSON_KEYS(`{{table}}`.changelog), ''), ' | ', `{{table}}`.state, ' | ', " +
		"COALESCE(JSON_EXTRACT(`{{table}}`.info, '$.tags'), ''), ' | ', " +
		"COALESCE(JSON_EXTRACT(`{{table}}`.locale, '$.events.*.{{lang}}.title'), ''), ' | ', " +
		"COALESCE(JSON_EXTRACT(`{{table}}`.locale, '$.module.{{lang}}.title'), ''), ' | ', " +
		"COALESCE(JSON_EXTRACT(`{{table}}`.locale, '$.module.{{lang}}.description'), ''), ' | ', " +
		"COALESCE(JSON_EXTRACT(`{{table}}`.locale, '$.tags.*.{{lang}}.title'), ''))",
	"name": "`{{table}}`.name",
}

type ModuleService struct {
	templatesDir     string
	db               *gorm.DB
	serverConnector  *client.AgentServerClient
	userActionWriter useraction.Writer
	modulesStorage   *storage.ModuleStorage
}

func NewModuleService(
	templatesDir string,
	db *gorm.DB,
	serverConnector *client.AgentServerClient,
	userActionWriter useraction.Writer,
	modulesStorage *storage.ModuleStorage,
) *ModuleService {
	return &ModuleService{
		templatesDir:     templatesDir,
		db:               db,
		serverConnector:  serverConnector,
		userActionWriter: userActionWriter,
		modulesStorage:   modulesStorage,
	}
}

// GetAgentModules is a function to return agent module list view on dashboard
// @Summary Retrieve agent modules by agent hash and by filters
// @Tags Agents,Modules
// @Produce json
// @Param hash path string true "agent hash in hex format (md5)" minlength(32) maxlength(32)
// @Param request query storage.TableQuery true "query table params"
// @Success 200 {object} response.successResp{data=agentModules} "agent modules received successful"
// @Failure 400 {object} response.errorResp "invalid query request data"
// @Failure 403 {object} response.errorResp "getting agent modules not permitted"
// @Failure 404 {object} response.errorResp "agent or modules not found"
// @Failure 500 {object} response.errorResp "internal error on getting agent modules"
// @Router /agents/{hash}/modules [get]
func (s *ModuleService) GetAgentModules(c *gin.Context) {
	var (
		hash  = c.Param("hash")
		pids  []uint64
		query storage.TableQuery
		resp  agentModules
		sv    *models.Service
	)

	if err := c.ShouldBindQuery(&query); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error binding query")
		response.Error(c, response.ErrModulesInvalidRequest, err)
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

	var agentPolicies models.AgentPolicies
	if err = iDB.Take(&agentPolicies, "hash = ?", hash).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding agent by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrGetAgentModulesAgentNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	}
	if err = iDB.Model(agentPolicies).Association("policies").Find(&agentPolicies.Policies).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding agent policies by agent model")
		response.Error(c, response.ErrGetAgentModulesAgentPoliciesNotFound, err)
		return
	}
	if err = agentPolicies.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating agent policies data '%s'", agentPolicies.Hash)
		response.Error(c, response.ErrGetAgentModulesInvalidAgentPoliciesData, err)
		return
	}

	for _, p := range agentPolicies.Policies {
		pids = append(pids, p.ID)
	}

	query.Init("modules", modulesSQLMappers)
	query.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Where("policy_id IN (?) AND status LIKE 'joined'", pids)
		},
		func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL")
		},
	})
	query.SetOrders([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Order("name asc")
		},
	})
	if resp.Total, err = query.Query(iDB, &resp.Modules); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding agent modules")
		response.Error(c, response.ErrGetAgentModulesInvalidQuery, err)
		return
	}

	for i := 0; i < len(resp.Modules); i++ {
		if err = resp.Modules[i].Valid(); err != nil {
			logger.FromContext(c).WithError(err).
				Errorf("error validating agent module data '%s'", resp.Modules[i].Info.Name)
			response.Error(c, response.ErrGetAgentModulesInvalidAgentData, err)
			return
		}
	}

	modNames := []string{""}
	for _, module := range resp.Modules {
		modNames = append(modNames, module.Info.Name)
	}
	moduleList := make([]models.ModuleS, 0)
	tid := c.GetUint64("tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return modules.LatestModulesQuery(db).
			Where("name IN (?) AND tenant_id = ? AND service_type = ?", modNames, tid, sv.Type)
	}

	if err = s.db.Scopes(scope).Find(&moduleList).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding system modules list by names")
		response.Error(c, response.ErrGetAgentsGetSystemModulesFail, err)
		return
	}

	var details []agentModuleDetails
	if err = iDB.Raw(sqlAgentModuleDetails, agentPolicies.ID).Scan(&details).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error loading agents modules details")
		response.Error(c, response.ErrGetAgentModulesDetailsNotFound, err)

		return
	}
	for _, ma := range resp.Modules {
		rmd := agentModuleDetails{Name: ma.Info.Name}
		for _, md := range details {
			if md.Name == ma.Info.Name {
				rmd = md
				break
			}
		}
		for _, pd := range agentPolicies.Policies {
			if pd.ID == ma.PolicyID {
				rmd.Policy = pd
			}
		}
		for _, ms := range moduleList {
			if ms.Info.Name == ma.Info.Name {
				rmd.Update = ma.Info.Version.String() != ms.Info.Version.String()
				break
			}
		}
		resp.Details = append(resp.Details, rmd)
	}

	response.Success(c, http.StatusOK, resp)
}

// GetAgentModule is a function to return agent module by name
// @Summary Retrieve agent module data by agent hash and module name
// @Tags Agents,Modules
// @Produce json
// @Param hash path string true "agent hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Success 200 {object} response.successResp{data=models.ModuleA} "agent module data received successful"
// @Failure 403 {object} response.errorResp "getting agent module data not permitted"
// @Failure 404 {object} response.errorResp "agent or module not found"
// @Failure 500 {object} response.errorResp "internal error on getting agent module"
// @Router /agents/{hash}/modules/{module_name} [get]
func (s *ModuleService) GetAgentModule(c *gin.Context) {
	var (
		hash       = c.Param("hash")
		module     models.ModuleA
		moduleName = c.Param("module_name")
		pids       []uint64
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

	var agentPolicies models.AgentPolicies
	if err = iDB.Take(&agentPolicies, "hash = ?", hash).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding agent by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrGetAgentModuleAgentNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	}
	if err = iDB.Model(agentPolicies).Association("policies").Find(&agentPolicies.Policies).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding agent policies by agent model")
		response.Error(c, response.ErrGetAgentModuleAgentPoliceNotFound, err)
		return
	}
	if err = agentPolicies.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating agent policies data '%s'", agentPolicies.Hash)
		response.Error(c, response.ErrGetAgentModuleInvalidAgentPoliceData, err)
		return
	}

	for _, p := range agentPolicies.Policies {
		pids = append(pids, p.ID)
	}

	scopeModule := func(db *gorm.DB) *gorm.DB {
		return db.Where("policy_id IN (?) AND name = ? AND status = 'joined'", pids, moduleName)
	}
	if err = iDB.Scopes(scopeModule).Take(&module).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy module by module name")
		response.Error(c, response.ErrModulesNotFound, err)
		return
	} else if err = module.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating module data '%s'", module.Info.Name)
		response.Error(c, response.ErrModulesInvalidData, err)
		return
	}

	response.Success(c, http.StatusOK, module)
}

// GetAgentBModule is a function to return bmodule vue code as a file
// @Summary Retrieve browser module vue code by agent hash and module name
// @Tags Agents,Modules
// @Produce text/javascript,application/javascript,json
// @Param hash path string true "agent hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Param file query string false "path to the browser module file" default(main.vue)
// @Success 200 {file} file "browser module vue code as a file"
// @Failure 403 {object} response.errorResp "getting agent module data not permitted"
// @Router /agents/{hash}/modules/{module_name}/bmodule.vue [get]
func (s *ModuleService) GetAgentBModule(c *gin.Context) {
	var (
		data       []byte
		filepath   = path.Join("/", c.DefaultQuery("file", "main.vue"))
		hash       = c.Param("hash")
		module     models.ModuleA
		moduleName = c.Param("module_name")
		pids       []uint64
		sv         *models.Service
	)

	defer func() {
		ctype := mime.TypeByExtension(path.Ext(filepath))
		if ctype == "" {
			ctype = "application/octet-stream"
		}
		c.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%q", path.Base(filepath)))
		c.Data(http.StatusOK, ctype, data)
	}()

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
		return
	}

	var agentPolicies models.AgentPolicies
	if err = iDB.Take(&agentPolicies, "hash = ?", hash).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding agent by hash")
		return
	}
	if err = iDB.Model(agentPolicies).Association("policies").Find(&agentPolicies.Policies).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding agent policies by agent model")
		return
	}
	if err = agentPolicies.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating agent policies data '%s'", agentPolicies.Hash)
		response.Error(c, response.ErrGetAgentBModuleInvalidAgentPoliceData, err)
		return
	}

	for _, p := range agentPolicies.Policies {
		pids = append(pids, p.ID)
	}

	scopeModule := func(db *gorm.DB) *gorm.DB {
		return db.Where("policy_id IN (?) AND name = ? AND status = 'joined'", pids, moduleName)
	}
	if err = iDB.Scopes(scopeModule).Take(&module).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy module by module name")
		return
	} else if err = module.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating module data '%s'", module.Info.Name)
		response.Error(c, response.ErrModulesInvalidData, err)
		return
	}

	s3Client, err := s3.New(sv.Info.S3.ToS3ConnParams())
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error openning connection to S3")
		return
	}

	path := path.Join(moduleName, module.Info.Version.String(), "bmodule", filepath)
	if data, err = s3Client.ReadFile(path); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error reading module file '%s'", path)
		return
	}
}

// GetGroupModules is a function to return group module list view on dashboard
// @Summary Retrieve group modules by group hash and by filters
// @Tags Groups,Modules
// @Produce json
// @Param hash path string true "group hash in hex format (md5)" minlength(32) maxlength(32)
// @Param request query storage.TableQuery true "query table params"
// @Success 200 {object} response.successResp{data=groupModules} "group modules received successful"
// @Failure 400 {object} response.errorResp "invalid query request data"
// @Failure 403 {object} response.errorResp "getting group modules not permitted"
// @Failure 404 {object} response.errorResp "group or modules not found"
// @Failure 500 {object} response.errorResp "internal error on getting group modules"
// @Router /groups/{hash}/modules [get]
func (s *ModuleService) GetGroupModules(c *gin.Context) {
	var (
		gps   models.GroupPolicies
		hash  = c.Param("hash")
		pids  []uint64
		query storage.TableQuery
		resp  groupModules
		sv    *models.Service
	)

	if err := c.ShouldBindQuery(&query); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error binding query")
		response.Error(c, response.ErrModulesInvalidRequest, err)
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

	if err = iDB.Take(&gps, "hash = ?", hash).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding group by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrGetGroupModulesGroupNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	}
	if err = iDB.Model(gps).Association("policies").Find(&gps.Policies).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding group policies by group model")
		response.Error(c, response.ErrGetGroupModulesGroupPoliciesNotFound, err)
		return
	}
	if err = gps.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating group policies data '%s'", gps.Hash)
		response.Error(c, response.ErrGetGroupModulesInvalidGroupPoliciesData, err)
		return
	}

	for _, p := range gps.Policies {
		pids = append(pids, p.ID)
	}

	query.Init("modules", modulesSQLMappers)
	query.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Where("policy_id IN (?) AND status LIKE 'joined'", pids)
		},
		func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL")
		},
	})
	query.SetOrders([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Order("name asc")
		},
	})
	if resp.Total, err = query.Query(iDB, &resp.Modules); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding group modules")
		response.Error(c, response.ErrGetGroupModulesInvalidGroupQuery, err)
		return
	}

	for i := 0; i < len(resp.Modules); i++ {
		if err = resp.Modules[i].Valid(); err != nil {
			logger.FromContext(c).WithError(err).
				Errorf("error validating group module data '%s'",
					resp.Modules[i].Info.Name)
			response.Error(c, response.ErrGetGroupModulesInvalidGroupData, err)
			return
		}
	}

	modNames := []string{""}
	for _, module := range resp.Modules {
		modNames = append(modNames, module.Info.Name)
	}
	moduleList := make([]models.ModuleS, 0)
	tid := c.GetUint64("tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return modules.LatestModulesQuery(db).Where("name IN (?) AND tenant_id = ? AND service_type = ?", modNames, tid, sv.Type)
	}

	if err = s.db.Scopes(scope).Find(&moduleList).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding system modules list by names")
		response.Error(c, response.ErrGetGroupsGetSystemModulesFail, err)
		return
	}

	var details []groupModuleDetails
	if err = iDB.Raw(sqlGroupModuleDetails, gps.ID).Scan(&details).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error loading group modules details")
		response.Error(c, response.ErrGetGroupModulesDetailsNotFound, err)
		return
	}
	for _, ma := range resp.Modules {
		rmd := groupModuleDetails{Name: ma.Info.Name}
		for _, md := range details {
			if md.Name == ma.Info.Name {
				rmd = md
				break
			}
		}
		for _, pd := range gps.Policies {
			if pd.ID == ma.PolicyID {
				rmd.Policy = pd
			}
		}
		for _, ms := range moduleList {
			if ms.Info.Name == ma.Info.Name {
				rmd.Update = ma.Info.Version.String() != ms.Info.Version.String()
				break
			}
		}
		resp.Details = append(resp.Details, rmd)
	}

	response.Success(c, http.StatusOK, resp)
}

// GetGroupModule is a function to return group module by name
// @Summary Retrieve group module data by group hash and module name
// @Tags Groups,Modules
// @Produce json
// @Param hash path string true "group hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Success 200 {object} response.successResp{data=models.ModuleA} "group module data received successful"
// @Failure 403 {object} response.errorResp "getting group module data not permitted"
// @Failure 404 {object} response.errorResp "group or module not found"
// @Failure 500 {object} response.errorResp "internal error on getting group"
// @Router /groups/{hash}/modules/{module_name} [get]
func (s *ModuleService) GetGroupModule(c *gin.Context) {
	var (
		gps        models.GroupPolicies
		hash       = c.Param("hash")
		module     models.ModuleA
		moduleName = c.Param("module_name")
		pids       []uint64
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

	if err = iDB.Take(&gps, "hash = ?", hash).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding group by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrGetGroupModuleGroupNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	}
	if err = iDB.Model(gps).Association("policies").Find(&gps.Policies).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding group policies by group model")
		response.Error(c, response.ErrGetGroupModuleGroupPoliciesNotFound, err)
		return
	}
	if err = gps.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating group policies data '%s'", gps.Hash)
		response.Error(c, response.ErrGetGroupModuleInvalidGroupPoliciesData, err)
		return
	}

	for _, p := range gps.Policies {
		pids = append(pids, p.ID)
	}

	scopeModule := func(db *gorm.DB) *gorm.DB {
		return db.Where("policy_id IN (?) AND name = ? AND status = 'joined'", pids, moduleName)
	}
	if err = iDB.Scopes(scopeModule).Take(&module).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy module by module name")
		response.Error(c, response.ErrModulesNotFound, err)

		return
	} else if err = module.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating module data '%s'", module.Info.Name)
		response.Error(c, response.ErrModulesInvalidData, err)
		return
	}

	response.Success(c, http.StatusOK, module)
}

// GetGroupBModule is a function to return bmodule vue code as a file
// @Summary Retrieve browser module vue code by group hash and module name
// @Tags Groups,Modules
// @Produce text/javascript,application/javascript,json
// @Param hash path string true "group hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Param file query string false "path to the browser module file" default(main.vue)
// @Success 200 {file} file "browser module vue code as a file"
// @Failure 403 {object} response.errorResp "getting group module data not permitted"
// @Router /groups/{hash}/modules/{module_name}/bmodule.vue [get]
func (s *ModuleService) GetGroupBModule(c *gin.Context) {
	var (
		data       []byte
		filepath   = path.Join("/", c.DefaultQuery("file", "main.vue"))
		gps        models.GroupPolicies
		hash       = c.Param("hash")
		module     models.ModuleA
		moduleName = c.Param("module_name")
		pids       []uint64
		sv         *models.Service
	)

	defer func() {
		ctype := mime.TypeByExtension(path.Ext(filepath))
		if ctype == "" {
			ctype = "application/octet-stream"
		}
		c.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%q", path.Base(filepath)))
		c.Data(http.StatusOK, ctype, data)
	}()

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
		return
	}

	if err = iDB.Take(&gps, "hash = ?", hash).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding group by hash")
		return
	}
	if err = iDB.Model(gps).Association("policies").Find(&gps.Policies).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding group policies by group model")
		return
	}
	if err = gps.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating group policies data '%s'", gps.Hash)
		response.Error(c, response.ErrGetGroupBModuleInvalidGroupPoliciesData, err)
		return
	}

	for _, p := range gps.Policies {
		pids = append(pids, p.ID)
	}

	scopeModule := func(db *gorm.DB) *gorm.DB {
		return db.Where("policy_id IN (?) AND name = ? AND status = 'joined'", pids, moduleName)
	}
	if err = iDB.Scopes(scopeModule).Take(&module).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy module by module name")
		return
	} else if err = module.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating module data '%s'", module.Info.Name)
		response.Error(c, response.ErrModulesInvalidData, err)
		return
	}

	s3Client, err := s3.New(sv.Info.S3.ToS3ConnParams())
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error openning connection to S3")
		return
	}

	path := path.Join(moduleName, module.Info.Version.String(), "bmodule", filepath)
	if data, err = s3Client.ReadFile(path); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error reading module file '%s'", path)
		return
	}
}

// GetPolicyModules is a function to return policy module list view on dashboard
// @Summary Retrieve policy modules by policy hash and by filters
// @Tags Policies,Modules
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Param request query storage.TableQuery true "query table params"
// @Success 200 {object} response.successResp{data=policyModules} "policy modules received successful"
// @Failure 400 {object} response.errorResp "invalid query request data
// @Failure 403 {object} response.errorResp "getting policy modules not permitted"
// @Failure 404 {object} response.errorResp "policy or modules not found"
// @Failure 500 {object} response.errorResp "internal error on getting policy modules"
// @Router /policies/{hash}/modules [get]
func (s *ModuleService) GetPolicyModules(c *gin.Context) {
	var (
		hash     = c.Param("hash")
		modulesA []models.ModuleA
		modulesS []models.ModuleS
		policy   models.Policy
		query    storage.TableQuery
		resp     policyModules
		sv       *models.Service
	)

	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, response.ErrModulesInvalidRequest, err)
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

	if err = iDB.Take(&policy, "hash = ?", hash).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrGetPolicyModulesPolicyNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = policy.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policy.Hash)
		response.Error(c, response.ErrGetPolicyModulesInvalidPolicyData, err)
		return
	}

	tid := c.GetUint64("tid")

	queryA := query
	queryA.Page = 0
	queryA.Size = 0
	queryA.Filters = []storage.TableFilter{}
	queryA.Init("modules", modulesSQLMappers)
	queryA.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Where("policy_id = ?", policy.ID)
		},
		func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL")
		},
	})
	if _, err = queryA.Query(iDB, &modulesA); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy modules")
		response.Error(c, response.ErrGetPolicyModulesInvalidPolicyQuery, err)
		return
	}

	queryS := query
	queryS.Init("modules", modulesSQLMappers)
	queryS.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Where("tenant_id IN (0, ?) AND service_type = ?", tid, sv.Type)
		},
	})
	queryS.SetOrders([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Order("status desc").Order("name asc")
		},
	})
	modNames := []string{""}
	for _, module := range modulesA {
		modNames = append(modNames, module.Info.Name)
	}
	funcs := []func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return modules.LatestModulesQuery(db).Omit("id").
				Select("`modules`.*, IF(`name` IN (?), 'joined', 'inactive') AS `status`", modNames)
		},
	}
	if resp.Total, err = queryS.Query(s.db, &modulesS, funcs...); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding system modules")
		response.Error(c, response.ErrGetPolicyModulesInvalidModulesQuery, err)
		return
	}

	var details []policyModuleDetails
	if err = iDB.Raw(sqlPolicyModuleDetails, policy.ID).Scan(&details).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error loading policies modules details")
		response.Error(c, response.ErrGetPolicyModulesDetailsNotFound, err)
		return
	}

	getModule := func(name string) *models.ModuleA {
		for _, m := range modulesA {
			if m.Info.Name == name {
				return &m
			}
		}
		return nil
	}
	getDetails := func(name string, ma *models.ModuleA, ms *models.ModuleS) policyModuleDetails {
		rmd := policyModuleDetails{Name: name}
		for _, md := range details {
			if md.Name == name {
				rmd = md
				break
			}
		}
		rmd.Update = ma.Info.Version.String() != ms.Info.Version.String()
		return rmd
	}
	for _, ms := range modulesS {
		ma := getModule(ms.Info.Name)
		if ma == nil {
			mt := ms.ToModuleA()
			mt.Status = "inactive"
			resp.Modules = append(resp.Modules, mt)
			resp.Details = append(resp.Details, getDetails(ms.Info.Name, &mt, &ms))
		} else {
			resp.Modules = append(resp.Modules, *ma)
			resp.Details = append(resp.Details, getDetails(ms.Info.Name, ma, &ms))
		}
	}

	for i := 0; i < len(resp.Modules); i++ {
		if err = resp.Modules[i].Valid(); err != nil {
			logger.FromContext(c).WithError(err).
				Errorf("error validating policy module data '%s'", resp.Modules[i].Info.Name)
			response.Error(c, response.ErrGetPolicyModulesInvalidPolicyData, err)
			return
		}
	}

	var moduleNames []string
	for _, v := range resp.Modules {
		if v.Status == "inactive" {
			moduleNames = append(moduleNames, v.Info.Name)
		}
	}
	duplicateMap, err := modules.CheckMultipleModulesDuplicate(iDB, moduleNames, policy.ID)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error checking duplicate modules")
		response.Error(c, response.ErrInternal, err)
		return
	}
	for i, v := range resp.Details {
		if _, ok := duplicateMap[v.Name]; ok {
			resp.Details[i].Duplicate = true
		}
	}
	response.Success(c, http.StatusOK, resp)
}

// GetPolicyModule is a function to return policy module by name
// @Summary Retrieve policy module data by policy hash and module name
// @Tags Policies,Modules
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Success 200 {object} response.successResp{data=models.ModuleA} "policy module data received successful"
// @Failure 403 {object} response.errorResp "getting policy module data not permitted"
// @Failure 404 {object} response.errorResp "policy or module not found"
// @Failure 500 {object} response.errorResp "internal error on getting policy module"
// @Router /policies/{hash}/modules/{module_name} [get]
func (s *ModuleService) GetPolicyModule(c *gin.Context) {
	var (
		hash       = c.Param("hash")
		moduleName = c.Param("module_name")
		module     models.ModuleA
		policy     models.Policy
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

	if err = iDB.Take(&policy, "hash = ?", hash).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrGetPolicyModulePolicyNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = policy.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policy.Hash)
		response.Error(c, response.ErrGetPolicyModuleInvalidPolicyData, err)
		return
	}

	if err = iDB.Take(&module, "policy_id = ? AND name = ?", policy.ID, moduleName).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy module by name")
		response.Error(c, response.ErrModulesNotFound, err)
		return
	} else if err = module.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating module data '%s'", module.Info.Name)
		response.Error(c, response.ErrModulesInvalidData, err)
		return
	}

	response.Success(c, http.StatusOK, module)
}

// GetPolicyBModule is a function to return bmodule vue code as a file
// @Summary Retrieve browser module vue code by policy hash and module name
// @Tags Policies,Modules
// @Produce text/javascript,application/javascript,json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Param file query string false "path to the browser module file" default(main.vue)
// @Success 200 {file} file "browser module vue code as a file"
// @Failure 403 {object} response.errorResp "getting policy module data not permitted"
// @Router /policies/{hash}/modules/{module_name}/bmodule.vue [get]
func (s *ModuleService) GetPolicyBModule(c *gin.Context) {
	var (
		data       []byte
		filepath   = path.Join("/", c.DefaultQuery("file", "main.vue"))
		hash       = c.Param("hash")
		module     models.ModuleA
		moduleName = c.Param("module_name")
		policy     models.Policy
		sv         *models.Service
	)

	defer func() {
		ctype := mime.TypeByExtension(path.Ext(filepath))
		if ctype == "" {
			ctype = "application/octet-stream"
		}
		c.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%q", path.Base(filepath)))
		c.Data(http.StatusOK, ctype, data)
	}()

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
		return
	}

	if err = iDB.Take(&policy, "hash = ?", hash).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		return
	}

	if err = iDB.Take(&module, "policy_id = ? AND name = ?", policy.ID, moduleName).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy module by name")
		return
	}

	s3Client, err := s3.New(sv.Info.S3.ToS3ConnParams())
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error openning connection to S3")
		return
	}

	path := path.Join(moduleName, module.Info.Version.String(), "bmodule", filepath)
	if data, err = s3Client.ReadFile(path); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error reading module file '%s'", path)
		return
	}
}

// PatchPolicyModule is a function to update policy module info and status
// @Summary Update or patch policy module data by policy hash and module name
// @Tags Policies,Modules
// @Accept json
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Param json body policyModulePatch true "action on policy module as JSON data (activate, deactivate, store, update)"
// @Success 200 {object} response.successResp "policy module patched successful"
// @Failure 403 {object} response.errorResp "updating policy module not permitted"
// @Failure 404 {object} response.errorResp "policy or module not found"
// @Failure 500 {object} response.errorResp "internal error on updating policy module"
// @Router /policies/{hash}/modules/{module_name} [put]
func (s *ModuleService) PatchPolicyModule(c *gin.Context) {
	var (
		form       policyModulePatch
		hash       = c.Param("hash")
		moduleA    models.ModuleA
		moduleName = c.Param("module_name")
		moduleS    models.ModuleS
		policy     models.Policy
		sv         *models.Service
		encryptor  crypto.IDBConfigEncryptor
	)

	uaf := useraction.NewFields(c, "policy", "policy", "editing", hash, useraction.UnknownObjectDisplayName)
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

	if err := c.ShouldBindJSON(&form); err != nil {
		name, nameErr := getPolicyName(iDB, hash)
		if nameErr == nil {
			uaf.ObjectDisplayName = name
		}
		logger.FromContext(c).WithError(err).Errorf("error binding JSON")
		response.Error(c, response.ErrModulesInvalidRequest, err)
		return
	}

	if encryptor = modules.GetDBEncryptor(c); encryptor == nil {
		response.Error(c, response.ErrInternalDBEncryptorNotFound, nil)
		return
	}

	if err = iDB.Take(&policy, "hash = ?", hash).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrPatchPolicyModulePolicyNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	}
	uaf.ObjectDisplayName = policy.Info.Name.En

	if err = policy.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policy.Hash)
		response.Error(c, response.ErrPatchPolicyModuleInvalidPolicyData, err)
		return
	}

	if sv = modules.GetService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, err)
		return
	}

	tid := c.GetUint64("tid")
	scope := func(db *gorm.DB) *gorm.DB {
		db = db.Where("name = ? AND tenant_id IN (0, ?) AND service_type = ?", moduleName, tid, sv.Type)
		switch form.Version {
		case "latest", "":
			return db.Order("ver_major DESC, ver_minor DESC, ver_patch DESC")
		default:
			return db.Where("version = ?", form.Version)
		}
	}

	if err = s.db.Scopes(scope).Take(&moduleS).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrModulesSystemModuleNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = moduleS.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", moduleS.Info.Name)
		response.Error(c, response.ErrModulesInvalidSystemModuleData, err)
		return
	}

	if err = iDB.Take(&moduleA, "policy_id = ? AND name = ?", policy.ID, moduleName).Error; err != nil {
		moduleA.FromModuleS(&moduleS)
		moduleA.PolicyID = policy.ID
	} else if err = moduleA.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating module data '%s'", moduleA.Info.Name)
		response.Error(c, response.ErrModulesInvalidData, err)
		return
	}

	if moduleA.ID == 0 && form.Action != "activate" {
		logger.FromContext(c).Errorf("error on %s module, policy module not found", form.Action)
		response.Error(c, response.ErrModulesInvalidSystemModuleData, err)
		return
	}

	incl := []interface{}{"status", "last_update"}
	excl := []string{"policy_id", "status", "join_date", "last_update"}
	switch form.Action {
	case "activate":
		if err = modules.CheckModulesDuplicate(iDB, &moduleA); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error checking duplicate modules")
			response.Error(c, response.ErrPatchPolicyModuleDuplicatedModule, err)
			return
		}

		if moduleA.ID == 0 {
			checksums, err := modules.CopyModuleAFilesToInstanceS3(&moduleA.Info, sv)
			if err != nil {
				logger.FromContext(c).WithError(err).Errorf("error copying module files to S3")
				response.Error(c, response.ErrInternal, err)
				return
			}

			moduleA.FilesChecksums = checksums

			if err = moduleA.ValidateEncryption(encryptor); err != nil {
				logger.FromContext(c).WithError(err).Errorf("module config not encrypted")
				response.Error(c, response.ErrModulesDataNotEncryptedOnDBInsert, nil)
				return
			}

			if err = iDB.Create(&moduleA).Error; err != nil {
				logger.FromContext(c).WithError(err).Errorf("error creating module")
				response.Error(c, response.ErrInternal, err)
				return
			}

			if moduleS.State == "draft" {
				if err = modules.UpdatePolicyModulesByModuleS(c, &moduleS, sv); err != nil {
					response.Error(c, response.ErrInternal, err)
					return
				}
			}
		} else {
			moduleA.Status = "joined"
			if err = iDB.Select("", incl...).Save(&moduleA).Error; err != nil {
				logger.FromContext(c).WithError(err).Errorf("error updating module")
				response.Error(c, response.ErrInternal, err)
				return
			}
		}

	case "deactivate":
		moduleA.Status = "inactive"
		if err = iDB.Select("", incl...).Save(&moduleA).Error; err != nil {
			logger.FromContext(c).WithError(err).Errorf("error updating module")
			response.Error(c, response.ErrInternal, err)
			return
		}

	case "store":
		if err = form.Module.Valid(); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error validating module")
			response.Error(c, response.ErrPatchPolicyModuleNewModuleInvalid, err)
			return
		}

		changes, err := modules.CompareModulesChanges(form.Module, moduleA, encryptor)
		if err != nil {
			logger.FromContext(c).WithError(err).Errorf("failed to compare modules changes")
			response.Error(c, response.ErrModulesFailedToCompareChanges, err)
			return
		}

		for _, ch := range changes {
			if ch {
				logger.FromContext(c).Errorf("error accepting module changes")
				response.Error(c, response.ErrPatchPolicyModuleAcceptFail, nil)
				return
			}
		}

		err = form.Module.EncryptSecureParameters(encryptor)
		if err != nil {
			logger.FromContext(c).WithError(err).Errorf("failed to encrypt module secure config")
			response.Error(c, response.ErrModulesFailedToEncryptSecureConfig, err)
			return
		}
		if err = iDB.Omit(excl...).Save(&form.Module).Error; err != nil {
			logger.FromContext(c).WithError(err).Errorf("error saving module")
			response.Error(c, response.ErrInternal, err)
			return
		}

	case "update":
		moduleVersion := moduleA.Info.Version.String()
		if moduleVersion == moduleS.Info.Version.String() {
			logger.FromContext(c).Errorf("error updating module to the same version: %s", moduleVersion)
			response.Error(c, response.ErrInternal, err)
			return
		}

		moduleA, err = modules.MergeModuleAConfigFromModuleS(&moduleA, &moduleS, encryptor)
		if err != nil {
			logger.FromContext(c).WithError(err).Errorf("invalid module state")
			response.Error(c, response.ErrInternal, err)
			return
		}

		if err = moduleA.Valid(); err != nil {
			logger.FromContext(c).WithError(err).Errorf("invalid module state")
			response.Error(c, response.ErrInternal, err)
			return
		}

		err = moduleA.EncryptSecureParameters(encryptor)
		if err != nil {
			logger.FromContext(c).WithError(err).Errorf("failed to encrypt module secure config")
			response.Error(c, response.ErrModulesFailedToEncryptSecureConfig, err)
			return
		}

		checksums, err := modules.CopyModuleAFilesToInstanceS3(&moduleA.Info, sv)
		if err != nil {
			logger.FromContext(c).WithError(err).Errorf("error copying module files to S3")
			response.Error(c, response.ErrInternal, err)
			return
		}
		moduleA.FilesChecksums = checksums

		if err = iDB.Omit(excl...).Save(&moduleA).Error; err != nil {
			logger.FromContext(c).WithError(err).Errorf("error updating module")
			response.Error(c, response.ErrInternal, err)
			return
		}

		if err = modules.RemoveUnusedModuleVersion(c, iDB, moduleName, moduleVersion, sv); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error removing unused module data")
			response.Error(c, response.ErrInternal, err)
			return
		}

	default:
		logger.FromContext(c).Errorf("error making unknown action on module")
		response.Error(c, response.ErrPatchPolicyModuleActionNotFound, nil)
		return
	}

	response.Success(c, http.StatusOK, struct{}{})
}

// DeletePolicyModule is a function to delete policy module instance
// @Summary Delete module instance by policy hash and module name
// @Tags Policies,Modules
// @Accept json
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Success 200 {object} response.successResp "policy module deleted successful"
// @Failure 403 {object} response.errorResp "deleting policy module not permitted"
// @Failure 404 {object} response.errorResp "policy or module not found"
// @Failure 500 {object} response.errorResp "internal error on deleting policy module"
// @Router /policies/{hash}/modules/{module_name} [delete]
func (s *ModuleService) DeletePolicyModule(c *gin.Context) {
	var (
		hash       = c.Param("hash")
		module     models.ModuleA
		moduleName = c.Param("module_name")
		policy     models.Policy
		sv         *models.Service
	)

	uaf := useraction.NewFields(c, "policy", "policy", "editing", hash, useraction.UnknownObjectDisplayName)
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

	if err = iDB.Take(&policy, "hash = ?", hash).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrDeletePolicyModulePolicyNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = policy.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policy.Hash)
		response.Error(c, response.ErrDeletePolicyModuleInvalidPolicyData, err)
		return
	}
	uaf.ObjectDisplayName = policy.Info.Name.En

	if sv = modules.GetService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	if err = iDB.Take(&module, "policy_id = ? AND name = ?", policy.ID, moduleName).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrModulesNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = module.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating module data '%s'", module.Info.Name)
		response.Error(c, response.ErrModulesInvalidData, err)
		return
	}

	if err = iDB.Delete(&module).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error deleting policy module by name '%s'", moduleName)
		response.Error(c, response.ErrInternal, err)
		return
	}

	moduleVersion := module.Info.Version.String()
	if err = modules.RemoveUnusedModuleVersion(c, iDB, moduleName, moduleVersion, sv); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error removing unused module data")
		response.Error(c, response.ErrInternal, err)
		return
	}

	response.Success(c, http.StatusOK, struct{}{})
}

// SetPolicyModuleSecureConfigValue is a function to set secured parameter value in policy module
// @Summary Set parameter value in secured current config for a module
// @Tags Policies,Modules
// @Accept json
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Param json body models.ModuleConfig true "param name and value to be set"
// @Success 200 {object} response.successResp "parameter updated successfully"
// @Success 400 {object} response.errorResp "bad request"
// @Failure 403 {object} response.errorResp "updating parameter not permitted"
// @Failure 404 {object} response.errorResp "policy or module not found"
// @Failure 500 {object} response.errorResp "internal error on updating secured parameter"
// @Router /policies/{hash}/modules/{module_name}/secure_config [post]
func (s *ModuleService) SetPolicyModuleSecureConfigValue(c *gin.Context) {
	var (
		payload    models.ModuleConfig
		moduleA    models.ModuleA
		moduleS    models.ModuleS
		policy     models.Policy
		sv         *models.Service
		hash       = c.Param("hash")
		moduleName = c.Param("module_name")
		encryptor  crypto.IDBConfigEncryptor
		paramName  string
	)

	uaf := useraction.NewFields(c, "policy", "policy", "setting value to module secure config", hash, useraction.UnknownObjectDisplayName)
	defer s.userActionWriter.WriteUserAction(c, uaf)

	err := c.ShouldBindJSON(&payload)
	switch {
	case err != nil:
		logger.FromContext(c).WithError(err).Errorf("error binding JSON")
		response.Error(c, response.ErrModulesInvalidRequest, err)
		return
	case len(payload) != 1:
		logger.FromContext(c).WithError(err).Errorf("only one key-value pair in body allowed")
		response.Error(c, response.ErrModulesInvalidRequest, err)
		return
	}

	for k := range payload {
		paramName = k
	}
	uaf.ActionCode = fmt.Sprintf("%s, key: %s", uaf.ActionCode, paramName)

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
		response.Error(c, response.ErrInternalServiceNotFound, err)
		return
	}

	if encryptor = modules.GetDBEncryptor(c); encryptor == nil {
		response.Error(c, response.ErrInternalDBEncryptorNotFound, nil)
		return
	}

	if err = iDB.Take(&policy, "hash = ?", hash).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrPatchPolicyModulePolicyNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = policy.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policy.Hash)
		response.Error(c, response.ErrPatchPolicyModuleInvalidPolicyData, err)
		return
	}

	tid := c.GetUint64("tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id IN (0, ?) AND service_type = ?", moduleName, tid, sv.Type).
			Order("ver_major DESC, ver_minor DESC, ver_patch DESC") // latest
	}

	if err = s.db.Scopes(scope).Take(&moduleS).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrModulesSystemModuleNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = moduleS.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", moduleS.Info.Name)
		response.Error(c, response.ErrModulesInvalidSystemModuleData, err)
		return
	}
	uaf.ObjectDisplayName = moduleS.Locale.Module["en"].Title

	if err = iDB.Take(&moduleA, "policy_id = ? AND name = ?", policy.ID, moduleName).Error; err != nil {
		moduleA.FromModuleS(&moduleS)
		moduleA.PolicyID = policy.ID
	} else if err = moduleA.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating module data '%s'", moduleA.Info.Name)
		response.Error(c, response.ErrModulesInvalidData, err)
		return
	}

	if err = moduleA.DecryptSecureParameters(encryptor); err != nil {
		logger.FromContext(c).WithError(err).
			Errorf("error decrypting module data '%s'", moduleA.Info.Name)
		response.Error(c, response.ErrModulesFailedToDecryptSecureConfig, err)
		return
	}

	param, ok := moduleA.SecureCurrentConfig[paramName]
	if !ok {
		logger.FromContext(c).WithError(err).Errorf("module secure parameter not exists")
		response.Error(c, response.ErrModulesInvalidRequest, err)
		return
	}

	moduleA.SecureCurrentConfig[paramName] = models.ModuleSecureParameter{
		ServerOnly: param.ServerOnly,
		Value:      payload[paramName],
	}
	if err = moduleA.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error saving module")
		response.Error(c, response.ErrModulesInvalidParameterValue, err)
		return
	}

	err = moduleA.EncryptSecureParameters(encryptor)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("failed to encrypt module secure config")
		response.Error(c, response.ErrModulesFailedToEncryptSecureConfig, err)
		return
	}
	if err = iDB.Save(&moduleA).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error saving module")
		response.Error(c, response.ErrInternal, err)
		return
	}

	response.Success(c, http.StatusOK, struct{}{})
}

// GetPolicyModuleSecureConfigValue is a function to get secured parameter value in policy module
// @Summary Get parameter value in secured current config for a module
// @Tags Policies,Modules
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Param param_name path string true "parameter name without spaces"
// @Success 200 {object} models.ModuleConfig "secured param value received successfully"
// @Failure 403 {object} response.errorResp "get secured parameter not permitted"
// @Failure 404 {object} response.errorResp "policy, module or parameter not found"
// @Failure 500 {object} response.errorResp "internal error on getting module secured parameter"
// @Router /policies/{hash}/modules/{module_name}/secure_config/{param_name} [get]
func (s *ModuleService) GetPolicyModuleSecureConfigValue(c *gin.Context) {
	var (
		moduleA    models.ModuleA
		moduleS    models.ModuleS
		policy     models.Policy
		sv         *models.Service
		hash       = c.Param("hash")
		moduleName = c.Param("module_name")
		paramName  = c.Param("param_name")
		encryptor  crypto.IDBConfigEncryptor
	)

	actionCode := fmt.Sprintf("retrieving value in module secure config, key: %s", paramName)
	uaf := useraction.NewFields(c, "policy", "policy", actionCode, hash, useraction.UnknownObjectDisplayName)
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
		response.Error(c, response.ErrInternalServiceNotFound, err)
		return
	}

	if encryptor = modules.GetDBEncryptor(c); encryptor == nil {
		response.Error(c, response.ErrInternalDBEncryptorNotFound, nil)
		return
	}

	if err = iDB.Take(&policy, "hash = ?", hash).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrPatchPolicyModulePolicyNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = policy.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policy.Hash)
		response.Error(c, response.ErrPatchPolicyModuleInvalidPolicyData, err)
		return
	}

	tid := c.GetUint64("tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id IN (0, ?) AND service_type = ?", moduleName, tid, sv.Type).
			Order("ver_major DESC, ver_minor DESC, ver_patch DESC") // latest
	}

	if err = s.db.Scopes(scope).Take(&moduleS).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrModulesSystemModuleNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = moduleS.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", moduleS.Info.Name)
		response.Error(c, response.ErrModulesInvalidSystemModuleData, err)
		return
	}
	uaf.ObjectDisplayName = moduleS.Locale.Module["en"].Title

	if err = iDB.Take(&moduleA, "policy_id = ? AND name = ?", policy.ID, moduleName).Error; err != nil {
		moduleA.FromModuleS(&moduleS)
		moduleA.PolicyID = policy.ID
	} else if err = moduleA.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating module data '%s'", moduleA.Info.Name)
		response.Error(c, response.ErrModulesInvalidData, err)
		return
	}

	if err = moduleA.DecryptSecureParameters(encryptor); err != nil {
		logger.FromContext(c).WithError(err).
			Errorf("error decrypting module data '%s'", moduleA.Info.Name)
		response.Error(c, response.ErrModulesFailedToDecryptSecureConfig, err)
		return
	}

	val, ok := moduleA.SecureCurrentConfig[paramName]
	if !ok {
		logger.FromContext(c).WithError(err).Errorf("module secure parameter not exists")
		response.Error(c, response.ErrModulesInvalidRequest, err)
		return
	}

	resp := make(models.ModuleConfig)
	resp[paramName] = val.Value
	response.Success(c, http.StatusOK, resp)
}

// GetModules is a function to return system module list
// @Summary Retrieve system modules by filters
// @Tags Modules
// @Produce json
// @Param request query storage.TableQuery true "query table params"
// @Success 200 {object} response.successResp{data=systemModules} "system modules received successful"
// @Failure 400 {object} response.errorResp "invalid query request data"
// @Failure 403 {object} response.errorResp "getting system modules not permitted"
// @Failure 500 {object} response.errorResp "internal error on getting system modules"
// @Router /modules/ [get]
func (s *ModuleService) GetModules(c *gin.Context) {
	var (
		query      storage.TableQuery
		sv         *models.Service
		resp       systemModules
		useVersion bool
		encryptor  crypto.IDBConfigEncryptor
	)

	if err := c.ShouldBindQuery(&query); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error binding query")
		response.Error(c, response.ErrModulesInvalidRequest, err)
		return
	}

	if sv = modules.GetService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	if encryptor = modules.GetDBEncryptor(c); encryptor == nil {
		response.Error(c, response.ErrInternalDBEncryptorNotFound, nil)
		return
	}

	tid := c.GetUint64("tid")

	query.Init("modules", modulesSQLMappers)

	setUsingTables := func(sfield string) {
		if sfield == "version" {
			useVersion = true
		}
	}
	setUsingTables(query.Sort.Prop)
	for _, filter := range query.Filters {
		setUsingTables(filter.Field)
	}
	query.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Where("tenant_id IN (0, ?) AND service_type = ?", tid, sv.Type)
		},
	})
	query.SetOrders([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Order("name asc")
		},
	})
	funcs := []func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			if !useVersion {
				db = modules.LatestModulesQuery(db)
			}
			return db
		},
	}
	total, err := query.Query(s.db, &resp.Modules, funcs...)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding system modules")
		response.Error(c, response.ErrGetModulesInvalidModulesQuery, err)
		return
	}
	resp.Total = total

	for _, module := range resp.Modules {
		if err = module.Valid(); err != nil {
			logger.FromContext(c).WithError(err).
				Errorf("error validating system module data '%s'", module.Info.Name)
			response.Error(c, response.ErrModulesInvalidSystemModuleData, err)
			return
		}

		if err = module.DecryptSecureParameters(encryptor); err != nil {
			logger.FromContext(c).WithError(err).
				Errorf("error decrypting module data '%s'", module.Info.Name)
			response.Error(c, response.ErrModulesFailedToDecryptSecureConfig, err)
			return
		}
	}

	response.Success(c, http.StatusOK, resp)
}

// CreateModule is a function to create new system module
// @Summary Create new system module from template
// @Tags Modules
// @Accept json
// @Produce json
// @Param json body models.ModuleInfo true "module info to create one"
// @Success 201 {object} response.successResp{data=models.ModuleS} "system module created successful"
// @Failure 400 {object} response.errorResp "invalid system module info"
// @Failure 403 {object} response.errorResp "creating system module not permitted"
// @Failure 500 {object} response.errorResp "internal error on creating system module"
// @Router /modules/ [post]
func (s *ModuleService) CreateModule(c *gin.Context) {
	var (
		count     int64
		info      models.ModuleInfo
		module    *models.ModuleS
		sv        *models.Service
		template  modules.Template
		encryptor crypto.IDBConfigEncryptor
	)

	uaf := useraction.NewFields(c, "module", "module", "creation", "", useraction.UnknownObjectDisplayName)
	defer s.userActionWriter.WriteUserAction(c, uaf)

	if sv = modules.GetService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	if encryptor = modules.GetDBEncryptor(c); encryptor == nil {
		response.Error(c, response.ErrInternalDBEncryptorNotFound, nil)
		return
	}

	tid := c.GetUint64("tid")

	if err := c.ShouldBindJSON(&info); err != nil || info.Valid() != nil {
		if err == nil {
			err = info.Valid()
		}
		logger.FromContext(c).WithError(err).Errorf("error binding JSON")
		response.Error(c, response.ErrCreateModuleInvalidInfo, err)
		return
	}
	uaf.ObjectID = info.Name

	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", info.Name, tid, sv.Type)
	}

	if err := s.db.Scopes(scope).Model(&module).Count(&count).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding number of system module")
		response.Error(c, response.ErrCreateModuleGetCountFail, err)
		return
	} else if count >= 1 {
		logger.FromContext(c).Errorf("error creating second system module")
		response.Error(c, response.ErrCreateModuleSecondSystemModule, err)
		return
	}

	info.System = false

	var err error
	if template, module, err = modules.LoadModuleSTemplate(&info, s.templatesDir); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error loading module")
		response.Error(c, response.ErrCreateModuleLoadFail, err)
		return
	}
	uaf.ObjectDisplayName = module.Locale.Module["en"].Title

	module.State = "draft"
	module.TenantID = tid
	module.ServiceType = sv.Type
	if err = module.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating module")
		response.Error(c, response.ErrCreateModuleValidationFail, err)
		return
	}

	if err = modules.StoreModuleSToGlobalS3(&info, template); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error storing module to S3")
		response.Error(c, response.ErrCreateModuleStoreS3Fail, err)
		return
	}

	if err = module.DecryptSecureParameters(encryptor); err != nil {
		response.Error(c, response.ErrModulesFailedToEncryptSecureConfig, nil)
		return
	}
	if err = s.db.Create(module).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error creating module")
		response.Error(c, response.ErrCreateModuleStoreDBFail, err)
		return
	}

	response.Success(c, http.StatusCreated, module)
}

// DeleteModule is a function to cascade delete system module
// @Summary Delete system module from all DBs and S3 storage
// @Tags Modules
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Success 200 {object} response.successResp "system module deleted successful"
// @Failure 403 {object} response.errorResp "deleting system module not permitted"
// @Failure 404 {object} response.errorResp "system module or services not found"
// @Failure 500 {object} response.errorResp "internal error on deleting system module"
// @Router /modules/{module_name} [delete]
func (s *ModuleService) DeleteModule(c *gin.Context) {
	var (
		err        error
		moduleList []models.ModuleS
		moduleName = c.Param("module_name")
		sv         *models.Service
		services   []models.Service
	)

	uaf := useraction.NewFields(c, "module", "module", "deletion", moduleName, useraction.UnknownObjectDisplayName)
	defer s.userActionWriter.WriteUserAction(c, uaf)

	if sv = modules.GetService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	tid := c.GetUint64("tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err = s.db.Scopes(scope).Find(&moduleList).Error; err != nil || len(moduleList) == 0 {
		logger.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if err == nil && len(moduleList) == 0 {
			response.Error(c, response.ErrModulesNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	}
	uaf.ObjectDisplayName = moduleList[len(moduleList)-1].Locale.Module["en"].Title

	deletePolicyModule := func(svc *models.Service) error {
		iDB, err := s.serverConnector.GetDB(c, svc.Hash)
		if err != nil {
			return err
		}

		var agentModules []models.ModuleA
		if err = iDB.Find(&agentModules, "name = ?", moduleName).Error; err != nil {
			logger.FromContext(c).WithError(err).Errorf("error finding modules by name")
			return err
		} else if len(agentModules) == 0 {
			return modules.UpdateDependenciesWhenModuleRemove(c, iDB, moduleName)
		} else if err = iDB.Where("name = ?", moduleName).Delete(&agentModules).Error; err != nil {
			logger.FromContext(c).WithError(err).Errorf("error deleting module by name '%s'", moduleName)
			return err
		}

		s3Client, err := s3.New(sv.Info.S3.ToS3ConnParams())
		if err != nil {
			logger.FromContext(c).WithError(err).Errorf("error openning connection to S3")
			return err
		}

		if err = s3Client.RemoveDir(moduleName + "/"); err != nil && err.Error() != "not found" {
			logger.FromContext(c).WithError(err).Errorf("error removing modules files")
			return err
		}

		if err = modules.UpdateDependenciesWhenModuleRemove(c, iDB, moduleName); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error updating module dependencies")
			return err
		}

		return nil
	}

	if err = s.db.Find(&services, "tenant_id = ? AND type = ?", tid, sv.Type).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding services")
		response.Error(c, response.ErrDeleteModuleServiceNotFound, err)
		return
	}

	for _, s := range services {
		if err = deletePolicyModule(&s); err != nil {
			response.Error(c, response.ErrDeleteModuleDeleteFail, err)
			return
		}
	}

	if err = s.db.Where("name = ?", moduleName).Delete(&moduleList).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error deleting system module by name '%s'", moduleName)
		response.Error(c, response.ErrInternal, err)
		return
	}

	s3Client, err := s3.New(nil)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error openning connection to S3")
		response.Error(c, response.ErrInternal, err)
		return
	}

	if err = s3Client.RemoveDir(moduleName + "/"); err != nil && err.Error() != "not found" {
		logger.FromContext(c).WithError(err).Errorf("error removing system modules files")
		response.Error(c, response.ErrDeleteModuleDeleteFilesFail, err)
		return
	}

	response.Success(c, http.StatusOK, struct{}{})
}

// GetModuleVersions is a function to return all versions for system module
// @Summary Retrieve all version for system module by filters and module name
// @Tags Modules
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param request query storage.TableQuery true "query table params"
// @Success 200 {object} response.successResp{data=systemShortModules} "system modules received successful"
// @Failure 400 {object} response.errorResp "invalid query request data"
// @Failure 403 {object} response.errorResp "getting system modules not permitted"
// @Failure 500 {object} response.errorResp "internal error on getting system modules"
// @Router /modules/{module_name}/versions/ [get]
func (s *ModuleService) GetModuleVersions(c *gin.Context) {
	var (
		moduleName = c.Param("module_name")
		query      storage.TableQuery
		sv         *models.Service
		resp       systemShortModules
	)

	if err := c.ShouldBindQuery(&query); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error binding query")
		response.Error(c, response.ErrModulesInvalidRequest, err)
		return
	}

	if sv = modules.GetService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	tid := c.GetUint64("tid")

	query.Init("modules", modulesSQLMappers)
	query.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Where("name = ? AND tenant_id IN (0, ?) AND service_type = ?", moduleName, tid, sv.Type)
		},
	})
	query.SetOrders([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Order("ver_major DESC, ver_minor DESC, ver_patch DESC")
		},
	})
	total, err := query.Query(s.db, &resp.Modules)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding system modules")
		response.Error(c, response.ErrGetModuleVersionsInvalidModulesQuery, err)
		return
	}
	resp.Total = total

	for i := 0; i < len(resp.Modules); i++ {
		if err = resp.Modules[i].Valid(); err != nil {
			logger.FromContext(c).WithError(err).
				Errorf("error validating system module data '%s'", resp.Modules[i].Info.Name)
			response.Error(c, response.ErrModulesInvalidSystemModuleData, err)
			return
		}
	}

	response.Success(c, http.StatusOK, resp)
}

// GetModuleVersion is a function to return system module by name and version
// @Summary Retrieve system module data by module name and version
// @Tags Modules
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format" default(latest)
// @Success 200 {object} response.successResp{data=models.ModuleS} "system module data received successful"
// @Failure 403 {object} response.errorResp "getting system module data not permitted"
// @Failure 404 {object} response.errorResp "system module not found"
// @Failure 500 {object} response.errorResp "internal error on getting system module"
// @Router /modules/{module_name}/versions/{version} [get]
func (s *ModuleService) GetModuleVersion(c *gin.Context) {
	var (
		module     models.ModuleS
		moduleName = c.Param("module_name")
		sv         *models.Service
		version    = c.Param("version")
		encryptor  crypto.IDBConfigEncryptor
	)

	if sv = modules.GetService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	if encryptor = modules.GetDBEncryptor(c); encryptor == nil {
		response.Error(c, response.ErrInternalDBEncryptorNotFound, nil)
		return
	}

	tid := c.GetUint64("tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err := s.db.Scopes(modules.FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrModulesNotFound, err)

		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = module.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		response.Error(c, response.ErrModulesInvalidSystemModuleData, err)
		return
	}

	if err := module.DecryptSecureParameters(encryptor); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error decrypting module data")
		response.Error(c, response.ErrModulesFailedToDecryptSecureConfig, err)
		return
	}

	response.Success(c, http.StatusOK, module)
}

// PatchModuleVersion is a function to update system module by name and version
// @Summary Update the version of system module to global DB and global S3 storage
// @Tags Modules
// @Accept json
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format" default(latest)
// @Param json body moduleVersionPatch true "module info to create one"
// @Success 200 {object} response.successResp "system module updated successful"
// @Failure 403 {object} response.errorResp "updating system module not permitted"
// @Failure 404 {object} response.errorResp "system module or services not found"
// @Failure 500 {object} response.errorResp "internal error on updating system module"
// @Router /modules/{module_name}/versions/{version} [put]
func (s *ModuleService) PatchModuleVersion(c *gin.Context) {
	var (
		cfiles     map[string][]byte
		module     models.ModuleS
		moduleName = c.Param("module_name")
		form       moduleVersionPatch
		sv         *models.Service
		services   []models.Service
		template   = make(modules.Template)
		version    = c.Param("version")
		encryptor  crypto.IDBConfigEncryptor
	)

	uaf := useraction.NewFields(c, "module", "module", "undefined action", moduleName, useraction.UnknownObjectDisplayName)
	defer s.userActionWriter.WriteUserAction(c, uaf)

	if err := c.ShouldBindJSON(&form); err != nil || form.Module.Valid() != nil {
		if err == nil {
			err = form.Module.Valid()
		}
		name, nameErr := modules.GetModuleName(c, s.db, moduleName, version)
		if nameErr == nil {
			uaf.ObjectDisplayName = name
		}
		logger.FromContext(c).WithError(err).Errorf("error binding JSON")
		response.Error(c, response.ErrModulesInvalidRequest, err)
		return
	}

	if form.Action == "release" {
		uaf.ActionCode = "release version release"
	} else {
		uaf.ActionCode = "module editing"
	}
	uaf.ObjectDisplayName = form.Module.Locale.Module["en"].Title

	if sv = modules.GetService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	if encryptor = modules.GetDBEncryptor(c); encryptor == nil {
		response.Error(c, response.ErrInternalDBEncryptorNotFound, nil)
		return
	}

	tid := c.GetUint64("tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err := s.db.Scopes(modules.FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrModulesSystemModuleNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = module.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		response.Error(c, response.ErrModulesInvalidSystemModuleData, err)
		return
	}

	if module.State == "release" {
		logger.FromContext(c).Errorf("error changing released system module")
		response.Error(c, response.ErrPatchModuleVersionAcceptReleaseChangesFail, nil)
		return
	}

	changes := []bool{
		module.ID != form.Module.ID,
		module.Info.Name != form.Module.Info.Name,
		module.Info.System != form.Module.Info.System,
		module.Info.Template != form.Module.Info.Template,
		module.Info.Version.String() != form.Module.Info.Version.String(),
		module.ServiceType != form.Module.ServiceType,
		module.TenantID != form.Module.TenantID,
	}
	for _, ch := range changes {
		if ch {
			logger.FromContext(c).Errorf("error accepting system module changes")
			response.Error(c, response.ErrPatchModuleVersionAcceptSystemChangesFail, nil)
			return
		}
	}

	err := form.Module.EncryptSecureParameters(encryptor)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("failed to encrypt module secure config")
		response.Error(c, response.ErrModulesFailedToEncryptSecureConfig, err)
		return
	}
	if sqlResult := s.db.Omit("last_update").Save(&form.Module); sqlResult.Error != nil {
		logger.FromContext(c).WithError(sqlResult.Error).Errorf("error saving system module")
		response.Error(c, response.ErrPatchModuleVersionUpdateFail, err)
		return
	} else if sqlResult.RowsAffected != 0 {
		if cfiles, err = modules.BuildModuleSConfig(&form.Module); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error building system module files")
			response.Error(c, response.ErrPatchModuleVersionBuildFilesFail, err)
			return
		}

		template["config"] = cfiles
		if err = modules.StoreModuleSToGlobalS3(&form.Module.Info, template); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error storing system module files to S3")
			response.Error(c, response.ErrPatchModuleVersionUpdateS3Fail, err)
			return
		}
	}

	if module.State == "draft" && form.Module.State == "release" {
		if err = s.db.Find(&services, "tenant_id = ? AND type = ?", tid, sv.Type).Error; err != nil {
			logger.FromContext(c).WithError(err).Errorf("error finding services")
			response.Error(c, response.ErrPatchModuleVersionServiceNotFound, err)
			return
		}

		if err = s.db.Model(&form.Module).Take(&module).Error; err != nil {
			logger.FromContext(c).WithError(err).Errorf("error finding system module by id '%d'", form.Module.ID)
			response.Error(c, response.ErrInternal, err)
			return
		}

		for _, svc := range services {
			if err = modules.UpdatePolicyModulesByModuleS(c, &module, &svc); err != nil {
				response.Error(c, response.ErrInternal, err)
				return
			}
		}
	}

	response.Success(c, http.StatusOK, struct{}{})
}

// CreateModuleVersion is a function to create new system module version
// @Summary Create new system module version from latest released version
// @Tags Modules
// @Accept json
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format"
// @Param json body models.ChangelogVersion true "module changelog to add to created module"
// @Success 201 {object} response.successResp{data=models.ModuleS} "system module created successful"
// @Failure 400 {object} response.errorResp "invalid system module info"
// @Failure 403 {object} response.errorResp "creating system module not permitted"
// @Failure 404 {object} response.errorResp "system module not found"
// @Failure 500 {object} response.errorResp "internal error on creating system module"
// @Router /modules/{module_name}/versions/{version} [post]
func (s *ModuleService) CreateModuleVersion(c *gin.Context) {
	var (
		cfiles     map[string][]byte
		clver      models.ChangelogVersion
		count      int64
		module     models.ModuleS
		moduleName = c.Param("module_name")
		sv         *models.Service
		template   modules.Template
		version    = c.Param("version")
		encryptor  crypto.IDBConfigEncryptor
	)

	uaf := useraction.NewFields(c, "module", "module", "creation of the draft", moduleName, useraction.UnknownObjectDisplayName)
	defer s.userActionWriter.WriteUserAction(c, uaf)

	if sv = modules.GetService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	if encryptor = modules.GetDBEncryptor(c); encryptor == nil {
		response.Error(c, response.ErrInternalDBEncryptorNotFound, nil)
		return
	}

	if err := c.ShouldBindJSON(&clver); err != nil || clver.Valid() != nil {
		if err == nil {
			err = clver.Valid()
		}
		name, nameErr := modules.GetModuleName(c, s.db, moduleName, "latest")
		if nameErr == nil {
			uaf.ObjectDisplayName = name
		}
		logger.FromContext(c).WithError(err).Errorf("error binding JSON")
		response.Error(c, response.ErrModulesInvalidRequest, err)
		return
	}

	tid := c.GetUint64("tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err := s.db.Scopes(modules.FilterModulesByVersion("latest"), scope).Take(&module).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrModulesSystemModuleNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	}

	uaf.ObjectDisplayName = module.Locale.Module["en"].Title

	if err := module.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		response.Error(c, response.ErrModulesInvalidSystemModuleData, err)
		return
	}

	draft := func(db *gorm.DB) *gorm.DB {
		return db.Where("state LIKE ?", "draft")
	}

	if err := s.db.Scopes(scope, draft).Model(&module).Count(&count).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding number of system module drafts")
		response.Error(c, response.ErrCreateModuleVersionGetDraftNumberFail, err)
		return
	} else if count >= 1 {
		logger.FromContext(c).Errorf("error creating system module second draft")
		response.Error(c, response.ErrCreateModuleVersionSecondSystemModuleDraft, err)
		return
	}

	newModuleVersion, err := semver.NewVersion(version)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error parsing new version '%s'", version)
		response.Error(c, response.ErrCreateModuleVersionInvalidModuleVersionFormat, err)
		return
	}

	switch semvertooling.CompareVersions(module.Info.Version.String(), version) {
	case semvertooling.TargetVersionGreat:
	default:
		logger.FromContext(c).Errorf("error validating new version '%s' -> '%s'",
			module.Info.Version.String(), version)
		response.Error(c, response.ErrCreateModuleVersionInvalidModuleVersion, nil)
		return
	}

	if template, err = modules.LoadModuleSFromGlobalS3(&module.Info); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error building system module files")
		response.Error(c, response.ErrCreateModuleVersionBuildFilesFail, err)
		return
	}

	module.ID = 0
	module.State = "draft"
	module.LastUpdate = time.Now()
	module.Info.Version.Major = newModuleVersion.Major()
	module.Info.Version.Minor = newModuleVersion.Minor()
	module.Info.Version.Patch = newModuleVersion.Patch()
	module.Changelog[module.Info.Version.String()] = clver
	if err = module.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating module")
		response.Error(c, response.ErrCreateModuleVersionValidationFail, err)
		return
	}

	if cfiles, err = modules.BuildModuleSConfig(&module); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error building system module files")
		response.Error(c, response.ErrCreateModuleVersionBuildFilesFail, err)
		return
	}

	template["config"] = cfiles
	if err = modules.StoreModuleSToGlobalS3(&module.Info, template); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error storing module to S3")
		response.Error(c, response.ErrCreateModuleVersionStoreS3Fail, err)
		return
	}

	err = module.EncryptSecureParameters(encryptor)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("failed to encrypt module secure config")
		response.Error(c, response.ErrModulesFailedToEncryptSecureConfig, err)
		return
	}
	if err = s.db.Create(&module).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error creating module")
		response.Error(c, response.ErrCreateModuleVersionStoreDBFail, err)
		return
	}

	response.Success(c, http.StatusCreated, module)
}

// DeleteModuleVersion is a function to delete the version system module
// @Summary Delete the version system module from global DB and global S3 storage
// @Tags Modules
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format" default(latest)
// @Success 200 {object} response.successResp "system module deleted successful"
// @Failure 403 {object} response.errorResp "deleting system module not permitted"
// @Failure 404 {object} response.errorResp "system module not found"
// @Failure 500 {object} response.errorResp "internal error on deleting system module"
// @Router /modules/{module_name}/versions/{version} [delete]
func (s *ModuleService) DeleteModuleVersion(c *gin.Context) {
	var (
		count      int64
		module     models.ModuleS
		moduleName = c.Param("module_name")
		sv         *models.Service
		version    = c.Param("version")
	)

	uaf := useraction.NewFields(c, "module", "module", "deletion of the version", moduleName, useraction.UnknownObjectDisplayName)
	defer s.userActionWriter.WriteUserAction(c, uaf)

	if sv = modules.GetService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	tid := c.GetUint64("tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err := s.db.Scopes(scope).Model(&module).Count(&count).Error; err != nil || count == 0 {
		logger.FromContext(c).WithError(err).Errorf("error finding number of system module versions")
		response.Error(c, response.ErrDeleteModuleVersionGetVersionNumberFail, err)
		return
	} else if count == 1 {
		logger.FromContext(c).Errorf("error deleting last system module version")
		response.Error(c, response.ErrDeleteModuleVersionDeleteLastVersionFail, nil)
		return
	}

	if err := s.db.Scopes(modules.FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrModulesSystemModuleNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = module.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		response.Error(c, response.ErrModulesInvalidSystemModuleData, err)
		return
	}
	uaf.ObjectDisplayName = module.Locale.Module["en"].Title

	if err := s.db.Delete(&module).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error deleting system module by name '%s'", moduleName)
		response.Error(c, response.ErrInternal, err)
		return
	}

	s3, err := s3.New(nil)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error openning connection to S3")
		response.Error(c, response.ErrInternal, err)
		return
	}

	path := moduleName + "/" + module.Info.Version.String() + "/"
	if err = s3.RemoveDir(path); err != nil && err.Error() != "not found" {
		logger.FromContext(c).WithError(err).Errorf("error removing system modules files")
		response.Error(c, response.ErrDeleteModuleVersionDeleteFilesFail, err)
		return
	}

	response.Success(c, http.StatusOK, struct{}{})
}

// GetModuleVersionUpdates is a function to return policy modules list ready to update
// @Summary Retrieve policy modules list ready to update by system module name and version
// @Tags Modules
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format" default(latest)
// @Success 200 {object} response.successResp{data=policyModulesUpdates} "policy modules list received successful"
// @Failure 403 {object} response.errorResp "getting policy modules list not permitted"
// @Failure 404 {object} response.errorResp "system module not found"
// @Failure 500 {object} response.errorResp "internal error on getting policy modules list to update"
// @Router /modules/{module_name}/versions/{version}/updates [get]
func (s *ModuleService) GetModuleVersionUpdates(c *gin.Context) {
	var (
		module     models.ModuleS
		moduleName = c.Param("module_name")
		pids       []uint64
		resp       policyModulesUpdates
		scope      func(db *gorm.DB) *gorm.DB
		sv         *models.Service
		version    = c.Param("version")
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
	scope = func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err = s.db.Scopes(modules.FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrModulesSystemModuleNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = module.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		response.Error(c, response.ErrModulesInvalidSystemModuleData, err)
		return
	}

	scope = func(db *gorm.DB) *gorm.DB {
		return db.Where("name LIKE ? AND version LIKE ? AND last_module_update != ?",
			module.Info.Name, module.Info.Version.String(), module.LastUpdate)
	}
	if err = iDB.Scopes(scope).Find(&resp.Modules).Error; err != nil {
		logger.FromContext(c).WithError(err).
			Errorf("error finding policy modules by name and version '%s' '%s'",
				module.Info.Name, module.Info.Version.String())
		response.Error(c, response.ErrInternal, err)
		return
	}

	for _, module := range resp.Modules {
		pids = append(pids, module.PolicyID)
	}

	if err = iDB.Find(&resp.Policies, "id IN (?)", pids).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding policies by IDs")
		response.Error(c, response.ErrInternal, err)
		return
	} else {
		for _, policy := range resp.Policies {
			if err = policy.Valid(); err != nil {
				logger.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policy.Hash)
				response.Error(c, response.ErrGetModuleVersionUpdatesInvalidPolicyData, err)
				return
			}
		}
	}

	response.Success(c, http.StatusOK, resp)
}

// CreateModuleVersionUpdates is a function to run policy modules update
// @Summary Run policy modules update by system module name and version
// @Tags Modules
// @Accept json
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format" default(latest)
// @Success 201 {object} response.successResp "policy modules update run successful"
// @Failure 403 {object} response.errorResp "running policy modules updates not permitted"
// @Failure 404 {object} response.errorResp "system module not found"
// @Failure 500 {object} response.errorResp "internal error on running policy modules updates"
// @Router /modules/{module_name}/versions/{version}/updates [post]
func (s *ModuleService) CreateModuleVersionUpdates(c *gin.Context) {
	var (
		moduleName = c.Param("module_name")
		module     models.ModuleS
		svc        *models.Service
		version    = c.Param("version")
	)

	uaf := useraction.NewFields(c, "module", "module", "version update in policies", moduleName, useraction.UnknownObjectDisplayName)
	defer s.userActionWriter.WriteUserAction(c, uaf)

	if svc = modules.GetService(c); svc == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	tid := c.GetUint64("tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, svc.Type)
	}

	if err := s.db.Scopes(modules.FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrModulesSystemModuleNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = module.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		response.Error(c, response.ErrModulesInvalidSystemModuleData, err)
		return
	}
	uaf.ObjectDisplayName = module.Locale.Module["en"].Title

	if err := modules.UpdatePolicyModulesByModuleS(c, &module, svc); err != nil {
		response.Error(c, response.ErrInternal, err)
		return
	}

	uaf.Success = true
	response.Success(c, http.StatusCreated, struct{}{})
}

// GetModuleVersionFiles is a function to return system module file list
// @Summary Retrieve system module files (relative path) by module name and version
// @Tags Modules
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format" default(latest)
// @Success 200 {object} response.successResp{data=[]string} "system module files received successful"
// @Failure 403 {object} response.errorResp "getting system module files not permitted"
// @Failure 404 {object} response.errorResp "system module not found"
// @Failure 500 {object} response.errorResp "internal error on getting system module files"
// @Router /modules/{module_name}/versions/{version}/files [get]
func (s *ModuleService) GetModuleVersionFiles(c *gin.Context) {
	var (
		files      []string
		module     models.ModuleS
		moduleName = c.Param("module_name")
		sv         *models.Service
		version    = c.Param("version")
	)

	if sv = modules.GetService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	tid := c.GetUint64("tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err := s.db.Scopes(modules.FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrModulesSystemModuleNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = module.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		response.Error(c, response.ErrModulesInvalidSystemModuleData, err)
		return
	}

	s3Client, err := s3.New(nil)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error openning connection to S3")
		response.Error(c, response.ErrInternal, err)
		return
	}

	path := moduleName + "/" + module.Info.Version.String()
	if files, err = modules.ReadDir(s3Client, path); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error listening module files from S3")
		response.Error(c, response.ErrGetModuleVersionFilesListenFail, err)
		return
	}

	response.Success(c, http.StatusOK, files)
}

// GetModuleVersionFile is a function to return system module file content
// @Summary Retrieve system module file content (in base64) by module name, version and relative path
// @Tags Modules
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format" default(latest)
// @Param path query string true "relative path to module file"
// @Success 200 {object} response.successResp{data=systemModuleFile} "system module file content received successful"
// @Failure 403 {object} response.errorResp "getting system module file content not permitted"
// @Failure 404 {object} response.errorResp "system module not found"
// @Failure 500 {object} response.errorResp "internal error on getting system module file"
// @Router /modules/{module_name}/versions/{version}/files/file [get]
func (s *ModuleService) GetModuleVersionFile(c *gin.Context) {
	var (
		data       string
		fileData   []byte
		filePath   = c.Query("path")
		module     models.ModuleS
		moduleName = c.Param("module_name")
		sv         *models.Service
		version    = c.Param("version")
	)

	if sv = modules.GetService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	tid := c.GetUint64("tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err := s.db.Scopes(modules.FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrModulesSystemModuleNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = module.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		response.Error(c, response.ErrModulesInvalidSystemModuleData, err)
		return
	}

	s3, err := s3.New(nil)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error openning connection to S3")
		response.Error(c, response.ErrInternal, err)
		return
	}

	prefix := moduleName + "/" + module.Info.Version.String() + "/"
	if !strings.HasPrefix(filePath, prefix) || strings.Contains(filePath, "..") {
		logger.FromContext(c).Errorf("error parsing path to file: mismatch base prefix")
		response.Error(c, response.ErrGetModuleVersionFileParsePathFail, nil)
		return
	}

	if fileData, err = s3.ReadFile(filePath); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error reading module file '%s'", filePath)
		response.Error(c, response.ErrGetModuleVersionFileReadFail, err)
		return
	}

	data = base64.StdEncoding.EncodeToString(fileData)
	response.Success(c, http.StatusOK, systemModuleFile{Path: filePath, Data: data})
}

// PatchModuleVersionFile is a function to save, move, remove of system module file and its content
// @Summary Patch system module file and content (in base64) by module name, version and relative path
// @Tags Modules
// @Accept json
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format" default(latest)
// @Param json body systemModuleFilePatch true "action, relative path and file content for module file"
// @Success 200 {object} response.successResp "action on system module file did successful"
// @Failure 403 {object} response.errorResp "making action on system module file not permitted"
// @Failure 404 {object} response.errorResp "system module not found"
// @Failure 500 {object} response.errorResp "internal error on making action system module file"
// @Router /modules/{module_name}/versions/{version}/files/file [put]
func (s *ModuleService) PatchModuleVersionFile(c *gin.Context) {
	var (
		data       []byte
		files      map[string]os.FileInfo
		form       systemModuleFilePatch
		info       os.FileInfo
		module     models.ModuleS
		moduleName = c.Param("module_name")
		sv         *models.Service
		version    = c.Param("version")
	)

	uaf := useraction.NewFields(c, "module", "module", "module editing", moduleName, useraction.UnknownObjectDisplayName)
	defer s.userActionWriter.WriteUserAction(c, uaf)

	if sv = modules.GetService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	tid := c.GetUint64("tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err := s.db.Scopes(modules.FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrModulesSystemModuleNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = module.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		response.Error(c, response.ErrModulesInvalidSystemModuleData, err)
		return
	}
	uaf.ObjectDisplayName = module.Locale.Module["en"].Title

	if module.State == "release" {
		logger.FromContext(c).Errorf("error patching released module")
		response.Error(c, response.ErrPatchModuleVersionFileUpdateFail, nil)
		return
	}

	s3, err := s3.New(nil)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error openning connection to S3")
		response.Error(c, response.ErrInternal, err)
		return
	}

	if err = c.ShouldBindJSON(&form); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error binding JSON")
		response.Error(c, response.ErrModulesInvalidRequest, err)
		return
	}

	prefix := moduleName + "/" + module.Info.Version.String() + "/"
	if !strings.HasPrefix(form.Path, prefix) || strings.Contains(form.Path, "..") {
		logger.FromContext(c).Errorf("error parsing path to file: mismatch base prefix")
		response.Error(c, response.ErrPatchModuleVersionFileParsePathFail, nil)
		return
	}

	switch form.Action {
	case "save":
		if data, err = base64.StdEncoding.DecodeString(form.Data); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error decoding file data")
			response.Error(c, response.ErrPatchModuleVersionFileParseModuleFileFail, err)
			return
		}

		if err = modules.ValidateFileOnSave(form.Path, data); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error validating file data before write to S3")
			response.Error(c, response.ErrPatchModuleVersionFileParseModuleFileFail, err)
			return
		}

		if err = s3.WriteFile(form.Path, data); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error writing file data to S3")
			response.Error(c, response.ErrPatchModuleVersionFileWriteModuleFileFail, err)
			return
		}

	case "remove":
		if err = s3.Remove(form.Path); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error removing file from S3")
			response.Error(c, response.ErrPatchModuleVersionFileWriteModuleObjectFail, err)
			return
		}

	case "move":
		if !strings.HasPrefix(form.NewPath, prefix) || strings.Contains(form.NewPath, "..") {
			logger.FromContext(c).Errorf("error parsing path to file: mismatch base prefix")
			response.Error(c, response.ErrPatchModuleVersionFileParseNewpathFail, nil)
			return
		}

		if info, err = s3.GetInfo(form.Path); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error getting file info from S3")
			response.Error(c, response.ErrPatchModuleVersionFileObjectNotFound, err)
			return
		} else if !info.IsDir() {
			if strings.HasSuffix(form.NewPath, "/") {
				form.NewPath += info.Name()
			}

			if form.Path == form.NewPath {
				logger.FromContext(c).Errorf("error moving file in S3: newpath is identical to path")
				response.Error(c, response.ErrPatchModuleVersionFilePathIdentical, nil)
				return
			}

			if err = s3.Rename(form.Path, form.NewPath); err != nil {
				logger.FromContext(c).WithError(err).Errorf("error renaming file in S3")
				response.Error(c, response.ErrPatchModuleVersionFileObjectMoveFail, err)
				return
			}
		} else {
			if !strings.HasSuffix(form.Path, "/") {
				form.Path += "/"
			}
			if !strings.HasSuffix(form.NewPath, "/") {
				form.NewPath += "/"
			}

			if form.Path == form.NewPath {
				logger.FromContext(c).Errorf("error moving file in S3: newpath is identical to path")
				response.Error(c, response.ErrPatchModuleVersionFilePathIdentical, nil)
				return
			}

			if files, err = s3.ListDirRec(form.Path); err != nil {
				logger.FromContext(c).WithError(err).Errorf("error getting files by path from S3")
				response.Error(c, response.ErrPatchModuleVersionFileGetFilesFail, err)
				return
			}

			for obj, info := range files {
				if !info.IsDir() {
					curfile := filepath.Join(form.Path, obj)
					newfile := filepath.Join(form.NewPath, obj)
					if err = s3.Rename(curfile, newfile); err != nil {
						logger.FromContext(c).WithError(err).Errorf("error moving file in S3")
						response.Error(c, response.ErrPatchModuleVersionFileObjectMoveFail, err)
						return
					}
				}
			}
		}

	default:
		logger.FromContext(c).Errorf("error making unknown action on module")
		response.Error(c, response.ErrPatchModuleVersionFileActionNotFound, nil)
		return
	}

	if err = s.db.Model(&module).UpdateColumn("last_update", gorm.Expr("NOW()")).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error updating system module")
		response.Error(c, response.ErrPatchModuleVersionFileSystemModuleUpdateFail, err)
		return
	}

	response.Success(c, http.StatusOK, struct{}{})
}

// GetModuleVersionOption is a function to return option of system module rendered on server side
// @Summary Retrieve rendered Event Config Schema of system module data by module name and version
// @Tags Modules
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format" default(latest)
// @Param option_name path string true "module option without spaces" Enums(id, tenant_id, service_type, state, config_schema, default_config, static_dependencies, fields_schema, action_config_schema, default_action_config, event_config_schema, default_event_config, changelog, locale, info, last_update, event_config_schema_definitions, action_config_schema_definitions)
// @Success 200 {object} response.successResp{data=interface{}} "module option received successful"
// @Failure 403 {object} response.errorResp "getting module option not permitted"
// @Failure 404 {object} response.errorResp "system module not found"
// @Failure 500 {object} response.errorResp "internal error on getting module option"
// @Router /modules/{module_name}/versions/{version}/options/{option_name} [get]
func (s *ModuleService) GetModuleVersionOption(c *gin.Context) {
	var (
		module     models.ModuleS
		moduleName = c.Param("module_name")
		optionName = c.Param("option_name")
		sv         *models.Service
		version    = c.Param("version")
	)

	if sv = modules.GetService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	tid := c.GetUint64("tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err := s.db.Scopes(modules.FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrModulesSystemModuleNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = module.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		response.Error(c, response.ErrModulesInvalidSystemModuleData, err)
		return
	}

	if optionName == "event_config_schema_definitions" {
		response.Success(c, http.StatusOK, models.GetECSDefinitions(nil))
		return
	}

	if optionName == "action_config_schema_definitions" {
		response.Success(c, http.StatusOK, models.GetACSDefinitions(nil))
		return
	}

	options := make(map[string]json.RawMessage)
	data, err := json.Marshal(module)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error building system module JSON")
		response.Error(c, response.ErrGetModuleVersionOptionMakeJsonFail, err)
		return
	} else if err = json.Unmarshal(data, &options); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error parsing system module JSON")
		response.Error(c, response.ErrGetModuleVersionOptionParseJsonFail, err)
		return
	} else if _, ok := options[optionName]; !ok {
		logger.FromContext(c).WithError(err).Errorf("error finding system module option by name")
		response.Error(c, response.ErrGetModuleVersionOptionNotFound, err)
		return
	}

	response.Success(c, http.StatusOK, options[optionName])
}
