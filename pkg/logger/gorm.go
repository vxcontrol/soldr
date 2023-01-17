package logger

import "github.com/sirupsen/logrus"

type GormLogger struct{}

func (*GormLogger) Print(v ...interface{}) {
	switch v[0] {
	case "sql":
		logrus.WithFields(
			logrus.Fields{
				"module":        "gorm",
				"type":          "sql",
				"rows_returned": v[5],
				"src":           v[1],
				"values":        v[4],
				"duration":      v[2],
			},
		).Info(v[3])
	case "log":
		logrus.WithFields(logrus.Fields{"module": "gorm"}).Info(v[2])
	case "info":
		// do not log validators
	}
}
