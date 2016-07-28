package connector

import (
	"fmt"
	"runtime"
)

type Hub struct {
	num        int64
	clients    map[*Client]bool
	Broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		num: 0,
		Broadcast:  make(chan []byte, runtime.NumCPU()),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			h.num ++;
			fmt.Printf("%s connected, total: %v\n", client.conn.RemoteAddr(), h.num)
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				func() {
					defer func() {
						recover()
					}()
					delete(h.clients, client)
					close(client.send)
				}()
				h.num --;
				fmt.Printf("%s close, total: %v\n", client.conn.RemoteAddr(), h.num)
			}
		case message := <-h.Broadcast:
			go func() {
				for client := range h.clients {
					func() {
						defer func() {
							recover()
						}()
						client.send <- message
					}()
				}
			}()
		}
	}
}
