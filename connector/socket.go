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
	chClosed chan int
	chClose  chan int
	chConn   chan *net.Conn
	socket   string
	listen   net.Listener
	startListen bool
}

var msgWorkers = make(map[string]MessageWorker)

func SetMsgWorker(workers map[string]MessageWorker) {
	msgWorkers = workers
}

func (l *Socket) Listen() {
	l.startListen = true
	go func() {
		seelog.Info("Local socket listening...")
		for {
			conn, err := l.listen.Accept()
			if err != nil {
				select {
				case <-l.chClose:
					l.chClosed <- 1
					return
				default:
					seelog.Errorf("%v connection error: ", err.Error())
					continue
				}
			}

			go func() {
				defer conn.Close()
				reader := bufio.NewReader(conn)
				for {
					data, err := reader.ReadBytes(MESSAGE_NEW_LINE_BYTE)
					if len(data) > 0 {
						msg := FormatMsg(data)
						if (msg == nil) {
							continue
						}
						SocketProcessMsg(msg, conn)
					}
					if err != nil {
						if err != io.EOF {
							seelog.Errorf("Read log error: %s", err)
						}
						break
					}
				}
			}()
		}
	}()
}

func (l *Socket) Stop() {
	if (l.startListen) {
		l.chClose <- 1
	}
	l.listen.Close()
	if (l.startListen) {
		<-l.chClosed
	}
	l.startListen = false
	seelog.Info("Local socket listen stoped")
}

var oSocket *Socket

func SocketProcessMsg(msg *Msg, conn net.Conn) {
	if _, ok := msgWorkers[msg.Category]; !ok {
		ProcessMsg(msgWorkers["*"], msg, conn);
	} else {
		ProcessMsg(msgWorkers[msg.Category], msg, conn);
	}
}

func NewSocket(socket string) *Socket {

	if oSocket != nil {
		return oSocket
	}

	defer util.PanicExit()

	//监听
	listen, err := net.Listen("unix", socket)
	if err != nil {
		panic(err)
	}

	if err := os.Chmod(socket, 0666); err != nil {
		panic(err)
	}

	oSocket = &Socket{
		chClosed: make(chan int, 1),
		chClose: make(chan int, 1),
		chConn : make(chan *net.Conn, 128),
		socket: socket,
		listen: listen,
	}

	return oSocket
}
