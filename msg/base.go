package msg

import (
	"net"
	"github.com/bupt1987/log-websocket/connector"
)

const (
	ONLINE_USER_AREA = "online_user_area"
	ONLINE_USER = "online_user"
	IP_TO_ISO = "ip_to_iso"
	ANY = "*"
)

type BaseProcesser struct {
	Hub *connector.Hub
}

func (m *BaseProcesser) Process(msg *connector.Msg, conn net.Conn) {
	m.Hub.Broadcast <- msg
}
