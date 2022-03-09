package amigo

import (
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tqcenglish/amigo-go/pkg"
	"github.com/tqcenglish/amigo-go/pkg/parse"
	"github.com/tqcenglish/amigo-go/utils"
)

// Amigo is a main package struct
type Amigo struct {
	settings *Settings
	ami      *amiAdapter

	eventEmitter pkg.EventEmmiter

	responses sync.Map

	connected bool
	mutex     *sync.RWMutex
}

// Settings represents connection settings for Amigo.
type Settings struct {
	Username string
	Password string
	Host     string
	Port     string

	DialTimeout       time.Duration
	ReconnectInterval time.Duration
	Keepalive         bool

	LogLevel logrus.Level
	Report   bool
}

// New creates new Amigo struct with credentials provided and returns pointer to it
// Usage: New(username string, secret string, [host string, [port string]])
// 建立连接
func New(settings *Settings, Log *logrus.Entry) *Amigo {
	if Log != nil {
		utils.Log = Log
	} else {
		utils.NewLog(settings.LogLevel, settings.Report)
	}

	eventEmitter := pkg.New()
	parse.Compile()

	if settings.DialTimeout == 0 {
		settings.DialTimeout = utils.DialTimeout
	}

	amiInstance := &Amigo{
		settings:     settings,
		eventEmitter: eventEmitter,
		mutex:        &sync.RWMutex{},
		connected:    false,
	}

	amiInstance.ConnectOn(func(payload ...interface{}) {
		status := payload[0].(pkg.ConnectStatus)
		if amiInstance.ami.reconnect && status != pkg.Connect_OK {
			<-time.After(utils.ReconnectInterval)
			amiInstance.initAMI()
		}
	})

	eventEmitter.On("namiEvent", func(payload ...interface{}) {
		event := payload[0].(*parse.Event)
		eventEmitter.Emit("AMI_Event", event.Data)
	})

	return amiInstance
}

// Send used to execute Actions in Asterisk. Returns immediately response from asterisk. Full response will follow.
// Usage amigo.Send(action map[string]string)
func (a *Amigo) Send(action map[string]string) (data map[string]string, event []parse.Event, err error) {
	utils.Log.Debugf("send action: %+v\n", action)
	if !a.Connected() {
		utils.Log.Warnf("ami not connected")
		return nil, nil, utils.ErrNotConnected
	}

	actionID := utils.NewV4()
	action["ActionID"] = actionID
	a.responses.Store(actionID, parse.NewResponse(""))
	a.ami.exec(action)

	// 响应处理(1.超时 2.连接断开)
	go func() {
		select {
		case <-a.ami.chanStop:
			if res, ok := a.responses.Load(actionID); ok {
				utils.Log.Warnf("action %+v %s wait complete chan failure CHAN-STOP", action, actionID)
				res.(*parse.Response).Complete <- struct{}{}
				return
			}
		case <-time.After(utils.ActionTimeout * time.Second):
			if res, ok := a.responses.Load(actionID); ok {
				utils.Log.Warnf("action %+v %s wait complete chan failure ActionTimeout: %d", action, actionID, utils.ActionTimeout)
				res.(*parse.Response).Complete <- struct{}{}
				return
			}

		}
	}()

	resInterface, _ := a.responses.Load(actionID)
	res := resInterface.(*parse.Response)
	<-res.Complete

	close(res.Complete)
	a.responses.Delete(actionID)

	res.RLock()
	if res.Data["Action"] == "logoff" {
		a.ami.reconnect = false
	}
	dataLen := len(res.Data)
	res.RUnlock()

	if dataLen == 0 {
		return nil, nil, errors.New("wait complete response failure")
	}
	return res.Data, res.Events, nil
}

// Connect with Asterisk.
// If connect fails, will try to reconnect every second.
func (a *Amigo) Connect() {
	a.mutex.RLock()
	if a.connected {
		return
	}
	a.mutex.RUnlock()

	a.initAMI()
}

func (a *Amigo) initAMI() {
	newAMIAdapter(a.settings, a.eventEmitter, a)
}

// Connected returns true if successfully connected and logged in Asterisk and false otherwise.
func (a *Amigo) Connected() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.ami != nil && a.ami.online()
}

//EventOn 暴露内部 Event 事件
func (a *Amigo) EventOn(fn func(...interface{})) {
	a.eventEmitter.AddListener("AMI_Event", fn)
}

//ConnectOn 暴露内部网络连接 事件
func (a *Amigo) ConnectOn(fn func(...interface{})) {
	a.eventEmitter.AddListener("AMI_Connect", fn)
}

func (a *Amigo) onRawMessage(message string) {
	if ok := parse.EventRegexp.MatchString(message); ok {
		event := parse.NewEvent(message)
		a.onRawEvent(event)
	} else if ok := parse.ResponseRegexp.MatchString(message); ok {
		response := parse.NewResponse(message)
		a.onRawResponse(response)
	} else {
		utils.Log.Warnf("Discarded: message %s", message)
	}
}
func (a *Amigo) onRawResponse(response *parse.Response) {
	actionID := response.Data["ActionID"]
	if actionID == "" {
		utils.Log.Warnf("No actionID Res %+v", response.Data)
		return
	}

	resInterface, existRes := a.responses.Load(actionID)
	if !existRes {
		utils.Log.Errorf("a.responses[actionID] is nil, actionID: %s", actionID)
		return
	}

	res := resInterface.(*parse.Response)
	if value, ok := response.Data["Message"]; ok && !utils.IsResponse(value) {
		res.Data = response.Data
		return
	}

	res.Complete <- struct{}{}
	res.Lock()
	res.Data = response.Data
	res.Unlock()
}
func (a *Amigo) onRawEvent(event *parse.Event) {
	if actionID, existID := event.Data["ActionID"]; existID {
		if resInterface, existRes := a.responses.Load(actionID); existRes {
			response := resInterface.(*parse.Response)
			response.Events = append(response.Events, *event)

			if utils.EventComplete(event.Data["Event"], event.Data["EventList"]) {
				response.Complete <- struct{}{}
			}
		} else {
			utils.Log.Warnf("actionID %s can't get response", actionID)
		}
		return
	}
	a.eventEmitter.Emit("namiEvent", event)
}

func (a *Amigo) handleMsg(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		case msg := <-a.ami.msg:
			a.onRawMessage(msg)
		}
	}
}
