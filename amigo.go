package amigo

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/tqcenglish/ami-go/events"
	"github.com/tqcenglish/ami-go/parse"
	"github.com/tqcenglish/ami-go/utils"
	"github.com/tqcenglish/ami-go/uuid"
)

var (
	version         = "0.1.9"
	errNotConnected = errors.New("Not connected to Asterisk")
)

type handlerFunc func(response parse.Response)
type eventHandlerFunc func(event map[string]string)

// Amigo is a main package struct
type Amigo struct {
	settings *Settings
	ami      *amiAdapter

	eventEmitter events.EventEmmiter

	responses map[string]*parse.Response

	connected bool
	mutex     *sync.RWMutex
}

// Settings represents connection settings for Amigo.
// Default:
// Username = admin,
// Password = amp111,
// Host = 127.0.0.1,
// Port = 5038,
// DialTimeout = 10s
type Settings struct {
	Username          string
	Password          string
	Host              string
	Port              string
	DialTimeout       time.Duration
	ReconnectInterval time.Duration
	Keepalive         bool
}

// New creates new Amigo struct with credentials provided and returns pointer to it
// Usage: New(username string, secret string, [host string, [port string]])
// 建立连接
func New(settings *Settings) *Amigo {
	eventEmitter := events.New()
	utils.NewLog()

	amiInstance := &Amigo{
		settings:     settings,
		eventEmitter: eventEmitter,
		mutex:        &sync.RWMutex{},
		connected:    false,
		responses:    make(map[string]*parse.Response),
	}

	eventEmitter.On("error", func(payload ...interface{}) {
		utils.Log.Errorf("ami error %s", payload[0].(string))
		eventEmitter.Emit("AMI_Connect", "ami connect error")
		if amiInstance.ami.reconnect {
			amiInstance.initAMI()
		}
	})
	eventEmitter.On("connect", func(payload ...interface{}) {
		eventEmitter.Emit("AMI_Connect", payload[0].(string))

		if err := amiInstance.login(); err != nil {
			amiInstance.eventEmitter.Emit("error", fmt.Sprintf("Asterisk login: %s", err.Error()))
			return
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
	utils.Log.Infof("send action: %+v\n", action)
	if !a.Connected() {
		utils.Log.Warnf("ami not connected")
		return nil, nil, errNotConnected
	}

	actionID := uuid.NewV4()
	action["ActionID"] = actionID
	a.responses[actionID] = parse.NewResponse("")
	a.ami.exec(action)

	// 超时处理
	time.AfterFunc(time.Duration(utils.ActionTimeout)*time.Second, func() {
		a.mutex.RLock()
		_, ok := a.responses[actionID]
		a.mutex.RUnlock()
		if ok {
			a.mutex.Lock()
			if res, ok := a.responses[actionID]; ok {
				utils.Log.Warnf("action wait complete chan failure ActionTimeout: %d", utils.ActionTimeout)
				res.Complete <- struct{}{}
				a.mutex.Unlock()
				return
			}
			a.mutex.Unlock()
		}
	})

	<-a.responses[actionID].Complete
	response := a.responses[actionID]

	close(a.responses[actionID].Complete)
	delete(a.responses, actionID)

	if response.Data["Action"] == "logoff" {
		a.ami.reconnect = false
	}
	if len((*response).Data) == 0 {
		return nil, nil, errors.New("不能等待到数据")
	}
	return (*response).Data, (*response).Events, nil
}

// Connect with Asterisk.
// If connect fails, will try to reconnect every second.
func (a *Amigo) Connect() {
	var connected bool
	a.mutex.RLock()
	connected = a.connected
	a.mutex.RUnlock()
	if connected {
		return
	}

	a.mutex.Lock()
	a.connected = true
	a.mutex.Unlock()

	a.initAMI()
}

func (a *Amigo) initAMI() {
	am, err := newAMIAdapter(a.settings, a.eventEmitter)
	if err != nil {
		go a.eventEmitter.Emit("error", fmt.Sprintf("AMI Connect error: %s", err.Error()))
	} else {
		a.mutex.Lock()
		a.ami = am
		a.mutex.Unlock()
		go func() {
			for {
				select {
				case msg := <-a.ami.msg:
					a.onRawMessage(msg)
				}
			}
		}()
	}
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
	if ok, _ := regexp.Match(`^Event: `, []byte(message)); ok {
		event := parse.NewEvent(message)
		a.onRawEvent(event)
	} else if ok, _ := regexp.Match(`^Response: `, []byte(message)); ok {
		response := parse.NewResponse(message)
		a.onRawResponse(response)
	} else {
		utils.Log.Warnf("Discarded: message %s", message)
	}
}
func (a *Amigo) onRawResponse(response *parse.Response) {
	actionID := response.Data["ActionID"]
	if value, ok := response.Data["Message"]; ok && strings.Contains(value, "follow") {
		a.responses[actionID].Data = response.Data
	} else {
		a.responses[actionID].Complete <- struct{}{}
		a.responses[actionID].Data = (*response).Data
	}
}
func (a *Amigo) onRawEvent(event *parse.Event) {
	if actionID, existID := event.Data["ActionID"]; existID {
		if _, existRes := a.responses[actionID]; existRes {
			response := a.responses[actionID]
			response.Events = append(response.Events, *event)
		}
		if strings.Contains(event.Data["Event"], "Complete") || strings.Contains(event.Data["Event"], "DBGetResponse") || (event.Data["EventList"] != "" && strings.Contains(event.Data["EventList"], "Complete")) {
			a.responses[actionID].Complete <- struct{}{}
		}
		return
	}
	a.eventEmitter.Emit("namiEvent", event)
}
