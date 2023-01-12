package public

import (
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"soldr/internal/app/api/models"
	srverrors "soldr/internal/app/api/server/errors"
	"soldr/internal/app/api/utils"
	"soldr/internal/version"
)

type info struct {
	Type                   string            `json:"type"`
	Service                *models.Service   `json:"service"`
	Develop                bool              `json:"develop"`
	User                   models.User       `json:"user"`
	Role                   models.Role       `json:"role"`
	Tenant                 models.Tenant     `json:"tenant"`
	Privs                  []string          `json:"privileges"`
	Services               []*models.Service `json:"services"`
	PasswordChangeRequired bool              `json:"password_change_required"`
}

func refreshCookie(c *gin.Context, resp *info, privs []string) error {
	session := sessions.Default(c)

	expires := utils.DefaultSessionTimeout
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
		Secure:   utils.IsUseSSL(),
		Path:     utils.PrefixPathAPI,
		MaxAge:   expires,
	})
	session.Save()

	utils.FromContext(c).
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

// Info is function to return settings and current information about system and config
// @Summary Retrieve current user and system settings
// @Tags Public
// @Produce json
// @Param refresh_cookie query boolean false "boolean arg to refresh current cookie, use explicit false"
// @Success 200 {object} utils.successResp{data=info} "info received successful"
// @Failure 403 {object} utils.errorResp "getting info not permitted"
// @Failure 404 {object} utils.errorResp "user not found"
// @Failure 500 {object} utils.errorResp "internal error on getting information about system and config"
// @Router /info [get]
func Info(c *gin.Context) {
	var (
		err   error
		expt  int64
		gtmt  int64
		gDB   *gorm.DB
		ok    bool
		privs []string
		resp  info
	)

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	nowt := time.Now().Unix()
	session := sessions.Default(c)
	uid := session.Get("uid")
	exp := session.Get("exp")
	gtm := session.Get("gtm")
	svc := session.Get("svc")
	resp.Develop = version.IsDevelop == "true"

	if privs := session.Get("prm"); privs != nil {
		if resp.Privs, ok = privs.([]string); !ok || resp.Privs == nil {
			resp.Privs = make([]string, 0)
		}
	}

	if exp != nil && gtm != nil {
		expt, _ = exp.(int64)
		gtmt, _ = gtm.(int64)
	}

	if uid == nil || expt == 0 || gtmt == 0 || nowt > expt {
		resp.Type = "guest"
		resp.Privs = make([]string, 0)
	} else {
		resp.Type = "user"
		err = gDB.Take(&resp.User, "id = ?", uid).
			Related(&resp.Role).Related(&resp.Tenant).Error
		if err != nil {
			utils.HTTPError(c, srverrors.ErrInfoUserNotFound, err)
			return
		} else if err = resp.User.Valid(); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error validating user data '%s'", resp.User.Hash)
			utils.HTTPError(c, srverrors.ErrInfoInvalidUserData, err)
			return
		}

		if resp.User.ID == 1 {
			var userPassword models.UserPassword
			if err = gDB.Take(&userPassword, "id = ?", resp.User.ID).Error; err != nil {
				utils.HTTPError(c, srverrors.ErrInfoUserNotFound, err)
				return
			}
			// NOTE: password from seed defaults, see db/api/seed.sql
			resp.PasswordChangeRequired =
				(userPassword.Password == "$2a$10$deVOk0o1nYRHpaVXjIcyCuRmaHvtoMN/2RUT7w5XbZTeiWKEbXx9q")
		}

		if err = gDB.Table("privileges").Where("role_id = ?", resp.User.RoleID).Pluck("name", &privs).Error; err != nil {
			utils.FromContext(c).WithError(err).Errorf("error getting user privileges list '%s'", resp.User.Hash)
			utils.HTTPError(c, srverrors.ErrInfoInvalidUserData, err)
			return
		}

		if err = gDB.Find(&resp.Services, "tenant_id = ?", resp.User.TenantID).Error; err != nil {
			utils.FromContext(c).WithError(err).Errorf("error getting user services list '%s'", resp.User.Hash)
			utils.HTTPError(c, srverrors.ErrInfoInvalidServiceData, err)
			return
		}

		for _, service := range resp.Services {
			service.Info = nil
			if svc == nil {
				continue
			}
			if svcHash, ok := svc.(string); ok && service.Hash == svcHash {
				resp.Service = service
			}
		}

		// check 5 minutes timeout to refresh current token
		var fiveMins int64 = 5 * 60
		if nowt >= gtmt+fiveMins && c.Query("refresh_cookie") != "false" {
			if err = refreshCookie(c, &resp, privs); err != nil {
				utils.FromContext(c).WithError(err).Errorf("failed to refresh token")
				// raise error when there is elapsing last five minutes
				if nowt >= gtmt+int64(utils.DefaultSessionTimeout)-fiveMins {
					utils.HTTPError(c, srverrors.ErrInternal, err)
					return
				}
			}
		}
	}

	// raise error when there is elapsing last five minutes
	// and user hasn't permissions in the session auth cookie
	if resp.Type != "guest" && resp.Privs == nil {
		utils.FromContext(c).
			WithFields(logrus.Fields{
				"uid": resp.User.ID,
				"rid": resp.User.RoleID,
				"tid": resp.User.TenantID,
			}).
			Errorf("failed to get user privileges for '%s' '%s'", resp.User.Mail, resp.User.Name)
		utils.HTTPError(c, srverrors.ErrInternal, err)
		return
	}

	utils.HTTPSuccess(c, http.StatusOK, resp)
}
