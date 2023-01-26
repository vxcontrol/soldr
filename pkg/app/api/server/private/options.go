package private

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	"soldr/pkg/app/api/logger"
	"soldr/pkg/app/api/models"
	"soldr/pkg/app/api/server/response"
	"soldr/pkg/app/api/storage"
)

const maxJsonArrayItemsAmount int = 50

type optionsActions struct {
	Actions []models.OptionsActions `json:"actions"`
	Total   uint64                  `json:"total"`
}

type optionsEvents struct {
	Events []models.OptionsEvents `json:"events"`
	Total  uint64                 `json:"total"`
}

type optionsFields struct {
	Fields []models.OptionsFields `json:"fields"`
	Total  uint64                 `json:"total"`
}

type optionsTags struct {
	Tags  []models.OptionsTags `json:"tags"`
	Total uint64               `json:"total"`
}

type optionsVersions struct {
	Versions []models.OptionsVersions `json:"versions"`
	Total    uint64                   `json:"total"`
}

func getOptionsMappers(option string) map[string]interface{} {
	var sqlMappers = map[string]interface{}{
		"name":                 "`list`.name",
		"module_name":          "`{{table}}`.name",
		"module_os":            storage.OptionsOSMapper,
		"module_os_arch":       storage.OptionsOSArchMapper,
		"module_os_type":       storage.OptionsOSTypeMapper,
		"version":              "`{{table}}`.version",
		"localizedName":        "JSON_UNQUOTE(JSON_EXTRACT(`{{table}}`.locale, '$." + option + "s.*.{{lang}}.title'))",
		"localizedDescription": "JSON_UNQUOTE(JSON_EXTRACT(`{{table}}`.locale, '$." + option + "s.*.{{lang}}.description'))",
		"data": "CONCAT(`{{table}}`.name, ' | ', `{{table}}`." + option + "s, ' | ', " +
			"COALESCE(JSON_EXTRACT(`{{table}}`.locale, '$." + option + "s.*.{{lang}}.title'), ''), ' | ', " +
			"COALESCE(JSON_EXTRACT(`{{table}}`.locale, '$." + option + "s.*.{{lang}}.title'), ''), ' | ', " +
			"COALESCE(JSON_EXTRACT(`{{table}}`.locale, '$.module.{{lang}}.title'), ''), ' | ', " +
			"COALESCE(JSON_EXTRACT(`{{table}}`.locale, '$.module.{{lang}}.description'), ''))",
	}

	switch option {
	case "action":
		fallthrough
	case "event":
		sqlMappers["fields"] = storage.OptionsFieldsMapper
	}

	return sqlMappers
}

func getBaseQueryForList(db *gorm.DB, option string) *gorm.SqlExpr {
	optionName := option + "s"
	groupCollumns := []string{"num.n"}
	selectCollumns := []string{}
	collumns := map[string]string{
		"module_name":    "name",
		"module_version": "version",
		"module_os":      "JSON_EXTRACT(info, '$.os')",
	}
	collumns["name"] = "JSON_UNQUOTE(JSON_EXTRACT(" + optionName + ", CONCAT('$[', n, ']')))"
	collumns["locale"] = "JSON_EXTRACT(locale, CONCAT('$." + optionName + ".\"'," + collumns["name"] + ",'\"'))"
	if option == "action" || option == "event" {
		collumns["config"] = "JSON_EXTRACT(default_" + option + "_config, CONCAT('$.\"'," + collumns["name"] + ",'\"'))"
	}

	for name, filter := range collumns {
		groupCollumns = append(groupCollumns, name)
		selectCollumns = append(selectCollumns, filter+" AS "+name)
	}

	unionList := "SELECT 0 n UNION ALL "
	for i := 1; i < maxJsonArrayItemsAmount-1; i++ {
		unionList += "SELECT " + strconv.Itoa(i) + " UNION ALL "
	}
	unionList += "SELECT " + strconv.Itoa(maxJsonArrayItemsAmount-1)

	return db.Model(&models.ModuleS{}).
		Select(selectCollumns).
		Group(strings.Join(groupCollumns, ", ")).
		Order("name asc").
		Joins("LEFT JOIN ? AS sq ON id = sq.sqid AND sq.js NOT LIKE '[]' AND NOT ISNULL(sq.js)",
			db.Model(&models.ModuleS{}).
				Select(optionName+" AS js, id AS sqid").
				SubQuery()).
		Joins(`INNER JOIN (` + unionList + `) num ON CHAR_LENGTH(sq.js)-CHAR_LENGTH(REPLACE(sq.js, ',', '')) >= num.n`).
		SubQuery()
}

