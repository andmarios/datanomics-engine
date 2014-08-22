/*
Datanomics™ — A web sink for your sensors
Copyright (C) 2014, Marios Andreopoulos

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

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
					senMet = d.Info(s)
					useInf, _ = udb.Info(senMet.Owner)
					go sendSensorOnline(senMet.Name, useInf.Email)
					log.Println(s + " back online")
				}
			}
		}
	}
}

func sendSensorOffline(s string, receiver string) {
	from := "Datanomics <" + emailSender + ">"
	subject := "Sensor " + s + " went offline."
	body := "Sensor <em>" + s + "</em> went offline. You won't receive other notifications until it come online again."
	server := emailServer + ":" + emailServerPort
	log.Println("Sending email to " + receiver)
	err := smtp.SendMail(
		server,
		mailAuth,
		emailSender, // This does nothing due to mailAuth
		[]string{receiver},
		[]byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s", from, receiver, subject, body)),
	)
	if err != nil {
		log.Println(err)
	}
}

func sendSensorOnline(s string, receiver string) {
	from := "Datanomics <" + emailSender + ">"
	subject := "Sensor " + s + " is live again."
	body := "Sensor <em>" + s + "</em> came online!"
	server := emailServer + ":" + emailServerPort
	log.Println("Sending email to " + receiver)
	err := smtp.SendMail(
		server,
		mailAuth, // This does nothing due to mailAuth
		emailSender,
		[]string{receiver},
		[]byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s", from, receiver, subject, body)),
	)
	if err != nil {
		log.Println(err)
	}
}
