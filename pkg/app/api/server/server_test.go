package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/memstore"
	"github.com/gin-gonic/gin"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/stretchr/testify/require"

	_ "soldr/pkg/app/api/docs"
	useraction "soldr/pkg/app/api/user_action"
	"soldr/pkg/app/api/utils"
)

var testUserActionFields = utils.UserActionFields{
	Domain:            "test objectDomain",
	ObjectType:        "test objectType",
	ObjectId:          "test objectId",
	ObjectDisplayName: "test objectDisplayName",
	ActionCode:        "test actionCode",
	Success:           true,
	FailReason:        "",
}

var testUserActionFieldsFailed = utils.UserActionFields{
	Domain:            "test objectDomain",
	ObjectType:        "test objectType",
	ObjectId:          "test objectId2",
	ObjectDisplayName: "test objectDisplayName2",
	ActionCode:        "test actionCode",
	Success:           false,
	FailReason:        "error",
}

func compareBeginEventAndUafields(uaBeginEvent useraction.BeginEvent, uaFields utils.UserActionFields) bool {
	if uaBeginEvent.ObjectDomain != uaFields.Domain ||
		uaBeginEvent.ObjectType != uaFields.ObjectType ||
		uaBeginEvent.ObjectID != uaFields.ObjectId ||
		uaBeginEvent.ObjectDisplayName != uaFields.ObjectDisplayName ||
		uaBeginEvent.ActionCode != uaFields.ActionCode {
		return false
	}
	return true
}

func compareSuccessEventAndUafields(uaSuccessEvent useraction.SuccessEvent, uaFields utils.UserActionFields) bool {
	if uaSuccessEvent.ObjectID != uaFields.ObjectId || uaSuccessEvent.ObjectDisplayName != uaFields.ObjectDisplayName {
		return false
	}
	return true
}

func compareFailEventAndUafields(uaFailEvent useraction.FailEvent, uaFields utils.UserActionFields) bool {
	return uaFailEvent.FailReason == uaFields.FailReason
}

func Test_logUserActions(t *testing.T) {
	t.Run("successful test", func(t *testing.T) {
		uachan := make(chan useraction.Converter)
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		store := memstore.NewStore([]byte("secret"))
		r.Use(sessions.Sessions("mysession", store))
		r.Use(func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set("uuid", "test user id")
			session.Set("uname", "test user name")
			c.Set("uaf", []utils.UserActionFields{testUserActionFields, testUserActionFieldsFailed})
		})
		r.Use(useraction.LogUserActions(uachan))
		req, err := http.NewRequestWithContext(c, http.MethodGet, "/", nil)
		require.NoError(t, err, "error in NewRequest")
		go r.ServeHTTP(w, req)

		userAction := <-uachan
		uaBeginEvent, ok := userAction.(useraction.BeginEvent)
		require.True(t, ok, "wrong format of sent event")
		require.True(t, compareBeginEventAndUafields(uaBeginEvent, testUserActionFields), "wrong fields in sent event, got %v want %v", uaBeginEvent, testUserActionFields)
		userAction = <-uachan
		uaSuccessEvent, ok := userAction.(useraction.SuccessEvent)
		require.True(t, ok, "wrong format of sent event")
		require.True(t, compareSuccessEventAndUafields(uaSuccessEvent, testUserActionFields), "wrong fields in sent event, got %v want %v", uaSuccessEvent, testUserActionFields)
		userAction = <-uachan
		uaBeginEvent, ok = userAction.(useraction.BeginEvent)
		require.True(t, ok, "wrong format of sent event")
		require.True(t, compareBeginEventAndUafields(uaBeginEvent, testUserActionFieldsFailed), "wrong fields in sent event, got %v want %v", uaBeginEvent, testUserActionFieldsFailed)
		userAction = <-uachan
		uaFailEvent, ok := userAction.(useraction.FailEvent)
		require.True(t, ok, "wrong format of sent event")
		require.True(t, compareFailEventAndUafields(uaFailEvent, testUserActionFieldsFailed), "wrong fields in sent event, got %v want %v", uaFailEvent, testUserActionFieldsFailed)
	})
}

