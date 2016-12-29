package connector

import (
	"net"
	"bytes"
	"github.com/cihub/seelog"
)

var comma = []byte{','}
const (
	MESSAGE_NEW_LINE_STRING = "\n"
	MESSAGE_NEW_LINE_BYTE = '\n'
)

type MsgProcess interface {
	Process(msg *Msg, conn net.Conn)
}

type MsgWorker struct {
	P MsgProcess
}

type Msg struct {
	Category string
	Data     []byte
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

func ProcessMsg(worker MsgWorker, msg *Msg, conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			seelog.Error("ProcessMsg error: ", err);
		}
	}()
	worker.P.Process(msg, conn)
}
