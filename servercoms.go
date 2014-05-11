package main

import (
	"net"
	"encoding/gob"
	"time"
	"log"
)

type remoteReading struct {
	S string
	V string
	T time.Time
}

func listenForRemoteReadings() {
	ln, err := net.Listen("tcp", ":" + scPort);
	if err != nil {
		log.Println("Could not start remote readings listener.")
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go func() {
			dec := gob.NewDecoder(conn)
			rrr := &[]remoteReading{}
			dec.Decode(rrr)
			for _, rr := range *rrr {
				if ! d.Exists(rr.S) { // Remove when you add code to add/delete sensors instead of adding them automatically.
					h.Pipe <- Hometicker{"New sensor: " + rr.S, "fa-check-circle", "success",
						"Sensor <em>" + rr.S + "</em> succesfully added."}
					d.AddT(rr.S, rr.T) // This is not needed. Sensors are added automatically upon first reading. It is here only to make the next command to work.
					sensorList()
				}
				d.StoreT(rr.S, rr.V, rr.T)
				t := d.Info(rr.S).Name
				h.Pipe <- Hometicker{"<a href='/view/" + rr.S + "'>" + t + "</a>: new reading", "fa-plus-circle", "info",
					t + "</em> sent value <em>" + rr.V + "</em> at <em>" + rr.T.String() + "</em>"}
				sh.Pipe <- rr.S
			}
		}()
	}
}

type SendReadingsCache struct {
	Readings []remoteReading
	Pipe chan remoteReading
}

func (s *SendReadingsCache) SendReadingsCron() {
        ticker := time.NewTicker(time.Duration(sendRemotePeriod) * time.Second)
        go func() {
                for {
                        select {
			case r := <- s.Pipe:
				s.Readings = append(s.Readings, r)
                        case <- ticker.C:
				sendRemoteReading(s.Readings)
				s.Readings = make([]remoteReading, 0, 0)
                        }
                }
        }()
}


func sendRemoteReading(sra []remoteReading) {
	for _, rHost := range remoteServers {
		conn, err := net.Dial("tcp", rHost)
		if err != nil {
			log.Println(err)
		} else {
			encoder := gob.NewEncoder(conn)
			rr := &sra
			encoder.Encode(rr)
			conn.Close()
		}
	}
}










// Nice code that never worked. :/

// type remoteServersQueue struct {
// 	Servers map[string]net.Conn
// }

// var rsq = remoteServersQueue{make(map[string]net.Conn)}

// func remoteServersConnect() {
// 	for _, rHost := range remoteServers {
//                 conn, err := net.Dial("tcp", rHost)
// 		if err != nil {
//                         log.Println(err)
//                 } else {
// 			rsq.Servers[rHost] = conn
// 			go remoteServerReceive(conn, rHost)
// 		}
// 	}
// }

// func remoteServerReceive(conn net.Conn, rh string) {
// 	var buffer []byte
// 	bytesRead, error := conn.Read(buffer)
// 	if error != nil {
// 		log.Println(bytesRead)
// 		conn.Close()
// 		delete(rsq.Servers, rh)
// 	}
// }

// func sendRemoteReading(s string, v string, t time.Time) {
// 	for _, rHost := range remoteServers {
// 		conn, exists := rsq.Servers[rHost]
// 		if exists {
// 			log.Println("sending");
// 			encoder := gob.NewEncoder(conn)
// 			rr := &remoteReading{s, v, t}
// 			encoder.Encode(rr)
// 		} else {
// 			log.Println("opening and sending");
// 			conn, err := net.Dial("tcp", rHost)
// 			if err != nil {
// 				log.Println(err)
// 			} else {
// 				conn.SetReadDeadline(time.Now().Add(120 * time.Second))
// 				rsq.Servers[rHost] = conn
// 				encoder := gob.NewEncoder(conn)
// 				rr := &remoteReading{s, v, t}
// 				encoder.Encode(rr)
// 				go remoteServerReceive(conn, rHost)
// 			}
// 		}
// 	}
//}











