package connector

import (
	"github.com/oschwald/geoip2-golang"
	"github.com/cihub/seelog"
	"sync"
	"net"
	"net/http"
	"os"
	"io/ioutil"
	"io"
	"crypto/md5"
	"compress/gzip"
	"encoding/hex"
	"time"
	"strings"
)

type GeoIp struct {
	geoip     *geoip2.Reader
	sDataFile string
	sMd5File  string
	md5       string
	l         *sync.RWMutex
}

const (
	GEOIP_DATA_URL = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-City.mmdb.gz"
	GEOIP_MD5_URL = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-City.md5"
)

var geoip *GeoIp

func InitGeoip(geoipdata string, sMd5File string) *GeoIp {
	if geoip != nil {
		return geoip
	}
	_md5, err := ioutil.ReadFile(sMd5File)
	if (err != nil) {
		panic(err)
	}

	_geoip, err := geoip2.Open(geoipdata)
	if err != nil {
		panic(err)
	}

	geoip = &GeoIp{
		geoip: _geoip,
		sDataFile: geoipdata,
		sMd5File: sMd5File,
		md5: strings.TrimRight(string(_md5), "\n"),
		l: new(sync.RWMutex),
	}

	geoip.Updata()

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
	iAfter := 3600
	seelog.Debugf("Check the update of geoip will run after %vs", iAfter)
	time.AfterFunc(time.Duration(iAfter) * time.Second, func() {
		defer g.Updata()

		seelog.Debug("======================  Start check geoip data update  ======================")

		//获取data的MD5文件匹配
		res, err := http.Get(GEOIP_MD5_URL)
		defer res.Body.Close()
		if (err != nil) {
			seelog.Errorf("Get geoip md5 file error: %v", err)
			return
		}

		_md5, err := ioutil.ReadAll(res.Body)
		if err != nil {
			seelog.Errorf("Get geoip md5 file error: %v", err)
			return
		}
		sNewMd5 := string(_md5)
		seelog.Debugf("The new md5: %v, old md5: %v", sNewMd5, g.md5)

		if sNewMd5 != g.md5 {
			seelog.Debug("Start download new geoip data")

			//下载data文件
			res, err := http.Get(GEOIP_DATA_URL)
			defer res.Body.Close()
			if err != nil {
				seelog.Errorf("Get geoip data file error: %v", err)
				return
			}

			sGzTmp := "/tmp/GeoLite2-City.mmdb.gz"
			gf, err := os.Create(sGzTmp)
			defer gf.Close()
			if err != nil {
				seelog.Errorf("Get geoip data file error: %v", err)
				return
			}
			defer os.Remove(sGzTmp)

			io.Copy(gf, res.Body)
			seelog.Debug("Download new geoip data finished")

			// gzip read
			gfr, err := os.Open(sGzTmp)
			defer gfr.Close()
			if err != nil {
				seelog.Errorf("Open geoip data gz file error: %v", err)
				return
			}

			gzr, err := gzip.NewReader(gfr)
			defer gzr.Close()
			if err != nil {
				seelog.Errorf("Gzip geoip data file error: %v", err)
				return
			}

			sTmp := "/tmp/GeoLite2-City.mmdb"
			f, err := os.Create(sTmp)
			defer f.Close()
			if err != nil {
				seelog.Errorf("Creat geoip data file error: %v", err)
				return
			}

			io.Copy(f, gzr)
			seelog.Debug("Ungz new geoip data finished")

			// data read
			fr, err := os.Open(sTmp)
			defer fr.Close()
			if err != nil {
				seelog.Errorf("Open geoip data file error: %v", err)
				return
			}

			//检查md5 是否一致
			md5h := md5.New()
			io.Copy(md5h, fr)
			dataMd5 := hex.EncodeToString(md5h.Sum(nil))

			seelog.Debugf("New geoip data md5 is %v", dataMd5)

			if dataMd5 != sNewMd5 {
				os.Remove(sTmp)
				seelog.Errorf("Download geoip data file md5 is not match: %v != %v", dataMd5, sNewMd5)
				return
			}

			sBakFile := g.sDataFile + "_bak"
			if err := os.Rename(g.sDataFile, sBakFile); err != nil {
				seelog.Errorf("Move old data file to bak error: ", err.Error())
				return
			}

			if err := os.Rename(sTmp, g.sDataFile); err != nil {
				seelog.Errorf("Move new data file error: ", err.Error())
				return
			}

			_geoip, err := geoip2.Open(g.sDataFile)
			if err != nil {
				seelog.Errorf("Load new data file error: ", err.Error())
				if err := os.Rename(sBakFile, g.sDataFile); err != nil {
					seelog.Errorf("Recover old data file error: ", err.Error())
					return
				}
				return
			}

			//移动md5文件
			if err := ioutil.WriteFile(g.sMd5File, []byte(sNewMd5), 0644); err != nil {
				seelog.Errorf("Write new md5 error: ", err);
			}
			defer os.Remove(sBakFile)

			//上锁
			g.l.Lock()
			defer g.l.Unlock()

			//替换新的geoip
			g.geoip.Close()
			g.geoip = _geoip
			g.md5 = sNewMd5

			seelog.Infof("Geoip data updata finished, md5: %v", sNewMd5)
		}
		seelog.Debug("====================================  End  ==================================")
	})
}
