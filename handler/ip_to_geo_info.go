package handler

import (
	"github.com/cihub/seelog"
	"net"
	"github.com/bupt1987/log-websocket/util"
	"bytes"
	"github.com/bupt1987/log-websocket/connector"
)

type IpToGeoInfo struct {

}

type GeoInfo struct {
	Iso   string
	CityName string
	Latitude float64
	Longitude float64
}

func (m *IpToGeoInfo) Process(msg *connector.Msg, conn net.Conn) {
	sIso := ""
	ip := string(bytes.TrimRight(msg.Data, connector.MESSAGE_NEW_LINE_STRING))

	info := map[string]interface{}{
		"iso": "",
		"city_name": "",
		"latitude": 0.0,
		"longitude": 0.0,
	}

	if ip != "" && ip != "unknown" {
		city, err := util.GetGeoIp().City(net.ParseIP(ip))
		if err != nil {
			seelog.Errorf("geoip '%v' error: %v", ip, err.Error())
		} else if city.Country.IsoCode != "" {
			sIso = city.Country.IsoCode
			info["iso"] = sIso;
			info["city_name"] = city.City.Names["en"]
			info["latitude"] = city.Location.Latitude
			info["longitude"] = city.Location.Longitude
		}
	}

	buff := bytes.NewBuffer(util.JsonEncode(info));
	buff.WriteByte(connector.MESSAGE_NEW_LINE_BYTE)

	conn.Write(buff.Bytes())
}
