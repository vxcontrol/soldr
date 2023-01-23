package private

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	"soldr/pkg/app/api/client"
	"soldr/pkg/app/api/logger"
	"soldr/pkg/app/api/server/response"
	"soldr/pkg/app/api/storage"
)

const maxJsonArrayTagsAmount int = 20

type tags struct {
	Tags  []string `json:"tags"`
	Total uint64   `json:"total"`
}

func getTagMappers(query *storage.TableQuery) (string, map[string]interface{}, error) {
	var table string
	var sqlMappers = map[string]interface{}{
		"id":   "`{{table}}`.id",
		"hash": "`{{table}}`.hash",
		"name": "JSON_UNQUOTE(JSON_EXTRACT(`{{table}}`.info, '$.name.{{lang}}'))",
		"tag":  "JSON_UNQUOTE(JSON_EXTRACT(`{{table}}`.info, CONCAT('$.tags[', n, ']')))",
	}

	for idx, filter := range query.Filters {
		if value, ok := filter.Value.(string); filter.Field == "type" && ok {
			table = value
			query.Filters = append(query.Filters[:idx], query.Filters[idx+1:]...)

			switch value {
			case "agents":
				sqlMappers["name"] = "`{{table}}`.description"
				sqlMappers["group_id"] = "`{{table}}`.group_id"
				sqlMappers["status"] = "`{{table}}`.status"
				sqlMappers["auth_status"] = "`{{table}}`.auth_status"
				sqlMappers["os"] = "CONCAT(`{{table}}`.os_type,':',`{{table}}`.os_arch)"
				sqlMappers["version"] = "`{{table}}`.version"
				sqlMappers["group_name"] = "JSON_UNQUOTE(JSON_EXTRACT(`groups`.info, '$.name.{{lang}}'))"
				sqlMappers["module_name"] = "`modules`.name"

			case "groups":
				sqlMappers["module_name"] = "`modules`.name"
				sqlMappers["policy_name"] = "JSON_UNQUOTE(JSON_EXTRACT(`policies`.info, '$.name.{{lang}}'))"
			case "policies":
				sqlMappers["group_name"] = "JSON_UNQUOTE(JSON_EXTRACT(`groups`.info, '$.name.{{lang}}'))"
				sqlMappers["module_name"] = "`modules`.name"
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

func getTagFilters(query *storage.TableQuery) []func(db *gorm.DB) *gorm.DB {
	return []func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.
				Select(query.Mappers()["tag"].(string) + " AS tag").
				Group(query.Mappers()["tag"].(string))
		},
		func(db *gorm.DB) *gorm.DB {
			return db.Joins("LEFT JOIN ? AS sq ON id = sq.sqid AND sq.js NOT LIKE '[]' AND NOT ISNULL(sq.js)",
				db.Table(query.Table()).
					Select("JSON_EXTRACT(info, CONCAT('$.tags')) AS js, id AS sqid").
					Group("js, sqid").
					SubQuery())
		},
		func(db *gorm.DB) *gorm.DB {
			unionList := "SELECT 0 n UNION ALL "
			for i := 1; i < maxJsonArrayTagsAmount-1; i++ {
				unionList += "SELECT " + strconv.Itoa(i) + " UNION ALL "
			}
			unionList += "SELECT " + strconv.Itoa(maxJsonArrayTagsAmount-1)
			return db.Joins(`INNER JOIN (` + unionList + `) num ON CHAR_LENGTH(sq.js)-CHAR_LENGTH(REPLACE(sq.js, ',', '')) >= num.n`)
		},
	}
}

func getTagJoinFuncs(table string, useGroup, useModule, usePolicy bool) []func(db *gorm.DB) *gorm.DB {
	if table == "agents" {
		return []func(db *gorm.DB) *gorm.DB{
			func(db *gorm.DB) *gorm.DB {
				if useGroup {
					db = db.Joins(`LEFT JOIN groups ON groups.id = agents.group_id AND groups.deleted_at IS NULL`)
				}
				if useModule {
					db = db.Joins(`LEFT JOIN groups_to_policies gtp ON gtp.group_id = agents.group_id`)
					db = db.Joins(`LEFT JOIN modules ON gtp.policy_id = modules.policy_id AND modules.status = 'joined' AND modules.deleted_at IS NULL`)
				}
				return db
			},
		}
	}
	if table == "groups" {
		return []func(db *gorm.DB) *gorm.DB{
			func(db *gorm.DB) *gorm.DB {
				if useModule || usePolicy {
					db = db.Joins(`LEFT JOIN groups_to_policies gtp ON gtp.group_id = groups.id`)
				}
				if useModule {
					db = db.Joins(`LEFT JOIN modules ON gtp.policy_id = modules.policy_id AND modules.status = 'joined' AND modules.deleted_at IS NULL`)
				}
				if usePolicy {
					db = db.Joins("LEFT JOIN policies ON gtp.policy_id = policies.id AND policies.deleted_at IS NULL")
				}
				return db
			},
		}
	}
	if table == "policies" {
		return []func(db *gorm.DB) *gorm.DB{
			func(db *gorm.DB) *gorm.DB {
				if useModule {
					db = db.Joins(`LEFT JOIN modules ON policies.id = modules.policy_id AND modules.status = 'joined' AND modules.deleted_at IS NULL`)
				}
				if useGroup {
					db = db.Joins(`LEFT JOIN groups_to_policies gtp ON gtp.policy_id = policies.id`)
					db = db.Joins(`LEFT JOIN groups ON groups.id = gtp.group_id AND groups.deleted_at IS NULL`)
				}
				return db
			},
		}
	}
	return []func(db *gorm.DB) *gorm.DB{}
}

type TagService struct {
	db              *gorm.DB
	serverConnector *client.AgentServerClient
}

func NewTagService(db *gorm.DB, serverConnector *client.AgentServerClient) *TagService {
	return &TagService{
		db:              db,
		serverConnector: serverConnector,
	}
}

// GetTags is a function to return tags list by type filter query key
// @Summary Retrieve tags list by filters
// @Tags Tags
// @Produce json
// @Param request query storage.TableQuery true "query table params"
// @Success 200 {object} response.successResp{data=tags} "tags list received successful"
// @Failure 400 {object} response.errorResp "invalid query request data"
// @Failure 403 {object} response.errorResp "getting tags not permitted"
// @Failure 500 {object} response.errorResp "internal error on getting tags"
// @Router /tags/ [get]
func (s *TagService) GetTags(c *gin.Context) {
	var (
		query      storage.TableQuery
		resp       tags
		sqlMappers map[string]interface{}
		table      string
		useModule  bool
		useGroup   bool
		usePolicy  bool
	)

	if err := c.ShouldBindQuery(&query); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error binding query")
		response.Error(c, response.ErrTagsInvalidRequest, err)
		return
	}

	serviceHash := c.GetString("svc")
	if serviceHash == "" {
		logger.FromContext(c).Errorf("could not get service hash")
		response.Error(c, response.ErrInternal, nil)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		logger.FromContext(c).WithError(err).Error()
		response.Error(c, response.ErrInternalDBNotFound, err)
		return
	}

	table, sqlMappers, err = getTagMappers(&query)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error getting tag mappers by query")
		response.Error(c, response.ErrTagsMappersNotFound, err)
		return
	}

	query.Init(table, sqlMappers)

	setUsingTables := func(sfield string) {
		if sfield == "module_name" {
			useModule = true
		}
		if sfield == "group_name" {
			useGroup = true
		}
		if sfield == "policy_name" {
			usePolicy = true
		}
	}
	setUsingTables(query.Sort.Prop)
	setUsingTables(query.Group)
	for _, filter := range query.Filters {
		setUsingTables(filter.Field)
	}
	query.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Where(table + ".deleted_at IS NULL")
		},
	})
	query.SetFind(func(out interface{}) func(*gorm.DB) *gorm.DB {
		return func(db *gorm.DB) *gorm.DB {
			return db.Pluck("tag", out)
		}
	})
	query.SetOrders([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Order("tag asc")
		},
	})

	funcs := getTagFilters(&query)
	funcs = append(funcs, getTagJoinFuncs(table, useGroup, useModule, usePolicy)...)
	if resp.Total, err = query.Query(iDB, &resp.Tags, funcs...); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding tags")
		response.Error(c, response.ErrTagsInvalidQuery, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}
