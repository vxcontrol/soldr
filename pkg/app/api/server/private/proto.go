package private

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"

	"soldr/pkg/app/api/logger"
	"soldr/pkg/app/api/models"
	"soldr/pkg/app/api/server/response"
	"soldr/pkg/app/api/storage"
)

func makeTokenClaims(c *gin.Context, cpt string) (*models.ProtoAuthTokenClaims, error) {
	rid := c.GetUint64("rid")
	if rid == 0 {
		return nil, fmt.Errorf("input RID invalid %d", rid)
	}

	sid := c.GetUint64("sid")
	if sid == 0 {
		return nil, fmt.Errorf("input SID invalid %d", sid)
	}

	tid := c.GetUint64("tid")
	if tid == 0 {
		return nil, fmt.Errorf("input TID invalid %d", tid)
	}

	uid := c.GetUint64("uid")
	if uid == 0 {
		return nil, fmt.Errorf("input UID invalid %d", uid)
	}

	return &models.ProtoAuthTokenClaims{
		RID: rid,
		SID: sid,
		TID: tid,
		UID: uid,
		CPT: cpt,
	}, nil
}

func MakeToken(c *gin.Context, req *models.ProtoAuthTokenRequest) (string, error) {
	claims, err := makeTokenClaims(c, req.Type)
	if err != nil {
		return "", fmt.Errorf("failed to get token claims: %w", err)
	}

	now := time.Now().Unix()
	claims.StandardClaims = jwt.StandardClaims{
		ExpiresAt: now + int64(req.TTL),
		IssuedAt:  now,
		Subject:   "vxproto",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(storage.MakeCookieStoreKey())
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenString, nil
}

func ValidateToken(tokenString string) (*models.ProtoAuthTokenClaims, error) {
	var claims models.ProtoAuthTokenClaims
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		return storage.MakeCookieStoreKey(), nil
	})

	if token != nil && token.Valid {
		return &claims, nil
	} else if ve, ok := err.(*jwt.ValidationError); ok {
		if ve.Errors&jwt.ValidationErrorMalformed != 0 {
			return nil, fmt.Errorf("token is malformed")
		} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
			return nil, fmt.Errorf("token is either expired or not active yet")
		} else {
			return nil, fmt.Errorf("token invalid: %w", err)
		}
	}
	return nil, fmt.Errorf("received data is not a token: %w", err)
}

// CreateAuthToken is a function to create new JWT token to authorize proto requests
// @Summary Create new JWT token to use it into vxproto connections
// @Tags Proto
// @Accept json
// @Produce json
// @Param json body models.ProtoAuthTokenRequest true "Proto auth token request JSON data"
// @Success 201 {object} response.successResp{data=models.ProtoAuthToken} "token created successful"
// @Failure 400 {object} response.errorResp "invalid requested token info"
// @Failure 403 {object} response.errorResp "creating token not permitted"
// @Failure 500 {object} response.errorResp "internal error on creating token"
// @Router /token/vxproto [post]
func CreateAuthToken(c *gin.Context) {
	var req models.ProtoAuthTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error binding JSON")
		response.Error(c, response.ErrProtoInvalidRequest, err)
		return
	} else if err := req.Valid(); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating JSON")
		response.Error(c, response.ErrProtoInvalidRequest, err)
		return
	}

	token, err := MakeToken(c, &req)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error on making token")
		response.Error(c, response.ErrProtoCreateTokenFail, err)
		return
	}
	if _, err := ValidateToken(token); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error on validating token")
		response.Error(c, response.ErrProtoInvalidToken, err)
		return
	}

	pat := models.ProtoAuthToken{
		Token:       token,
		TTL:         req.TTL,
		CreatedDate: time.Now(),
	}
	response.Success(c, http.StatusCreated, pat)
}
