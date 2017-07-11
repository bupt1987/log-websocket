package handler

import (
	"net"
	"github.com/bupt1987/log-websocket/connector"
)

const (
	ONLINE_USER_AREA = "online_user_area"
	ONLINE_USER = "online_user"
	IP_TO_ISO = "ip_to_iso"
	IP_TO_GEO_INFO = "ip_to_geo_info"
	ANY = "*"
)

type Base struct {
	Group *connector.WsGroup
}

func (m *Base) Process(msg *connector.Msg, conn net.Conn) {
	m.Group.Broadcast <- msg
}
