package connector

import (
	"bytes"
	"github.com/cihub/seelog"
	"net"
)

var comma = []byte{','}

const (
	LOG_TYPE_ONLINE_USER_AREA = "online_user_area"
	LOG_TYPE_ONLINE_USER = "online_user"
	LOG_TYPE_IP_TO_ISO = "ip_to_iso"
	LOG_TYPE_NORMAL = "*"

	MESSAGE_NEW_LINE_STRING = "\n"
	MESSAGE_NEW_LINE_BYTE = '\n'
)

type MessageProcess interface {
	Process(msg *Msg, conn net.Conn)
}

type MessageWorker struct {
	P MessageProcess
}

type Msg struct {
	Category string
	Data     []byte
}

type BaseMessageProcesser struct {
	Hub *Hub
}

func (m *BaseMessageProcesser) Process(msg *Msg, conn net.Conn) {
	m.Hub.Broadcast <- msg
}

func FormatMsg(data []byte) *Msg {
	var message = bytes.SplitN(data, comma, 2)

	if (len(message) != 2) {
		seelog.Errorf("received message format is error: %s", bytes.TrimRight(data, MESSAGE_NEW_LINE_STRING))
		return nil
	}

	return &Msg{Category: string(message[0]), Data: message[1]};
}

func RevertMsg(msg *Msg) []byte {
	return bytes.Join([][]byte{[]byte(msg.Category), msg.Data}, comma)
}

func ProcessMsg(worker MessageWorker, msg *Msg, conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			seelog.Error("ProcessMsg error: ", err);
		}
	}()
	worker.P.Process(msg, conn)
}
