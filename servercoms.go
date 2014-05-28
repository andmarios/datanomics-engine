package main

import (
	"encoding/gob"
	"log"
	"net"
	"time"
)

type remoteReading struct {
	S string
	V string
	T time.Time
}

func listenForRemoteReadings() {
	ln, err := net.Listen("tcp", address+":"+scPort)
	if err != nil {
		log.Println("Could not start remote readings listener.")
		log.Println(err)
		return
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
				if !d.Exists(rr.S) { // Since this sensor comes from another datanomics server, we trust is it ok.
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
	Pipe     chan remoteReading
}

func (s *SendReadingsCache) SendReadingsCron() {
	ticker := time.NewTicker(time.Duration(sendRemotePeriod) * time.Second)
	go func() {
		for {
			select {
			case r := <-s.Pipe:
				s.Readings = append(s.Readings, r)
			case <-ticker.C:
				s.Readings = sendRemoteReading(s.Readings)
			}
		}
	}()
}

// TODO: If we can send to someone, store the readings (up to a size) to try later.
func sendRemoteReading(sra []remoteReading) []remoteReading {
	srb := make([]remoteReading, 0, 0)
	for _, rHost := range remoteServers {
		conn, err := net.Dial("tcp", rHost)
		if err != nil {
			log.Println(err)
			// TODO. If we don't find a remote server, we return the array of readings.
			// Next time we will try to send both old and new values. If we send to more than
			// one servers, some servers will receive the old values twice.
			// This doesn't cause problems since these servers will ignore the old values.
			// It just isn't optimal when there are many sensors.
			srb = sra
		} else {
			encoder := gob.NewEncoder(conn)
			rr := &sra
			encoder.Encode(rr)
			conn.Close()
		}
	}
	// TODO FIX HARDLIMIT : 100000 below should be configurable
	if len(srb) > 100000 {
		srb = srb[len(srb)-100000:]
	}
	return srb
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
