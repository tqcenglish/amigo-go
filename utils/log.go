package utils

import "github.com/sirupsen/logrus"

//Log logrus instance
var Log *logrus.Entry

//NewLog new log
func NewLog(level logrus.Level, report bool) {
	Log = logrus.New().WithFields(
		logrus.Fields{
			"libname": "ami-go",
		},
	)
	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
	}
	Log.Logger.SetFormatter(formatter)
	Log.Logger.SetReportCaller(report)
	Log.Logger.SetLevel(level)
}
