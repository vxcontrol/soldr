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

	"soldr/internal/app/api/models"
	"soldr/internal/app/api/server/context"
	srverrors "soldr/internal/app/api/server/response"
	"soldr/internal/app/api/utils"
)

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
func AuthLogin(c *gin.Context) {
	var (
		data    models.Login
		err     error
		gDB     *gorm.DB
		privs   []string
		service *models.Service
		user    models.UserPassword
	)

	if err = c.ShouldBindJSON(&data); err != nil || data.Valid() != nil {
		if err == nil {
			err = data.Valid()
		}
		utils.FromContext(c).WithError(err).Errorf("error validating request data")
		utils.HTTPError(c, srverrors.ErrAuthInvalidLoginRequest, err)
		return
	}

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	if err = gDB.Take(&user, "(mail = ? OR name = ?) AND password IS NOT NULL", data.Mail, data.Mail).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error getting user by mail '%s'", data.Mail)
		utils.HTTPError(c, srverrors.ErrAuthInvalidCredentials, err)
		return
	} else if err = user.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating user data '%s'", user.Hash)
		utils.HTTPError(c, srverrors.ErrAuthInvalidUserData, err)
		return
	} else if user.RoleID == 100 {
		utils.FromContext(c).WithError(err).Errorf("can't authorize external user '%s'", user.Hash)
		utils.HTTPError(c, srverrors.ErrAuthInvalidUserData, fmt.Errorf("user is external"))
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(data.Password)); err != nil {
		utils.FromContext(c).Errorf("error matching user input password")
		utils.HTTPError(c, srverrors.ErrAuthInvalidCredentials, err)
		return
	}

	if user.Status != "active" {
		utils.FromContext(c).Errorf("error checking active state for user '%s'", user.Status)
		utils.HTTPError(c, srverrors.ErrAuthInactiveUser, fmt.Errorf("user is inactive"))
		return
	}

	if service, err = getService(c, gDB, data.Service, &user.User); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error loading service data by hash '%s'", data.Service)
		utils.HTTPError(c, srverrors.ErrAuthInvalidServiceData, err)
		return
	} else if err = service.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating service data '%s'", service.Hash)
		utils.HTTPError(c, srverrors.ErrAuthInvalidServiceData, err)
		return
	}

	if err = gDB.Table("privileges").Where("role_id = ?", user.RoleID).Pluck("name", &privs).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error getting user privileges list '%s'", user.Hash)
		utils.HTTPError(c, srverrors.ErrAuthInvalidServiceData, err)
		return
	}

	uuid, err := utils.MakeUuidStrFromHash(user.Hash)
	if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating user data '%s'", user.Hash)
		utils.HTTPError(c, srverrors.ErrAuthInvalidUserData, err)
		return
	}

	expires := utils.DefaultSessionTimeout
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
		Secure:   utils.IsUseSSL(),
		Path:     utils.PrefixPathAPI,
		MaxAge:   expires,
	})
	session.Save()

	utils.FromContext(c).
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

	utils.HTTPSuccess(c, http.StatusOK, struct{}{})
}

// AuthLogout is function to logout current user
// @Summary Logout current user via HTTP redirect
// @Tags Public
// @Produce json
// @Param return_uri query string false "URI to redirect user there after logout" default(/)
// @Success 307 "redirect to input return_uri path"
// @Router /auth/logout [get]
func AuthLogout(c *gin.Context) {
	returnURI := "/"
	if returnURL, err := url.Parse(c.Query("return_uri")); err == nil {
		if uri := returnURL.RequestURI(); uri != "" {
			returnURI = path.Clean(path.Join("/", uri))
		}
	}

	session := sessions.Default(c)
	utils.FromContext(c).
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

	resetSession(c)
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
func AuthSwitchService(c *gin.Context) {
	var (
		err     error
		gDB     *gorm.DB
		service models.Service
		tenant  models.Tenant
	)

	serviceHash := c.PostForm("service")
	if err := models.GetValidator().Var(serviceHash, "len=32,hexadecimal,lowercase,required"); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating input data")
		utils.HTTPError(c, srverrors.ErrAuthInvalidSwitchServiceHash, err)
		return
	}

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	tid, _ := context.GetUint64(c, "tid")
	exp, _ := context.GetInt64(c, "exp")
	expires := int(time.Until(time.Unix(exp, 0)) / time.Second)

	if err = gDB.Take(&tenant, "id = ?", tid).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error loading tenant data '%d'", tid)
		utils.HTTPError(c, srverrors.ErrAuthInvalidTenantData, err)
		return
	} else if err = tenant.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating tenant data '%d:%s'", tid, tenant.Hash)
		utils.HTTPError(c, srverrors.ErrAuthInvalidTenantData, err)
		return
	}

	if err = gDB.Take(&service, "hash LIKE ? AND status = 'active'", serviceHash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error loading service data '%s'", serviceHash)
		utils.HTTPError(c, srverrors.ErrAuthInvalidServiceData, err)
		return
	} else if err = service.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating service data '%s'", serviceHash)
		utils.HTTPError(c, srverrors.ErrAuthInvalidServiceData, err)
		return
	}

	session := sessions.Default(c)
	session.Set("sid", service.ID)
	session.Set("svc", service.Hash)
	session.Options(sessions.Options{
		HttpOnly: true,
		Secure:   utils.IsUseSSL(),
		Path:     utils.PrefixPathAPI,
		MaxAge:   expires,
	})
	session.Save()

	utils.FromContext(c).
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
	utils.HTTPSuccess(c, http.StatusOK, service)
}

func resetSession(c *gin.Context) {
	now := time.Now().Add(-1 * time.Second)
	session := sessions.Default(c)
	session.Set("gtm", now.Unix())
	session.Set("exp", now.Unix())
	session.Options(sessions.Options{
		HttpOnly: true,
		Secure:   utils.IsUseSSL(),
		Path:     utils.PrefixPathAPI,
		MaxAge:   -1,
	})
	session.Save()
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
