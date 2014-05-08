package main

import (
	"time"
	"sync"
)

type Query interface {
	Add(string)
	Delete(string)
	List() []string
	Store(string, string)
	StoreT(string, string, time.Time)
//	Store(string, int)
//	Store(string, string)
	Load(string) sensorlog
	Exists(string) bool
	Last(string) (string, time.Time)

}

type sensorlog struct {
	Data []string
	Timestamp []time.Time
	Info map[string] string
}


type Database struct {
	Db map[string]sensorlog
}

var mutexA = &sync.Mutex{}

func (d Database) Add(s string) {
	t := sensorlog{}
	t.Data = make([]string, 0, 1)
	t.Timestamp = make([]time.Time, 0, 1)
	mutexA.Lock()
	d.Db[s] = t
	mutexA.Unlock()
}

func (d Database) Delete(s string) {
	delete(d.Db, s)
}

func (d Database) List() []string {
	var s []string
	for key, _ := range d.Db {
		s = append(s, key)
	}
	return s
}

func (d Database) Store(s string, v string) {
	c := d.Db[s]
	c.Data = append(d.Db[s].Data, v)
	c.Timestamp = append(d.Db[s].Timestamp, time.Now())
	d.Db[s] = c
}

func (d Database) StoreT(s string, v string, t time.Time) {
	c := d.Db[s]
	c.Data = append(d.Db[s].Data, v)
	c.Timestamp = append(d.Db[s].Timestamp, t)
	mutexA.Lock()
	d.Db[s] = c
	mutexA.Unlock()
}

func (d Database) Load(s string) sensorlog {
	return d.Db[s]
}

func (d Database) Exists(s string) (b bool) {
	_, b = d.Db[s]
	return b
}

func (d Database) Last(s string) (string, time.Time) {
	l := len(d.Db[s].Data) - 1
	if l < 0 {
		return "", time.Unix(0, 0)
	}
	return d.Db[s].Data[l], d.Db[s].Timestamp[l]
}



















