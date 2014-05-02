package main

import (
        "code.google.com/p/go.net/websocket"
	"time"
)

type Hub struct {
	Connections map[*Socket]bool
	Pipe chan HometickerJson
}

type  HometickerJson struct {
        Title string
        Icon string
        Color string
        Message string
}

func (h *Hub) Broadcast() {
	for {
		select {
		case str := <-h.Pipe:
			for s, _ := range h.Connections {
				err := websocket.JSON.Send(s.Ws, str)
				if err != nil {
					s.Ws.Close()
					delete(h.Connections, s)
				}
			}
		}
	}
}

type Socket struct {
	Ws *websocket.Conn
}

func (s *Socket) ReceiveMessage() {
	for {
		var x []byte
		err := websocket.Message.Receive(s.Ws, &x)
		if err != nil {
			break
		}
	}
	s.Ws.Close()
}

var htj = HometickerJson{"Welcome", "fa-thumbs-up", "primary", "Connected to datanomics."}
func homeTickerHandler(ws *websocket.Conn) {
//        fmt.Fprintf(ws, "hello")
	h.Pipe <- HometickerJson{"Client Connected", "fa-smile-o", "warning", "Address " + string(ws.Request().RemoteAddr) + " joined the party."}
	time.Sleep(100 * time.Millisecond) // This way the new client won't receive the message above (which is async, so it is delayed a bit).
	s := &Socket{ws}
	h.Connections[s] = true
	websocket.JSON.Send(s.Ws, htj)
	s.ReceiveMessage() // Only way to keep socket open?
	debug(string(ws.Request().RemoteAddr) + " connected to hometicker websocket")
}



















