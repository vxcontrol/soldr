package server

import (
	"crypto/tls"
	"encoding/gob"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	srvevents "soldr/internal/app/api/server/events"
	"soldr/internal/app/api/server/private"
	"soldr/internal/app/api/server/proto"
	"soldr/internal/app/api/server/public"
	"soldr/internal/app/api/storage/mem"
	"soldr/internal/app/api/utils"
)

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
	db *gorm.DB,
	exchanger *srvevents.Exchanger,
	serviceDBConns *mem.ServiceDBConnectionStorage,
	serviceS3Conns *mem.ServiceS3ConnectionStorage,

) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	if _, exists := os.LookupEnv("DEBUG"); exists {
		gin.SetMode(gin.DebugMode)
	}

	// Register privileges model that will be used into session cookie
	gob.Register([]string{})
	gob.Register(map[string]interface{}{})

	cookieStore := cookie.NewStore(utils.MakeCookieStoreKey())

	logDir := "logs"
	if dir, ok := os.LookupEnv("LOG_DIR"); ok {
		logDir = dir
	}
	apiLogFile := &lumberjack.Logger{
		Filename:   path.Join(logDir, "api.log"),
		MaxSize:    100,
		MaxBackups: 7,
		MaxAge:     14,
		Compress:   true,
	}
	defer apiLogFile.Close()

	router := gin.New()
	router.Use(otelgin.Middleware("vxapi"))
	router.Use(gin.LoggerWithWriter(io.MultiWriter(apiLogFile, os.Stdout)))
	router.Use(gin.Recovery())
	router.Use(sessions.Sessions("auth", cookieStore))

	router.Static("/js", "./static/js")
	router.Static("/css", "./static/css")
	router.Static("/fonts", "./static/fonts")
	router.Static("/images", "./static/images")

	if uiStaticAddr, ok := os.LookupEnv("API_STATIC_URL"); ok {
		uiStaticUrl, err := url.Parse(uiStaticAddr)
		if err != nil {
			logrus.WithError(err).Error("error on parsing URL to redirect requests to the UI static")
		}
		router.NoRoute(func() gin.HandlerFunc {
			return func(c *gin.Context) {
				if strings.HasPrefix(c.Request.URL.String(), "/app/") {
					index(c)
					return
				}
				director := func(req *http.Request) {
					*req = *c.Request
					req.URL.Scheme = uiStaticUrl.Scheme
					req.URL.Host = uiStaticUrl.Host
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
		// static files
		router.StaticFile("/favicon.ico", "./static/favicon.ico")
		router.StaticFile("/apple-touch-icon.png", "./static/apple-touch-icon.png")

		router.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusTemporaryRedirect, "/app/")
		})

		router.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.String(), "/app/") {
				index(c)
			}
		})
	}

	// services
	agentsService := private.NewAgentService(
		db,
		serviceDBConns,
		serviceS3Conns,
	)

	// set api handlers
	api := router.Group(utils.PrefixPathAPI)
	api.Use(setGlobalDB(db))
	{
		setPublicGroup(api)

		setSwaggerGroup(api)

		setVXProtoGroup(api)
	}

	privateGroup := api.Group("/")
	privateGroup.Use(authRequired())
	privateGroup.Use(setServiceInfo())
	{
		setTokenGroup(privateGroup)

		setBinariesGroup(privateGroup)
		setUpgradesGroup(privateGroup)
		setAgentsGroup(privateGroup, agentsService)

		setGroupsGroup(privateGroup)

		setPoliciesGroup(privateGroup)

		// collected events by policy modules
		setEventsGroup(privateGroup)

		// system modules groups
		setSystemModulesGroup(privateGroup)
		setExportGroup(privateGroup)
		setImportGroup(privateGroup)
		setOptionsGroup(privateGroup)

		setNotificationsGroup(privateGroup, exchanger)

		setTagsGroup(privateGroup)
		setVersionsGroup(privateGroup)

		// system objects
		setRolesGroup(privateGroup)
		setServicesGroup(privateGroup)
		setTenanesGroup(privateGroup)
		setUsersGroup(privateGroup)
	}

	return router
}

