// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"net/http"
	"net"
	"os"
	"runtime"
	"bufio"
	"io"
	"os/signal"
	"syscall"
	"fmt"

	"github.com/bupt1987/log-websockt/connector"
)

var addr = flag.String("addr", ":9090", "http service address")
var socket = flag.String("socket", "/tmp/log-stock.socket", "Listen socket address")
var hub = connector.NewHub()

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

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
			log.Fatal("ListenAndServe: ", err)
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
		for {
			conn, err := listen.Accept()
			if err != nil {
				fmt.Println("connection error:", err)
				continue
			}
			chConn <- conn
		}
	}()

	for {
		select {
		case <-chSig:
			if err := os.Remove(*socket); err != nil {
				panic(err)
			}
			return
		case conn := <-chConn:
			go func() {
				defer conn.Close()
				reader := bufio.NewReader(conn)
				for {
					data, err := reader.ReadBytes('\n')
					if len(data) > 0 {
						hub.Broadcast <- data
					}
					if err != nil {
						if err != io.EOF {
							fmt.Println("read log error:", err.Error())
						}
						break
					}
				}
			}()
		}
	}

}
