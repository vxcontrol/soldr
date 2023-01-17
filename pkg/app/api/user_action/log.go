package useraction

import (
	"github.com/sirupsen/logrus"
)

type LogWriter struct{}

func NewLogWriter() *LogWriter {
	return &LogWriter{}
}

func (w *LogWriter) WriteUserAction(uaf Fields) error {
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
	}
	logrus.WithFields(fields).Info()
	return nil
}
