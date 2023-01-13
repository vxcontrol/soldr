package server

import (
	"fmt"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"soldr/pkg/app/api/server/private"
	srverrors "soldr/pkg/app/api/server/response"
	"soldr/pkg/app/api/utils"
)

func getPrms(c *gin.Context) ([]string, error) {
	session := sessions.Default(c)
	prm := session.Get("prm")
	if prm == nil {
		return nil, fmt.Errorf("privileges is not set")
	}
	prms, ok := prm.([]string)
	if !ok {
		return nil, fmt.Errorf("privileges is malformed")
	}
	return prms, nil
}

func privilegesRequiredByQueryTypeField(mprivs map[string][]string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.IsAborted() {
			return
		}

		prms, err := getPrms(c)
		if err != nil {
			utils.HTTPError(c, srverrors.ErrPrivilegesRequired, err)
			return
		}

		var query utils.TableQuery
		if err := c.ShouldBindQuery(&query); err != nil {
			utils.HTTPError(c, srverrors.ErrPrivilegesRequired, fmt.Errorf("error binding query: %w", err))
			return
		}
		for _, filter := range query.Filters {
			if value, ok := filter.Value.(string); filter.Field == "type" && ok {
				if privs, ok := mprivs[value]; ok {
					for _, priv := range privs {
						if !lookupPerm(prms, priv) {
							utils.HTTPError(c, srverrors.ErrPrivilegesRequired, fmt.Errorf("'%s' is not set", priv))
							return
						}
					}
				} else {
					utils.HTTPError(c, srverrors.ErrPrivilegesRequired, fmt.Errorf("'%s' is not specified", value))
					return
				}
			}
		}
		c.Next()
	}
}

func privilegesRequiredPatchAgents() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.IsAborted() {
			return
		}

		prms, err := getPrms(c)
		if err != nil {
			utils.HTTPError(c, srverrors.ErrPrivilegesRequired, err)
			return
		}

		var action private.AgentsAction
		if err := c.ShouldBindBodyWith(&action, binding.JSON); err != nil {
			utils.HTTPError(c, srverrors.ErrPrivilegesRequired, fmt.Errorf("error binding query: %w", err))
			return
		}
		if action.Action == "delete" {
			if !lookupPerm(prms, "vxapi.agents.api.delete") {
				utils.HTTPError(c, srverrors.ErrPrivilegesRequired, fmt.Errorf("'%s' is not set", "vxapi.agents.api.delete"))
				return
			}
		} else {
			if !lookupPerm(prms, "vxapi.agents.api.edit") {
				utils.HTTPError(c, srverrors.ErrPrivilegesRequired, fmt.Errorf("'%s' is not set", "vxapi.agents.api.edit"))
				return
			}
		}

		c.Next()
	}
}

func privilegesRequired(privs ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.IsAborted() {
			return
		}

		prms, err := getPrms(c)
		if err != nil {
			utils.HTTPError(c, srverrors.ErrPrivilegesRequired, err)
			return
		}

		for _, priv := range append([]string{}, privs...) {
			if !lookupPerm(prms, priv) {
				utils.HTTPError(c, srverrors.ErrPrivilegesRequired, fmt.Errorf("'%s' is not set", priv))
				return
			}
		}
		c.Next()
	}
}

func lookupPerm(prm []string, perm string) bool {
	for _, p := range prm {
		if p == perm {
			return true
		}
	}
	return false
}
