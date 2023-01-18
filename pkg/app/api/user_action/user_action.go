package useraction

import (
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const UnknownObjectDisplayName = "Undefined object"

type Writer interface {
	WriteUserAction(uaf Fields) error
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
