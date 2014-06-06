package main

import (
	"fmt"
	"log"
	"net/smtp"
	"time"
)

var mailAuth smtp.Auth

func checkSensorStatus(db *DatabaseRRD, checkPeriod int) {
	ticker := time.NewTicker(time.Duration(checkPeriod) * time.Second)
	offline := make(map[string]bool)
	var lastUp graphPoint
	var senMet sensorMetadata
	var useInf User
	mailAuth = smtp.PlainAuth(
		"",
		emailUser,
		emailPass,
		emailServer,
	)

	for {
		select {
		case <-ticker.C:
			// TODO graphPoint returns int64 and we reconvert it to time, overhead
			checkPoint := time.Now().Add(-time.Duration(checkPeriod) * time.Second)
			for s := range db.Open {
				lastUp = db.LoadR(s)
				if time.Unix(lastUp.Time, 0).Before(checkPoint) {
					if offline[s] != true {
						offline[s] = true
						senMet = d.Info(s)
						useInf, _ = udb.Info(senMet.Owner)
						go sendSensorOffline(senMet.Name, useInf.Email)
						log.Println(s + " sensor closed")
					}
				} else if offline[s] == true {
					delete(offline, s)
					log.Println(s + " back online")
				}
			}
		}
	}
}

func sendSensorOffline(s string, receiver string) {
	subject := "Sensor " + s + " went offline."
	body := "Sensor " + s + " went offline. You won't receive other notifications until it come online again."
	server := emailServer + ":" + emailServerPort
	log.Println("Sending email to " + receiver)
	err := smtp.SendMail(
		server,
		mailAuth,
		emailSender,
		[]string{receiver},
		[]byte(fmt.Sprintf("Subject: %s\r\n\r\n%s", subject, body)),
	)
	if err != nil {
		log.Println(err)
	}
}
