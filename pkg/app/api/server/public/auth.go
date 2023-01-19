package public

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"soldr/pkg/app/api/logger"
	"soldr/pkg/app/api/models"
	"soldr/pkg/app/api/server/context"
	"soldr/pkg/app/api/server/response"
	"soldr/pkg/app/api/storage"
)

type AuthServiceConfig struct {
	APIBaseURL     string
	SessionTimeout int
	SecureCookie   bool
}

type AuthService struct {
	cfg AuthServiceConfig
	db  *gorm.DB
}

func NewAuthService(cfg AuthServiceConfig, db *gorm.DB) *AuthService {
	return &AuthService{
		cfg: cfg,
		db:  db,
	}
}

// AuthLogin is function to login user in the system
// @Summary Login user into system
// @Tags Public
// @Accept json
// @Produce json
// @Param json body models.Login true "Login form JSON data"
// @Success 200 {object} utils.successResp "login successful"
// @Failure 400 {object} utils.errorResp "invalid login data"
// @Failure 401 {object} utils.errorResp "invalid login or password"
// @Failure 403 {object} utils.errorResp "login not permitted"
// @Failure 500 {object} utils.errorResp "internal error on login"
// @Router /auth/login [post]
func (s *AuthService) AuthLogin(c *gin.Context) {
	var data models.Login
	if err := c.ShouldBindJSON(&data); err != nil || data.Valid() != nil {
		if err == nil {
			err = data.Valid()
		}
		logger.FromContext(c).WithError(err).Errorf("error validating request data")
		response.Error(c, response.ErrAuthInvalidLoginRequest, err)
		return
	}

	var user models.UserPassword
	if err := s.db.Take(&user, "(mail = ? OR name = ?) AND password IS NOT NULL", data.Mail, data.Mail).Error; err != nil {
		logrus.WithError(err).Errorf("error getting user by mail '%s'", data.Mail)
		response.Error(c, response.ErrAuthInvalidCredentials, err)
		return
	} else if err = user.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating user data '%s'", user.Hash)
		response.Error(c, response.ErrAuthInvalidUserData, err)
		return
	} else if user.RoleID == 100 {
		logger.FromContext(c).WithError(err).Errorf("can't authorize external user '%s'", user.Hash)
		response.Error(c, response.ErrAuthInvalidUserData, fmt.Errorf("user is external"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(data.Password)); err != nil {
		logger.FromContext(c).Errorf("error matching user input password")
		response.Error(c, response.ErrAuthInvalidCredentials, err)
		return
	}

	if user.Status != "active" {
		logger.FromContext(c).Errorf("error checking active state for user '%s'", user.Status)
		response.Error(c, response.ErrAuthInactiveUser, fmt.Errorf("user is inactive"))
		return
	}

	service, err := getService(c, s.db, data.Service, &user.User)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error loading service data by hash '%s'", data.Service)
		response.Error(c, response.ErrAuthInvalidServiceData, err)
		return
	} else if err = service.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating service data '%s'", service.Hash)
		response.Error(c, response.ErrAuthInvalidServiceData, err)
		return
	}

	var privs []string
	err = s.db.Table("privileges").
		Where("role_id = ?", user.RoleID).
		Pluck("name", &privs).Error
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error getting user privileges list '%s'", user.Hash)
		response.Error(c, response.ErrAuthInvalidServiceData, err)
		return
	}

	uuid, err := storage.MakeUuidStrFromHash(user.Hash)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating user data '%s'", user.Hash)
		response.Error(c, response.ErrAuthInvalidUserData, err)
		return
	}

	expires := s.cfg.SessionTimeout
	session := sessions.Default(c)
	session.Set("uid", user.ID)
	session.Set("rid", user.RoleID)
	session.Set("tid", user.TenantID)
	session.Set("sid", service.ID)
	session.Set("svc", service.Hash)
	session.Set("prm", privs)
	session.Set("gtm", time.Now().Unix())
	session.Set("exp", time.Now().Add(time.Duration(expires)*time.Second).Unix())
	session.Set("uuid", uuid)
	session.Set("uname", user.Name)
	session.Options(sessions.Options{
		HttpOnly: true,
		Secure:   s.cfg.SecureCookie,
		Path:     s.cfg.APIBaseURL,
		MaxAge:   expires,
	})
	session.Save()

	logger.FromContext(c).
		WithFields(logrus.Fields{
			"age": expires,
			"uid": user.ID,
			"rid": user.RoleID,
			"tid": user.TenantID,
			"sid": session.Get("sid"),
			"gtm": session.Get("gtm"),
			"exp": session.Get("exp"),
			"prm": session.Get("prm"),
		}).
		Infof("user made successful local login for '%s'", data.Mail)

	response.Success(c, http.StatusOK, struct{}{})
}

// AuthLogout is function to logout current user
// @Summary Logout current user via HTTP redirect
// @Tags Public
// @Produce json
// @Param return_uri query string false "URI to redirect user there after logout" default(/)
// @Success 307 "redirect to input return_uri path"
// @Router /auth/logout [get]
func (s *AuthService) AuthLogout(c *gin.Context) {
	returnURI := "/"
	if returnURL, err := url.Parse(c.Query("return_uri")); err == nil {
		if uri := returnURL.RequestURI(); uri != "" {
			returnURI = path.Clean(path.Join("/", uri))
		}
	}

	session := sessions.Default(c)
	logger.FromContext(c).
		WithFields(logrus.Fields{
			"uid": session.Get("uid"),
			"rid": session.Get("rid"),
			"tid": session.Get("tid"),
			"sid": session.Get("sid"),
			"gtm": session.Get("gtm"),
			"exp": session.Get("exp"),
			"prm": session.Get("prm"),
		}).
		Info("user made successful logout")

	now := time.Now().Add(-1 * time.Second)
	session.Set("gtm", now.Unix())
	session.Set("exp", now.Unix())
	session.Options(sessions.Options{
		HttpOnly: true,
		Secure:   s.cfg.SecureCookie,
		Path:     s.cfg.APIBaseURL,
		MaxAge:   -1,
	})
	session.Save()

	http.Redirect(c.Writer, c.Request, returnURI, http.StatusTemporaryRedirect)
}

