package events

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"soldr/internal/app/api/models"
	"soldr/internal/app/api/utils"
)

type EventName string

const (
	CreateAgentEvent EventName = "create-agent"
	UpdateAgentEvent EventName = "update-agent"
	DeleteAgentEvent EventName = "delete-agent"

	CreateGroupEvent EventName = "create-group"
	UpdateGroupEvent EventName = "update-group"
	DeleteGroupEvent EventName = "delete-group"

	CreatePolicyEvent EventName = "create-policy"
	UpdatePolicyEvent EventName = "update-policy"
	DeletePolicyEvent EventName = "delete-policy"

	CreateModuleEvent EventName = "create-module"
	UpdateModuleEvent EventName = "update-module"
	DeleteModuleEvent EventName = "delete-module"

	CreateGroupToPolicyEvent EventName = "create-group-to-policy"
	DeleteGroupToPolicyEvent EventName = "delete-group-to-policy"
)

type Event struct {
	Name    EventName   `json:"event"`
	Payload interface{} `json:"payload"`
}

// EventPoller poll all available events from database. Polled events then reported to exchanger.
type EventPoller struct {
	exchanger    *Exchanger
	tickInterval time.Duration

	lastPollAt time.Time

	services map[uint64]*service

	db *gorm.DB

	logger *logrus.Entry

	// list of service_id -> GroupToPolicy_id, useful for detect new GroupToPolicy records
	groupToPolicyLastID map[uint64]uint64
	// list of service_id -> module_id, useful for detect new module records
	moduleLastID map[uint64]uint64
}

// NewEventPoller create EventPoller.
func NewEventPoller(exchanger *Exchanger, tickInterval time.Duration, db *gorm.DB) *EventPoller {
	return &EventPoller{
		exchanger:           exchanger,
		tickInterval:        tickInterval,
		db:                  db,
		lastPollAt:          time.Now().UTC().Add(-tickInterval),
		services:            make(map[uint64]*service),
		groupToPolicyLastID: make(map[uint64]uint64),
		moduleLastID:        make(map[uint64]uint64),
		logger: logrus.WithFields(logrus.Fields{
			"component": "event_poller",
		}),
	}
}

// Run start polling loop.
func (ep *EventPoller) Run(ctx context.Context) error {
	ep.logger.Debugf("Start DB polling, interval: %s", ep.tickInterval.String())

	if ep.tickInterval <= 0 {
		return errors.New("expected event poller tick interval greater than 0")
	}

	ticker := time.NewTicker(ep.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now().UTC()
			from, to := ep.lastPollAt, now
			ep.lastPollAt = now

			ep.poll(from, to)
		case <-ctx.Done():
			return nil
		}
	}
}

