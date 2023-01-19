package server

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"soldr/pkg/app/api/models"
	"soldr/pkg/app/api/server/context"
	"soldr/pkg/app/api/server/protected"
	"soldr/pkg/app/api/server/response"
	"soldr/pkg/app/api/utils/dbencryptor"
	obs "soldr/pkg/observability"
)

func authTokenProtoRequired(apiBaseURL string) gin.HandlerFunc {
	privInteractive := "vxapi.modules.interactive"
	connTypeRegexp := regexp.MustCompile(
		fmt.Sprintf("%s/vxpws/(aggregate|browser|external)/.*", apiBaseURL),
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
					response.Error(c, response.ErrNotPermitted, errors.New(msg))
					return
				}
			}

			if prms, ok := prm.([]string); !ok {
				response.Error(c, response.ErrNotPermitted, nil)
				return
			} else {
				if !lookupPerm(prms, privInteractive) {
					response.Error(c, response.ErrNotPermitted, nil)
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
		claims, err := protected.ValidateToken(token)
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

		attrs := []interface{}{uid, rid, sid, tid, prm, exp, gtm, uname, svc}
		for _, attr := range attrs {
			if attr == nil {
				response.Error(c, response.ErrAuthRequired, errors.New("token claim invalid"))
				return
			}
		}

		if prms, ok := prm.([]string); !ok {
			response.Error(c, response.ErrAuthRequired, nil)
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
			response.Error(c, response.ErrLocalUserRequired, nil)
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

// Deprecated
func setServiceInfo(db *gorm.DB) gin.HandlerFunc {
	var mu sync.Mutex
	serviceCache := make(map[uint64]*models.Service)

	getService := func(c *gin.Context) (*models.Service, error) {
		mu.Lock()
		defer mu.Unlock()

		sid, ok := context.GetUint64(c, "sid")
		if !ok || sid == 0 {
			return nil, errors.New("sid cannot be 0 or absent")
		}

		service, ok := serviceCache[sid]
		if !ok {
			var s models.Service
			if err := db.Take(&s, "id = ?", sid).Error; err != nil {
				return nil, fmt.Errorf("could not fetch service: %w", err)
			}
			serviceCache[sid] = &s
		}
		return service, nil
	}

	return func(c *gin.Context) {
		if c.IsAborted() {
			return
		}

		service, err := getService(c)
		if err != nil {
			response.Error(c, response.ErrInternalServiceNotFound, nil)
			return
		}

		c.Set("SV", service)
		c.Next()
	}
}

func WithLogger(skipPaths []string) gin.HandlerFunc {
	skip := make(map[string]struct{}, len(skipPaths))
	for _, path := range skipPaths {
		skip[path] = struct{}{}
	}

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		ctx, span := obs.Observer.NewSpan(c.Request.Context(), obs.SpanKindServer, "http_server")
		defer span.End()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		if _, ok := skip[path]; ok {
			return
		}
		if raw != "" {
			path = path + "?" + raw
		}
		logrus.WithContext(ctx).WithFields(logrus.Fields{
			"component":        "api",
			"net_peer_ip":      c.ClientIP(),
			"duration":         time.Since(start),
			"http_uri":         path,
			"http_route":       c.Request.URL.Path,
			"http_host_name":   c.Request.Host,
			"http_method":      c.Request.Method,
			"http_status_code": c.Writer.Status(),
		}).Info("http request handled")
	}
}
