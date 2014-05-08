package main

import (
        "code.google.com/p/go.net/websocket"
	"time"
//	"regexp"
)

type Hub struct {
	Connections map[*Socket]bool
	Pipe chan Hometicker
}

type Hometicker struct {
        Title string
        Icon string
        Color string
        Message string
}

// ---------------- READ TO IMPLEMENT
// Sensors ws will be at /_sensorticker
// When someone connects, he should send the sensor he is interested in.
// If the sensor exists we add him to SuperHub.Connections[sensor]
// When a conn close, if the sensor has 0 listeners, delete him from SuperHub.Connections
// -----------------------------------

type SuperHub struct {
	Connections map[string]map[*Socket]bool
	Pipe chan Sensorticker
}

type Sensorticker struct {
	Sensor string
	Value string
	Timestamp string
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
					// Note for sensorticker implementation: here may go code that deletes the hub when it is empty
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

var htj = Hometicker{"Welcome", "fa-thumbs-up", "primary", "Connected to datanomics."}

func homeTickerHandler(ws *websocket.Conn) {
//        fmt.Fprintf(ws, "hello")
	h.Pipe <- Hometicker{"Client Connected", "fa-smile-o", "warning", "Address " + string(ws.Request().RemoteAddr) + " joined the party."}
	time.Sleep(100 * time.Millisecond) // This way the new client won't receive the message above (which is async, so it is delayed a bit).
	s := &Socket{ws}
	h.Connections[s] = true
	websocket.JSON.Send(s.Ws, htj)
	s.ReceiveMessage() // Only way to keep socket open?
	debug(string(ws.Request().RemoteAddr) + " connected to hometicker websocket")
}

// var validSensorWS = regexp.MustCompile("^/sws/([a-zA-Z0-9-]+)/?$")

// func sensorTickerHandler(ws *websocket.Conn) {
// 	m := validSensorWS.FindStringSubmatch(ws.Path)
//         if len(m) == 0 {
// 		ws.Close()
//                 return
//         }
// 	if ! d.Exists(m[1]) {
// 		ws.Close()
// 		return
// 	}
// 	s := &Socket{ws}
// }



















