package connector

import (
	"bytes"
	"encoding/json"
	"net"
	"github.com/cihub/seelog"
)

var comma = []byte{','}

const (
	LOG_TYPE_ONLINE_USER_AREA = "online_user_area"
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
	var isoCode = ""
	var countryName = ""
	userLog := UserLog{}
	json.Unmarshal(msg.Data, &userLog)

	if userLog.Ip != "" {
		ip := net.ParseIP(userLog.Ip)
		city, err := GetGeoIp().City(ip)
		if err != nil {
			seelog.Errorf("geoip error: %v ,", userLog.Ip, err.Error())
		} else if city.Country.IsoCode != "" {
			isoCode = city.Country.IsoCode
			countryName = city.Country.Names["en"]
		}
	}

	m.UserSet.NewUser(&User{
		Uid: userLog.Uid,
		Ip: userLog.Ip,
		IsoCode: isoCode,
		CountryName: countryName,
		StartTime: userLog.Start_Time,
		EndTime: userLog.End_Time,
	})
}
