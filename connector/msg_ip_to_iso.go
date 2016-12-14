package connector

import (
	"github.com/cihub/seelog"
	"net"
	"github.com/bupt1987/log-websocket/util"
	"bytes"
)

type IpToIsoMessageProcesser struct {

}

type NewUserLog struct {
	Ip string
}

func (m *IpToIsoMessageProcesser) Process(msg *Msg, conn net.Conn) {
	var sIso = MESSAGE_NEW_LINE_STRING
	ip := string(bytes.TrimRight(msg.Data, MESSAGE_NEW_LINE_STRING))

	if ip != "" && ip != "unknown" {
		city, err := util.GetGeoIp().City(net.ParseIP(ip))
		if err != nil {
			seelog.Errorf("geoip '%v' error: %v", ip, err.Error())
		} else if city.Country.IsoCode != "" {
			sIso = city.Country.IsoCode + MESSAGE_NEW_LINE_STRING
		}
	}

	conn.Write([]byte(sIso))
}
