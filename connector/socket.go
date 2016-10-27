package connector

import (
	"net"
	"os"
	"github.com/cihub/seelog"
	"bufio"
	"io"
	"github.com/bupt1987/log-websocket/util"
)

type Socket struct {
	chConn chan *net.Conn
	socket string
	listen net.Listener
}

func (l *Socket) Listen(workers map[string]MessageWorker) {
	go func() {
		seelog.Info("Push running...")
		for {
			select {
			case conn := <-l.chConn:
				go func() {
					defer (*conn).Close()
					reader := bufio.NewReader(*conn)
					for {
						data, err := reader.ReadBytes(MESSAGE_NEW_LINE_BYTE)
						if len(data) > 0 {
							msg := FormatMsg(data)
							if (msg == nil) {
								continue
							}
							if _, ok := workers[msg.Category]; !ok {
								ProcessMsg(workers["*"], msg, conn);
							} else {
								ProcessMsg(workers[msg.Category], msg, conn);
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
}

func (l *Socket) Stop() {
	l.listen.Close()
	seelog.Info("Push stoped")
}

func NewSocket(socket string) *Socket {
	defer util.PanicExit()

	//监听
	listen, err := net.Listen("unix", socket)
	if err != nil {
		panic(err)
	}

	if err := os.Chmod(socket, 0666); err != nil {
		panic(err)
	}

	l := &Socket{
		chConn : make(chan *net.Conn, 128),
		socket: socket,
		listen: listen,
	}

	go func() {
		for {
			conn, err := listen.Accept()
			if err != nil {
				seelog.Error("connection error:", err)
				continue
			}
			l.chConn <- &conn
			seelog.Debugf("new connect %v", conn.LocalAddr())
		}
	}()

	return l
}
