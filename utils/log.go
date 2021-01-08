package utils

import "github.com/sirupsen/logrus"

//Log logrus instance
var Log *logrus.Entry

//NewLog new log
func NewLog(level logrus.Level, report bool) {
	Log = logrus.WithFields(
		logrus.Fields{
			"libname": "ami-go",
		},
	)
	Log.Logger.SetReportCaller(report)
	Log.Logger.SetLevel(level)
}
