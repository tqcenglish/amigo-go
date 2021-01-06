package utils

import (
	"errors"
	"time"
)

//EOL 换行
var EOL = "\r\n"

//EOM 换行
var EOM = "\r\n\r\n"

//ErrEOM EOM error
var ErrEOM = errors.New("EOM")

//ActionTimeout 超时(s)
var ActionTimeout = 60

const (
	//PingActionID ID
	PingActionID = "AmigoPing"
	//AmigoConnIDKey amiID
	AmigoConnIDKey = "AmigoConnID"
)

var (
	//DialTimeout TCP Dial
	DialTimeout = 10 * time.Second
	//PingInterval Ping
	PingInterval = 5 * time.Second
)