func (ep *EventPoller) poll(from, to time.Time) {
	ep.services = loadServices(ep.db, ep.services)

	wg := &sync.WaitGroup{}
	for _, serv := range ep.services {
		wg.Add(1)
		go func(s *service) {
			defer wg.Done()

			newAgents, updatedAgents, deletedAgents := ep.pollAgents(s, from, to)
			agentCreateEvents := makeAgentEvents(CreateAgentEvent, newAgents)
			agentUpdateEvents := makeAgentEvents(UpdateAgentEvent, updatedAgents)
			agentDeleteEvents := makeAgentEvents(DeleteAgentEvent, deletedAgents)

			ep.exchanger.fireEvents(s.sv.ID, CreateAgentsChannel, agentCreateEvents...)
			ep.exchanger.fireEvents(s.sv.ID, UpdateAgentsChannel, agentUpdateEvents...)
			ep.exchanger.fireEvents(s.sv.ID, DeleteAgentsChannel, agentDeleteEvents...)

			newGroups, updatedGroups, deletedGroups := ep.pollGroups(s, from, to)
			groupCreateEvents := makeGroupEvents(CreateGroupEvent, newGroups)
			groupUpdateEvents := makeGroupEvents(UpdateGroupEvent, updatedGroups)
			groupDeleteEvents := makeGroupEvents(DeleteGroupEvent, deletedGroups)

			ep.exchanger.fireEvents(s.sv.ID, CreateGroupsChannel, groupCreateEvents...)
			ep.exchanger.fireEvents(s.sv.ID, UpdateGroupsChannel, groupUpdateEvents...)
			ep.exchanger.fireEvents(s.sv.ID, DeleteGroupsChannel, groupDeleteEvents...)

			newPolicies, updatedPolicies, deletedPolicies := ep.pollPolicies(s, from, to)
			policyCreateEvents := makePolicyEvents(CreatePolicyEvent, newPolicies)
			policyUpdateEvents := makePolicyEvents(UpdatePolicyEvent, updatedPolicies)
			policyDeleteEvents := makePolicyEvents(DeletePolicyEvent, deletedPolicies)

			ep.exchanger.fireEvents(s.sv.ID, CreatePoliciesChannel, policyCreateEvents...)
			ep.exchanger.fireEvents(s.sv.ID, UpdatePoliciesChannel, policyUpdateEvents...)
			ep.exchanger.fireEvents(s.sv.ID, DeletePoliciesChannel, policyDeleteEvents...)

			newGroupToPolicy := ep.pollGroupToPolicy(s)
			groupToPolicyCreateEvents := makeGroupToPoliciesEvents(CreateGroupToPolicyEvent, newGroupToPolicy)

			ep.exchanger.fireEvents(s.sv.ID, CreateGroupToPolicyChannel, groupToPolicyCreateEvents...)

			newModules, updatedModules, deletedModules := ep.pollModules(s, from, to)
			moduleCreateEvents := makeModuleEvents(CreateModuleEvent, newModules)
			moduleUpdateEvents := makeModuleEvents(UpdateModuleEvent, updatedModules)
			moduleDeleteEvents := makeModuleEvents(DeleteModuleEvent, deletedModules)

			ep.exchanger.fireEvents(s.sv.ID, CreateModulesChannel, moduleCreateEvents...)
			ep.exchanger.fireEvents(s.sv.ID, UpdateModulesChannel, moduleUpdateEvents...)
			ep.exchanger.fireEvents(s.sv.ID, DeleteModulesChannel, moduleDeleteEvents...)

			allEvents := flattenEvents(
				agentCreateEvents, agentUpdateEvents, agentDeleteEvents,
				groupCreateEvents, groupUpdateEvents, groupDeleteEvents,
				policyCreateEvents, policyUpdateEvents, policyDeleteEvents,
				groupToPolicyCreateEvents,
				moduleCreateEvents, moduleUpdateEvents, moduleDeleteEvents,
			)
			ep.exchanger.fireEvents(s.sv.ID, AllEventsChannel, allEvents...)
		}(serv)
	}

	wg.Wait()
}

func (ep *EventPoller) pollAgents(s *service, from, to time.Time) (created, updated, deleted []models.Agent) {
	if err := s.db.Find(&created, "created_date >= ? && created_date < ?", from, to).Error; err != nil {
		ep.logger.WithError(err).Error("poll new agents fail")
	}

	if err := s.db.Find(&updated, "updated_at >= ? && updated_at < ? && created_date < ?", from, to, from.Add(-time.Second)).Error; err != nil {
		ep.logger.WithError(err).Error("poll updated agents fail")
	}

	if err := s.db.Unscoped().Find(&deleted, "deleted_at >= ? && deleted_at < ?", from, to).Error; err != nil {
		ep.logger.WithError(err).Error("poll deleted agents fail")
	}

	return created, updated, deleted
}

func makeAgentEvents(event EventName, agents []models.Agent) []Event {
	result := make([]Event, len(agents))
	for i := range agents {
		result[i] = Event{Name: event, Payload: agents[i]}
	}

	return result
}

func (ep *EventPoller) pollGroups(s *service, from, to time.Time) (created, updated, deleted []models.Group) {
	if err := s.db.Find(&created, "created_date >= ? && created_date < ?", from, to).Error; err != nil {
		ep.logger.WithError(err).Error("poll new groups fail")
	}

	if err := s.db.Find(&updated, "updated_at >= ? && updated_at < ? && created_date < ?", from, to, from.Add(-time.Second)).Error; err != nil {
		ep.logger.WithError(err).Error("poll updated groups fail")
	}

	if err := s.db.Unscoped().Find(&deleted, "deleted_at >= ? && deleted_at < ?", from, to).Error; err != nil {
		ep.logger.WithError(err).Error("poll deleted groups fail")
	}

	return created, updated, deleted
}

func makeGroupEvents(event EventName, groups []models.Group) []Event {
	result := make([]Event, len(groups))
	for i := range groups {
		result[i] = Event{Name: event, Payload: groups[i]}
	}

	return result
}

