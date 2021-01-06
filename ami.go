package amigo

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/tqcenglish/ami-go/events"
	"github.com/tqcenglish/ami-go/utils"
)

type amiAdapter struct {
	received []byte
	id       string

	dialString string
	username   string
	password   string

	connected bool
	reconnect bool

	dialTimeout time.Duration

	actionsChan chan map[string]string
	pingerChan  chan struct{}
	mutex       *sync.RWMutex

	eventEmitter events.EventEmmiter
}

func newAMIAdapter(s *Settings, eventEmitter events.EventEmmiter) (*amiAdapter, error) {
	a := &amiAdapter{
		dialString:   fmt.Sprintf("%s:%s", s.Host, s.Port),
		username:     s.Username,
		password:     s.Password,
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
	// 持续读取数据
	buf := make([]byte, 1024*20)
	var data []byte
	result := bytes.NewBuffer(nil)
	for {
		n, err := conn.Read(buf)
		if err == io.EOF {
			continue
		}
		if err != nil && err != io.EOF {
			utils.Log.Errorf("socket  error %+v", err)
			break
		}
		old := string(buf[:n])
		if a.received != nil {
			data = append(a.received, buf[:n]...)
			utils.Log.Infof("补充上一次数据\n原数据%s补充后%s", buf[:n], data)
			a.received = nil
		} else {
			data = buf[:n]
		}
		result.Write(data)
		// utils.Log.Infof("Write len %d value %s", p, data)

		scanner := bufio.NewScanner(result)
		scanner.Split(utils.PacketSlitFunc)
		for scanner.Scan() {
			msg := scanner.Text()
			if len(old) > len(msg) {
				old = string(old[len(msg):])
			}
			go a.eventEmitter.Emit("namiRawMessage", msg)
		}
		// msg 不是一个完整的消息
		if err := scanner.Err(); err == utils.ErrEOM {
			utils.Log.Infof("重复使用 %s", old)
			// fmt.Printf("%+v\n", buf[:n])
			// fmt.Printf("%+v\n", old)
			a.received = []byte(old)
			// fmt.Printf("%+v\n", a.received)
		}
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