func setPublicGroup(parent *gin.RouterGroup) {
	publicGroup := parent.Group("/")
	{
		publicGroup.GET("/info", public.Info)
		authGroup := publicGroup.Group("/auth")
		{
			authGroup.POST("/login", public.AuthLogin)
			authGroup.GET("/logout", public.AuthLogout)
		}

		authPrivateGroup := publicGroup.Group("/auth")
		authPrivateGroup.Use(authRequired())
		{
			authPrivateGroup.POST("/switch-service", public.AuthSwitchService)
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

func setVXProtoGroup(parent *gin.RouterGroup) {
	vxProtoGroup := parent.Group("/")
	vxProtoGroup.Use(authTokenProtoRequired())
	vxProtoGroup.Use(setServiceInfo())
	{
		protoBrowserGroup := vxProtoGroup.Group("/vxpws")
		{
			protoBrowserGroup.GET("/browser/:agent_id/", proto.BrowserWSConnect)
		}
		protoExternalGroup := vxProtoGroup.Group("/vxpws")
		{
			protoExternalGroup.GET("/external/:agent_id/", proto.ExternalWSConnect)
		}
	}
}

func setTokenGroup(parent *gin.RouterGroup) {
	tokenGroup := parent.Group("/token")
	tokenGroup.Use(privilegesRequired("vxapi.modules.interactive"))
	{
		tokenGroup.POST("/vxproto", private.CreateAuthToken)
	}
}

func setBinariesGroup(parent *gin.RouterGroup) {
	binariesGroup := parent.Group("/binaries")
	binariesGroup.Use(privilegesRequired("vxapi.agents.downloads"))
	{
		binariesGroup.GET("/vxagent", private.GetAgentBinaries)
		binariesGroup.GET("/vxagent/:os/:arch/:version", private.GetAgentBinaryFile)
	}
}

func setUpgradesGroup(parent *gin.RouterGroup) {
	upgradesGroup := parent.Group("/upgrades")
	upgradesGroup.Use(privilegesRequired("vxapi.agents.api.edit"))
	{
		upgradesGroup.GET("/agents", private.GetAgentsUpgrades)
		upgradesGroup.POST("/agents", private.CreateAgentsUpgrades)
		upgradesGroup.GET("/agents/:hash/last", private.GetLastAgentUpgrade)
		upgradesGroup.PUT("/agents/:hash/last", private.PatchLastAgentUpgrade)
	}
}

func setAgentsGroup(parent *gin.RouterGroup, service *private.AgentService) {
	agentsCreateGroup := parent.Group("/agents")
	agentsCreateGroup.Use(privilegesRequired("vxapi.agents.api.create"))
	{
		agentsCreateGroup.POST("/", service.CreateAgent)
	}

	agentsDeleteGroup := parent.Group("/agents")
	agentsDeleteGroup.Use(privilegesRequired("vxapi.agents.api.delete"))
	{
		agentsDeleteGroup.DELETE("/:hash", service.DeleteAgent)
	}

	agentsEditGroup := parent.Group("/agents")
	agentsEditGroup.Use(privilegesRequired("vxapi.agents.api.edit"))
	{
		agentsEditGroup.PUT("/:hash", service.PatchAgent)
	}

	agentsEditOrDeleteGroup := parent.Group("/agents")
	agentsEditOrDeleteGroup.Use(privilegesRequiredPatchAgents())
	{
		agentsEditOrDeleteGroup.PUT("/", service.PatchAgents)
	}

	agentsViewGroup := parent.Group("/agents")
	agentsViewGroup.Use(privilegesRequired("vxapi.agents.api.view"))
	{
		agentsViewGroup.GET("/", service.GetAgents)
		agentsViewGroup.GET("/:hash", service.GetAgent)
	}

	agentsModulesViewGroup := parent.Group("/agents")
	agentsModulesViewGroup.Use(privilegesRequired("vxapi.policies.api.view"))
	{
		agentsModulesViewGroup.GET("/:hash/modules", private.GetAgentModules)
		agentsModulesViewGroup.GET("/:hash/modules/:module_name", private.GetAgentModule)
		agentsModulesViewGroup.GET("/:hash/modules/:module_name/bmodule.vue", private.GetAgentBModule)
	}
}

func setGroupsGroup(parent *gin.RouterGroup) {
	groupsCreateGroup := parent.Group("/groups")
	groupsCreateGroup.Use(privilegesRequired("vxapi.groups.api.create"))
	{
		groupsCreateGroup.POST("/", private.CreateGroup)
	}

	groupsDeleteGroup := parent.Group("/groups")
	groupsDeleteGroup.Use(privilegesRequired("vxapi.groups.api.delete"))
	{
		groupsDeleteGroup.DELETE("/:hash", private.DeleteGroup)
	}

	groupsEditGroup := parent.Group("/groups")
	groupsEditGroup.Use(privilegesRequired("vxapi.groups.api.edit"))
	{
		groupsEditGroup.PUT("/:hash", private.PatchGroup)
	}

	groupsViewGroup := parent.Group("/groups")
	groupsViewGroup.Use(privilegesRequired("vxapi.groups.api.view"))
	{
		groupsViewGroup.GET("/", private.GetGroups)
		groupsViewGroup.GET("/:hash", private.GetGroup)
	}

	groupsModulesViewGroup := parent.Group("/groups")
	groupsModulesViewGroup.Use(privilegesRequired("vxapi.policies.api.view"))
	{
		groupsModulesViewGroup.GET("/:hash/modules", private.GetGroupModules)
		groupsModulesViewGroup.GET("/:hash/modules/:module_name", private.GetGroupModule)
		groupsModulesViewGroup.GET("/:hash/modules/:module_name/bmodule.vue", private.GetGroupBModule)
	}

	groupsPoliciesLinkGroup := parent.Group("/groups")
	groupsPoliciesLinkGroup.Use(privilegesRequired("vxapi.policies.control.link"))
	{
		groupsPoliciesLinkGroup.PUT("/:hash/policies", private.PatchGroupPolicy)
	}
}

func setPoliciesGroup(parent *gin.RouterGroup) {
	parent = parent.Group("/")
	parent.Use(setSecureConfigEncryptor())

	policiesCreateGroup := parent.Group("/policies")
	policiesCreateGroup.Use(privilegesRequired("vxapi.policies.api.create"))
	{
		policiesCreateGroup.POST("/", private.CreatePolicy)
	}

	policiesDeleteGroup := parent.Group("/policies")
	policiesDeleteGroup.Use(privilegesRequired("vxapi.policies.api.delete"))
	{
		policiesDeleteGroup.DELETE("/:hash", private.DeletePolicy)
	}

	policiesEditGroup := parent.Group("/policies")
	policiesEditGroup.Use(privilegesRequired("vxapi.policies.api.edit"))
	{
		policiesEditGroup.PUT("/:hash", private.PatchPolicy)
		policiesEditGroup.DELETE("/:hash/modules/:module_name", private.DeletePolicyModule)
	}

	policiesInconcurrentEditGroup := policiesEditGroup.Group("/")
	policiesInconcurrentEditGroup.Use(inconcurrentRequest())
	{
		policiesInconcurrentEditGroup.PUT("/:hash/modules/:module_name", private.PatchPolicyModule)
	}

	policiesViewGroup := parent.Group("/policies")
	policiesViewGroup.Use(privilegesRequired("vxapi.policies.api.view"))
	{
		policiesViewGroup.GET("/", private.GetPolicies)
		policiesViewGroup.GET("/:hash", private.GetPolicy)
	}

	policiesModulesViewGroup := parent.Group("/policies")
	policiesModulesViewGroup.Use(privilegesRequired("vxapi.policies.api.view"))
	{
		policiesModulesViewGroup.GET("/:hash/modules", private.GetPolicyModules)
		policiesModulesViewGroup.GET("/:hash/modules/:module_name", private.GetPolicyModule)
		policiesModulesViewGroup.GET("/:hash/modules/:module_name/bmodule.vue", private.GetPolicyBModule)
	}

	policiesGroupsLinkGroup := parent.Group("/policies")
	policiesGroupsLinkGroup.Use(privilegesRequired("vxapi.policies.control.link"))
	{
		policiesGroupsLinkGroup.PUT("/:hash/groups", private.PatchPolicyGroup)
	}

	policiesSecureConfigViewGroup := parent.Group("/policies")
	policiesSecureConfigViewGroup.Use(privilegesRequired("vxapi.policies.api.edit", "vxapi.modules.secure-config.view"))
	{
		policiesSecureConfigViewGroup.GET("/:hash/modules/:module_name/secure_config/:param_name", private.GetPolicyModuleSecureConfigValue)
	}

	policiesSecureConfigEditGroup := parent.Group("/policies")
	policiesSecureConfigEditGroup.Use(privilegesRequired("vxapi.policies.api.edit", "vxapi.modules.secure-config.edit"))
	{
		policiesSecureConfigEditGroup.POST("/:hash/modules/:module_name/secure_config", private.SetPolicyModuleSecureConfigValue)
	}
}

func setEventsGroup(parent *gin.RouterGroup) {
	eventsGroup := parent.Group("/events")
	eventsGroup.Use(privilegesRequired("vxapi.modules.events"))
	{
		eventsGroup.GET("/", private.GetEvents)
	}
}

func setSystemModulesGroup(parent *gin.RouterGroup) {
	parent = parent.Group("/")
	parent.Use(setSecureConfigEncryptor())

	systemModulesCreateGroup := parent.Group("/modules")
	systemModulesCreateGroup.Use(privilegesRequired("vxapi.modules.api.create"))
	{
		systemModulesCreateGroup.POST("/", private.CreateModule)
	}

	systemModulesDeleteGroup := parent.Group("/modules")
	systemModulesDeleteGroup.Use(privilegesRequired("vxapi.modules.api.delete"))
	{
		systemModulesDeleteGroup.DELETE("/:module_name", private.DeleteModule)
	}

	systemModulesEditGroup := parent.Group("/modules")
	systemModulesEditGroup.Use(privilegesRequired("vxapi.modules.api.edit"))
	{
		systemModulesEditGroup.POST("/:module_name/versions/:version", private.CreateModuleVersion)
		systemModulesEditGroup.PUT("/:module_name/versions/:version", private.PatchModuleVersion)
		systemModulesEditGroup.DELETE("/:module_name/versions/:version", private.DeleteModuleVersion)

		systemModulesEditGroup.GET("/:module_name/versions/:version/files", private.GetModuleVersionFiles)
		systemModulesEditGroup.GET("/:module_name/versions/:version/files/file", private.GetModuleVersionFile)
		systemModulesEditGroup.PUT("/:module_name/versions/:version/files/file", private.PatchModuleVersionFile)

		systemModulesEditGroup.GET("/:module_name/versions/:version/updates", private.GetModuleVersionUpdates)
		systemModulesEditGroup.POST("/:module_name/versions/:version/updates", private.CreateModuleVersionUpdates)
	}

	systemModulesViewGroup := parent.Group("/modules")
	systemModulesViewGroup.Use(privilegesRequired("vxapi.modules.api.view"))
	{
		systemModulesViewGroup.GET("/", private.GetModules)
		systemModulesViewGroup.GET("/:module_name/versions/", private.GetModuleVersions)
		systemModulesViewGroup.GET("/:module_name/versions/:version", private.GetModuleVersion)
		systemModulesViewGroup.GET("/:module_name/versions/:version/options/:option_name", private.GetModuleVersionOption)
	}
}

func setExportGroup(parent *gin.RouterGroup) {
	exportGroup := parent.Group("/export")
	exportGroup.Use(privilegesRequired("vxapi.modules.control.export"))
	{
		exportGroup.POST("/modules/:module_name/versions/:version", private.ExportModule)
	}
}

func setImportGroup(parent *gin.RouterGroup) {
	importGroup := parent.Group("/import")
	importGroup.Use(privilegesRequired("vxapi.modules.control.import"))
	{
		importGroup.POST("/modules/:module_name/versions/:version", private.ImportModule)
	}
}

func setOptionsGroup(parent *gin.RouterGroup) {
	optionsGroup := parent.Group("/options")
	optionsGroup.Use(privilegesRequired("vxapi.modules.api.view"))
	{
		optionsGroup.GET("/actions", private.GetOptionsActions)
		optionsGroup.GET("/events", private.GetOptionsEvents)
		optionsGroup.GET("/fields", private.GetOptionsFields)
		optionsGroup.GET("/tags", private.GetOptionsTags)
		optionsGroup.GET("/versions", private.GetOptionsVersions)
	}
}

func setNotificationsGroup(parent *gin.RouterGroup, exchanger *srvevents.Exchanger) {
	notificationsGroup := parent.Group("/notifications")
	premsFilter := func(c *gin.Context, name srvevents.EventChannelName) bool {
		prms, ok := utils.GetStringArray(c, "prm")
		if !ok {
			return false
		}
		var privs []string
		switch name {
		case srvevents.CreateAgentsChannel, srvevents.UpdateAgentsChannel, srvevents.DeleteAgentsChannel:
			privs = append(privs, "vxapi.agents.api.view")
		case srvevents.CreateGroupsChannel, srvevents.UpdateGroupsChannel, srvevents.DeleteGroupsChannel:
			privs = append(privs, "vxapi.groups.api.view")
		case srvevents.CreatePoliciesChannel, srvevents.UpdatePoliciesChannel, srvevents.DeletePoliciesChannel:
			privs = append(privs, "vxapi.policies.api.view")
		case srvevents.CreateModulesChannel, srvevents.UpdateModulesChannel, srvevents.DeleteModulesChannel:
			privs = append(privs, "vxapi.policies.api.view")
		case srvevents.CreateGroupToPolicyChannel, srvevents.DeleteGroupToPolicyChannel:
			privs = append(privs, "vxapi.groups.api.view")
			privs = append(privs, "vxapi.policies.api.view")
		case srvevents.AllEventsChannel:
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

func setTagsGroup(parent *gin.RouterGroup) {
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
		tagsGroup.GET("/", private.GetTags)
	}
}

func setVersionsGroup(parent *gin.RouterGroup) {
	versionsGroup := parent.Group("/versions")
	versionsGroup.Use(privilegesRequiredByQueryTypeField(
		map[string][]string{
			"agents":  {"vxapi.agents.api.view"},
			"modules": {"vxapi.policies.api.view"},
		},
	))
	{
		versionsGroup.GET("/", private.GetVersions)
	}
}

func setRolesGroup(parent *gin.RouterGroup) {
	rolesGroup := parent.Group("/roles")
	rolesGroup.Use(privilegesRequired("vxapi.roles.api.view"))
	{
		rolesGroup.GET("/", private.GetRoles)
	}
}

func setServicesGroup(parent *gin.RouterGroup) {
	servicesCreateGroup := parent.Group("/services")
	servicesCreateGroup.Use(privilegesRequired("vxapi.services.api.create"))
	{
		servicesCreateGroup.POST("/", private.CreateService)
	}

	servicesDeleteGroup := parent.Group("/services")
	servicesDeleteGroup.Use(privilegesRequired("vxapi.services.api.delete"))
	{
		servicesDeleteGroup.DELETE("/:hash", private.DeleteService)
	}

	servicesEditGroup := parent.Group("/services")
	servicesEditGroup.Use(privilegesRequired("vxapi.services.api.edit"))
	{
		servicesEditGroup.PUT("/:hash", private.PatchService)
	}

	servicesViewGroup := parent.Group("/services")
	servicesViewGroup.Use(privilegesRequired("vxapi.services.api.view"))
	{
		servicesViewGroup.GET("/", private.GetServices)
		servicesViewGroup.GET("/:hash", private.GetService)
	}
}

func setTenanesGroup(parent *gin.RouterGroup) {
	tenantsCreateGroup := parent.Group("/tenants")
	tenantsCreateGroup.Use(privilegesRequired("vxapi.tenants.api.create"))
	{
		tenantsCreateGroup.POST("/", private.CreateTenant)
	}

	tenantsDeleteGroup := parent.Group("/tenants")
	tenantsDeleteGroup.Use(privilegesRequired("vxapi.tenants.api.delete"))
	{
		tenantsDeleteGroup.DELETE("/:hash", private.DeleteTenant)
	}

	tenantsEditGroup := parent.Group("/tenants")
	tenantsEditGroup.Use(privilegesRequired("vxapi.tenants.api.edit"))
	{
		tenantsEditGroup.PUT("/:hash", private.PatchTenant)
	}

	tenantsViewGroup := parent.Group("/tenants")
	tenantsViewGroup.Use(privilegesRequired("vxapi.tenants.api.view"))
	{
		tenantsViewGroup.GET("/", private.GetTenants)
		tenantsViewGroup.GET("/:hash", private.GetTenant)
	}
}

func setUsersGroup(parent *gin.RouterGroup) {
	usersCreateGroup := parent.Group("/users")
	usersCreateGroup.Use(privilegesRequired("vxapi.users.api.create"))
	{
		usersCreateGroup.POST("/", private.CreateUser)
	}

	usersDeleteGroup := parent.Group("/users")
	usersDeleteGroup.Use(privilegesRequired("vxapi.users.api.delete"))
	{
		usersDeleteGroup.DELETE("/:hash", private.DeleteUser)
	}

	usersEditGroup := parent.Group("/users")
	usersEditGroup.Use(privilegesRequired("vxapi.users.api.edit"))
	{
		usersEditGroup.PUT("/:hash", private.PatchUser)
	}

	usersViewGroup := parent.Group("/users")
	usersViewGroup.Use(privilegesRequired("vxapi.users.api.view"))
	{
		usersViewGroup.GET("/", private.GetUsers)
		usersViewGroup.GET("/:hash", private.GetUser)
	}

	userViewGroup := parent.Group("/user")
	{
		userViewGroup.GET("/", private.GetCurrentUser)
	}

	userEditGroup := parent.Group("/user")
	userEditGroup.Use(localUserRequired())
	{
		userEditGroup.PUT("/password", private.ChangePasswordCurrentUser)
	}
}

func index(c *gin.Context) {
	data, err := ioutil.ReadFile("./static/index.html")
	if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error loading index.html")
		return
	}
	c.Data(200, "text/html", []byte(data))
}
