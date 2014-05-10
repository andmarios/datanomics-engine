package main

import (
        "code.google.com/p/go.net/websocket"
	"time"
	"log"
//	"encoding/json"
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

type SensorHub struct {
	Connections map[string]map[*Socket]bool
	Pipe chan string
}

type Sensorticker struct {
	Sensor string
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

type sensorMsg struct {
	C string
	T int64
	V float64
}

func (h *SensorHub) Broadcast() {
	for {
		select {
		case str := <-h.Pipe:
			if d.Exists(str) {
				_, exists := h.Connections[str]
				if exists {
					var m sensorMsg
					v := d.LoadR(str)
					m.C = "u"
					m.T = v.Time
					m.V = v.Value
					for s, _ := range h.Connections[str] {
						err := websocket.JSON.Send(s.Ws, m)
						if err != nil {
							s.Ws.Close()
							delete(h.Connections[str], s)
							if len(h.Connections[str]) == 0 {
								delete(h.Connections, str)
							}
						}
					}
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

type incomingMsg struct {
	Type string
	Sensor string
	Start int64
	End int64
}

func (s *Socket) ReceiveSensorMessage() {
	for {
//		var x []byte
		var rec incomingMsg
                err := websocket.JSON.Receive(s.Ws, &rec)
                if err != nil {
                        break
                }
		if rec.Type == "range" {
			m := sensorMsg{"d", 0, 0}
			err := websocket.JSON.Send(s.Ws, m)
			if err != nil {
				s.Ws.Close()
			}
			if d.Exists(rec.Sensor) {
				t := d.LoadMR(rec.Sensor, rec.Start, rec.End)
				for i, _ := range t {
					_ = websocket.JSON.Send(s.Ws, t[i])
					log.Println(t[i])
				}
			}
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

func sensorTickerHandler(ws *websocket.Conn) {
	//        fmt.Fprintf(ws, "hello")
        h.Pipe <- Hometicker{"Client Connected", "fa-smile-o", "warning", "Address " + string(ws.Request().RemoteAddr) + " started monitoring a sensor."}
	s := &Socket{ws}

	var  x string
	err := websocket.Message.Receive(s.Ws, &x)
	if err != nil {
		log.Println(err)
	}
	if d.Exists(x) {
		_, exists := sh.Connections[x]
		if ! exists {
			t := make(map[*Socket]bool)
			sh.Connections[x] = t
		}
		sh.Connections[x][s] = true
		s.ReceiveSensorMessage() // Only way to keep socket open?
		debug(string(ws.Request().RemoteAddr) + " connected to sensorticker websocket")
	}
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



















