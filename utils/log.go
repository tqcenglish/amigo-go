package utils

import "github.com/sirupsen/logrus"

//Log logrus instance
var Log *logrus.Entry

//NewLog new log
func NewLog() {
	Log = logrus.WithFields(
		logrus.Fields{
			// "libname": "ami-go",
		},
	)
	Log.Logger.SetReportCaller(true)
	Log.Logger.SetLevel(logrus.DebugLevel)
}
