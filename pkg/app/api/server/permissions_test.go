package server

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestPrivilegesRequiredPatchAgents(t *testing.T) {
	server := newTestServer(t, "/test", authRequired, privilegesRequiredPatchAgents)
	defer server.Close()

	server.SetSessionCheckFunc(func(t *testing.T, c *gin.Context) {
		t.Helper()
		assert.Equal(t, uint64(1), c.GetUint64("uid"))
	})

	assert.False(t, server.CallAndGetStatus(t))

	server.Authorize(t, []string{"some.permission"})
	id, ok := server.TestCallWithData(t, "not a json")
	assert.False(t, ok)
	assert.False(t, server.Called(id))

	server.Authorize(t, []string{"some.permission"})
	id, ok = server.TestCallWithData(t, `{"action": "delete"}`)
	assert.False(t, ok)
	assert.False(t, server.Called(id))

	server.Authorize(t, []string{"vxapi.agents.api.edit"})
	id, ok = server.TestCallWithData(t, `{"action": "delete"}`)
	assert.False(t, ok)
	assert.False(t, server.Called(id))

	server.Authorize(t, []string{"vxapi.agents.api.delete"})
	id, ok = server.TestCallWithData(t, `{"action": "delete"}`)
	assert.False(t, ok)
	assert.True(t, server.Called(id))

	for _, action := range []string{"authorize", "block", "delete", "unauthorize", "move"} {
		server.Authorize(t, []string{"vxapi.agents.api.delete"})
		id, ok = server.TestCallWithData(t, `{"action": "`+action+`"}`)
		assert.False(t, ok)
		assert.False(t, server.Called(id))

		server.Authorize(t, []string{"vxapi.agents.api.edit"})
		id, ok = server.TestCallWithData(t, `{"action": "`+action+`"}`)
		assert.False(t, ok)
		assert.True(t, server.Called(id))
	}
}

func TestPrivilegesRequired(t *testing.T) {
	privilegesMiddleware := func() gin.HandlerFunc {
		return privilegesRequired("priv1", "priv2")
	}
	server := newTestServer(t, "/test", authRequired, privilegesMiddleware)
	defer server.Close()

	server.SetSessionCheckFunc(func(t *testing.T, c *gin.Context) {
		t.Helper()
		assert.Equal(t, uint64(1), c.GetUint64("uid"))
	})

	assert.False(t, server.CallAndGetStatus(t))

	server.Authorize(t, []string{"some.permission"})
	assert.False(t, server.CallAndGetStatus(t))

	server.Authorize(t, []string{"priv1"})
	assert.False(t, server.CallAndGetStatus(t))

	server.Authorize(t, []string{"priv1", "priv2"})
	assert.True(t, server.CallAndGetStatus(t))
}