// AuthSwitchService is function to change current authorization cookie while switching current service there
// @Summary Switch current agent server for all next requiest
// @Tags Public
// @Accept multipart/form-data
// @Produce json
// @Param service formData string true "New service hash to change current one and return new cookie"
// @Success 200 {object} utils.successResp{data=models.Service} "switch successful"
// @Failure 400 {object} utils.errorResp "invalid service data"
// @Failure 403 {object} utils.errorResp "switch service not permitted"
// @Failure 500 {object} utils.errorResp "internal error on switch service"
// @Router /auth/switch-service [post]
func (s *AuthService) AuthSwitchService(c *gin.Context) {
	serviceHash := c.PostForm("service")
	if err := models.GetValidator().Var(serviceHash, "len=32,hexadecimal,lowercase,required"); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating input data")
		response.Error(c, response.ErrAuthInvalidSwitchServiceHash, err)
		return
	}

	tid, _ := context.GetUint64(c, "tid")
	exp, _ := context.GetInt64(c, "exp")
	expires := int(time.Until(time.Unix(exp, 0)) / time.Second)

	var tenant models.Tenant
	err := s.db.Take(&tenant, "id = ?", tid).Error
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error loading tenant data '%d'", tid)
		response.Error(c, response.ErrAuthInvalidTenantData, err)
		return
	}
	if err = tenant.Valid(); err != nil {
		logrus.WithError(err).Errorf("error validating tenant data '%d:%s'", tid, tenant.Hash)
		response.Error(c, response.ErrAuthInvalidTenantData, err)
		return
	}

	var service models.Service
	err = s.db.Take(&service, "hash LIKE ? AND status = 'active'", serviceHash).Error
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error loading service data '%s'", serviceHash)
		response.Error(c, response.ErrAuthInvalidServiceData, err)
		return
	}
	if err = service.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating service data '%s'", serviceHash)
		response.Error(c, response.ErrAuthInvalidServiceData, err)
		return
	}

	session := sessions.Default(c)
	session.Set("sid", service.ID)
	session.Set("svc", service.Hash)
	session.Options(sessions.Options{
		HttpOnly: true,
		Secure:   s.cfg.SecureCookie,
		Path:     s.cfg.APIBaseURL,
		MaxAge:   expires,
	})
	session.Save()

	logger.FromContext(c).
		WithFields(logrus.Fields{
			"age": expires,
			"uid": session.Get("uid"),
			"rid": session.Get("rid"),
			"tid": session.Get("tid"),
			"sid": session.Get("sid"),
			"gtm": session.Get("gtm"),
			"exp": session.Get("exp"),
			"prm": session.Get("prm"),
		}).
		Infof("session was refreshed and service was switched to '%s'", serviceHash)

	service.Info = nil
	response.Success(c, http.StatusOK, service)
}

func getService(c *gin.Context, gDB *gorm.DB, hash string, user *models.User) (*models.Service, error) {
	var service models.Service

	sqlGetService := gDB.
		Model(&models.Service{}).
		Order("name ASC")
	if err := models.GetValidator().Var(hash, "len=32,hexadecimal,lowercase,required"); err == nil {
		sqlGetService = sqlGetService.Where("hash LIKE ?", hash)
	}
	if err := sqlGetService.Take(&service, "tenant_id = ?", user.TenantID).Error; err == nil {
		return &service, nil
	} else if errors.Is(err, gorm.ErrRecordNotFound) && hash != "" {
		return getService(c, gDB, "", user)
	} else {
		return nil, fmt.Errorf("failed to get service '%s' from DB: %w", hash, err)
	}
}

func (s *AuthService) refreshCookie(c *gin.Context, resp *info, privs []string) error {
	session := sessions.Default(c)

	expires := s.cfg.SessionTimeout
	session.Set("prm", privs)
	session.Set("gtm", time.Now().Unix())
	session.Set("exp", time.Now().Add(time.Duration(expires)*time.Second).Unix())
	resp.Privs = privs

	session.Set("uid", resp.User.ID)
	session.Set("rid", resp.User.RoleID)
	session.Set("tid", resp.User.TenantID)
	session.Set("sid", session.Get("sid"))
	session.Set("svc", session.Get("svc"))
	session.Options(sessions.Options{
		HttpOnly: true,
		Secure:   s.cfg.SecureCookie,
		Path:     s.cfg.APIBaseURL,
		MaxAge:   expires,
	})
	session.Save()

	logger.FromContext(c).
		WithFields(logrus.Fields{
			"age": expires,
			"uid": resp.User.ID,
			"rid": resp.User.RoleID,
			"tid": resp.User.TenantID,
			"sid": session.Get("sid"),
			"gtm": session.Get("gtm"),
			"exp": session.Get("exp"),
			"prm": session.Get("prm"),
		}).
		Infof("session was refreshed for '%s' '%s'", resp.User.Mail, resp.User.Name)

	return nil
}
