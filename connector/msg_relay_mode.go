package connector

import (
	"github.com/cihub/seelog"
	"net"
	"github.com/bupt1987/log-websocket/util"
	"net/url"
	"net/http"
	"github.com/gorilla/websocket"
	"time"
	"strings"
)

const (
	RELAY_SEND_MAX_LENGTH = 1024
)

type WsRelayClient struct {
	url    string
	header http.Header
	conn   *websocket.Conn
	send   chan []byte
	close  chan int
}

func NewRelay(addr string, access_token string) *WsRelayClient {
	header := make(http.Header);
	header.Add("access_token", access_token);
	header.Add("client_mode", "relay")

	u := url.URL{Scheme: "ws", Host: addr, Path: "/"}
	return &WsRelayClient{
		header: header,
		url: u.String(),
		send: make(chan []byte, RELAY_SEND_MAX_LENGTH),
		close: make(chan int, 1),
	}
}

func (c *WsRelayClient) Listen() {
	c.connect(3)
}

func (c *WsRelayClient) connect(retry int) {
	if c.conn != nil {
		return
	}

	defer util.PanicExit()

	var conn *websocket.Conn;
	var err error

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = time.Millisecond * 100

	for i := 0; i < retry; i++ {
		conn, _, err = dialer.Dial(c.url, c.header)
		if err != nil {
			time.Sleep(time.Second * 1)
			seelog.Errorf("Can not connect master: %v", err)
			continue
		}

		c.conn = conn
		c.conn.SetReadLimit(maxMessageSize)
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.conn.SetPongHandler(func(string) error {
			if util.IsDev() {
				seelog.Debug("Get pong msg")
			}
			c.conn.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})
		break
	}

	if err != nil {
		panic(err)
	}

	go func() {
		for {
			if (c.conn == nil) {
				time.Sleep(time.Second * 1)
				continue
			}
			_, _, err := c.conn.ReadMessage()
			if err != nil {
				seelog.Errorf("Master connect error: %v", err)
				c.close <- 1
				return
			}
		}
	}()

	//创建push携程
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer func() {
			ticker.Stop()
			c.conn.Close()
			c.conn = nil
			c.connect(999999999999)
		}()
		for {
			select {
			case message := <-c.send:
				if (message == nil) {
					continue
				}

				c.conn.SetWriteDeadline(time.Now().Add(writeWait))
				w, err := c.conn.NextWriter(websocket.TextMessage)
				if err != nil {
					seelog.Errorf("Get ws writer error: %v", err)
					return
				}

				w.Write(message)

				if util.IsDev() {
					seelog.Debugf("Send msg to master: %s", strings.TrimRight(string(message), MESSAGE_NEW_LINE_STRING))
				}

				if err := w.Close(); err != nil {
					seelog.Errorf("close write error: %v", err)
					return
				}
			case <-c.close:
				return
			case <-ticker.C:
				c.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := c.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
					seelog.Errorf("send ping error: %v", err)
					return
				}
				if util.IsDev() {
					seelog.Debug("Send ping msg")
				}
			}
		}
	}()

	seelog.Infof("Master connected %s", c.url)
}

type RelayMessageProcesser struct {
	Client *WsRelayClient
}

func (m *RelayMessageProcesser) Process(msg *Msg, conn net.Conn) {
	//todo 以后优化如果master连接断掉之后还有消息输入时消息如何处理, 是直接抛弃还是记录下来, 因为现在只有online类型的数据,如果
	if m.Client.conn == nil {
		seelog.Error("Master not connect, ignore msg");
		return
	}
	m.Client.send <- RevertMsg(msg)
}
