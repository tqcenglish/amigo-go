package amigo

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/tqcenglish/amigo-go/events"
	"github.com/tqcenglish/amigo-go/utils"
)

type amiAdapter struct {
	received chan string
	msg      chan string

	id string

	username string
	password string

	connected bool
	reconnect bool

	dialString  string
	dialTimeout time.Duration

	actionsChan chan map[string]string
	pingerChan  chan struct{}
	mutex       *sync.RWMutex

	eventEmitter events.EventEmmiter
}

func newAMIAdapter(s *Settings, eventEmitter events.EventEmmiter) (*amiAdapter, error) {
	a := &amiAdapter{
		dialString: fmt.Sprintf("%s:%s", s.Host, s.Port),
		username:   s.Username,
		password:   s.Password,

		received: make(chan string, 1024),
		msg:      make(chan string, 1024),

		dialTimeout:  s.DialTimeout,
		mutex:        &sync.RWMutex{},
		eventEmitter: eventEmitter,

		actionsChan: make(chan map[string]string),
		pingerChan:  make(chan struct{}),
	}

	go a.initializeSocket()

	return a, nil
}

func (a *amiAdapter) initializeSocket() {
	a.id = utils.NextID()
	var err error
	var conn net.Conn
	readErrChan := make(chan error)
	writeErrChan := make(chan error)
	pingErrChan := make(chan error)
	chanStop := make(chan struct{})

	for {
		conn, err = a.openConnection()
		if err == nil {
			defer conn.Close()
			greetings := make([]byte, 100)
			n, err := conn.Read(greetings)
			if err != nil {
				a.eventEmitter.Emit("error", fmt.Sprintf("Asterisk connection error: %s", err.Error()))
				time.Sleep(time.Second)
				return
			}

			if n > 2 {
				greetings = greetings[:n-2]
			}

			a.mutex.Lock()
			a.connected = true
			a.mutex.Unlock()
			go a.eventEmitter.Emit("connect", string(greetings))
			break
		}

		a.eventEmitter.Emit("error", "AMI Reconnect failed")
		return
	}

	go a.reader(conn, chanStop, readErrChan)
	go a.writer(conn, chanStop, writeErrChan)
	// if s.Keepalive {
	// 	go a.pinger(chanStop, pingErrChan)
	// }

	select {
	case err = <-readErrChan:
	case err = <-writeErrChan:
	case err = <-pingErrChan:
	}

	close(chanStop)
	a.mutex.Lock()
	a.connected = false
	a.mutex.Unlock()

	a.eventEmitter.Emit("error", fmt.Sprintf("AMI TCP ERROR: %s", err.Error()))
	time.Sleep(time.Second)
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
		for {
			select {
			case msg := <-a.received:
				result = append(result, []byte(msg)...)
				for {
					index := strings.Index(string(result), utils.EOM)
					if index != -1 {
						// 获取结束位置
						endIndex := index + len(utils.EOM)
						skippedEolChars := 0
						for endIndex+skippedEolChars+1 <= len(result) {
							nextChar := result[endIndex+skippedEolChars+1]
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
			utils.Log.Errorf("socket  error %+v", err)
			readErrChan <- errors.New("socket error")
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
