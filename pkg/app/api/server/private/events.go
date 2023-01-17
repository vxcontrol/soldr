package private

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	"soldr/pkg/app/api/client"
	"soldr/pkg/app/api/models"
	srvcontext "soldr/pkg/app/api/server/context"
	"soldr/pkg/app/api/server/response"
	"soldr/pkg/app/api/utils"
)

type events struct {
	Events   []models.Event        `json:"events"`
	Agents   []models.Agent        `json:"agents"`
	Groups   []models.Group        `json:"groups"`
	Modules  []models.ModuleAShort `json:"modules"`
	Policies []models.Policy       `json:"policies"`
	Total    uint64                `json:"total"`
}

var eventsSQLMappers = map[string]interface{}{
	"id":                  "`{{table}}`.id",
	"date":                "`{{table}}`.date",
	"name":                "`{{table}}`.name",
	"data":                dataMapper,
	"agent_id":            "`agents`.id",
	"agent_hash":          "`agents`.hash",
	"agent_name":          "`agents`.description",
	"group_id":            "`groups`.id",
	"group_hash":          "`groups`.hash",
	"group_name":          "JSON_UNQUOTE(JSON_EXTRACT(`groups`.info, '$.name.{{lang}}'))",
	"module_id":           "`modules`.id",
	"module_name":         "`modules`.name",
	"policy_id":           "`modules`.policy_id",
	"policy_name":         "JSON_UNQUOTE(JSON_EXTRACT(`policies`.info, '$.name.{{lang}}'))",
	"policy_hash":         "`policies`.hash",
	"localizedDate":       "`{{table}}`.date",
	"localizedModuleName": "JSON_UNQUOTE(JSON_EXTRACT(`modules`.locale, '$.module.{{lang}}.title'))",
	"localizedEventName": "JSON_UNQUOTE(JSON_EXTRACT(`modules`.locale, " +
		"CONCAT('$.events.', `{{table}}`.name, '.{{lang}}.title')))",
	"localizedEventDescription": "JSON_UNQUOTE(JSON_EXTRACT(`modules`.locale, " +
		"CONCAT('$.events.', `{{table}}`.name, '.{{lang}}.description')))",
}

func dataMapper(q *utils.TableQuery, db *gorm.DB, value interface{}) *gorm.DB {
	return db.Where("`events`.data_text LIKE ? OR `events`.name LIKE ?", value, value)
}

func getAgentIDs(iDB *gorm.DB, query *utils.TableQuery, epids []uint64) ([]uint64, error) {
	var (
		eaids         []uint64
		eventsFilters []utils.TableFilter
		eventsSort    utils.TableSort
		aidsFilters   []utils.TableFilter
		aidsSort      utils.TableSort
	)

	midsKeys := []string{
		"agent_id",
		"agent_hash",
		"agent_name",
		"group_id",
		"group_hash",
		"group_name",
	}
	for _, filter := range query.Filters {
		if utils.StringInSlice(filter.Field, midsKeys) {
			aidsFilters = append(aidsFilters, filter)
		} else {
			eventsFilters = append(eventsFilters, filter)
		}
	}

	eventsSort = query.Sort
	if utils.StringInSlice(query.Sort.Prop, midsKeys) {
		aidsSort = query.Sort
	}

	query.Filters = aidsFilters
	query.Sort = aidsSort
	err := iDB.Table("agents").
		Joins("JOIN groups ON groups.id = agents.group_id AND groups.deleted_at IS NULL").
		Joins("JOIN groups_to_policies AS gtp ON groups.id = gtp.group_id").
		Scopes(query.DataFilter(), query.Ordering()).
		Where("gtp.policy_id IN (?) AND agents.deleted_at IS NULL", epids).
		Pluck("agents.id", &eaids).Error

	query.Filters = eventsFilters
	query.Sort = eventsSort

	return utils.UniqueUint64InSlice(eaids), err
}

