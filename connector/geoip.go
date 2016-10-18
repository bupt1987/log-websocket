package connector

import (
	"github.com/oschwald/geoip2-golang"
	"github.com/cihub/seelog"
)

var geoip *geoip2.Reader
var err error

func InitGeoip(geoipdata string) {
	if geoip != nil {
		return
	}
	geoip, err = geoip2.Open(geoipdata)
	if err != nil {
		seelog.Error("load geoip data error:", err.Error())
		return
	}
}

func GetGeoIp() *geoip2.Reader {
	return geoip
}