func (ep *EventPoller) pollPolicies(s *service, from, to time.Time) (created, updated, deleted []models.Policy) {
	if err := s.db.Find(&created, "created_date >= ? && created_date < ?", from, to).Error; err != nil {
		ep.logger.WithError(err).Error("poll new policies fail")
	}

	if err := s.db.Find(&updated, "updated_at >= ? && updated_at < ? && created_date < ?", from, to, from.Add(-time.Second)).Error; err != nil {
		ep.logger.WithError(err).Error("poll updated policies fail")
	}

	if err := s.db.Unscoped().Find(&deleted, "deleted_at >= ? && deleted_at < ?", from, to).Error; err != nil {
		ep.logger.WithError(err).Error("poll deleted policies fail")
	}

	return created, updated, deleted
}

func makePolicyEvents(event EventName, policies []models.Policy) []Event {
	result := make([]Event, len(policies))
	for i := range policies {
		result[i] = Event{Name: event, Payload: policies[i]}
	}
	return result
}

func (ep *EventPoller) pollModules(s *service, from, to time.Time) (created, updated, deleted []models.ModuleA) {
	if _, exists := ep.moduleLastID[s.sv.ID]; !exists {
		var module models.ModuleA
		s.db.Order("id DESC").First(&module)
		ep.moduleLastID[s.sv.ID] = module.ID
	} else {
		if err := s.db.Order("id ASC").Find(&created, "id > ?", ep.moduleLastID[s.sv.ID]).Error; err != nil {
			ep.logger.WithError(err).Error("poll new modules fail")
		}
		if len(created) > 0 {
			ep.moduleLastID[s.sv.ID] = created[len(created)-1].ID
		}
	}

	createdIDs := make([]uint64, 0, len(created))
	for _, module := range created {
		createdIDs = append(createdIDs, module.ID)
	}

	if err := s.db.Not(createdIDs).Where("last_update >= ? && last_update < ?", from, to).Find(&updated).Error; err != nil {
		ep.logger.WithError(err).Error("poll updated modules fail")
	}

	if err := s.db.Unscoped().Find(&deleted, "deleted_at >= ? && deleted_at < ?", from, to).Error; err != nil {
		ep.logger.WithError(err).Error("poll deleted modules fail")
	}

	return created, updated, deleted
}

func makeModuleEvents(event EventName, modules []models.ModuleA) []Event {
	result := make([]Event, len(modules))
	for i := range modules {
		result[i] = Event{Name: event, Payload: modules[i]}
	}
	return result
}

func (ep *EventPoller) pollGroupToPolicy(s *service) (created []models.GroupToPolicy) {
	if _, exists := ep.groupToPolicyLastID[s.sv.ID]; !exists {
		var gtp models.GroupToPolicy
		s.db.Order("id DESC").First(&gtp)
		ep.groupToPolicyLastID[s.sv.ID] = gtp.ID
		return nil
	}

	if err := s.db.Order("id ASC").Find(&created, "id > ?", ep.groupToPolicyLastID[s.sv.ID]).Error; err != nil {
		ep.logger.WithError(err).Error("poll new group-to-policy fail")
	}

	if len(created) > 0 {
		ep.groupToPolicyLastID[s.sv.ID] = created[len(created)-1].ID
	}

	return created
}

func makeGroupToPoliciesEvents(event EventName, gtp []models.GroupToPolicy) []Event {
	result := make([]Event, len(gtp))
	for i := range gtp {
		result[i] = Event{Name: event, Payload: gtp[i]}
	}
	return result
}

func flattenEvents(events ...[]Event) []Event {
	var resultLen int
	for _, ev := range events {
		resultLen += len(ev)
	}

	result := make([]Event, 0, resultLen)
	for _, ev := range events {
		result = append(result, ev...)
	}

	return result
}

type service struct {
	db *gorm.DB
	sv *models.Service
}

func loadServices(gDB *gorm.DB, existingServices map[uint64]*service) map[uint64]*service {
	var svs []models.Service
	if err := gDB.Find(&svs).Error; err != nil {
		return existingServices
	}
	for _, sv := range svs {
		if _, ok := existingServices[sv.ID]; ok {
			continue
		}
		db := utils.GetDB(sv.Info.DB.User, sv.Info.DB.Pass, sv.Info.DB.Host,
			strconv.Itoa(int(sv.Info.DB.Port)), sv.Info.DB.Name)
		if db != nil {
			existingServices[sv.ID] = &service{
				db: db,
				sv: &sv,
			}
		}
	}
	return existingServices
}
