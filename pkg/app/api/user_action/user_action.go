package useraction

import (
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"soldr/pkg/app/api/logger"
)

const UnknownObjectDisplayName = "Undefined object"

type Writer interface {
	WriteUserAction(c *gin.Context, uaf Fields) error
}

type Fields struct {
	StartTime         time.Time
	UserName          string
	UserUUID          string
	Domain            string
	ObjectType        string
	ObjectID          string
	ObjectDisplayName string
	ActionCode        string
	Success           bool
	FailReason        string
}

func NewFields(c *gin.Context, domain, objectType, actionCode, objectID, objectDisplayName string) Fields {
	session := sessions.Default(c)
	uuid := (session.Get("uuid")).(string)
	userName := (session.Get("uname")).(string)
	return Fields{
		StartTime:         time.Now(),
		UserName:          userName,
		UserUUID:          uuid,
		Domain:            domain,
		ObjectType:        objectType,
		ObjectID:          objectID,
		ObjectDisplayName: objectDisplayName,
		ActionCode:        actionCode,
	}
}

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (w *Logger) WriteUserAction(c *gin.Context, uaf Fields) error {
	fields := logrus.Fields{
		"start_time":          uaf.StartTime,
		"user_name":           uaf.UserName,
		"user_uuid":           uaf.UserUUID,
		"domain":              uaf.Domain,
		"object_type":         uaf.ObjectType,
		"object_id":           uaf.ObjectID,
		"object_display_name": uaf.ObjectDisplayName,
		"action_code":         uaf.ActionCode,
		"success":             uaf.Success,
		"fail_reason":         uaf.FailReason,
		"component":           "user_action",
	}
	logger.FromContext(c).WithFields(fields).Info()
	return nil
}