func getBaseQueryForItem(db *gorm.DB, option string) *gorm.SqlExpr {
	groupCollumns := []string{}
	selectCollumns := []string{}
	collumns := map[string]string{
		"name":           option,
		"module_name":    "name",
		"module_version": "version",
		"module_os":      "JSON_EXTRACT(info, '$.os')",
	}

	for name, filter := range collumns {
		groupCollumns = append(groupCollumns, name)
		selectCollumns = append(selectCollumns, filter+" AS "+name)
	}

	return db.Model(&models.ModuleS{}).
		Select(selectCollumns).
		Group(strings.Join(groupCollumns, ", ")).
		Order(option + " asc").
		SubQuery()
}

func validOptions(c *gin.Context, value interface{}) bool {
	var vlist []models.IValid
	switch tvalue := value.(type) {
	case *[]models.OptionsActions:
		for _, v := range *tvalue {
			vlist = append(vlist, v)
		}
	case *[]models.OptionsEvents:
		for _, v := range *tvalue {
			vlist = append(vlist, v)
		}
	case *[]models.OptionsFields:
		for _, v := range *tvalue {
			vlist = append(vlist, v)
		}
	case *[]models.OptionsTags:
		for _, v := range *tvalue {
			vlist = append(vlist, v)
		}
	case *[]models.OptionsVersions:
		for _, v := range *tvalue {
			vlist = append(vlist, v)
		}
	}
	for i := 0; i < len(vlist); i++ {
		if err := vlist[i].Valid(); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error validating response data")
			return false
		}
	}

	return true
}

func getOption(c *gin.Context, db *gorm.DB, option string, value interface{}) (uint64, *response.HttpError) {
	var (
		err   error
		query storage.TableQuery
		sv    *models.Service
		total uint64
	)

	if err = c.ShouldBindQuery(&query); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error binding query")
		return 0, response.ErrOptionsInvalidRequestData
	}

	if sv = getService(c); sv == nil {
		return 0, response.ErrInternalServiceNotFound
	}

	tid := c.GetUint64("tid")

	query.Init("modules", getOptionsMappers(option))
	query.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Where("tenant_id IN (0, ?) AND service_type = ?", tid, sv.Type)
		},
	})
	query.SetOrders([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Order("name asc")
		},
	})
	funcs := []func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			var subQuery *gorm.SqlExpr
			cond_name := gorm.Expr("`modules`.name LIKE `list`.module_name")
			cond_version := gorm.Expr("`modules`.version LIKE `list`.module_version")
			if option == "version" {
				subQuery = getBaseQueryForItem(db, option)
			} else {
				subQuery = getBaseQueryForList(db, option)
				db = LatestModulesQuery(db)
			}
			return db.Select("`list`.*").
				Joins("INNER JOIN ? AS list ON ? AND ?", subQuery, cond_name, cond_version)
		},
	}
	if total, err = query.Query(db, value, funcs...); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding global %s list", option)
		return 0, response.ErrOptionsInvalidQuery
	}

	if !validOptions(c, value) {
		return 0, response.ErrOptionsInvalidData
	}

	return total, nil
}

type OptionService struct {
	db *gorm.DB
}

func NewOptionService(db *gorm.DB) *OptionService {
	return &OptionService{db: db}
}

