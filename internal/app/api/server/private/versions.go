package private

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	"soldr/internal/app/api/client"
	srvcontext "soldr/internal/app/api/server/context"
	srverrors "soldr/internal/app/api/server/response"
	"soldr/internal/app/api/utils"
)

type versions struct {
	Versions []string `json:"versions"`
	Total    uint64   `json:"total"`
}

func getVersionMappers(query *utils.TableQuery) (string, map[string]interface{}, error) {
	var table string
	var sqlMappers = map[string]interface{}{
		"id":      "`{{table}}`.id",
		"hash":    "`{{table}}`.hash",
		"version": "`{{table}}`.version",
	}

	for idx, filter := range query.Filters {
		if value, ok := filter.Value.(string); filter.Field == "type" && ok {
			table = value
			query.Filters = append(query.Filters[:idx], query.Filters[idx+1:]...)

			switch value {
			case "agents":
				sqlMappers["name"] = "`{{table}}`.description"
				sqlMappers["group_id"] = "`{{table}}`.group_id"
			case "modules":
				sqlMappers["name"] = "`{{table}}`.name"
				sqlMappers["policy_id"] = "`{{table}}`.policy_id"
				delete(sqlMappers, "hash")
			default:
				return "", nil, errors.New("unknown model type")
			}
		}
	}
	if table == "" {
		return "", nil, errors.New("model type not found in filters")
	}

	return table, sqlMappers, nil
}

type VersionService struct {
	db              *gorm.DB
	serverConnector *client.AgentServerClient
}

func NewVersionService(db *gorm.DB, serverConnector *client.AgentServerClient) *VersionService {
	return &VersionService{
		db:              db,
		serverConnector: serverConnector,
	}
}

// GetVersions is a function to return versions list by type filter query key
// @Summary Retrieve versions list by filters
// @Tags Versions
// @Produce json
// @Param request query utils.TableQuery true "query table params"
// @Success 200 {object} utils.successResp{data=versions} "versions list received successful"
// @Failure 400 {object} utils.errorResp "invalid query request data"
// @Failure 403 {object} utils.errorResp "getting versions not permitted"
// @Failure 500 {object} utils.errorResp "internal error on getting versions"
// @Router /versions/ [get]
func (s *VersionService) GetVersions(c *gin.Context) {
	var (
		query      utils.TableQuery
		resp       versions
		sqlMappers map[string]interface{}
		table      string
	)

	if err := c.ShouldBindQuery(&query); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding query")
		utils.HTTPError(c, srverrors.ErrVersionsInvalidRequest, err)
		return
	}

	serviceHash, ok := srvcontext.GetString(c, "svc")
	if !ok {
		utils.FromContext(c).Errorf("could not get service hash")
		utils.HTTPError(c, srverrors.ErrInternal, nil)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		utils.FromContext(c).WithError(err).Error()
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, err)
		return
	}

	table, sqlMappers, err = getVersionMappers(&query)
	if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error getting version mappers by query")
		utils.HTTPError(c, srverrors.ErrVersionsMapperNotFound, err)
		return
	}

	query.Init(table, sqlMappers)
	query.SetFind(func(out interface{}) func(*gorm.DB) *gorm.DB {
		return func(db *gorm.DB) *gorm.DB {
			return db.Pluck("version", out)
		}
	})
	query.SetOrders([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Order("version asc")
		},
	})
	query.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL")
		},
	})
	funcs := []func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Group("version")
		},
	}

	if resp.Total, err = query.Query(iDB, &resp.Versions, funcs...); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding versions")
		utils.HTTPError(c, srverrors.ErrVersionsInvalidQuery, err)
		return
	}

	utils.HTTPSuccess(c, http.StatusOK, resp)
}