func getModuleIDs(iDB *gorm.DB, query *utils.TableQuery) ([]uint64, []uint64, error) {
	var (
		emids         []uint64
		epids         []uint64
		eventsFilters []utils.TableFilter
		eventsSort    utils.TableSort
		midsFilters   []utils.TableFilter
		midsSort      utils.TableSort
	)

	midsKeys := []string{
		"module_id",
		"module_name",
		"policy_id",
		"policy_name",
		"policy_hash",
		"localizedModuleName",
	}
	for _, filter := range query.Filters {
		if utils.StringInSlice(filter.Field, midsKeys) {
			midsFilters = append(midsFilters, filter)
		} else {
			eventsFilters = append(eventsFilters, filter)
		}
	}

	eventsSort = query.Sort
	if utils.StringInSlice(query.Sort.Prop, midsKeys) {
		midsSort = query.Sort
	}

	query.Filters = midsFilters
	query.Sort = midsSort
	err := iDB.Table("modules").
		Joins("JOIN policies ON policies.id = modules.policy_id AND policies.deleted_at IS NULL").
		Scopes(query.DataFilter(), query.Ordering()).
		Where("`modules`.status = 'joined' AND modules.deleted_at IS NULL").
		Pluck("modules.id", &emids).
		Pluck("policies.id", &epids).Error

	query.Filters = eventsFilters
	query.Sort = eventsSort

	return utils.UniqueUint64InSlice(emids), utils.UniqueUint64InSlice(epids), err
}

func getFilters(query *utils.TableQuery) []func(db *gorm.DB) *gorm.DB {
	var (
		useAgent  bool
		useGroup  bool
		useModule bool
		usePolicy bool
	)

	agentKeys := []string{
		"group_id",
		"agent_hash",
		"agent_name",
	}
	groupKeys := []string{
		"group_hash",
		"group_name",
	}
	moduleKeys := []string{
		"policy_id",
		"module_name",
		"localizedModuleName",
		"localizedEventName",
		"localizedEventDescription",
	}
	policyKeys := []string{
		"policy_hash",
		"policy_name",
	}
	setUsingTables := func(sfield string) {
		if utils.StringInSlice(sfield, agentKeys) {
			useAgent = true
		}
		if utils.StringInSlice(sfield, groupKeys) {
			useGroup = true
		}
		if utils.StringInSlice(sfield, moduleKeys) {
			useModule = true
		}
		if utils.StringInSlice(sfield, policyKeys) {
			usePolicy = true
		}
	}
	setUsingTables(query.Sort.Prop)
	setUsingTables(query.Group)
	for _, filter := range query.Filters {
		if filter.Value == nil {
			continue
		}
		if v, ok := filter.Value.(string); ok && v == "" {
			continue
		}
		if v, ok := filter.Value.([]interface{}); ok && len(v) == 0 {
			continue
		}
		setUsingTables(filter.Field)
	}

	funcs := []func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			if useAgent || useGroup {
				db = db.Joins("JOIN agents ON agents.id = agent_id")
			}
			if useGroup {
				db = db.Joins("JOIN groups ON groups.id = agents.group_id")
			}
			if useModule || usePolicy {
				db = db.Joins("JOIN modules ON modules.id = module_id")
			}
			if usePolicy {
				db = db.Joins("JOIN policies ON modules.policy_id = policies.id")
			}
			return db
		},
	}

	return funcs
}

func doQuery(iDB *gorm.DB, query *utils.TableQuery, resp *events, funcs []func(db *gorm.DB) *gorm.DB) error {
	var (
		eids []uint64
		err  error
	)

	if query.Size < 0 || query.Size > 500 {
		resp.Total, err = query.Query(iDB, &resp.Events, funcs...)
		return err
	}

	query.SetFind(func(out interface{}) func(*gorm.DB) *gorm.DB {
		return func(db *gorm.DB) *gorm.DB {
			return db.Pluck("`events`.id", out)
		}
	})
	resp.Events = make([]models.Event, 0)
	if resp.Total, err = query.Query(iDB, &eids, funcs...); err != nil {
		return err
	}
	if len(eids) == 0 {
		return nil
	}

	eventsList := []*models.Event{}
	if err = iDB.Find(&eventsList, "id IN (?)", eids).Error; err != nil {
		return err
	}
	eventsMap := make(map[uint64]*models.Event, len(eids))
	for _, event := range eventsList {
		eventsMap[event.ID] = event
	}
	for _, eventID := range eids {
		resp.Events = append(resp.Events, *eventsMap[eventID])
	}

	return err
}

type EventService struct {
	serverConnector *client.AgentServerClient
}

func NewEventService(serverConnector *client.AgentServerClient) *EventService {
	return &EventService{
		serverConnector: serverConnector,
	}
}

