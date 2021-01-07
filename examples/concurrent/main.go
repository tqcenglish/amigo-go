package main

import (
	"time"

	log "github.com/sirupsen/logrus"
	amigo "github.com/tqcenglish/ami-go"
)

var a *amigo.Amigo

func amiTest() {
	settings := &amigo.Settings{Host: "192.168.17.66", Port: "5038", Username: "openapi", Password: "e845116521d590069f285ddde46ee2cf"}
	a = amigo.New(settings)
	a.EventOn(func(payload ...interface{}) {
		log.Infof("ami event on %+v", payload[0])
	})
	a.ConnectOn(func(payload ...interface{}) {
		log.Infof("ami connect on %s", payload[0].(string))
	})
	a.Connect()

	// 每 10s 运行
	go func() {
		count := 0
		for {
			now := time.Now()
			// 计算下一个零点
			next := now.Add(time.Second * 6)
			t := time.NewTimer(next.Sub(now))
			<-t.C

			result, events, err := a.Send(map[string]string{"Action": "SIPpeers"})
			if err != nil {
				log.Error(err)
			} else {
				log.Infof("*******************%d**************************", count)
				log.Infof("SIPpeers res %+v", result)
				log.Infof("SIPpeers events %d", len(events))
				for _, event := range events {
					if event.Data["Event"] == "PeerEntry" {
						log.Infof("SIPpeers ObjectName %s", event.Data["ObjectName"])
					}
				}
				log.Infof("*********************************************")
			}
			count = count + 1
		}
	}()

}

func main() {

	go amiTest()

	// ch := make(chan int)
	// <-ch
	select {}

}
