package connector

import (
	"time"
	"github.com/gorilla/websocket"
	"github.com/cihub/seelog"
)

const (
	// Time allowed to write a message to the peer.
	WriteWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	PongWait = 60 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	PingPeriod = (PongWait * 9) / 10
	// Maximum message size allowed from peer.
	MaxMessageSize = 1024 * 1024
)

type Client struct {
	mode    string
	listens []string
	hub     *Hub
	conn    *websocket.Conn
	send    chan []byte
}

func (c *Client) write(mt int, payload []byte) error {
	c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
	return c.conn.WriteMessage(mt, payload)
}

func (c *Client) listen() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(MaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(PongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(PongWait))
		return nil
	})
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				seelog.Errorf("close error: %v", err)
			}
			break
		}
		msg := FormatMsg(data)
		if (msg == nil) {
			continue
		}
		if (c.mode == CLIENT_MODE_RELAY) {
			SocketProcessMsg(msg, nil)
		} else {
			c.hub.Broadcast <- msg
		}
	}
}

func (c *Client) push() {
	ticker := time.NewTicker(PingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}

			c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

