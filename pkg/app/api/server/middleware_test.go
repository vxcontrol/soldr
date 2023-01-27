package server

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"soldr/pkg/app/api/models"
	"soldr/pkg/app/api/server/private"
	"soldr/pkg/app/api/storage"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthTokenProtoRequiredAuthWithCookie(t *testing.T) {
	authMiddleware := NewAuthMiddleware("/base/url")

	t.Run("test URL", func(t *testing.T) {
		server := newTestServer(t, "/test", authMiddleware.AuthTokenProtoRequired)
		defer server.Close()

		server.SetSessionCheckFunc(func(t *testing.T, c *gin.Context) {
			t.Helper()
			assert.Equal(t, uint64(1), c.GetUint64("uid"))
			assert.Equal(t, uint64(2), c.GetUint64("rid"))
			assert.Equal(t, uint64(4), c.GetUint64("sid"))
			assert.Equal(t, uint64(3), c.GetUint64("tid"))
			assert.Empty(t, c.GetString("svc"))
			assert.NotNil(t, c.GetStringSlice("prm"))
			assert.NotNil(t, c.GetInt64("gtm"))
			assert.NotNil(t, c.GetInt64("exp"))
			assert.Empty(t, c.GetString("uuid"))
			assert.Equal(t, "browser", c.GetString("cpt"))
			assert.Equal(t, "name1", c.GetString("uname"))
		})

		assert.False(t, server.CallAndGetStatus(t))

		server.Authorize(t, []string{})
		assert.False(t, server.CallAndGetStatus(t))

		server.Authorize(t, []string{"wrong.permission"})
		assert.False(t, server.CallAndGetStatus(t))

		server.Authorize(t, []string{"vxapi.modules.interactive"})
		assert.True(t, server.CallAndGetStatus(t))

		server.Authorize(t, []string{"wrong.permission", "vxapi.modules.interactive"})
		assert.True(t, server.CallAndGetStatus(t))
	})

	for _, kind := range []string{"aggregate", "browser", "external"} {
		t.Run(kind+" URL", func(t *testing.T) {
			server := newTestServer(t, "/base/url/vxpws/"+kind+"/test", authMiddleware.AuthTokenProtoRequired)
			defer server.Close()

			server.SetSessionCheckFunc(func(t *testing.T, c *gin.Context) {
				t.Helper()
				assert.Equal(t, kind, c.GetString("cpt"))
			})
			server.Authorize(t, []string{"vxapi.modules.interactive"})
			assert.True(t, server.CallAndGetStatus(t))
		})
	}
}

func TestAuthTokenProtoRequiredAuthWithToken(t *testing.T) {
	authMiddleware := NewAuthMiddleware("/base/url")

	for _, kind := range []string{"aggregate", "browser", "external"} {
		t.Run(kind+" type", func(t *testing.T) {
			server := newTestServer(t, "/test", authMiddleware.AuthTokenProtoRequired)
			defer server.Close()

			server.Authorize(t, []string{"vxapi.modules.interactive"})
			token := server.GetToken(t, kind)
			require.NotEmpty(t, token)

			server.Unauthorize(t)
			assert.False(t, server.CallAndGetStatus(t))

			assert.False(t, server.CallAndGetStatus(t, token))
			assert.False(t, server.CallAndGetStatus(t, "not a bearer "+token))
			assert.False(t, server.CallAndGetStatus(t, "Bearer"+token))
			assert.False(t, server.CallAndGetStatus(t, "Bearer not_a_token"))

			server.SetSessionCheckFunc(func(t *testing.T, c *gin.Context) {
				t.Helper()
				assert.Equal(t, uint64(1), c.GetUint64("uid"))
				assert.Equal(t, uint64(2), c.GetUint64("rid"))
				assert.Equal(t, uint64(4), c.GetUint64("sid"))
				assert.Equal(t, uint64(3), c.GetUint64("tid"))
				assert.Empty(t, c.GetString("svc"))
				assert.NotNil(t, c.GetStringSlice("prm"))
				assert.Zero(t, c.GetInt64("gtm"))
				assert.Zero(t, c.GetInt64("exp"))
				assert.Empty(t, c.GetString("uuid"))
				assert.Equal(t, kind, c.GetString("cpt"))
				assert.Empty(t, c.GetString("uname"))
			})

			assert.True(t, server.CallAndGetStatus(t, "Bearer "+token))
		})
	}
}

func TestAuthRequiredAuthWithCookie(t *testing.T) {
	authMiddleware := NewAuthMiddleware("/base/url")

	server := newTestServer(t, "/test", authMiddleware.AuthRequired)
	defer server.Close()

	server.SetSessionCheckFunc(func(t *testing.T, c *gin.Context) {
		t.Helper()
		assert.Equal(t, uint64(1), c.GetUint64("uid"))
		assert.Equal(t, uint64(2), c.GetUint64("rid"))
		assert.Equal(t, uint64(3), c.GetUint64("tid"))
		assert.Equal(t, uint64(4), c.GetUint64("sid"))
		assert.Equal(t, "svc1", c.GetString("svc"))
		assert.NotNil(t, c.GetStringSlice("prm"))
		assert.NotNil(t, c.GetInt64("gtm"))
		assert.NotNil(t, c.GetInt64("exp"))
		assert.Empty(t, c.GetString("uuid"))
		assert.Equal(t, "name1", c.GetString("uname"))
	})

	assert.False(t, server.CallAndGetStatus(t))

	server.Authorize(t, []string{"some.permission"})
	assert.True(t, server.CallAndGetStatus(t))
}

