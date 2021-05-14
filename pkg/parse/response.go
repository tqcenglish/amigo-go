package parse

import "fmt"

//Response 命令响应
type Response struct {
	*Message
	Events   []Event
	Complete chan struct{}
}

//NewResponse 新建事件
func NewResponse(data string) *Response {
	messages := &Message{
		Data: make(map[string]string),
	}
	response := &Response{
		Message:  messages,
		Events:   make([]Event, 0),
		Complete: make(chan struct{}, 1),
	}
	if data != "" {
		response.unMarshall(data)
	}
	return response
}

//String 定义 toString
func (res *Response) String() string {
	return fmt.Sprintf("{Response: %s, ActionID: %s, Message: %s, Event:%+v}", res.Data["Response"], res.Data["ActionID"], res.Data["Message"], res.Events)
}
