package main

import "time"

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

func (d Database) Add(s string) {
	t := sensorlog{}
	t.Data = make([]string, 0, 100000)
	t.Timestamp = make([]time.Time, 0, 100000)
	d.Db[s] = t
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
	d.Db[s] = c
}

func (d Database) Load(s string) sensorlog {
	return d.Db[s]
}

func (d Database) Exists(s string) (b bool) {
	_, b = d.Db[s]
	return b
}

func (d Database) Last(s string) (v string, t time.Time) {
	l := len(d.Db[s].Data)
	return d.Db[s].Data[l], d.Db[s].Timestamp[l]
}



















