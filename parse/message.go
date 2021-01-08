package parse

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/tqcenglish/ami-go/utils"
)

//Message ami 消息
type Message struct {
	lines    []string
	variable map[string]string
	Data     map[string]string
}

func (message *Message) unMarshall(data string) {
	/* Setlet contains Variable: header, but value should not include '=' in this case
	 */
	//     if (keySafe.match(/variable/) !== null && valueSafe.match(/=/) !== null) {
	//         let variable = valueSafe.split("=");
	//         this.variables[variable[0]] = variable[1];
	//     } else {
	//         this.set(keySafe, valueSafe);
	//     }
	// }
	message.lines = strings.Split(data, utils.EOL)
	for i := 0; i < len(message.lines); i++ {
		parts := strings.Split(message.lines[i], ":")
		if len(parts) <= 1 {
			log.Errorf("Error message foramt")
		}
		key := strings.ReplaceAll(strings.TrimSpace(parts[0]), "-", "")
		value := strings.Join(parts[1:], ":")
		message.Data[key] = strings.TrimSpace(value)
	}
}
