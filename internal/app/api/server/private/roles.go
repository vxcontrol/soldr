package private

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	"soldr/internal/app/api/models"
	srverrors "soldr/internal/app/api/server/errors"
	"soldr/internal/app/api/utils"
)

type roles struct {
	Roles []models.Role `json:"roles"`
	Total uint64        `json:"total"`
}

var rolesSQLMappers = map[string]interface{}{
	"id":   "`{{table}}`.id",
	"name": "`{{table}}`.name",
	"data": "`{{table}}`.name",
}

// GetRoles is a function to return roles list
// @Summary Retrieve roles list
// @Tags Roles
// @Produce json
// @Param request query utils.TableQuery true "query table params"
// @Success 200 {object} utils.successResp{data=roles} "roles list received successful"
// @Failure 400 {object} utils.errorResp "invalid query request data"
// @Failure 403 {object} utils.errorResp "getting roles not permitted"
// @Failure 500 {object} utils.errorResp "internal error on getting roles"
// @Router /roles/ [get]
func GetRoles(c *gin.Context) {
	var (
		err   error
		gDB   *gorm.DB
		query utils.TableQuery
		resp  roles
	)

	if err = c.ShouldBindQuery(&query); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding query")
		utils.HTTPError(c, srverrors.ErrRolesInvalidRequest, err)
		return
	}

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	query.Init("roles", rolesSQLMappers)

	if resp.Total, err = query.Query(gDB, &resp.Roles); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding roles")
		utils.HTTPError(c, srverrors.ErrInternal, err)
		return
	}

	for i := 0; i < len(resp.Roles); i++ {
		if err = resp.Roles[i].Valid(); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error validating role data '%d'", resp.Roles[i].ID)
			utils.HTTPError(c, srverrors.ErrRolesInvalidData, err)
			return
		}
	}

	utils.HTTPSuccess(c, http.StatusOK, resp)
}
