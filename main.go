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
)

func main() {
	addr := flag.String("addr", ":9090", "http service address")
	socket := flag.String("socket", "/tmp/log-stock.socket", "Listen socket address")
	geoipdata := flag.String("geoip", "./GeoLite2-City.mmdb", "GeoIp data file path")
	geoipdatamd5 := flag.String("md5", "./GeoLite2-City.md5", "GeoIp data md5 file path")
	level := flag.String("level", "debug", "Logger level")
	sInfoFile := flag.String("info", "./info.log", "Info log file")
	sErrorFile := flag.String("error", "./error.log", "Error log file")
	sPanicFile := flag.String("panic", "./panic.dump", "Panic log file")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	//init logger
	newLogger, err := seelog.LoggerFromConfigAsString(
		"<seelog minlevel=\"" + *level + "\">" +
			"<outputs formatid=\"main\">" +
				"<console />" +
				"<filter levels=\"info\">" +
					"<rollingfile type=\"size\" filename=\"" + *sInfoFile + "\" maxsize=\"10485760\" maxrolls=\"2\" />" +
				"</filter>" +
				"<filter levels=\"warn,error\">" +
					"<rollingfile type=\"date\" filename=\"" + *sErrorFile + "\" datepattern=\"2006.01.02\" />" +
				"</filter>" +
				"<filter levels=\"critical\">" +
					"<rollingfile type=\"size\" filename=\"" + *sPanicFile + "\" maxsize=\"10485760\" />" +
				"</filter>" +
			"</outputs>" +
			"<formats>" +
				"<format id=\"main\" format=\"[%Date %Time][%Level]: %Msg%n\"/>" +
			"</formats>" +
		"</seelog>")
	if err != nil {
		panic(err)
	}
	seelog.ReplaceLogger(newLogger);
	defer seelog.Flush()

	defer util.PanicExit()

	seelog.Debug("Server begin to start")

	//init geoip
	geoip := util.InitGeoip(*geoipdata, *geoipdatamd5)
	defer geoip.Close()

	// close redis connect
	defer util.GetRedis().Close()

	//websocket 连接的客户端集合
	hub := connector.NewHub()
	hub.Run()

	//local socket
	oLocalSocket := connector.NewSocket(*socket)
	defer oLocalSocket.Stop()

	// websocket listen
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
	userSet := connector.NewUserSet("dw_online_user", hub)
	defer userSet.Dump()
	userSet.Run()

	//开始处理socket数据
	msgWorkers := map[string]connector.MessageWorker{
		connector.LOG_TYPE_ONLINE_USER: {P: &connector.OnlineUserMessage{UserSet: userSet}},
		connector.LOG_TYPE_NORMAL: {P: &connector.BaseMessage{Hub:hub}},
	}
	oLocalSocket.Listen(msgWorkers)

	chSig := make(chan os.Signal)
	signal.Notify(chSig, os.Interrupt)
	signal.Notify(chSig, os.Kill)
	signal.Notify(chSig, syscall.SIGTERM)

	for {
		select {
		case <-chSig:
			return
		case <-time.After(60 * time.Second):
		/**
		HeapSys：程序向应用程序申请的内存
		HeapAlloc：堆上目前分配的内存
		HeapIdle：堆上目前没有使用的内存
		Alloc : 已经被配并仍在使用的字节数
		NumGC : GC次数
		HeapReleased：回收到操作系统的内存

			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			seelog.Debugf("%d,%d,%d,%d,%d,%d\n", m.HeapSys, m.HeapAlloc, m.HeapIdle, m.Alloc, m.NumGC, m.HeapReleased)
		*/
		}
	}

}
