package connector

import (
	"bytes"
	"encoding/json"
	"net"
	"github.com/cihub/seelog"
)

var comma = []byte{','}

const (
	LOG_TYPE_ONLINE_USER = "online_user"
	LOG_TYPE_NORMAL = "*"
)

type MessageProcess interface {
	Process(msg *Msg)
}

type MessageWorker struct {
	P MessageProcess
}

type Msg struct {
	Category string
	Data     []byte
}

type BaseMessage struct {
	Hub *Hub
}

func (m *BaseMessage) Process(msg *Msg) {
	m.Hub.Broadcast <- msg
}

func FormatMsg(data []byte) *Msg {
	var message = bytes.SplitN(data, comma, 2)

	if (len(message) != 2) {
		seelog.Errorf("received message format is error: %s", bytes.TrimRight(data, "\n"))
		return nil
	}

	return &Msg{Category: string(message[0]), Data: message[1]};
}

func ProcessMsg(worker MessageWorker, msg *Msg) {
	defer func() {
		if err := recover(); err != nil {
			seelog.Error("ProcessMsg error: ", err);
		}
	}()
	worker.P.Process(msg)
}

type OnlineUserMessage struct {
	UserSet *UserSet
}

func (m *OnlineUserMessage) Process(msg *Msg) {
	userLog := UserLog{}
	json.Unmarshal(msg.Data, &userLog)
	ip := net.ParseIP(userLog.Ip)
	city, err := GetGeoIp().City(ip)
	var iso = ""
	var name = ""
	if err != nil {
		seelog.Error("geoip error: ", err.Error())
	} else if city.Country.IsoCode != "" {
		iso = city.Country.IsoCode
		name = city.Subdivisions[0].Names["en"]
	}
	m.UserSet.NewUser(&User{
		Uid: userLog.Uid,
		Ip: userLog.Ip,
		CountryIsoCode: iso,
		CountryName: name,
		StartTime: userLog.Start_Time,
		EndTime: userLog.End_Time,
	})
}
