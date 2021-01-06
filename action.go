package amigo

import (
	"errors"
	"time"

	"github.com/tqcenglish/ami-go/utils"
)

func (a *amiAdapter) pinger(stop <-chan struct{}, errChan chan error) {
	ticker := time.NewTicker(utils.PingInterval)
	defer ticker.Stop()
	ping := map[string]string{"Action": "Ping", "ActionID": utils.PingActionID, utils.AmigoConnIDKey: a.id}
	for {
		select {
		case <-stop:
			return
		default:
		}

		select {
		case <-stop:
			return
		case <-ticker.C:
		}

		if !a.online() {
			// when stop chan didn't received before ticker
			return
		}

		a.actionsChan <- ping
		timer := time.NewTimer(3 * time.Second)
		select {
		case <-a.pingerChan:
			timer.Stop()
			continue
		case <-timer.C:
			errChan <- errors.New("ping timeout")
			return
		}
	}
}

func (a *Amigo) login() error {
	var action = map[string]string{
		"Action":   "Login",
		"Username": a.ami.username,
		"Secret":   a.ami.password,
	}

	if data, _, err := a.Send(action); err != nil {
		return err
	} else if data["Response"] != "Success" && data["Message"] != "Authentication accepted" {
		utils.Log.Errorf("ami login failure by username:%s password:%s", a.ami.username, a.ami.password)
		return errors.New(data["Message"])
	}
	return nil
}
