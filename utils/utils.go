package utils

import (
	"fmt"
	"strconv"
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
