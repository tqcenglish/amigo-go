package amigo

import (
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/tqcenglish/amigo-go/pkg"
	"github.com/tqcenglish/amigo-go/utils"
)

type amiAdapter struct {
	id string

	received chan string
	msg      chan string

	username string
	password string

	connected bool
	reconnect bool
	chanStop  chan struct{}

	dialString  string
	dialTimeout time.Duration

	actionsChan chan map[string]string
	mutex       *sync.RWMutex
	amigo       *Amigo

	eventEmitter pkg.EventEmmiter
}

func newAMIAdapter(s *Settings, eventEmitter pkg.EventEmmiter, amigo *Amigo) {
	adapter := &amiAdapter{
		dialString: fmt.Sprintf("%s:%s", s.Host, s.Port),
		username:   s.Username,
		password:   s.Password,

		reconnect: true,
		chanStop:  make(chan struct{}),

		received: make(chan string, 1024),
		msg:      make(chan string, 1024),

		dialTimeout:  s.DialTimeout,
		mutex:        &sync.RWMutex{},
		eventEmitter: eventEmitter,
		amigo:        amigo,

		actionsChan: make(chan map[string]string),
	}

	amigo.mutex.Lock()
	amigo.ami = adapter
	amigo.mutex.Unlock()

	go adapter.initializeSocket()
	go amigo.handleMsg(adapter.chanStop)
}

func (a *amiAdapter) initializeSocket() {
	a.id = utils.NextID()
	var err error
	var conn net.Conn

	readErrChan := make(chan error)
	writeErrChan := make(chan error)
	pingErrChan := make(chan error)

	conn, err = a.openConnection()
	if err != nil {
		utils.Log.Errorf("ami init socket %s", err)
		a.eventEmitter.Emit("AMI_Connect", pkg.Connect_Network_Error)
		return
	}
	defer conn.Close()
	greetings := make([]byte, 100)
	n, err := conn.Read(greetings)
	if err != nil {
		utils.Log.Errorf("ami read socket %s", err)
		a.eventEmitter.Emit("AMI_Connect", pkg.Disconnect_Network_Error)
		time.Sleep(time.Second)
		return
	}

	if n > 2 {
		greetings = greetings[:n-2]
	}

	a.mutex.Lock()
	a.connected = true
	a.mutex.Unlock()

	utils.Log.Infof("ami connect: %s", string(greetings))

	go a.reader(conn, a.chanStop, readErrChan)
	go a.writer(conn, a.chanStop, writeErrChan)

	go func() {
		if err := a.login(); err != nil {
			utils.Log.Errorf("ami login %s", pkg.Connect_Password_Error)
			a.reconnect = false
			return
		}
		a.eventEmitter.Emit("AMI_Connect", pkg.Connect_OK)
		a.pinger(a.chanStop, pingErrChan)
	}()

	select {
	case err = <-readErrChan:
	case err = <-writeErrChan:
	case err = <-pingErrChan:
	}

	close(a.chanStop)
	a.mutex.Lock()
	a.connected = false
	a.mutex.Unlock()

	utils.Log.Errorf("ami read/write/ping socket %s", err.Error())
	a.eventEmitter.Emit("AMI_Connect", pkg.Disconnect_Network_Error)
}

func (a *amiAdapter) online() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.connected
}

func (a *amiAdapter) openConnection() (net.Conn, error) {
	return net.DialTimeout("tcp", a.dialString, a.dialTimeout)
}

func (a *amiAdapter) reader(conn net.Conn, stop <-chan struct{}, readErrChan chan error) {
	go func() {
		var result []byte
		for msg := range a.received {
			result = append(result, []byte(msg)...)
			for {
				index := strings.Index(string(result), utils.EOM)
				if index != -1 {
					// 获取结束位置
					endIndex := index + len(utils.EOM)
					skippedEolChars := 0
					nextIndex := endIndex + skippedEolChars + 1
					for nextIndex < len(result) {
						nextChar := result[nextIndex]
						if nextChar == '\r' || nextChar == '\n' {
							skippedEolChars++
							continue
						}
						break
					}
					a.msg <- string(result[:index])
					result = result[endIndex:]
					continue
				}
				break
			}
		}

	}()

	// 持续读取数据
	buf := make([]byte, 1024*4)
	for {
		n, err := conn.Read(buf)
		if err == io.EOF {
			continue
		}
		if err != nil && err != io.EOF {
			readErrChan <- err
			return
		}
		a.received <- string(buf[:n])
	}

}

func (a *amiAdapter) writer(conn net.Conn, stop <-chan struct{}, writeErrChan chan error) {
	for {
		select {
		case <-stop:
			return
		case action := <-a.actionsChan:
			if action[utils.AmigoConnIDKey] != a.id {
				// action sent before reconnect, need to be ignored
				continue
			}

			data := utils.Marshall(utils.StringMapToInterface(action))
			_, err := conn.Write([]byte(data))
			if err != nil {
				writeErrChan <- err
				return
			}
		}
	}
}

func (a *amiAdapter) exec(action map[string]string) {
	action[utils.AmigoConnIDKey] = a.id
	a.actionsChan <- action
}
