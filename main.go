package main

import (
	"flag"
	"net/http"
	"os"
	"fmt"
	"syscall"
	"os/signal"
	"github.com/bupt1987/log-websocket/connector"
	"github.com/cihub/seelog"
	"time"
)

var addr = flag.String("addr", ":9090", "http service address")
var socket = flag.String("socket", "/tmp/log-stock.socket", "Listen socket address")
var geoipdata = flag.String("geoipdata", "./GeoLite2-City.mmdb", "GeoIp data file path")
var level = flag.String("level", "debug", "Logger level")

func init() {
	//init logger
	newLogger, err := seelog.LoggerFromConfigAsString(
		"<seelog minlevel=\"" + *level + "\">" +
			"<outputs formatid=\"main\">" +
			"<console />" +
			"</outputs>" +
			"<formats>" +
			"<format id=\"main\" format=\"[%Date %Time][%Level] %Msg%n\"/>" +
			"</formats>" +
			"</seelog>")
	if err != nil {
		panic(err)
	}
	seelog.ReplaceLogger(newLogger);
}

func main() {
	defer seelog.Flush()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	//init geoip
	geoip := connector.InitGeoip(*geoipdata)
	defer geoip.Close()

	hub := connector.NewHub()
	go hub.Run()

	userSet := connector.NewUserSet("dw_online_user", hub)
	go userSet.Run()

	msgWorkers := map[string]connector.MessageWorker{
		connector.LOG_TYPE_ONLINE_USER: {P: &connector.OnlineUserMessage{UserSet: userSet}},
		connector.LOG_TYPE_NORMAL: {P: &connector.BaseMessage{Hub:hub}},
	}

	//local socket
	oLocalSocket := connector.NewSocket(*socket, msgWorkers)
	go oLocalSocket.Listen()
	defer oLocalSocket.Stop()

	// websocket listen
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			connector.ServeWs(hub, w, r)
		})
		err := http.ListenAndServe(*addr, nil)
		if err != nil {
			panic(err)
		}
	}()

	chSig := make(chan os.Signal)
	signal.Notify(chSig, os.Interrupt)
	signal.Notify(chSig, os.Kill)
	signal.Notify(chSig, syscall.SIGTERM)

	//dump
	defer userSet.Dump()

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
