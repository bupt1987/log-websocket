package connector

import (
	"net"
	"os"
	"github.com/cihub/seelog"
	"bufio"
	"io"
	"runtime"
)

type Socket struct {
	chConn  chan net.Conn
	socket  string
	workers map[string]MessageWorker
}

func (l *Socket) Listen() {
	//监听
	listen, err := net.Listen("unix", l.socket)
	if err != nil {
		panic(err)
	}

	if err := os.Chmod(l.socket, 0777); err != nil {
		panic(err)
	}
	go func() {
		for {
			conn, err := listen.Accept()
			if err != nil {
				seelog.Error("connection error:", err)
				continue
			}
			l.chConn <- conn
		}
	}()

	seelog.Info("Push running...")

	for {
		select {
		case conn := <-l.chConn:
			go func() {
				defer conn.Close()
				reader := bufio.NewReader(conn)
				for {
					data, err := reader.ReadBytes('\n')
					if len(data) > 0 {
						msg := FormatMsg(data)
						if (msg == nil) {
							continue
						}
						if _, ok := l.workers[msg.Category]; !ok {
							ProcessMsg(l.workers["*"], msg);
						} else {
							ProcessMsg(l.workers[msg.Category], msg);
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
}

func NewSocket(socket string, workers map[string]MessageWorker) *Socket {
	chConn := make(chan net.Conn, runtime.NumCPU())
	return &Socket{
		chConn : chConn,
		socket: socket,
		workers: workers,
	}
}