func Test_logUserActions_withAbortedContext(t *testing.T) {
	t.Run("test with aborted context", func(t *testing.T) {
		uachan := make(chan useraction.Converter)
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		store := memstore.NewStore([]byte("secret"))
		r.Use(sessions.Sessions("mysession", store))
		r.Use(func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set("uuid", "test user id")
			session.Set("uname", "test user name")
			c.Set("uaf", []utils.UserActionFields{testUserActionFields, testUserActionFieldsFailed})
			c.Abort()
		})
		r.Use(useraction.LogUserActions(uachan))
		req, err := http.NewRequestWithContext(c, http.MethodGet, "/", nil)
		require.NoError(t, err, "error in NewRequest")
		r.ServeHTTP(w, req)

	})
}

func Test_logUserActions_withNoUaf(t *testing.T) {
	t.Run("test with no uaf", func(t *testing.T) {
		uachan := make(chan useraction.Converter)
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		store := memstore.NewStore([]byte("secret"))
		r.Use(sessions.Sessions("mysession", store))
		r.Use(func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set("uuid", "test user id")
			session.Set("uname", "test user name")
		})
		r.Use(useraction.LogUserActions(uachan))
		req, err := http.NewRequestWithContext(c, http.MethodGet, "/", nil)
		require.NoError(t, err, "error in NewRequest")
		r.ServeHTTP(w, req)
	})
}

func Test_logUserActions_withIncorrectUaf(t *testing.T) {
	t.Run("test with incorrect uaf", func(t *testing.T) {
		uachan := make(chan useraction.Converter)
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		store := memstore.NewStore([]byte("secret"))
		r.Use(sessions.Sessions("mysession", store))
		r.Use(func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set("uuid", "test user id")
			session.Set("uname", "test user name")
			c.Set("uaf", "incorrect")
		})
		r.Use(useraction.LogUserActions(uachan))
		req, err := http.NewRequestWithContext(c, http.MethodGet, "/", nil)
		require.NoError(t, err, "error in NewRequest")
		r.ServeHTTP(w, req)
	})
}

func Test_logUserActions_withEmptyFieldInUaf(t *testing.T) {
	t.Run("test with empty field in uaf", func(t *testing.T) {
		uachan := make(chan useraction.Converter)
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		store := memstore.NewStore([]byte("secret"))
		r.Use(sessions.Sessions("mysession", store))
		r.Use(func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set("uuid", "test user id")
			session.Set("uname", "test user name")
			uafs := []utils.UserActionFields{testUserActionFields, testUserActionFieldsFailed}
			uafs[0].Domain = ""
			c.Set("uaf", uafs)
		})
		r.Use(useraction.LogUserActions(uachan))
		req, err := http.NewRequestWithContext(c, http.MethodGet, "/", nil)
		require.NoError(t, err, "error in NewRequest")
		r.ServeHTTP(w, req)
	})
}

func Test_logUserActions_withIncorrectUserId(t *testing.T) {
	t.Run("test with incorrect user id", func(t *testing.T) {
		uachan := make(chan useraction.Converter)
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		store := memstore.NewStore([]byte("secret"))
		r.Use(sessions.Sessions("mysession", store))
		r.Use(func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set("uuid", 7)
			session.Set("uname", "test user name")
			c.Set("uaf", []utils.UserActionFields{testUserActionFields, testUserActionFieldsFailed})
		})
		r.Use(useraction.LogUserActions(uachan))
		req, err := http.NewRequestWithContext(c, http.MethodGet, "/", nil)
		require.NoError(t, err, "error in NewRequest")
		r.ServeHTTP(w, req)

	})
}

func Test_logUserActions_withIncorrectUserName(t *testing.T) {
	t.Run("test with incorrect user name", func(t *testing.T) {
		uachan := make(chan useraction.Converter)
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		store := memstore.NewStore([]byte("secret"))
		r.Use(sessions.Sessions("mysession", store))
		r.Use(func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set("uuid", "test user id")
			session.Set("uname", 7)
			c.Set("uaf", []utils.UserActionFields{testUserActionFields, testUserActionFieldsFailed})
		})
		r.Use(useraction.LogUserActions(uachan))
		req, err := http.NewRequestWithContext(c, http.MethodGet, "/", nil)
		require.NoError(t, err, "error in NewRequest")
		r.ServeHTTP(w, req)

	})
}
