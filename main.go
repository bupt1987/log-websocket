package main

import (
	"flag"
	"net/http"
	"os"
	"fmt"
	"syscall"
	"os/signal"
	"time"
	"github.com/bupt1987/log-websocket/connector"
	"github.com/bupt1987/log-websocket/util"
	"github.com/cihub/seelog"
	"runtime"
	"github.com/bupt1987/log-websocket/analysis"
	"github.com/bupt1987/log-websocket/controller"
)

const (
	CLIENT_MODE_ALL = "all"
	CLIENT_MODE_MASTER = "master"
	CLIENT_MODE_RELAY = "relay"
)

func main() {
	addr := flag.String("addr", ":9090", "http service address")
	masterAddr := flag.String("master", "127.0.0.1:9090", "http service address")
	socket := flag.String("socket", "/tmp/log-stock.socket", "Listen socket address")
	geoipdata := flag.String("geoip", "./_tmp/GeoLite2-City.mmdb", "GeoIp data file path")
	geoipdatamd5 := flag.String("md5", "./_tmp/GeoLite2-City.md5", "GeoIp data md5 file path")
	sDumpPath := flag.String("dump", "./_tmp/", "Dump file path")
	sLoggerConfig := flag.String("log", "./logger.xml", "log config file")
	mode := flag.String("mode", CLIENT_MODE_ALL, "Run model: master or relay, all")
	accessToken := flag.String("access_token", "oQjcVqVIWYx81YW1wc6CbQf0ZUOqcENn", "websocket access token")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	//init logger
	newLogger, err := seelog.LoggerFromConfigAsFile(*sLoggerConfig)
	if err != nil {
		panic(err)
	}
	defer util.PanicExit()

	seelog.ReplaceLogger(newLogger);
	defer seelog.Flush()

	// close redis connect
	defer util.GetRedis().Close()

	//init geoip
	geoip := util.InitGeoip(*geoipdata, *geoipdatamd5)
	defer geoip.Close()

	var oLocalSocket *connector.Socket;
	bInitLocalSocket := *mode == CLIENT_MODE_RELAY || *mode == CLIENT_MODE_ALL

	if (bInitLocalSocket) {
		//local socket
		oLocalSocket = connector.NewSocket(*socket)
		defer oLocalSocket.Stop()
	}

	// msg worker
	msgWorkers := make(map[string]connector.MsgWorker)

	if (*mode != CLIENT_MODE_RELAY) {
		//websocket 连接的客户端集合
		hub := connector.NewWsGroup()
		hub.Run()

		// websocket listen
		connector.SetAccessToken(*accessToken)
		go func() {
			defer util.PanicExit()
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				connector.ServeWs(hub, w, r)
			})
			err := http.ListenAndServe(*addr, nil)
			if err != nil {
				panic(err)
			}
		}()

		// 在线用户
		userSet := controller.NewUserSet("dw_online_user", *sDumpPath, hub)
		defer userSet.Dump()
		defer analysis.PushSessionImmediately()
		userSet.Run()

		msgWorkers = map[string]connector.MsgWorker{
			controller.ANY: {P: &controller.Base{Group:hub}},
			controller.ONLINE_USER: {P: &controller.OnlineUser{UserSet: userSet}},
			controller.IP_TO_ISO: {P:&controller.IpToIso{}},
		}
	} else {
		// relay mode
		wsClient := controller.NewRelay(*masterAddr, *accessToken)
		wsClient.Listen()

		msgWorkers = map[string]connector.MsgWorker{
			controller.ANY: {P: &controller.RelayMode{Client: wsClient}},
			controller.IP_TO_ISO: {P:&controller.IpToIso{}},
		}
	}

	connector.SetSocketMsgWorker(msgWorkers)

	//开始处理socket数据
	if (bInitLocalSocket) {
		oLocalSocket.Listen()
	}

	chSig := make(chan os.Signal)
	signal.Notify(chSig, os.Interrupt)
	signal.Notify(chSig, os.Kill)
	signal.Notify(chSig, syscall.SIGTERM)

	for {
		select {
		case <-chSig:
			return
		case <-time.After(60 * time.Second):
			if util.IsDev() && *mode == CLIENT_MODE_MASTER {
				/**
				HeapSys：程序向应用程序申请的内存
				HeapAlloc：堆上目前分配的内存
				HeapIdle：堆上目前没有使用的内存
				Alloc : 已经被配并仍在使用的字节数
				NumGC : GC次数
				HeapReleased：回收到操作系统的内存
				*/
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				seelog.Debugf("MemStat => HeapSys: %d, HeapAlloc: %d, HeapIdle: %d, Alloc: %d, NumGC: %d, HeapReleased: %d",
						m.HeapSys, m.HeapAlloc, m.HeapIdle, m.Alloc, m.NumGC, m.HeapReleased)
			}
		}
	}

}
