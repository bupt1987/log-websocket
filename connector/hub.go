package connector

import (
	"runtime"
	"github.com/cihub/seelog"
)

type Hub struct {
	num        int64
	clients    map[*Client]bool
	listens    map[string]map[*Client]bool
	Broadcast  chan [][]byte
	register   chan *Client
	unregister chan *Client
}

var comma = []byte{','}

func NewHub() *Hub {
	return &Hub{
		num: 0,
		Broadcast:  make(chan [][]byte, runtime.NumCPU()),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		listens:    make(map[string]map[*Client]bool),
	}
}

func (h *Hub) push(client *Client, msg []byte) {
	defer func() {
		recover()
	}()
	client.send <- msg
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			for _, listen := range client.listens {
				if len(listen) == 0 {
					continue
				}
				if _, ok := h.listens[listen]; !ok {
					h.listens[listen] = make(map[*Client]bool)
				}
				h.listens[listen][client] = true
			}
			h.num ++;
			seelog.Infof("%s connected, listen : %v, total connected: %v", client.conn.RemoteAddr(), client.listens, h.num)
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				func() {
					defer func() {
						recover()
					}()
					for _, listen := range client.listens {
						if len(listen) == 0 {
							continue
						}
						delete(h.listens[listen], client)
					}
					delete(h.clients, client)
					close(client.send)
				}()
				h.num --;
				seelog.Infof("%s close, total connected: %v", client.conn.RemoteAddr(), h.num)
			}
		case message := <-h.Broadcast:
			go func() {
				if string(message[0]) != "*" {
					if listens, ok := h.listens[string(message[0])]; ok {
						for client := range listens {
							h.push(client, message[1])
						}
					}
					//再给监听所有的客户端发送数据
					if listens, ok := h.listens["*"]; ok {
						for client := range listens {
							h.push(client, message[1])
						}
					}
				} else {
					for client := range h.clients {
						h.push(client, message[1])
					}
				}
			}()
		}
	}
}
