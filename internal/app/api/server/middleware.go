package server

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	"soldr/internal/app/api/models"
	srverrors "soldr/internal/app/api/server/errors"
	"soldr/internal/app/api/server/private"
	"soldr/internal/app/api/utils"
	"soldr/internal/app/api/utils/dbencryptor"
)

func authTokenProtoRequired() gin.HandlerFunc {
	privInteractive := "vxapi.modules.interactive"
	connTypeRegexp := regexp.MustCompile(
		fmt.Sprintf("%s/vxpws/(aggregate|browser|external)/.*", utils.PrefixPathAPI),
	)
	return func(c *gin.Context) {
		if c.IsAborted() {
			return
		}

		authFallback := func(msg string) {
			session := sessions.Default(c)
			uid := session.Get("uid")
			rid := session.Get("rid")
			sid := session.Get("sid")
			tid := session.Get("tid")
			prm := session.Get("prm")
			exp := session.Get("exp")
			gtm := session.Get("gtm")
			uname := session.Get("uname")

			attrs := []interface{}{uid, rid, sid, tid, prm, exp, gtm, uname}
			for _, attr := range attrs {
				if attr == nil {
					utils.HTTPError(c, srverrors.ErrNotPermitted, errors.New(msg))
					return
				}
			}

			if prms, ok := prm.([]string); !ok {
				utils.HTTPError(c, srverrors.ErrNotPermitted, nil)
				return
			} else {
				if !lookupPerm(prms, privInteractive) {
					utils.HTTPError(c, srverrors.ErrNotPermitted, nil)
					return
				}
				c.Set("prm", prms)
			}

			connTypeVal := connTypeRegexp.ReplaceAllString(c.Request.URL.Path, "$1")
			switch connTypeVal {
			case "aggregate", "browser", "external":
			default:
				connTypeVal = "browser"
			}

			c.Set("uid", uid.(uint64))
			c.Set("rid", rid.(uint64))
			c.Set("sid", sid.(uint64))
			c.Set("tid", tid.(uint64))
			c.Set("exp", exp.(int64))
			c.Set("gtm", gtm.(int64))
			c.Set("cpt", connTypeVal)
			c.Set("uname", uname.(string))

			c.Next()
		}
		auth := c.Request.Header.Get("Authorization")
		if auth == "" {
			authFallback("token required")
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == auth {
			authFallback("must be used bearer schema")
			return
		}
		claims, err := private.ValidateToken(token)
		if err != nil {
			authFallback("token invalid")
			return
		}

		c.Set("uid", claims.UID)
		c.Set("rid", claims.RID)
		c.Set("sid", claims.SID)
		c.Set("tid", claims.TID)
		c.Set("cpt", claims.CPT)
		c.Set("prm", []string{privInteractive})

		c.Next()
	}
}

func authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.IsAborted() {
			return
		}

		session := sessions.Default(c)
		uid := session.Get("uid")
		rid := session.Get("rid")
		sid := session.Get("sid")
		tid := session.Get("tid")
		prm := session.Get("prm")
		exp := session.Get("exp")
		gtm := session.Get("gtm")
		uname := session.Get("uname")
		svc := session.Get("svc")

		attrs := []interface{}{uid, rid, sid, tid, prm, exp, gtm, uname}
		for _, attr := range attrs {
			if attr == nil {
				utils.HTTPError(c, srverrors.ErrAuthRequired, errors.New("token claim invalid"))
				return
			}
		}

		if prms, ok := prm.([]string); !ok {
			utils.HTTPError(c, srverrors.ErrAuthRequired, nil)
			return
		} else {
			c.Set("prm", prms)
		}

		c.Set("uid", uid.(uint64))
		c.Set("rid", rid.(uint64))
		c.Set("sid", sid.(uint64))
		c.Set("tid", tid.(uint64))
		c.Set("exp", exp.(int64))
		c.Set("gtm", gtm.(int64))
		c.Set("uname", uname.(string))
		c.Set("svc", svc.(string))

		c.Next()
	}
}

func localUserRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.IsAborted() {
			return
		}

		session := sessions.Default(c)
		rid := session.Get("rid")

		if rid == nil || rid.(uint64) == models.RoleExternal {
			utils.HTTPError(c, srverrors.ErrLocalUserRequired, nil)
			return
		}

		c.Next()
	}
}

func inconcurrentRequest() gin.HandlerFunc {
	var reqLock sync.Mutex
	return func(c *gin.Context) {
		if c.IsAborted() {
			return
		}

		reqLock.Lock()
		defer reqLock.Unlock()

		c.Next()
	}
}

func setGlobalDB(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.IsAborted() {
			return
		}

		c.Set("gDB", db)
		c.Next()
	}
}

func setSecureConfigEncryptor() gin.HandlerFunc {
	encryptor := dbencryptor.NewSecureConfigEncryptor(dbencryptor.GetKey)

	return func(c *gin.Context) {
		if c.IsAborted() {
			return
		}

		c.Set("crp", encryptor)
		c.Next()
	}
}

func setServiceInfo() gin.HandlerFunc {
	var mx sync.Mutex
	mDB := make(map[uint64]*gorm.DB)
	mSV := make(map[uint64]*models.Service)
	getInstanceDB := func(c *gin.Context) (*gorm.DB, *models.Service) {
		mx.Lock()
		defer mx.Unlock()

		var gDB *gorm.DB
		if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
			return nil, nil
		}

		sid, ok := utils.GetUint64(c, "sid")
		if !ok || sid == 0 {
			return nil, nil
		}

		if iDB, ok := mDB[sid]; !ok {
			var s models.Service
			if err := gDB.Take(&s, "id = ?", sid).Error; err != nil {
				return nil, nil
			}

			iDB = utils.GetDB(s.Info.DB.User, s.Info.DB.Pass, s.Info.DB.Host,
				strconv.Itoa(int(s.Info.DB.Port)), s.Info.DB.Name)
			if iDB != nil {
				mDB[sid] = iDB
				mSV[sid] = &s
				return iDB, &s
			}
		} else {
			if sv, ok := mSV[sid]; ok {
				return iDB, sv
			}
		}

		return nil, nil
	}

	return func(c *gin.Context) {
		if c.IsAborted() {
			return
		}

		iDB, sv := getInstanceDB(c)
		c.Set("iDB", iDB)
		c.Set("SV", sv)
		c.Next()
	}
}