// GetEvents is a function to return event list view on dashboard
// @Summary Retrieve events list by filters
// @Tags Events
// @Produce json
// @Param request query utils.TableQuery true "query table params"
// @Success 200 {object} utils.successResp{data=events} "events list received successful"
// @Failure 400 {object} utils.errorResp "invalid query request data"
// @Failure 403 {object} utils.errorResp "getting events not permitted"
// @Failure 500 {object} utils.errorResp "internal error on getting events"
// @Router /events/ [get]
func (s *EventService) GetEvents(c *gin.Context) {
	var (
		aids        []uint64
		eaids       []uint64
		emids       []uint64
		epids       []uint64
		gids        []uint64
		mids        []uint64
		pids        []uint64
		query       utils.TableQuery
		resp        events
		groupedResp utils.GroupedData
	)

	if err := c.ShouldBindQuery(&query); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding query")
		response.Error(c, response.ErrEventsInvalidRequest, err)
		return
	}

	serviceHash, ok := srvcontext.GetString(c, "svc")
	if !ok {
		utils.FromContext(c).Errorf("could not get service hash")
		response.Error(c, response.ErrInternal, nil)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		utils.FromContext(c).WithError(err).Error()
		response.Error(c, response.ErrInternalDBNotFound, err)
		return
	}

	if err = query.Init("events", eventsSQLMappers); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding query")
		response.Error(c, response.ErrEventsInvalidRequest, err)
		return
	}

	emids, epids, err = getModuleIDs(iDB, &query)
	if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error getting modules list by filter")
		response.Error(c, response.ErrEventsInvalidQuery, err)
		return
	}

	eaids, err = getAgentIDs(iDB, &query, epids)
	if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error getting agents list by filter")
		response.Error(c, response.ErrEventsInvalidQuery, err)
		return
	}

	copyEventsSQLMappers := make(map[string]interface{}, len(eventsSQLMappers))
	for key, value := range eventsSQLMappers {
		copyEventsSQLMappers[key] = value
	}

	copyEventsSQLMappers["agent_id"] = "`events`.agent_id"
	copyEventsSQLMappers["group_id"] = "`agents`.group_id"
	copyEventsSQLMappers["module_id"] = "`events`.module_id"
	copyEventsSQLMappers["policy_id"] = "`modules`.policy_id"
	if err = query.Init("events", copyEventsSQLMappers); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding query")
		response.Error(c, response.ErrEventsInvalidRequest, err)
		return
	}

	funcs := getFilters(&query)
	query.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Where("`events`.agent_id IN (?) AND `events`.module_id IN (?)", eaids, emids)
		},
	})

	if query.Group == "" {
		if err = doQuery(iDB, &query, &resp, funcs); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error finding events")
			response.Error(c, response.ErrEventsInvalidQuery, err)
			return
		}
	} else {
		if groupedResp.Total, err = query.QueryGrouped(iDB, &groupedResp.Grouped, funcs...); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error finding grouped events")
			response.Error(c, response.ErrEventsInvalidQuery, err)
			return
		}
		response.Success(c, http.StatusOK, groupedResp)
		return
	}

	for i := 0; i < len(resp.Events); i++ {
		aids = append(aids, resp.Events[i].AgentID)
		mids = append(mids, resp.Events[i].ModuleID)
		if resp.Events[i].Valid() != nil {
			utils.FromContext(c).WithError(err).Errorf("error validating event data")
			response.Error(c, response.ErrEventsInvalidData, err)
			return
		}
	}
	aids = utils.UniqueUint64InSlice(aids)
	if err = iDB.Where("id IN (?)", aids).Find(&resp.Agents).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding linked agents")
		response.Error(c, response.ErrEventsInvalidQuery, err)
		return
	}
	mids = utils.UniqueUint64InSlice(mids)
	if err = iDB.Where("id IN (?)", mids).Find(&resp.Modules).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding linked modules")
		response.Error(c, response.ErrEventsInvalidQuery, err)
		return
	}

	for i := 0; i < len(resp.Agents); i++ {
		gids = append(gids, resp.Agents[i].GroupID)
	}
	gids = utils.UniqueUint64InSlice(gids)
	if err = iDB.Where("id IN (?)", gids).Find(&resp.Groups).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding linked gloups")
		response.Error(c, response.ErrEventsInvalidQuery, err)
		return
	}

	for i := 0; i < len(resp.Modules); i++ {
		pids = append(pids, resp.Modules[i].PolicyID)
	}
	pids = utils.UniqueUint64InSlice(pids)
	if err = iDB.Where("id IN (?)", pids).Find(&resp.Policies).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding linked policies")
		response.Error(c, response.ErrEventsInvalidQuery, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}