// GetOptionsActions is a function to return global action list
// @Summary Retrieve global action list by filters
// @Tags Modules,Options
// @Produce json
// @Param request query storage.TableQuery true "query table params"
// @Success 200 {object} response.successResp{data=optionsActions} "global action list received successful"
// @Failure 400 {object} response.errorResp "invalid query request data"
// @Failure 403 {object} response.errorResp "getting global action list not permitted"
// @Failure 500 {object} response.errorResp "internal error on getting global action list"
// @Router /options/actions [get]
func (s *OptionService) GetOptionsActions(c *gin.Context) {
	var resp optionsActions
	ext, err := getOption(c, s.db, "action", &resp.Actions)
	if err != nil {
		response.Error(c, err, nil)
	}

	resp.Total = ext
	response.Success(c, http.StatusOK, resp)
}

// GetOptionsEvents is a function to return global event list
// @Summary Retrieve global event list by filters
// @Tags Modules,Options
// @Produce json
// @Param request query storage.TableQuery true "query table params"
// @Success 200 {object} response.successResp{data=optionsEvents} "global event list received successful"
// @Failure 400 {object} response.errorResp "invalid query request data"
// @Failure 403 {object} response.errorResp "getting global event list not permitted"
// @Failure 500 {object} response.errorResp "internal error on getting global event list"
// @Router /options/events [get]
func (s *OptionService) GetOptionsEvents(c *gin.Context) {
	var resp optionsEvents
	ext, err := getOption(c, s.db, "event", &resp.Events)
	if err != nil {
		response.Error(c, err, nil)
	}

	resp.Total = ext
	response.Success(c, http.StatusOK, resp)
}

// GetOptionsFields is a function to return global field list
// @Summary Retrieve global field list by filters
// @Tags Modules,Options
// @Produce json
// @Param request query storage.TableQuery true "query table params"
// @Success 200 {object} response.successResp{data=optionsFields} "global field list received successful"
// @Failure 400 {object} response.errorResp "invalid query request data"
// @Failure 403 {object} response.errorResp "getting global field list not permitted"
// @Failure 500 {object} response.errorResp "internal error on getting global field list"
// @Router /options/fields [get]
func (s *OptionService) GetOptionsFields(c *gin.Context) {
	var resp optionsFields
	ext, err := getOption(c, s.db, "field", &resp.Fields)
	if err != nil {
		response.Error(c, err, nil)
	}

	resp.Total = ext
	response.Success(c, http.StatusOK, resp)
}

// GetOptionsTags is a function to return global tag list
// @Summary Retrieve global tag list by filters
// @Tags Modules,Options
// @Produce json
// @Param request query storage.TableQuery true "query table params"
// @Success 200 {object} response.successResp{data=optionsTags} "global tag list received successful"
// @Failure 400 {object} response.errorResp "invalid query request data"
// @Failure 403 {object} response.errorResp "getting global tag list not permitted"
// @Failure 500 {object} response.errorResp "internal error on getting global tag list"
// @Router /options/tags [get]
func (s *OptionService) GetOptionsTags(c *gin.Context) {
	var resp optionsTags
	ext, err := getOption(c, s.db, "tag", &resp.Tags)
	if err != nil {
		response.Error(c, err, nil)
	}

	resp.Total = ext
	response.Success(c, http.StatusOK, resp)
}

// GetOptionsVersions is a function to return global version list
// @Summary Retrieve global version list by filters
// @Tags Modules,Options
// @Produce json
// @Param request query storage.TableQuery true "query table params"
// @Success 200 {object} response.successResp{data=optionsVersions} "global version list received successful"
// @Failure 400 {object} response.errorResp "invalid query request data"
// @Failure 403 {object} response.errorResp "getting global version list not permitted"
// @Failure 500 {object} response.errorResp "internal error on getting global version list"
// @Router /options/versions [get]
func (s *OptionService) GetOptionsVersions(c *gin.Context) {
	var resp optionsVersions
	ext, err := getOption(c, s.db, "version", &resp.Versions)
	if err != nil {
		response.Error(c, err, nil)
	}

	resp.Total = ext
	response.Success(c, http.StatusOK, resp)
}
