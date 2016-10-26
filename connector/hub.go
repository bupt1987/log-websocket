package connector

import (
	"runtime"
	"github.com/cihub/seelog"
)

type Hub struct {
	num        int64
	clients    map[*Client]bool
	listens    map[string]map[*Client]bool
	Broadcast  chan *Msg
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		num: 0,
		Broadcast:  make(chan *Msg, runtime.NumCPU()),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		listens:    make(map[string]map[*Client]bool),
	}
}

func (h *Hub) push(client *Client, msg []byte) {
	defer func() {
		if err := recover(); err != nil {
			seelog.Error("Hub.push error: ", err);
		}
	}()
	client.send <- msg
}

func (h *Hub) Run() {
	go func() {
		defer PanicHandler()
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
			case msg := <-h.Broadcast:
				go func() {
					if msg.Category != "*" {
						if listens, ok := h.listens[msg.Category]; ok {
							for client := range listens {
								h.push(client, msg.Data)
							}
						}
						//再给监听所有的客户端发送数据
						if listens, ok := h.listens["*"]; ok {
							for client := range listens {
								h.push(client, msg.Data)
							}
						}
					} else {
						for client := range h.clients {
							h.push(client, msg.Data)
						}
					}
				}()
			}
		}
	}()
}
