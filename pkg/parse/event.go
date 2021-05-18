package parse

import (
	"fmt"
	"regexp"
)

var EventRegexp *regexp.Regexp
var ResponseRegexp *regexp.Regexp

//Event 事件
type Event struct {
	*Message
}

//NewEvent 新建事件
func NewEvent(data string) *Event {
	messages := &Message{
		Data: make(map[string]string),
	}
	event := &Event{
		Message: messages,
	}
	event.unMarshall(data)
	return event
}

//String 定义 toString
func (event Event) String() string {
	return fmt.Sprintf("%+v", event.Data)
}

func Compile() {
	EventRegexp, _ = regexp.Compile(`^Event: `)
	ResponseRegexp, _ = regexp.Compile(`^Response: `)
}
