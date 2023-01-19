package private

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"soldr/pkg/app/api/models"
	"soldr/pkg/app/api/server/response"
	"soldr/pkg/app/api/utils"
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

type RoleService struct {
	db *gorm.DB
}

func NewRoleService(db *gorm.DB) *RoleService {
	return &RoleService{
		db: db,
	}
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
func (s *RoleService) GetRoles(c *gin.Context) {
	var (
		err   error
		query utils.TableQuery
		resp  roles
	)

	if err = c.ShouldBindQuery(&query); err != nil {
		logrus.WithContext(c).WithError(err).Errorf("error binding query")
		response.Error(c, response.ErrRolesInvalidRequest, err)
		return
	}

	query.Init("roles", rolesSQLMappers)

	if resp.Total, err = query.Query(s.db, &resp.Roles); err != nil {
		logrus.WithContext(c).WithError(err).Errorf("error finding roles")
		response.Error(c, response.ErrInternal, err)
		return
	}

	for i := 0; i < len(resp.Roles); i++ {
		if err = resp.Roles[i].Valid(); err != nil {
			logrus.WithContext(c).WithError(err).Errorf("error validating role data '%d'", resp.Roles[i].ID)
			response.Error(c, response.ErrRolesInvalidData, err)
			return
		}
	}

	response.Success(c, http.StatusOK, resp)
}
