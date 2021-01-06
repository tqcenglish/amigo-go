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
	// node code
	// let value, parts, key, line = 0;
	// this.lines = data.split(this.EOL);
	// for (; line < this.lines.length; line = line + 1) {
	//     parts = this.lines[line].split(":");
	//     key = parts.shift();
	//     /*
	//      * This is so, because if this message is a response, specifically a response to
	//      * something like "ListCommands", the value of the keys, can contain the semicolon
	//      * ":", which happens to be token to be used to split keys and values. AMI does not
	//      * specify anything like an escape character, so we cant distinguish wether we're
	//      * dealing with a multi semicolon line or a standard key/value line.
	//      */
	//     if (parts.length > 1) {
	//         value = parts.join(':');
	//     } else if (parts.length === 1) {
	//         value = parts[0];
	//     }
	//     let keySafe = key.replace(/-/, '_').toLowerCase();
	//     let valueSafe = value.replace(/^\s+/g, '').replace(/\s+$/g, '');
	//     /*
	//      * Setlet contains Variable: header, but value should not include '=' in this case
	//      */
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
		key := strings.TrimSpace(parts[0])
		value := strings.Join(parts[1:], ":")
		message.Data[key] = strings.TrimSpace(value)
	}
}
