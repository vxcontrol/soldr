package utils

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	srverrors "soldr/internal/app/api/server/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var testUserActionFields = UserActionFields{
	Domain:            "test objectDomain",
	ObjectType:        "test objectType",
	ObjectId:          "test objectId",
	ObjectDisplayName: "test objectDisplayName",
	ActionCode:        "test actionCode",
	Success:           true,
	FailReason:        "",
}

var testUserActionFieldsSuccess = UserActionFields{
	Domain:            "test objectDomain",
	ObjectType:        "test objectType",
	ObjectId:          "test objectId",
	ObjectDisplayName: "test objectDisplayName",
	ActionCode:        "test actionCode",
	Success:           true,
	FailReason:        "",
}

var testUserActionFieldsFail = UserActionFields{
	Domain:            "test objectDomain",
	ObjectType:        "test objectType",
	ObjectId:          "test objectId",
	ObjectDisplayName: "test objectDisplayName",
	ActionCode:        "test actionCode",
	Success:           false,
	FailReason:        "internal server error",
}

func TestMakeUuidStrFromHash(t *testing.T) {
	uid := strings.Replace(uuid.New().String(), "-", "", -1)
	type args struct {
		hash string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "successful test",
			args: args{
				hash: uid,
			},
			wantErr: false,
		},
		{
			name: "test with error in decode",
			args: args{
				hash: "a",
			},
			wantErr: true,
		},
		{
			name: "test with error creating error",
			args: args{
				hash: "123456789012345678901234567890",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := MakeUuidStrFromHash(tt.args.hash)
			if !tt.wantErr {
				require.NoError(t, err, "MakeUuidStrFromHash() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHTTPSuccessWithUAFieldsSlice(t *testing.T) {

	t.Run("successful test", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		testUafs := []UserActionFields{testUserActionFields, testUserActionFields}
		HTTPSuccessWithUAFieldsSlice(c, http.StatusOK, "some data", testUafs)

		require.True(t, c.IsAborted(), "context is not aborted")
		uafint, ok := c.Get("uaf")
		require.True(t, ok, "user action fields value is not set")
		uaf, ok := uafint.([]UserActionFields)
		require.True(t, ok, "user action fields value in wrong format")
		testSuccessUafs := []UserActionFields{testUserActionFieldsSuccess, testUserActionFieldsSuccess}
		require.True(t, reflect.DeepEqual(uaf, testSuccessUafs), "wrong user action fields value, got %v, want %v", uaf, testUafs)
	})
}

func TestHTTPSuccessWithUAFieldsSlice_withAbortedContext(t *testing.T) {

	t.Run("successful test with aborted context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Abort()
		testUafs := []UserActionFields{testUserActionFields, testUserActionFields}
		HTTPSuccessWithUAFieldsSlice(c, http.StatusOK, "some data", testUafs)

		_, ok := c.Get("uaf")
		require.False(t, ok, "user action fields value is set")
	})
}

func TestHTTPSuccessWithUAFieldsSlice_withEmptyUafields(t *testing.T) {

	t.Run("successful test with empty uafields", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		HTTPSuccessWithUAFieldsSlice(c, http.StatusOK, "some data", nil)

		require.True(t, c.IsAborted(), "context is not aborted")
		_, ok := c.Get("uaf")
		require.False(t, ok, "user action fields value is set")
	})
}

func TestHTTPErrorWithUAFieldsSlice(t *testing.T) {
	t.Run("successful test", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		var err error
		c.Request, err = http.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte("{}")))
		require.NoError(t, err, "error in NewRequest")
		testUafs := []UserActionFields{testUserActionFields, testUserActionFields}
		HTTPErrorWithUAFieldsSlice(c, srverrors.ErrInternal, nil, testUafs)
		require.True(t, c.IsAborted(), "context is not aborted")
		uafint, ok := c.Get("uaf")
		require.True(t, ok, "user action fields value is not set")
		uaf, ok := uafint.([]UserActionFields)
		require.True(t, ok, "user action fields value in wrong format")
		testSuccessUafs := []UserActionFields{testUserActionFieldsFail, testUserActionFieldsFail}
		require.True(t, reflect.DeepEqual(uaf, testSuccessUafs), "wrong user action fields value, got %v, want %v", uaf, testUafs)
	})

}

func TestHTTPErrorWithUAFieldsSlice_withAbortedContext(t *testing.T) {

	t.Run("successful test with aborted context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Abort()
		testUafs := []UserActionFields{testUserActionFields, testUserActionFields}
		HTTPErrorWithUAFieldsSlice(c, srverrors.ErrInternal, nil, testUafs)
		_, ok := c.Get("uaf")
		require.False(t, ok, "user action fields value is set")
	})
}

func TestHTTPErrorWithUAFieldsSlice_withEmptyUafields(t *testing.T) {
	t.Run("successful test with empty uafields", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		var err error
		c.Request, err = http.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte("{}")))
		require.NoError(t, err, "error in NewRequest")
		HTTPErrorWithUAFieldsSlice(c, srverrors.ErrInternal, nil, nil)
		if !c.IsAborted() {
			t.Errorf("context is not aborted")
		}
		_, ok := c.Get("uaf")
		require.False(t, ok, "user action fields value is set")
	})

}
