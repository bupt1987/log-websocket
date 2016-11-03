package util

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

func InitGeoip(geoipdata string, md5File string) *GeoIp {
	if geoip != nil {
		return geoip
	}

	geoip = &GeoIp{
		sDataFile: geoipdata,
		sMd5File: md5File,
		l: new(sync.RWMutex),
	}

	var _md5 string

	if _, err := os.Stat(md5File); err != nil {
		_md5, err = geoip.getNewMd5()
		if (err != nil) {
			panic(err)
		}
		geoip.writeMd5File(_md5)
	} else {
		var _md5byte []byte
		_md5byte, err = ioutil.ReadFile(md5File)
		_md5 = string(_md5byte)
		if (err != nil) {
			panic(err)
		}
	}

	if _, err := os.Stat(geoipdata); err != nil {
		err = geoip.downloadData(geoipdata)
		if (err != nil) {
			panic(err)
		}
	}

	_geoip, err := geoip2.Open(geoipdata)
	if err != nil {
		panic(err)
	}

	geoip.md5 = strings.TrimRight(_md5, "\n")
	geoip.geoip = _geoip

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
	var iAfter int;
	if (IsDev()) {
		iAfter = 600
	} else {
		iAfter = 3600
	}
	seelog.Debugf("Check the update of geoip will run after %vs", iAfter)
	time.AfterFunc(time.Duration(iAfter) * time.Second, func() {
		defer g.Updata()

		seelog.Debug("======================  Start check geoip data update  ======================")

		//获取data的MD5文件匹配
		sNewMd5, err := g.getNewMd5()
		if err != nil {
			return
		}
		seelog.Debugf("The new md5: %v, old md5: %v, match: %v", sNewMd5, g.md5, sNewMd5 == g.md5)

		if sNewMd5 != g.md5 {
			//下载data文件
			root, _ := os.Getwd()
			sTmp := root + "/GeoLite2-City.mmdb.up"
			err := g.downloadData(sTmp)
			if err != nil {
				return
			}

			//data md5
			dataMd5 := g.getDataMd5(sTmp)

			if dataMd5 != sNewMd5 {
				os.Remove(sTmp)
				seelog.Errorf("Download geoip data file md5 is not match: %v != %v", dataMd5, sNewMd5)
				return
			}

			sBakFile := g.sDataFile + ".bak"
			if err := os.Rename(g.sDataFile, sBakFile); err != nil {
				seelog.Errorf("Move old data file to bak error: %v", err.Error())
				return
			}

			if err := os.Rename(sTmp, g.sDataFile); err != nil {
				seelog.Errorf("Move new data file error: %v", err.Error())
				if err := os.Rename(sBakFile, g.sDataFile); err != nil {
					seelog.Errorf("Recover old data file error: %v", err.Error())
				}
				return
			}

			_geoip, err := geoip2.Open(g.sDataFile)
			if err != nil {
				seelog.Errorf("Load new data file error: ", err.Error())
				if err := os.Rename(sBakFile, g.sDataFile); err != nil {
					seelog.Errorf("Recover old data file error: %v", err.Error())
				}
				return
			}

			//写入新的md5
			g.writeMd5File(sNewMd5)
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

func (g *GeoIp)downloadData(toFile string) error {
	seelog.Debug("Start download geoip data")
	//下载data文件
	res, err := http.Get(GEOIP_DATA_URL)
	defer res.Body.Close()
	if err != nil {
		seelog.Errorf("Get geoip data file error: %v", err)
		return err
	}

	root, _ := os.Getwd()

	sGzTmp := root + "/GeoLite2-City.mmdb.gz"
	gf, err := os.Create(sGzTmp)
	defer gf.Close()
	if err != nil {
		seelog.Errorf("Get geoip data file error: %v", err)
		return err
	}
	defer os.Remove(sGzTmp)

	io.Copy(gf, res.Body)
	seelog.Debug("Download geoip data finished")

	// gzip read
	gfr, err := os.Open(sGzTmp)
	defer gfr.Close()
	if err != nil {
		seelog.Errorf("Open geoip data gz file error: %v", err)
		return err
	}

	gzr, err := gzip.NewReader(gfr)
	defer gzr.Close()
	if err != nil {
		seelog.Errorf("Gzip geoip data file error: %v", err)
		return err
	}

	f, err := os.Create(toFile)
	defer f.Close()
	if err != nil {
		seelog.Errorf("Creat geoip data file error: %v", err)
		return err
	}

	io.Copy(f, gzr)
	seelog.Debug("Decompression geoip data finished")

	return nil
}

func (g *GeoIp)getNewMd5() (string, error) {
	seelog.Debug("Start get geoip md5")
	//获取data的MD5文件
	res, err := http.Get(GEOIP_MD5_URL)
	defer res.Body.Close()
	if (err != nil) {
		seelog.Errorf("Get geoip md5 file error: %v", err)
		return "", err
	}

	_md5, err := ioutil.ReadAll(res.Body)
	if err != nil {
		seelog.Errorf("Get geoip md5 file error: %v", err)
		return "", err
	}
	seelog.Debugf("Get geoip md5: %s", _md5)
	return string(_md5), nil
}

func (g *GeoIp)writeMd5File(sNewMd5 string) {
	if err := ioutil.WriteFile(g.sMd5File, []byte(sNewMd5), 0644); err != nil {
		seelog.Errorf("Write new md5 error: %v", err);
	}
}

func (g *GeoIp)getDataMd5(dateFile string) string {
	// data read
	fr, err := os.Open(dateFile)
	defer fr.Close()
	if err != nil {
		seelog.Errorf("Open geoip data file error: %v", err)
		return ""
	}

	//检查md5 是否一致
	md5h := md5.New()
	io.Copy(md5h, fr)
	dataMd5 := hex.EncodeToString(md5h.Sum(nil))

	seelog.Debugf("Geoip data md5 is %v", dataMd5)

	return dataMd5
}