type testServer struct {
	testEndpoint     string
	client           *http.Client
	calls            map[string]struct{}
	sessionCheckFunc func(t *testing.T, c *gin.Context)
	*httptest.Server
}

func newTestServer(t *testing.T, testEndpoint string, middlewares ...gin.HandlerFunc) *testServer {
	t.Helper()

	server := &testServer{}

	router := gin.New()
	cookieStore := cookie.NewStore(storage.MakeCookieStoreKey())
	router.Use(sessions.Sessions("auth", cookieStore))

	server.calls = map[string]struct{}{}

	if testEndpoint == "" {
		testEndpoint = "/test"
	}
	server.testEndpoint = testEndpoint

	router.GET("/auth", func(c *gin.Context) {
		t.Helper()
		privs, _ := c.GetQueryArray("privileges")
		expString, ok := c.GetQuery("expiration")
		assert.True(t, ok)
		exp, err := strconv.Atoi(expString)
		assert.NoError(t, err)
		setTestSession(t, c, privs, exp)
	})

	authRoutes := router.Group("")
	for _, middleware := range middlewares {
		authRoutes.Use(middleware)
	}
	authRoutes.GET(server.testEndpoint, func(c *gin.Context) {
		t.Helper()
		id, _ := c.GetQuery("id")
		require.NotEmpty(t, id)

		if server.sessionCheckFunc != nil {
			server.sessionCheckFunc(t, c)
		}
		server.calls[id] = struct{}{}
	})
	authRoutes.GET("/auth_token", func(c *gin.Context) {
		t.Helper()
		cpt, ok := c.GetQuery("cpt")
		assert.True(t, ok)
		token, err := private.MakeToken(c, &models.ProtoAuthTokenRequest{
			TTL:  3600,
			Type: cpt,
		})
		require.NoError(t, err)
		c.Writer.Write([]byte(token))
	})

	server.Server = httptest.NewServer(router)
	server.client = server.Client()
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	server.client.Jar = jar

	return server
}

func (s *testServer) Authorize(t *testing.T, privileges []string) {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, s.URL+"/auth", nil)
	require.NoError(t, err)
	query := url.Values{}
	for _, p := range privileges {
		query.Add("privileges", p)
	}
	query.Add("expiration", strconv.Itoa(5*60))
	request.URL.RawQuery = query.Encode()

	resp, err := s.client.Do(request)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func (s *testServer) GetToken(t *testing.T, cpt string) string {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, s.URL+"/auth_token", nil)
	require.NoError(t, err)
	query := url.Values{}
	query.Add("cpt", cpt)
	request.URL.RawQuery = query.Encode()

	resp, err := s.client.Do(request)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	token, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(token)
}

func (s *testServer) SetSessionCheckFunc(f func(t *testing.T, c *gin.Context)) {
	s.sessionCheckFunc = f
}

func (s *testServer) Unauthorize(t *testing.T) {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, s.URL+"/auth", nil)
	require.NoError(t, err)
	query := url.Values{}
	query.Add("expiration", strconv.Itoa(-1))
	request.URL.RawQuery = query.Encode()

	resp, err := s.client.Do(request)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func (s *testServer) TestCall(t *testing.T, token ...string) (string, bool) {
	t.Helper()
	id := strconv.Itoa(rand.Int())

	request, err := http.NewRequest(http.MethodGet, s.URL+s.testEndpoint+"?id="+id, nil)
	require.NoError(t, err)
	if token != nil && len(token) == 1 {
		request.Header.Add("Authorization", token[0])
	}

	resp, err := s.client.Do(request)
	require.NoError(t, err)

	assert.True(t, resp.StatusCode == http.StatusOK ||
		resp.StatusCode == http.StatusForbidden)

	return id, resp.StatusCode == http.StatusOK
}

func (s *testServer) TestCallWithData(t *testing.T, data string) (string, bool) {
	t.Helper()
	id := strconv.Itoa(rand.Int())

	request, err := http.NewRequest(http.MethodGet, s.URL+s.testEndpoint+"?id="+id, bytes.NewBufferString(data))
	require.NoError(t, err)

	resp, err := s.client.Do(request)
	require.NoError(t, err)

	assert.True(t, resp.StatusCode == http.StatusOK ||
		resp.StatusCode == http.StatusForbidden)

	return id, resp.StatusCode == http.StatusOK
}

func (s *testServer) Called(id string) bool {
	_, ok := s.calls[id]
	return ok
}

func (s *testServer) CallAndGetStatus(t *testing.T, token ...string) bool {
	t.Helper()
	id, ok := s.TestCall(t, token...)
	assert.Equal(t, ok, s.Called(id))
	return ok
}

func setTestSession(t *testing.T, c *gin.Context, privileges []string, expires int) {
	t.Helper()
	session := sessions.Default(c)
	session.Set("uid", uint64(1))
	session.Set("rid", uint64(2))
	session.Set("tid", uint64(3))
	session.Set("sid", uint64(4))
	session.Set("svc", "svc1")
	session.Set("prm", privileges)
	session.Set("gtm", time.Now().Unix())
	session.Set("exp", time.Now().Add(time.Duration(expires)*time.Second).Unix())
	session.Set("uuid", "uuid1")
	session.Set("uname", "name1")
	session.Options(sessions.Options{
		HttpOnly: true,
		MaxAge:   expires,
	})
	require.NoError(t, session.Save())
}
