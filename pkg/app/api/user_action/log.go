package useraction

import (
	"github.com/sirupsen/logrus"

	"soldr/pkg/log"
)

type LogWriter struct {
	logger log.Logger
}

func NewLogWriter(logger log.Logger) *LogWriter {
	return &LogWriter{logger: logger}
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
	w.logger.WithFields(fields).Info()
	return nil
}
