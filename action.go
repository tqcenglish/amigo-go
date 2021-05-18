package amigo

import (
	"errors"
	"time"

	"github.com/tqcenglish/amigo-go/utils"
)

func (a *amiAdapter) pinger(stop <-chan struct{}, errChan chan error) {
	ticker := time.NewTicker(utils.PingInterval)
	defer ticker.Stop()
	ping := map[string]string{
		"Action": "Ping",
	}
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
		}

		if _, _, err := a.amigo.Send(ping); err != nil {
			errChan <- errors.New("ping timeout")
		}
	}
}

func (a *amiAdapter) login() error {
	var action = map[string]string{
		"Action":   "Login",
		"Username": a.username,
		"Secret":   a.password,
	}

	utils.Log.Infof("ami login action: %+v", action)
	if data, _, err := a.amigo.Send(action); err != nil {
		return err
	} else if data["Response"] != "Success" && data["Message"] != "Authentication accepted" {
		utils.Log.Errorf("ami login failure by username:%s password:%s", a.username, a.password)
		return errors.New(data["Message"])
	}
	return nil
}

//SIPPeersRes respons
// Response: Success
// ActionID: 043a71d0-4ee1-419f-9b0b-70751d6274c3
// EventList: start
// Message: Peer status list will follow
type SIPPeersRes struct {
	Response  string `json:"response"`
	ActionID  string `json:"action_id"`
	EventList string `json:"event_list"`
	Message   string `json:"message"`
}

//SIPpeersEvent event
// Event: PeerEntry
// ActionID: 043a71d0-4ee1-419f-9b0b-70751d6274c3
// Channeltype: SIP
// ObjectName: 701
// ChanObjectType: peer
// IPaddress: 192.168.17.77
// IPport: 5160
// Dynamic: yes
// AutoForcerport: no
// Forcerport: no
// AutoComedia: no
// Comedia: no
// VideoSupport: no
// TextSupport: no
// ACL: yes
// Status: OK (7 ms)
// RealtimeDevice: no
// Description:
// Accountcode:
type SIPpeersEvent struct {
	Event          string `json:"event"`
	ActionID       string `json:"action_id"`
	Channeltype    string `json:"channel_type"`
	ObjectName     string `json:"object_name"`
	ChanObjectType string `json:"chan_object_type"`
	IPaddress      string `json:"ip_address"`
	IPport         string `json:"ip_port"`
	Status         string `json:"status"`
	ACL            string `json:"acl"`
}

//SIPpeers sip show peers
func (a *Amigo) SIPpeers() (response *SIPPeersRes, events []*SIPpeersEvent, err error) {
	var action = map[string]string{
		"Action": "SIPpeers",
	}
	responseMap, eventsMapArray, err := a.Send(action)

	if err != nil {
		return &SIPPeersRes{}, nil, err
	}

	response = &SIPPeersRes{}
	for k, v := range responseMap {
		if err := utils.SetField(response, k, v); err != nil {
			utils.Log.Errorf("Response SetField error %+v", err)
		}
	}

	events = make([]*SIPpeersEvent, 0)
	for _, eventMap := range eventsMapArray {
		event := &SIPpeersEvent{}
		for k, v := range eventMap.Data {
			if err := utils.SetField(event, k, v); err != nil {
				utils.Log.Errorf("Event SetField error %+v", err)
				continue
			}
		}
		events = append(events, event)
	}
	return response, events, nil
}
