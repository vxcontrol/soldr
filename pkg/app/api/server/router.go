package server

import (
	"crypto/tls"
	"encoding/gob"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"soldr/pkg/app/api/client"
	"soldr/pkg/app/api/logger"
	"soldr/pkg/app/api/server/private"
	"soldr/pkg/app/api/server/proto"
	"soldr/pkg/app/api/server/public"
	"soldr/pkg/app/api/storage"
	"soldr/pkg/app/api/useraction"
	"soldr/pkg/app/api/worker/events"
)

type RouterConfig struct {
	Debug        bool
	UseSSL       bool
	BaseURL      string
	StaticPath   string
	StaticURL    *url.URL
	TemplatesDir string
	CertsPath    string
}

// @title SOLDR Swagger API
// @version 1.0
// @description Swagger API for VXControl SOLDR backend product.
// @termsOfService http://swagger.io/terms/

// @contact.url https://vxcontrol.com
// @contact.name Dmitry Nagibin
// @contact.email admin@vxcontrol.com

// @license.name Proprietary License
// @license.url https://raw.githubusercontent.com/vxcontrol/soldr/master/LICENSE

// @query.collection.format multi

// @BasePath /api/v1
func NewRouter(
	cfg RouterConfig,
	db *gorm.DB,
	exchanger *events.Exchanger,
	userActionWriter useraction.Writer,
	dbConns *storage.DBConnectionStorage,
	s3Conns *storage.S3ConnectionStorage,
	modulesStorage *storage.ModuleStorage,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	if cfg.Debug {
		gin.SetMode(gin.DebugMode)
	}

	// Register privileges model that will be used into session cookie
	gob.Register([]string{})
	gob.Register(map[string]interface{}{})

	cookieStore := cookie.NewStore(storage.MakeCookieStoreKey())

	index := func(c *gin.Context) {
		data, err := ioutil.ReadFile(path.Join(cfg.StaticPath, "/index.html"))
		if err != nil {
			logger.FromContext(c).WithError(err).Errorf("error loading index.html")
			return
		}
		c.Data(200, "text/html", data)
	}

	router := gin.New()
	router.Use(WithLogger("vxapi"))
	router.Use(gin.Recovery())
	router.Use(sessions.Sessions("auth", cookieStore))

	router.Static("/js", path.Join(cfg.StaticPath, "js"))
	router.Static("/css", path.Join(cfg.StaticPath, "css"))
	router.Static("/fonts", path.Join(cfg.StaticPath, "fonts"))
	router.Static("/images", path.Join(cfg.StaticPath, "images"))

	// TODO: should be moved to the web service
	router.StaticFile("/favicon.ico", path.Join(cfg.StaticPath, "favicon.ico"))
	router.StaticFile("/apple-touch-icon.png", path.Join(cfg.StaticPath, "apple-touch-icon.png"))

	if cfg.StaticURL.Scheme != "" && cfg.StaticURL.Host != "" {
		router.NoRoute(func() gin.HandlerFunc {
			return func(c *gin.Context) {
				if strings.HasPrefix(c.Request.URL.String(), "/app/") {
					index(c)
					return
				}
				director := func(req *http.Request) {
					*req = *c.Request
					req.URL.Scheme = cfg.StaticURL.Scheme
					req.URL.Host = cfg.StaticURL.Host
				}
				proxy := &httputil.ReverseProxy{
					Director: director,
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{
							InsecureSkipVerify: true,
						},
					},
				}
				proxy.ServeHTTP(c.Writer, c.Request)
			}
		}())
	} else {
		router.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusTemporaryRedirect, "/app/")
		})

		router.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.String(), "/app/") {
				index(c)
			}
		})
	}

	serverConnector := client.NewAgentServerClient(db, dbConns, s3Conns)

	// services
	authService := public.NewAuthService(public.AuthServiceConfig{
		SessionTimeout: 3 * 3600, // 3 hours
		APIBaseURL:     cfg.BaseURL,
		SecureCookie:   cfg.UseSSL,
	}, db)
	protoService := proto.NewProtoService(db, serverConnector, userActionWriter, cfg.CertsPath)
	agentService := private.NewAgentService(db, serverConnector, userActionWriter, modulesStorage)
	binariesService := private.NewBinariesService(db, userActionWriter)
	eventService := private.NewEventService(serverConnector)
	groupService := private.NewGroupService(serverConnector, userActionWriter, modulesStorage)
	moduleService := private.NewModuleService(cfg.TemplatesDir, db, serverConnector, userActionWriter, modulesStorage)
	optionService := private.NewOptionService(db)
	policyService := private.NewPolicyService(db, serverConnector, userActionWriter)
	portingService := private.NewPortingService(db, userActionWriter)
	roleService := private.NewRoleService(db)
	upgradeService := private.NewUpgradeService(db, serverConnector, userActionWriter)
	tagService := private.NewTagService(db, serverConnector)
	versionService := private.NewVersionService(db, serverConnector)
	servicesService := private.NewServicesService(db)
	tenantService := private.NewTenantService(db)
	userService := private.NewUserService(db)

	// set api handlers
	api := router.Group(cfg.BaseURL)
	api.Use(setGlobalDB(db))
	{
		setPublicGroup(api, authService)

		setSwaggerGroup(api)

		setVXProtoGroup(api, db, protoService, cfg.BaseURL)
	}

	privateGroup := api.Group("/")
	privateGroup.Use(authRequired())
	privateGroup.Use(setServiceInfo(db))
	{
		setTokenGroup(privateGroup)

		setBinariesGroup(privateGroup, binariesService)
		setUpgradesGroup(privateGroup, upgradeService)
		setAgentsGroup(privateGroup, agentService, moduleService)

		setGroupsGroup(privateGroup, groupService, moduleService)

		setPoliciesGroup(privateGroup, policyService, moduleService)

		// collected events by policy modules
		setEventsGroup(privateGroup, eventService)

		// system modules groups
		setSystemModulesGroup(privateGroup, moduleService)
		setExportGroup(privateGroup, portingService)
		setImportGroup(privateGroup, portingService)
		setOptionsGroup(privateGroup, optionService)

		setNotificationsGroup(privateGroup, exchanger)

		setTagsGroup(privateGroup, tagService)
		setVersionsGroup(privateGroup, versionService)

		// system objects
		setRolesGroup(privateGroup, roleService)
		setServicesGroup(privateGroup, servicesService)
		setTenanesGroup(privateGroup, tenantService)
		setUsersGroup(privateGroup, userService)
	}

	return router
}

