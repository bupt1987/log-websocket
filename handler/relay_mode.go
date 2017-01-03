package handler

import (
	"github.com/cihub/seelog"
	"net"
	"github.com/bupt1987/log-websocket/util"
	"net/url"
	"net/http"
	"github.com/gorilla/websocket"
	"time"
	"strings"
	"math"
	"github.com/bupt1987/log-websocket/connector"
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
	c.connect(1)
}

func (c *WsRelayClient) connect(retry int) {
	if c.conn != nil {
		return
	}

	defer func() {
		util.PanicExit()
	}()

	var conn *websocket.Conn;
	var err error

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = time.Millisecond * 100

	for i := 1; i <= retry || retry < 0; i++ {
		conn, _, err = dialer.Dial(c.url, c.header)
		if err != nil {
			if (i % 100 == 1) {
				seelog.Errorf("No.%d Can not connect master: %v", i, err)
			}

			sleep := time.Duration(math.Min((math.Floor(float64(i / 100)) + 1) * 100, 10000))
			time.Sleep(time.Millisecond * sleep)
			continue
		}

		c.conn = conn
		c.conn.SetReadLimit(connector.MaxMessageSize)
		c.conn.SetReadDeadline(time.Now().Add(connector.PongWait))
		c.conn.SetPongHandler(func(string) error {
			if util.IsDev() {
				seelog.Debug("Get pong msg")
			}
			c.conn.SetReadDeadline(time.Now().Add(connector.PongWait))
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
		ticker := time.NewTicker(connector.PingPeriod)
		defer func() {
			ticker.Stop()
			c.conn.Close()
			c.conn = nil
			c.connect(-1)
		}()
		for {
			select {
			case message := <-c.send:
				if (message == nil) {
					continue
				}

				c.conn.SetWriteDeadline(time.Now().Add(connector.WriteWait))
				w, err := c.conn.NextWriter(websocket.TextMessage)
				if err != nil {
					seelog.Errorf("Get ws writer error: %v", err)
					return
				}

				w.Write(message)

				if util.IsDev() {
					seelog.Debugf("Send msg to master: %s", strings.TrimRight(string(message), connector.MESSAGE_NEW_LINE_STRING))
				}

				if err := w.Close(); err != nil {
					seelog.Errorf("close write error: %v", err)
					return
				}
			case <-c.close:
				return
			case <-ticker.C:
				c.conn.SetWriteDeadline(time.Now().Add(connector.WriteWait))
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

	seelog.Infof("Master %s connected", c.url)
}

type RelayMode struct {
	Client *WsRelayClient
}

func (m *RelayMode) Process(msg *connector.Msg, conn net.Conn) {
	//todo 以后优化如果master连接断掉之后还有消息输入时消息如何处理, 是直接抛弃还是记录下来, 因为现在只有online类型的数据,如果
	if m.Client.conn == nil {
		seelog.Error("Master not connect, ignore msg");
		return
	}
	m.Client.send <- connector.RevertMsg(msg)
}
