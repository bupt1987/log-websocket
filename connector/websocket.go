package connector

import (
	"github.com/gorilla/websocket"
	"net/http"
	"github.com/cihub/seelog"
	"strings"
)

const (
	CLIENT_MODE_RELAY = "relay"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var wsAuthKey = "oQjcVqVIWYx81YW1wc6CbQf0ZUOqcENn";

func SetAccessToken(accessToken string) {
	seelog.Infof("access token : %s", accessToken)
	wsAuthKey = accessToken
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		conn.Close();
		seelog.Error(err)
		return
	}

	r.ParseForm()
	authKey := r.Header.Get("access_token")
	if (authKey == "") {
		authKey = r.Form.Get("access_token")
	}

	if (authKey != wsAuthKey) {
		conn.Close();
		seelog.Errorf("WS access_token error: %s", authKey)
		return
	}

	mode := r.Header.Get("client_mode")
	listens := ""

	if (mode != CLIENT_MODE_RELAY) {
		listens = strings.TrimSpace(r.Form.Get("listens"))
		if len(listens) == 0 {
			seelog.Errorf("WS listens is empty: %v", conn.LocalAddr())
			conn.Close()
			return
		}
		//如果listens里面有*的话则只保留*
		var checkListens = "," + listens + ",";
		if checkListens != ",*," && strings.Index(checkListens, ",*,") != -1 {
			listens = "*"
		}
	}

	client := &Client{
		mode: mode,
		listens: strings.Split(listens, ","),
		hub: hub,
		conn: conn,
		send: make(chan []byte, 256),
	}
	hub.register <- client
	go client.push()
	client.listen()
}
