package main

import (
	"time"

	log "github.com/sirupsen/logrus"
	amigo "github.com/tqcenglish/amigo-go"
	"github.com/tqcenglish/amigo-go/pkg"
	"github.com/tqcenglish/amigo-go/utils"
)

var a *amigo.Amigo

func amiTest() {
	start := make(chan bool, 1)
	settings := &amigo.Settings{
		Host:     "127.0.0.1",
		Port:     "5038",
		Username: "admin",
		Password: "admin",
		LogLevel: log.WarnLevel}
	a = amigo.New(settings, nil)
	log.SetLevel(log.InfoLevel)
	// a.EventOn(func(payload ...interface{}) {
	// 	log.Infof("Event on %+v", payload[0])
	// })
	a.ConnectOn(func(payload ...interface{}) {
		status := payload[0].(pkg.ConnectStatus)
		if status == pkg.Connect_OK {
			start <- true
			log.Infof("连接成功")
		}
	})
	a.Connect()

	<-start

	// 每 60s 运行
	go func() {
		count := 0
		for {
			now := time.Now()
			// 计算下一个零点
			next := now.Add(time.Second * 60)
			t := time.NewTimer(next.Sub(now))
			<-t.C

			result, events, err := a.Send(map[string]string{"Action": "PJSIPShowContacts"})
			if err != nil {
				log.Error(err)
			} else {
				log.Infof("******************* 当前运行次数 %d**************************", count)
				log.Infof("PJSIPShowContacts res %+v", result)
				log.Infof("PJSIPShowContacts events %d %+v", len(events), events)
				log.Infof("************************************************************")
			}
			count = count + 1
		}
	}()

}

func main() {
	go amiTest()
	go utils.HTTPServerPProf()
	select {}
}
