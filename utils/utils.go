package utils

import (
	"fmt"
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

//PacketSlitFunc 解析 ami 返回消息
//按 EOM 处理
func PacketSlitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Log.Infof("packetSlitFunc %s", string(data))
	// 检查 atEOF 参数
	if !atEOF && len(data) > 1 {
		index := strings.Index(string(data), EOM)
		if index == -1 {
			return 0, nil, ErrEOM
		}

		// 获取结束位置
		endIndex := index + len(EOM)
		skippedEolChars := 0
		for endIndex+skippedEolChars+1 <= len(data) {
			nextChar := data[endIndex+skippedEolChars+1]
			if nextChar == '\r' || nextChar == '\n' {
				skippedEolChars++
				continue
			}
			break
		}
		// Log.Infof("endIndex:%d skippedEolChars:%d result %s left %s", endIndex, skippedEolChars, data[:index], data[endIndex+skippedEolChars:])
		return endIndex + skippedEolChars, data[:index], nil
	}
	return
}
