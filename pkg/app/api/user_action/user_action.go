package useraction

import (
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"soldr/pkg/app/api/utils"
)

type Action struct {
	Mtype             string `json:"type"`
	ID                string `json:"id"`
	Time              int64  `json:"time"`
	UtcOffset         string `json:"utcOffset"`
	UserID            string `json:"userId,omitempty"`
	UserDisplayName   string `json:"userDisplayName,omitempty"`
	ObjectDomain      string `json:"objectDomain,omitempty"`
	ObjectType        string `json:"objectType,omitempty"`
	ObjectID          string `json:"objectId,omitempty"`
	ObjectDisplayName string `json:"objectDisplayName,omitempty"`
	ActionCode        string `json:"actionCode,omitempty"`
	FailReason        string `json:"failReason,omitempty"`
}

type Converter interface {
	ToUserAction() Action
}

type BeginEvent struct {
	ID                uuid.UUID
	Time              int64
	UTCOffset         string
	UserID            string
	UserDisplayName   string
	ObjectDomain      string
	ObjectType        string
	ObjectID          string
	ObjectDisplayName string
	ActionCode        string
}

func (e BeginEvent) ToUserAction() Action {
	return Action{
		Mtype:             "UserActionBeginEvent",
		ID:                e.ID.String(),
		Time:              e.Time,
		UtcOffset:         e.UTCOffset,
		UserID:            e.UserID,
		UserDisplayName:   e.UserDisplayName,
		ObjectDomain:      e.ObjectDomain,
		ObjectType:        e.ObjectType,
		ObjectID:          e.ObjectID,
		ObjectDisplayName: e.ObjectDisplayName,
		ActionCode:        e.ActionCode,
	}
}

type FailEvent struct {
	ID         uuid.UUID
	Time       int64
	UTCOffset  string
	FailReason string
}

func (e FailEvent) ToUserAction() Action {
	return Action{
		Mtype:      "UserActionFailEvent",
		ID:         e.ID.String(),
		Time:       e.Time,
		UtcOffset:  e.UTCOffset,
		FailReason: e.FailReason,
	}
}

type SuccessEvent struct {
	ID                uuid.UUID
	Time              int64
	UTCOffset         string
	ObjectID          string
	ObjectDisplayName string
}

func (e SuccessEvent) ToUserAction() Action {
	return Action{
		Mtype:             "UserActionSuccessEvent",
		ID:                e.ID.String(),
		Time:              e.Time,
		UtcOffset:         e.UTCOffset,
		ObjectID:          e.ObjectID,
		ObjectDisplayName: e.ObjectDisplayName,
	}
}

func LogUserActions(actionCh chan Converter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.IsAborted() {
			return
		}

		tStart := time.Now()

		c.Next()

		fields, ok := c.Get("uaf")
		if !ok {
			// action doesn't need to be logged
			return
		}

		uafs, ok := fields.([]utils.UserActionFields)
		if !ok {
			utils.FromContext(c).Errorf("wrong fields format for action logging")
			return
		}
		for _, uaf := range uafs {
			if uaf.ActionCode == "" || uaf.Domain == "" || uaf.ObjectType == "" {
				utils.FromContext(c).Errorf("empty required field or wrong action code for action logging")
				return
			}
		}

		tFinish := time.Now()
		tstrarr := strings.Fields(tFinish.Format("01/02 03:04:05PM '06 -07:00"))

		session := sessions.Default(c)
		uid := session.Get("uuid")
		uuidstr, ok := uid.(string)
		if !ok {
			utils.FromContext(c).Errorf("error getting user id from session")
			return
		}

		userName := session.Get("uname")
		userNamestr, ok := userName.(string)
		if !ok {
			utils.FromContext(c).Errorf("error getting user name from session")
			return
		}

		for _, uaf := range uafs {
			eventId := uuid.New()

			beginAction := BeginEvent{
				ID:                eventId,
				Time:              tStart.Unix(),
				UTCOffset:         tstrarr[3],
				UserID:            uuidstr,
				UserDisplayName:   userNamestr,
				ObjectDomain:      uaf.Domain,
				ObjectType:        uaf.ObjectType,
				ObjectID:          uaf.ObjectId,
				ObjectDisplayName: uaf.ObjectDisplayName,
				ActionCode:        uaf.ActionCode,
			}
			actionCh <- beginAction

			if uaf.Success {
				successAction := SuccessEvent{
					ID:                eventId,
					Time:              tFinish.Unix(),
					UTCOffset:         tstrarr[3],
					ObjectID:          uaf.ObjectId,
					ObjectDisplayName: uaf.ObjectDisplayName,
				}
				actionCh <- successAction

			} else {
				failAction := FailEvent{
					ID:         eventId,
					Time:       tFinish.Unix(),
					UTCOffset:  tstrarr[3],
					FailReason: uaf.FailReason,
				}
				actionCh <- failAction
			}
		}
	}
}
