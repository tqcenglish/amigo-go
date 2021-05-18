package pkg

type ConnectStatus int32

const (
	Connect_OK               ConnectStatus = 0
	Connect_Password_Error   ConnectStatus = 1
	Connect_Network_Error    ConnectStatus = 2
	Disconnect_Network_Error ConnectStatus = 3
)

func (status ConnectStatus) String() string {
	switch status {
	case Connect_OK:
		return "Connect OK"
	case Connect_Password_Error:
		return "Password Error"
	case Connect_Network_Error:
		return "Network Error"
	case Disconnect_Network_Error:
		return "Network Disconnect Error"
	default:
		return "Unknown Error"
	}
}
