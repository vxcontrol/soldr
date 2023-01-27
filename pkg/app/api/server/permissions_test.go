package server

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestPrivilegesRequiredPatchAgents(t *testing.T) {
	authMiddleware := NewAuthMiddleware("/base/url")
	server := newTestServer(t, "/test", authMiddleware.AuthRequired, privilegesRequiredPatchAgents())
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
	assert.True(t, ok)
	assert.True(t, server.Called(id))

	for _, action := range []string{"authorize", "block", "unauthorize", "move"} {
		server.Authorize(t, []string{"vxapi.agents.api.delete"})
		id, ok = server.TestCallWithData(t, `{"action": "`+action+`"}`)
		assert.False(t, ok)
		assert.False(t, server.Called(id))

		server.Authorize(t, []string{"vxapi.agents.api.edit"})
		id, ok = server.TestCallWithData(t, `{"action": "`+action+`"}`)
		assert.True(t, ok)
		assert.True(t, server.Called(id))
	}
}

func TestPrivilegesRequired(t *testing.T) {
	authMiddleware := NewAuthMiddleware("/base/url")
	server := newTestServer(t, "/test", authMiddleware.AuthRequired, privilegesRequired("priv1", "priv2"))
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
