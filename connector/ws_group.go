package connector

import (
	"runtime"
	"github.com/cihub/seelog"
	"github.com/bupt1987/log-websocket/util"
)

type WsGroup struct {
	num        int64
	clients    map[*WsClient]bool
	listens    map[string]map[*WsClient]bool
	Broadcast  chan *Msg
	register   chan *WsClient
	unregister chan *WsClient
}

func NewWsGroup() *WsGroup {
	return &WsGroup{
		num: 0,
		Broadcast:  make(chan *Msg, runtime.NumCPU()),
		register:   make(chan *WsClient),
		unregister: make(chan *WsClient),
		clients:    make(map[*WsClient]bool),
		listens:    make(map[string]map[*WsClient]bool),
	}
}

func (h *WsGroup) push(client *WsClient, msg []byte) {
	defer func() {
		if err := recover(); err != nil {
			seelog.Error("Hub.push error: ", err);
		}
	}()
	client.send <- msg
}

func (h *WsGroup) Run() {
	go func() {
		defer util.PanicExit()
		for {
			select {
			case client := <-h.register:
				h.clients[client] = true
				for _, listen := range client.listens {
					if len(listen) == 0 {
						continue
					}
					if _, ok := h.listens[listen]; !ok {
						h.listens[listen] = make(map[*WsClient]bool)
					}
					h.listens[listen][client] = true
				}
				h.num ++;
				seelog.Infof("%s connected, mode: %s , listen : %v, total connected: %v",
					client.conn.RemoteAddr(), client.mode, client.listens, h.num)
			case client := <-h.unregister:
				if _, ok := h.clients[client]; ok {
					for _, listen := range client.listens {
						if len(listen) == 0 {
							continue
						}
						if _, ok := h.listens[listen]; !ok {
							continue
						}
						delete(h.listens[listen], client)
					}
					delete(h.clients, client)
					close(client.send)
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
							if (client.mode != CLIENT_MODE_RELAY) {
								h.push(client, msg.Data)
							}
						}
					}
				}()
			}
		}
	}()
}
