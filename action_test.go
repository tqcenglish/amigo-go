package amigo

import (
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

var a *Amigo

func TestMain(m *testing.M) {
	wait := make(chan struct{})
	go initAMI()
	time.AfterFunc(5*time.Second, func() {
		m.Run()
		wait <- struct{}{}
	})
	<-wait
}
func initAMI() {
	settings := &Settings{Host: "192.168.17.66", Port: "5038", Username: "openapi", Password: "e845116521d590069f285ddde46ee2cf", Report: true, LogLevel: logrus.DebugLevel}
	a = New(settings)
	a.EventOn(func(payload ...interface{}) {
		// log.Infof("ami event on %+v", payload[0])
	})
	a.ConnectOn(func(payload ...interface{}) {
		// log.Infof("ami connect on %s", payload[0].(string))
	})
	a.Connect()
}
func Test_SIPpeers(t *testing.T) {
	fmt.Printf("Test_SIPpeers\n")
	_, _, err := a.SIPpeers()
	if err != nil {
		t.Errorf("sip peers error %+v", err)
		return
	}
	// fmt.Printf("%+v %+v", res, events)
}
