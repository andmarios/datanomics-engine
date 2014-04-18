package main

import "time"

type Query interface {
	Add(string)
	Delete(string)
	List() string
	Store(string, float64)
	StoreT(string, float64, time.Time)
//	Store(string, int)
//	Store(string, string)
	Load(string) sensorlog
	Exists(string) bool
}

type sensorlog struct {
	Data []float64
	Timestamp []time.Time
	Info map[string] string
}


type Database struct {
	Db map[string]sensorlog
}

func (d Database) Add(s string) {
	t := sensorlog{}
	t.Data = make([]float64, 0, 100000)
	t.Timestamp = make([]time.Time, 0, 100000)
	d.Db[s] = t
}

func (d Database) Delete(s string) {
	delete(d.Db, s)
}

func (d Database) List() string {
	return "not implemented"
}

func (d Database) Store(s string, f float64) {
	c := d.Db[s]
	c.Data = append(d.Db[s].Data, f)
	c.Timestamp = append(d.Db[s].Timestamp, time.Now())
	d.Db[s] = c
}

func (d Database) StoreT(s string, f float64, t time.Time) {
	c := d.Db[s]
	c.Data = append(d.Db[s].Data, f)
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
