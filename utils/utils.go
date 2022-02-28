package utils

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
)

//Marshall 构建 action
func Marshall(action map[string]interface{}) string {
	output := ""
	for key, value := range action {
		if key == "variables" {
			continue
		}
		output = fmt.Sprintf("%s%s:%s%s", output, key, value, EOL)
	}
	if variables, ok := action["variables"].(map[string]string); ok {
		for key, value := range variables {
			output = fmt.Sprintf("%sVariable: %s=%s%s", output, key, value, EOL)
		}
	}
	output = fmt.Sprintf("%s%s", output, EOL)
	return output
}

//StringMapToInterface 转换
func StringMapToInterface(src map[string]string) (dst map[string]interface{}) {
	dst = make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = interface{}(v)
	}
	return
}

//NextID id
func NextID() string {
	var sequence uint64
	i := atomic.AddUint64(&sequence, 1)
	return strconv.Itoa(int(i))
}

//SetField 反射设置值
func SetField(obj interface{}, name string, value interface{}) error {
	structValue := reflect.ValueOf(obj).Elem()
	structFieldValue := structValue.FieldByName(name)

	if !structFieldValue.IsValid() {
		return nil //fmt.Errorf("No such field: %s in obj", name)
	}

	if !structFieldValue.CanSet() {
		return fmt.Errorf("cannot set %s field value", name)
	}

	structFieldType := structFieldValue.Type()
	val := reflect.ValueOf(value)
	if structFieldType != val.Type() {
		return errors.New("provided value type didn't match obj field type")
	}

	structFieldValue.Set(val)
	return nil
}

//IsResponse 通过 Message 判断当前响应属于 Event 还是 Response
func IsResponse(message string) bool {
	if strings.Contains(message, "Command output follows") {
		return true
	}
	if strings.Contains(message, "follow") {
		return false
	}
	if strings.Contains(message, "Follow") {
		return false
	}
	return true
}

//EventComplete 判断事件是否结束
func EventComplete(event, list string) bool {
	if strings.Contains(event, "Complete") {
		return true
	}
	if strings.Contains(event, "DBGetResponse") {
		return true
	}
	if list != "" && strings.Contains(list, "Complete") {
		return true
	}
	return false
}
