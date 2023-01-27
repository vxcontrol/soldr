package server

import (
	"fmt"
	"regexp"
	"strings"

	"soldr/pkg/app/api/server/private"
	"soldr/pkg/app/api/server/response"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type authResult int

const (
	authResultOk authResult = iota
	authResultSkip
	authResultFail
	authResultAbort
)

type AuthMiddleware struct {
	connectionTypeRegexp *regexp.Regexp
}

func NewAuthMiddleware(baseURL string) *AuthMiddleware {
	return &AuthMiddleware{
		connectionTypeRegexp: regexp.MustCompile(
			fmt.Sprintf("%s/vxpws/(aggregate|browser|external)/.*", baseURL),
		),
	}
}

func (p *AuthMiddleware) AuthRequired(c *gin.Context) {
	p.tryAuth(c,
		p.tryUserCookieAuthentication,
	)
}

func (p *AuthMiddleware) AuthTokenProtoRequired(c *gin.Context) {
	p.tryAuth(c,
		p.tryProtoTokenAuthentication,
		p.tryProtoCookieAuthentication,
	)
}

func (p *AuthMiddleware) tryAuth(c *gin.Context, authMethods ...func(c *gin.Context) authResult) {
	if c.IsAborted() {
		return
	}

	result := authResultSkip
	for _, authMethod := range authMethods {
		result = authMethod(c)
		if result != authResultSkip {
			break
		}
	}

	if result != authResultOk {
		response.Error(c, response.ErrAuthRequired, nil)
		c.Abort()
		return
	}
	c.Next()
}

func (p *AuthMiddleware) tryUserCookieAuthentication(c *gin.Context) authResult {
	sessionObject, exists := c.Get(sessions.DefaultKey)
	if !exists {
		return authResultSkip
	}

	session, ok := sessionObject.(sessions.Session)
	if !ok {
		return authResultFail
	}

	uid := session.Get("uid")
	rid := session.Get("rid")
	sid := session.Get("sid")
	tid := session.Get("tid")
	prm := session.Get("prm")
	exp := session.Get("exp")
	gtm := session.Get("gtm")
	uname := session.Get("uname")
	svc := session.Get("svc")

	for _, attr := range []interface{}{uid, rid, sid, tid, prm, exp, gtm, uname, svc} {
		if attr == nil {
			return authResultFail
		}
	}

	prms, ok := prm.([]string)
	if !ok {
		return authResultFail
	}

	c.Set("prm", prms)
	c.Set("uid", uid.(uint64))
	c.Set("rid", rid.(uint64))
	c.Set("sid", sid.(uint64))
	c.Set("tid", tid.(uint64))
	c.Set("exp", exp.(int64))
	c.Set("gtm", gtm.(int64))
	c.Set("uname", uname.(string))
	c.Set("svc", svc.(string))

	return authResultOk
}

const privilegeInteractive = "vxapi.modules.interactive"

func (p *AuthMiddleware) tryProtoCookieAuthentication(c *gin.Context) authResult {
	sessionObject, exists := c.Get(sessions.DefaultKey)
	if !exists {
		return authResultSkip
	}

	session, ok := sessionObject.(sessions.Session)
	if !ok {
		return authResultFail
	}

	uid := session.Get("uid")
	rid := session.Get("rid")
	sid := session.Get("sid")
	tid := session.Get("tid")
	prm := session.Get("prm")
	exp := session.Get("exp")
	gtm := session.Get("gtm")
	uname := session.Get("uname")

	for _, attr := range []interface{}{uid, rid, sid, tid, prm, exp, gtm, uname} {
		if attr == nil {
			return authResultFail // response.Error(c, response.ErrNotPermitted, errors.New(msg))
		}
	}

	prms, ok := prm.([]string)
	if !ok {
		return authResultFail // response.Error(c, response.ErrNotPermitted, nil)
	}
	if !lookupPerm(prms, privilegeInteractive) {
		return authResultFail // response.Error(c, response.ErrNotPermitted, nil)
	}
	c.Set("prm", prms)

	connectionType := p.connectionTypeRegexp.ReplaceAllString(c.Request.URL.Path, "$1")
	switch connectionType {
	case "aggregate", "browser", "external":
	default:
		connectionType = "browser"
	}

	c.Set("uid", uid.(uint64))
	c.Set("rid", rid.(uint64))
	c.Set("sid", sid.(uint64))
	c.Set("tid", tid.(uint64))
	c.Set("exp", exp.(int64))
	c.Set("gtm", gtm.(int64))
	c.Set("cpt", connectionType)
	c.Set("uname", uname.(string))

	return authResultOk
}

func (p *AuthMiddleware) tryProtoTokenAuthentication(c *gin.Context) authResult {
	authHeader := c.Request.Header.Get("Authorization")
	if authHeader == "" {
		return authResultSkip // "token required"
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return authResultSkip // "must be used bearer schema"
	}
	token := authHeader[7:]
	if token == "" {
		return authResultSkip
	}

	claims, err := private.ValidateToken(token)
	if err != nil {
		return authResultFail // "token invalid"
	}

	c.Set("uid", claims.UID)
	c.Set("rid", claims.RID)
	c.Set("sid", claims.SID)
	c.Set("tid", claims.TID)
	c.Set("cpt", claims.CPT)
	c.Set("prm", []string{privilegeInteractive})

	c.Next()

	return authResultOk
}