func setPublicGroup(parent *gin.RouterGroup, svc *public.AuthService) {
	publicGroup := parent.Group("/")
	{
		publicGroup.GET("/info", svc.Info)
		authGroup := publicGroup.Group("/auth")
		{
			authGroup.POST("/login", svc.AuthLogin)
			authGroup.GET("/logout", svc.AuthLogout)
		}

		authPrivateGroup := publicGroup.Group("/auth")
		authPrivateGroup.Use(authRequired())
		{
			authPrivateGroup.POST("/switch-service", svc.AuthSwitchService)
		}
	}
}

func setSwaggerGroup(parent *gin.RouterGroup) {
	swaggerGroup := parent.Group("/")
	swaggerGroup.Use(authRequired())
	{
		swaggerGroup.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
}

func setVXProtoGroup(parent *gin.RouterGroup, db *gorm.DB, svc *proto.ProtoService, apiBaseURL string) {
	vxProtoGroup := parent.Group("/vxpws")
	vxProtoGroup.Use(authTokenProtoRequired(apiBaseURL))
	vxProtoGroup.Use(setServiceInfo(db))
	{
		vxProtoGroup.GET("/aggregate/:group_id/", svc.AggregateWSConnect)
		vxProtoGroup.GET("/browser/:agent_id/", svc.BrowserWSConnect)
		vxProtoGroup.GET("/external/:agent_id/", svc.ExternalWSConnect)
	}
}

func setTokenGroup(parent *gin.RouterGroup) {
	tokenGroup := parent.Group("/token")
	tokenGroup.Use(privilegesRequired("vxapi.modules.interactive"))
	{
		tokenGroup.POST("/vxproto", private.CreateAuthToken)
	}
}

func setBinariesGroup(parent *gin.RouterGroup, svc *private.BinariesService) {
	binariesGroup := parent.Group("/binaries")
	binariesGroup.Use(privilegesRequired("vxapi.agents.downloads"))
	{
		binariesGroup.GET("/vxagent", svc.GetAgentBinaries)
		binariesGroup.GET("/vxagent/:os/:arch/:version", svc.GetAgentBinaryFile)
	}
}

func setUpgradesGroup(parent *gin.RouterGroup, svc *private.UpgradeService) {
	upgradesGroup := parent.Group("/upgrades")
	upgradesGroup.Use(privilegesRequired("vxapi.agents.api.edit"))
	{
		upgradesGroup.GET("/agents", svc.GetAgentsUpgrades)
		upgradesGroup.POST("/agents", svc.CreateAgentsUpgrades)
		upgradesGroup.GET("/agents/:hash/last", svc.GetLastAgentUpgrade)
		upgradesGroup.PUT("/agents/:hash/last", svc.PatchLastAgentUpgrade)
	}
}

func setAgentsGroup(
	parent *gin.RouterGroup,
	agentService *private.AgentService,
	moduleService *private.ModuleService,
) {
	agentsCreateGroup := parent.Group("/agents")
	agentsCreateGroup.Use(privilegesRequired("vxapi.agents.api.create"))
	{
		agentsCreateGroup.POST("/", agentService.CreateAgent)
	}

	agentsDeleteGroup := parent.Group("/agents")
	agentsDeleteGroup.Use(privilegesRequired("vxapi.agents.api.delete"))
	{
		agentsDeleteGroup.DELETE("/:hash", agentService.DeleteAgent)
	}

	agentsEditGroup := parent.Group("/agents")
	agentsEditGroup.Use(privilegesRequired("vxapi.agents.api.edit"))
	{
		agentsEditGroup.PUT("/:hash", agentService.PatchAgent)
	}

	agentsEditOrDeleteGroup := parent.Group("/agents")
	agentsEditOrDeleteGroup.Use(privilegesRequiredPatchAgents())
	{
		agentsEditOrDeleteGroup.PUT("/", agentService.PatchAgents)
	}

	agentsViewGroup := parent.Group("/agents")
	agentsViewGroup.Use(privilegesRequired("vxapi.agents.api.view"))
	{
		agentsViewGroup.GET("/", agentService.GetAgents)
		agentsViewGroup.GET("/:hash", agentService.GetAgent)
		agentsViewGroup.GET("/count", agentService.GetAgentsCount)
	}

	agentsModulesViewGroup := parent.Group("/agents")
	agentsModulesViewGroup.Use(privilegesRequired("vxapi.policies.api.view"))
	{
		agentsModulesViewGroup.GET("/:hash/modules", moduleService.GetAgentModules)
		agentsModulesViewGroup.GET("/:hash/modules/:module_name", moduleService.GetAgentModule)
		agentsModulesViewGroup.GET("/:hash/modules/:module_name/bmodule.vue", moduleService.GetAgentBModule)
	}
}

func setGroupsGroup(
	parent *gin.RouterGroup,
	groupService *private.GroupService,
	moduleService *private.ModuleService,
) {
	groupsCreateGroup := parent.Group("/groups")
	groupsCreateGroup.Use(privilegesRequired("vxapi.groups.api.create"))
	{
		groupsCreateGroup.POST("/", groupService.CreateGroup)
	}

	groupsDeleteGroup := parent.Group("/groups")
	groupsDeleteGroup.Use(privilegesRequired("vxapi.groups.api.delete"))
	{
		groupsDeleteGroup.DELETE("/:hash", groupService.DeleteGroup)
	}

	groupsEditGroup := parent.Group("/groups")
	groupsEditGroup.Use(privilegesRequired("vxapi.groups.api.edit"))
	{
		groupsEditGroup.PUT("/:hash", groupService.PatchGroup)
	}

	groupsViewGroup := parent.Group("/groups")
	groupsViewGroup.Use(privilegesRequired("vxapi.groups.api.view"))
	{
		groupsViewGroup.GET("/", groupService.GetGroups)
		groupsViewGroup.GET("/:hash", groupService.GetGroup)
	}

	groupsModulesViewGroup := parent.Group("/groups")
	groupsModulesViewGroup.Use(privilegesRequired("vxapi.policies.api.view"))
	{
		groupsModulesViewGroup.GET("/:hash/modules", moduleService.GetGroupModules)
		groupsModulesViewGroup.GET("/:hash/modules/:module_name", moduleService.GetGroupModule)
		groupsModulesViewGroup.GET("/:hash/modules/:module_name/bmodule.vue", moduleService.GetGroupBModule)
	}

	groupsPoliciesLinkGroup := parent.Group("/groups")
	groupsPoliciesLinkGroup.Use(privilegesRequired("vxapi.policies.control.link"))
	{
		groupsPoliciesLinkGroup.PUT("/:hash/policies", groupService.PatchGroupPolicy)
	}
}

func setPoliciesGroup(
	parent *gin.RouterGroup,
	policyService *private.PolicyService,
	moduleService *private.ModuleService,
) {
	parent = parent.Group("/")
	parent.Use(setSecureConfigEncryptor())

	policiesCreateGroup := parent.Group("/policies")
	policiesCreateGroup.Use(privilegesRequired("vxapi.policies.api.create"))
	{
		policiesCreateGroup.POST("/", policyService.CreatePolicy)
	}

	policiesDeleteGroup := parent.Group("/policies")
	policiesDeleteGroup.Use(privilegesRequired("vxapi.policies.api.delete"))
	{
		policiesDeleteGroup.DELETE("/:hash", policyService.DeletePolicy)
	}

	policiesEditGroup := parent.Group("/policies")
	policiesEditGroup.Use(privilegesRequired("vxapi.policies.api.edit"))
	{
		policiesEditGroup.PUT("/:hash", policyService.PatchPolicy)
		policiesEditGroup.DELETE("/:hash/modules/:module_name", moduleService.DeletePolicyModule)
	}

	policiesInconcurrentEditGroup := policiesEditGroup.Group("/")
	policiesInconcurrentEditGroup.Use(inconcurrentRequest())
	{
		policiesInconcurrentEditGroup.PUT("/:hash/modules/:module_name", moduleService.PatchPolicyModule)
	}

	policiesViewGroup := parent.Group("/policies")
	policiesViewGroup.Use(privilegesRequired("vxapi.policies.api.view"))
	{
		policiesViewGroup.GET("/", policyService.GetPolicies)
		policiesViewGroup.GET("/:hash", policyService.GetPolicy)
		policiesViewGroup.GET("/count", policyService.GetPoliciesCount)
	}

	policiesModulesViewGroup := parent.Group("/policies")
	policiesModulesViewGroup.Use(privilegesRequired("vxapi.policies.api.view"))
	{
		policiesModulesViewGroup.GET("/:hash/modules", moduleService.GetPolicyModules)
		policiesModulesViewGroup.GET("/:hash/modules/:module_name", moduleService.GetPolicyModule)
		policiesModulesViewGroup.GET("/:hash/modules/:module_name/bmodule.vue", moduleService.GetPolicyBModule)
	}

	policiesGroupsLinkGroup := parent.Group("/policies")
	policiesGroupsLinkGroup.Use(privilegesRequired("vxapi.policies.control.link"))
	{
		policiesGroupsLinkGroup.PUT("/:hash/groups", policyService.PatchPolicyGroup)
	}

	policiesSecureConfigViewGroup := parent.Group("/policies")
	policiesSecureConfigViewGroup.Use(privilegesRequired("vxapi.policies.api.edit", "vxapi.modules.secure-config.view"))
	{
		policiesSecureConfigViewGroup.GET("/:hash/modules/:module_name/secure_config/:param_name", moduleService.GetPolicyModuleSecureConfigValue)
	}

	policiesSecureConfigEditGroup := parent.Group("/policies")
	policiesSecureConfigEditGroup.Use(privilegesRequired("vxapi.policies.api.edit", "vxapi.modules.secure-config.edit"))
	{
		policiesSecureConfigEditGroup.POST("/:hash/modules/:module_name/secure_config", moduleService.SetPolicyModuleSecureConfigValue)
	}
}

func setEventsGroup(parent *gin.RouterGroup, svc *private.EventService) {
	eventsGroup := parent.Group("/events")
	eventsGroup.Use(privilegesRequired("vxapi.modules.events"))
	{
		eventsGroup.GET("/", svc.GetEvents)
	}
}

func setSystemModulesGroup(parent *gin.RouterGroup, svc *private.ModuleService) {
	parent = parent.Group("/")
	parent.Use(setSecureConfigEncryptor())

	systemModulesCreateGroup := parent.Group("/modules")
	systemModulesCreateGroup.Use(privilegesRequired("vxapi.modules.api.create"))
	{
		systemModulesCreateGroup.POST("/", svc.CreateModule)
	}

	systemModulesDeleteGroup := parent.Group("/modules")
	systemModulesDeleteGroup.Use(privilegesRequired("vxapi.modules.api.delete"))
	{
		systemModulesDeleteGroup.DELETE("/:module_name", svc.DeleteModule)
	}

	systemModulesEditGroup := parent.Group("/modules")
	systemModulesEditGroup.Use(privilegesRequired("vxapi.modules.api.edit"))
	{
		systemModulesEditGroup.POST("/:module_name/versions/:version", svc.CreateModuleVersion)
		systemModulesEditGroup.PUT("/:module_name/versions/:version", svc.PatchModuleVersion)
		systemModulesEditGroup.DELETE("/:module_name/versions/:version", svc.DeleteModuleVersion)

		systemModulesEditGroup.GET("/:module_name/versions/:version/files", svc.GetModuleVersionFiles)
		systemModulesEditGroup.GET("/:module_name/versions/:version/files/file", svc.GetModuleVersionFile)
		systemModulesEditGroup.PUT("/:module_name/versions/:version/files/file", svc.PatchModuleVersionFile)

		systemModulesEditGroup.GET("/:module_name/versions/:version/updates", svc.GetModuleVersionUpdates)
		systemModulesEditGroup.POST("/:module_name/versions/:version/updates", svc.CreateModuleVersionUpdates)
	}

	systemModulesViewGroup := parent.Group("/modules")
	systemModulesViewGroup.Use(privilegesRequired("vxapi.modules.api.view"))
	{
		systemModulesViewGroup.GET("/", svc.GetModules)
		systemModulesViewGroup.GET("/:module_name/versions/", svc.GetModuleVersions)
		systemModulesViewGroup.GET("/:module_name/versions/:version", svc.GetModuleVersion)
		systemModulesViewGroup.GET("/:module_name/versions/:version/options/:option_name", svc.GetModuleVersionOption)
	}
}

func setExportGroup(parent *gin.RouterGroup, svc *private.PortingService) {
	exportGroup := parent.Group("/export")
	exportGroup.Use(privilegesRequired("vxapi.modules.control.export"))
	{
		exportGroup.POST("/modules/:module_name/versions/:version", svc.ExportModule)
	}
}

func setImportGroup(parent *gin.RouterGroup, svc *private.PortingService) {
	importGroup := parent.Group("/import")
	importGroup.Use(privilegesRequired("vxapi.modules.control.import"))
	{
		importGroup.POST("/modules/:module_name/versions/:version", svc.ImportModule)
	}
}

func setOptionsGroup(parent *gin.RouterGroup, svc *private.OptionService) {
	optionsGroup := parent.Group("/options")
	optionsGroup.Use(privilegesRequired("vxapi.modules.api.view"))
	{
		optionsGroup.GET("/actions", svc.GetOptionsActions)
		optionsGroup.GET("/events", svc.GetOptionsEvents)
		optionsGroup.GET("/fields", svc.GetOptionsFields)
		optionsGroup.GET("/tags", svc.GetOptionsTags)
		optionsGroup.GET("/versions", svc.GetOptionsVersions)
	}
}

func setNotificationsGroup(parent *gin.RouterGroup, exchanger *events.Exchanger) {
	notificationsGroup := parent.Group("/notifications")
	premsFilter := func(c *gin.Context, name events.EventChannelName) bool {
		prms := c.GetStringSlice("prm")
		if len(prms) == 0 {
			return false
		}
		var privs []string
		switch name {
		case events.CreateAgentsChannel, events.UpdateAgentsChannel, events.DeleteAgentsChannel:
			privs = append(privs, "vxapi.agents.api.view")
		case events.CreateGroupsChannel, events.UpdateGroupsChannel, events.DeleteGroupsChannel:
			privs = append(privs, "vxapi.groups.api.view")
		case events.CreatePoliciesChannel, events.UpdatePoliciesChannel, events.DeletePoliciesChannel:
			privs = append(privs, "vxapi.policies.api.view")
		case events.CreateModulesChannel, events.UpdateModulesChannel, events.DeleteModulesChannel:
			privs = append(privs, "vxapi.policies.api.view")
		case events.CreateGroupToPolicyChannel, events.DeleteGroupToPolicyChannel:
			privs = append(privs, "vxapi.groups.api.view")
			privs = append(privs, "vxapi.policies.api.view")
		case events.AllEventsChannel:
			privs = append(privs, "vxapi.agents.api.view")
			privs = append(privs, "vxapi.groups.api.view")
			privs = append(privs, "vxapi.policies.api.view")
		default:
			return false
		}
		for _, priv := range privs {
			if !lookupPerm(prms, priv) {
				return false
			}
		}
		return true
	}
	{
		notificationsGroup.GET("/subscribe/", private.SubscribeHandler(exchanger, premsFilter))
	}
}

func setTagsGroup(parent *gin.RouterGroup, svc *private.TagService) {
	tagsGroup := parent.Group("/tags")
	tagsGroup.Use(privilegesRequiredByQueryTypeField(
		map[string][]string{
			"agents":   {"vxapi.agents.api.view"},
			"groups":   {"vxapi.groups.api.view"},
			"policies": {"vxapi.policies.api.view"},
			"modules":  {"vxapi.policies.api.view"},
		},
	))
	{
		tagsGroup.GET("/", svc.GetTags)
	}
}

func setVersionsGroup(parent *gin.RouterGroup, svc *private.VersionService) {
	versionsGroup := parent.Group("/versions")
	versionsGroup.Use(privilegesRequiredByQueryTypeField(
		map[string][]string{
			"agents":  {"vxapi.agents.api.view"},
			"modules": {"vxapi.policies.api.view"},
		},
	))
	{
		versionsGroup.GET("/", svc.GetVersions)
	}
}

func setRolesGroup(parent *gin.RouterGroup, svc *private.RoleService) {
	rolesGroup := parent.Group("/roles")
	rolesGroup.Use(privilegesRequired("vxapi.roles.api.view"))
	{
		rolesGroup.GET("/", svc.GetRoles)
	}
}

func setServicesGroup(parent *gin.RouterGroup, svc *private.ServicesService) {
	servicesCreateGroup := parent.Group("/services")
	servicesCreateGroup.Use(privilegesRequired("vxapi.services.api.create"))
	{
		servicesCreateGroup.POST("/", svc.CreateService)
	}

	servicesDeleteGroup := parent.Group("/services")
	servicesDeleteGroup.Use(privilegesRequired("vxapi.services.api.delete"))
	{
		servicesDeleteGroup.DELETE("/:hash", svc.DeleteService)
	}

	servicesEditGroup := parent.Group("/services")
	servicesEditGroup.Use(privilegesRequired("vxapi.services.api.edit"))
	{
		servicesEditGroup.PUT("/:hash", svc.PatchService)
	}

	servicesViewGroup := parent.Group("/services")
	servicesViewGroup.Use(privilegesRequired("vxapi.services.api.view"))
	{
		servicesViewGroup.GET("/", svc.GetServices)
		servicesViewGroup.GET("/:hash", svc.GetService)
	}
}

func setTenanesGroup(parent *gin.RouterGroup, svc *private.TenantService) {
	tenantsCreateGroup := parent.Group("/tenants")
	tenantsCreateGroup.Use(privilegesRequired("vxapi.tenants.api.create"))
	{
		tenantsCreateGroup.POST("/", svc.CreateTenant)
	}

	tenantsDeleteGroup := parent.Group("/tenants")
	tenantsDeleteGroup.Use(privilegesRequired("vxapi.tenants.api.delete"))
	{
		tenantsDeleteGroup.DELETE("/:hash", svc.DeleteTenant)
	}

	tenantsEditGroup := parent.Group("/tenants")
	tenantsEditGroup.Use(privilegesRequired("vxapi.tenants.api.edit"))
	{
		tenantsEditGroup.PUT("/:hash", svc.PatchTenant)
	}

	tenantsViewGroup := parent.Group("/tenants")
	tenantsViewGroup.Use(privilegesRequired("vxapi.tenants.api.view"))
	{
		tenantsViewGroup.GET("/", svc.GetTenants)
		tenantsViewGroup.GET("/:hash", svc.GetTenant)
	}
}

func setUsersGroup(parent *gin.RouterGroup, svc *private.UserService) {
	usersCreateGroup := parent.Group("/users")
	usersCreateGroup.Use(privilegesRequired("vxapi.users.api.create"))
	{
		usersCreateGroup.POST("/", svc.CreateUser)
	}

	usersDeleteGroup := parent.Group("/users")
	usersDeleteGroup.Use(privilegesRequired("vxapi.users.api.delete"))
	{
		usersDeleteGroup.DELETE("/:hash", svc.DeleteUser)
	}

	usersEditGroup := parent.Group("/users")
	usersEditGroup.Use(privilegesRequired("vxapi.users.api.edit"))
	{
		usersEditGroup.PUT("/:hash", svc.PatchUser)
	}

	usersViewGroup := parent.Group("/users")
	usersViewGroup.Use(privilegesRequired("vxapi.users.api.view"))
	{
		usersViewGroup.GET("/", svc.GetUsers)
		usersViewGroup.GET("/:hash", svc.GetUser)
	}

	userViewGroup := parent.Group("/user")
	{
		userViewGroup.GET("/", svc.GetCurrentUser)
	}

	userEditGroup := parent.Group("/user")
	userEditGroup.Use(localUserRequired())
	{
		userEditGroup.PUT("/password", svc.ChangePasswordCurrentUser)
	}
}
