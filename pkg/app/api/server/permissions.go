package server

import (
	"fmt"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"soldr/pkg/app/api/server/protected"
	"soldr/pkg/app/api/server/response"
	"soldr/pkg/app/api/storage"
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
			response.Error(c, response.ErrPrivilegesRequired, err)
			return
		}

		var query storage.TableQuery
		if err := c.ShouldBindQuery(&query); err != nil {
			response.Error(c, response.ErrPrivilegesRequired, fmt.Errorf("error binding query: %w", err))
			return
		}
		for _, filter := range query.Filters {
			if value, ok := filter.Value.(string); filter.Field == "type" && ok {
				if privs, ok := mprivs[value]; ok {
					for _, priv := range privs {
						if !lookupPerm(prms, priv) {
							response.Error(c, response.ErrPrivilegesRequired, fmt.Errorf("'%s' is not set", priv))
							return
						}
					}
				} else {
					response.Error(c, response.ErrPrivilegesRequired, fmt.Errorf("'%s' is not specified", value))
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
			response.Error(c, response.ErrPrivilegesRequired, err)
			return
		}

		var action protected.AgentsAction
		if err := c.ShouldBindBodyWith(&action, binding.JSON); err != nil {
			response.Error(c, response.ErrPrivilegesRequired, fmt.Errorf("error binding query: %w", err))
			return
		}
		if action.Action == "delete" {
			if !lookupPerm(prms, "vxapi.agents.api.delete") {
				response.Error(c, response.ErrPrivilegesRequired, fmt.Errorf("'%s' is not set", "vxapi.agents.api.delete"))
				return
			}
		} else {
			if !lookupPerm(prms, "vxapi.agents.api.edit") {
				response.Error(c, response.ErrPrivilegesRequired, fmt.Errorf("'%s' is not set", "vxapi.agents.api.edit"))
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
			response.Error(c, response.ErrPrivilegesRequired, err)
			return
		}

		for _, priv := range append([]string{}, privs...) {
			if !lookupPerm(prms, priv) {
				response.Error(c, response.ErrPrivilegesRequired, fmt.Errorf("'%s' is not set", priv))
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
