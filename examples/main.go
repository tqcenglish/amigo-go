package main

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	amigo "github.com/tqcenglish/amigo-go"
	"github.com/tqcenglish/amigo-go/pkg"
)

var a *amigo.Amigo

func amiTest() {
	start := make(chan bool, 1)
	settings := &amigo.Settings{
		Host:     "192.168.17.66",
		Port:     "5038",
		Username: "openapi",
		Password: "e845116521d590069f285ddde46ee2cf",
		LogLevel: log.WarnLevel}
	a = amigo.New(settings)
	log.SetLevel(log.InfoLevel)
	a.EventOn(func(payload ...interface{}) {
		log.Infof("Event on %+v", payload[0])
	})
	a.ConnectOn(func(payload ...interface{}) {
		status := payload[0].(pkg.ConnectStatus)
		if(status == pkg.Connect_OK){
			start <- true
		}
	})
	a.Connect()

	fmt.Print("wait start start\n")
	<- start
	fmt.Print("wait start end\n")

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
			// res, events, err := a.SIPpeers()
			// if err != nil {
			// 	utils.Log.Errorf("sip peers error %+v", err)
			// } else {
			// 	utils.Log.Infof("sip res %+v", res)
			// 	for _, v := range events {
			// 		utils.Log.Infof("sip event %+v", v)
			// 	}
			// }
			count = count + 1
		}
	}()

}

func main() {

	go amiTest()

	select {}

}
