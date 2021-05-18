package utils

import (
	"errors"
	"time"
)

const (
	//AmigoConnIDKey amiID
	AmigoConnIDKey = "AmigoConnID"

	//DialTimeout TCP Dial
	DialTimeout = 5 * time.Second
	//ReconnectInterval reconnect interval
	ReconnectInterval = 5 * time.Second
	//PingInterval Ping
	PingInterval = 5 * time.Second

	//ActionTimeout 超时(s)
	ActionTimeout = 60

	//EOL 换行
	EOL = "\r\n"
	//EOM 换行
	EOM = "\r\n\r\n"
)

var (
	ErrNotConnected = errors.New("not connected to asterisk")
	//ErrEOM EOM error
	ErrEOM = errors.New("eom")
)
