package handler

import (
	"github.com/cihub/seelog"
	"net"
	"github.com/bupt1987/log-websocket/util"
	"bytes"
	"github.com/bupt1987/log-websocket/connector"
)

type IpToIso struct {

}

func (m *IpToIso) Process(msg *connector.Msg, conn net.Conn) {
	var sIso = connector.MESSAGE_NEW_LINE_STRING
	ip := string(bytes.TrimRight(msg.Data, connector.MESSAGE_NEW_LINE_STRING))

	if ip != "" && ip != "unknown" {
		city, err := util.GetGeoIp().City(net.ParseIP(ip))
		if err != nil {
			seelog.Errorf("geoip '%v' error: %v", ip, err.Error())
		} else if city.Country.IsoCode != "" {
			sIso = city.Country.IsoCode + connector.MESSAGE_NEW_LINE_STRING
		}
	}

	conn.Write([]byte(sIso))
}
