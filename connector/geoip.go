package connector

import (
	"github.com/oschwald/geoip2-golang"
	"github.com/cihub/seelog"
	"sync"
	"net"
	"time"
	"net/http"
)

type GeoIp struct {
	geoip *geoip2.Reader
	l     *sync.RWMutex
}

const (
	GEOIP_DATA_URL = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-City.mmdb.gz"
	GEOIP_MD5_URL = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-City.md5"
)

var geoip *GeoIp

func InitGeoip(geoipdata string) *GeoIp {
	if geoip != nil {
		return geoip
	}
	_geoip, err := geoip2.Open(geoipdata)
	if err != nil {
		panic(err)
	}
	geoip = &GeoIp{
		geoip: _geoip,
		l: new(sync.RWMutex),
	}

	return geoip
}

func GetGeoIp() *GeoIp {
	return geoip
}

func (g *GeoIp)City(ip net.IP) (*geoip2.City, error) {
	g.l.RLock()
	defer g.l.RUnlock()
	return g.geoip.City(ip)
}

func (g *GeoIp)Close() {
	g.geoip.Close()
}

func (g *GeoIp)Updata() {
	time.AfterFunc(3600 * time.Second, func(){
		g.Updata()

		//获取data的MD5文件匹配
		_, err := http.Get(GEOIP_MD5_URL)
		if (err != nil) {
			seelog.Errorf("Get geoip md5 file error: %v", err)
		}

		//res.Body
		//g.l.Lock()
		//defer g.l.Unlock()
	})
}
