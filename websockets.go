package main

import (
        "code.google.com/p/go.net/websocket"
)

type Hub struct {
	Connections map[*Socket]bool
	Pipe chan string
}

func (h *Hub) Broadcast() {
	for {
		select {
		case str := <-h.Pipe:
			for s, _ := range h.Connections {
				err := websocket.Message.Send(s.Ws, string(str))
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
func homeTickerHandler(ws *websocket.Conn) {
//        fmt.Fprintf(ws, "hello")
	s := &Socket{ws}
	h.Connections[s] = true
	websocket.Message.Send(s.Ws, "Welcome to Datanomics.")
	s.ReceiveMessage() // Only way to keep socket open?
}



















