package main

import (
	"flag"
	"net/http"
	"net"
	"os"
	"runtime"
	"bufio"
	"io"
	"os/signal"
	"syscall"
	"fmt"
	"github.com/bupt1987/log-websocket/connector"
	"github.com/cihub/seelog"
	"github.com/oschwald/geoip2-golang"
	"encoding/json"
	"github.com/bupt1987/log-websocket/analysis"
)

var (
	addr = flag.String("addr", ":9090", "http service address")
	socket = flag.String("socket", "/tmp/log-stock.socket", "Listen socket address")
	geoipdata = flag.String("geoipdata", "./GeoLite2-City.mmdb", "Listen socket address")
	hub = connector.NewHub()
)

func init() {
	newLogger, err := seelog.LoggerFromConfigAsString(
		"<seelog>" +
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

	geoip, err := geoip2.Open(*geoipdata)
	if err != nil {
		seelog.Error("load geoip data error:", err.Error())
		return
	}
	defer geoip.Close()

	go analysis.Run()

	chConn := make(chan net.Conn, runtime.NumCPU())
	chSig := make(chan os.Signal)
	signal.Notify(chSig, os.Interrupt)
	signal.Notify(chSig, os.Kill)
	signal.Notify(chSig, syscall.SIGTERM)

	go hub.Run()

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

	// local socket listen
	go func() {
		//监听
		listen, err := net.Listen("unix", *socket)
		if err != nil {
			panic(err)
		}

		if err := os.Chmod(*socket, 0777); err != nil {
			panic(err)
		}
		go func() {
			for {
				conn, err := listen.Accept()
				if err != nil {
					seelog.Error("connection error:", err)
					continue
				}
				chConn <- conn
			}
		}()

		for {
			select {
			case conn := <-chConn:
				go func() {
					defer conn.Close()
					reader := bufio.NewReader(conn)
					for {
						data, err := reader.ReadBytes('\n')
						if len(data) > 0 {
							var message = connector.FormatMsg(data)
							if (message == nil) {
								continue
							}

							if string(message[0]) == "online_user" {
								res := analysis.UserLog{}
								json.Unmarshal(message[1], &res)
								ip := net.ParseIP(res.Ip)
								city, err := geoip.City(ip)
								var iso = ""
								var name = ""
								if err != nil {
									seelog.Error("geoip error: ", err.Error())
								} else {
									iso = city.Country.IsoCode
									name = city.Subdivisions[0].Names["en"]
								}
								analysis.ActiveUser(&analysis.OnlineUser{
									Uid: res.Uid,
									Ip: res.Ip,
									CountryIsoCode: iso,
									CountryName: name,
									Time: res.Time,
								})
							} else {
								hub.Broadcast <- message
							}
						}
						if err != nil {
							if err != io.EOF {
								seelog.Error("read log error:", err.Error())
							}
							break
						}
					}
				}()
			}
		}
	}()

	seelog.Info("Push running...")

	for {
		select {
		case <-chSig:
			if err := os.Remove(*socket); err != nil {
				panic(err)
			}
			seelog.Info("Push stoped")
			return
		}
	}

}
