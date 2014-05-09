package main

import (
	"time"
	"sync"
	"log"
	"strconv"
	"bytes"
	"fmt"
	"github.com/ziutek/rrd"
)

type Query interface {
	Add(string)
	AddT(string, time.Time)
	Delete(string)
	List() []string
	Store(string, string)
	StoreT(string, string, time.Time)
	Load(string) string
	Exists(string) bool
	Last(string) (string, time.Time)
	Close()
	Graph(string)
}

// RRD database implementation

var (
	step = uint(30) // seconds
	heartbeat = 2 * step
)

type DatabaseRRD struct {
	Sensor map[string]string
	Open map[string]*rrd.Updater
}

func (d DatabaseRRD) Add(s string) {
	d.AddT(s, time.Now())
}

var mutexRRD = &sync.Mutex{}

func (d DatabaseRRD) AddT(s string, t time.Time) {
	mutexRRD.Lock()
	dbfile := sensorDataDir + "/" + s
	c := rrd.NewCreator(dbfile, t.Add(-time.Second), step)
	c.DS("g", "GAUGE", heartbeat, 0, 60) // See what these numbers are!!!!
	c.RRA("AVERAGE", 0.5, 1, 720) // Hold 720 datapoints at fine resolution
	c.RRA("AVERAGE", 0.5, 4, 720) // Hold 720 datapoints at resolution/4
	c.RRA("AVERAGE", 0.5, 10, 8640)
	c.RRA("AVERAGE", 0.5, 60, 17532)
	err := c.Create(true)
	if err != nil {
		log.Println(err)
	}
	d.Sensor[s] = s
	d.Open[s] = rrd.NewUpdater(dbfile)
	mutexRRD.Unlock()
}

func (d DatabaseRRD) Delete(s string) {
	_, exists := d.Sensor[s]
	if exists {
		mutexRRD.Lock()
		_, open := d.Open[s]
		if open {
			delete(d.Open, s)
		}
		// TODO: Delete FILE
		delete(d.Sensor, s)
		mutexRRD.Unlock()
	}
}

func (d DatabaseRRD) List() []string {
        var s []string
        for key, _ := range d.Sensor {
                s = append(s, key)
        }
        return s
}

func (d DatabaseRRD) Store(s string, v string) {
	d.StoreT(s, v, time.Now())
}

func (d DatabaseRRD) StoreT(s string, v string, t time.Time) {
	dbfile := sensorDataDir + "/" + s
	_, open := d.Open[s]
	if ! open {
		mutexRRD.Lock()
		d.Open[s] = rrd.NewUpdater(dbfile)
		mutexRRD.Unlock()
	}
	f, _ := strconv.ParseFloat(v, 0)
	mutexRRD.Lock()
	d.Open[s].Cache(t, f)
	err := d.Open[s].Update() // TODO: Skip this step and run it periodically
	mutexRRD.Unlock()
	if err != nil {
		log.Println(err)
	}
}

func (d DatabaseRRD) Load(s string) string {
	dbfile := sensorDataDir + "/" + s
	inf, err := rrd.Info(dbfile)
        if err != nil {
                log.Println(err)
        }
	end := time.Unix(int64(inf["last_update"].(uint)), 0)
	start := end.Add(-60 * 60 * 6 * time.Second)
	data, err := rrd.Fetch(dbfile, "AVERAGE", start, end, time.Duration(step) * time.Second)
	defer data.FreeValues()
        if err != nil {
                log.Println(err)
        }

	var buffer bytes.Buffer
        row := 0
	buffer.WriteString("[")
        for ti := data.Start.Add(data.Step); ti.Before(end) || ti.Equal(end); ti = ti.Add(data.Step) {
                for i := 0; i < len(data.DsNames); i++ {
                        v := data.ValueAt(i, row)
			buffer.WriteString(fmt.Sprintf("[%d000, %f],", ti.Unix(), v))
                }
                row++
        }
	buffer.Truncate(buffer.Len() - 1)
	buffer.WriteString("]")

	return buffer.String()
}

func (d DatabaseRRD) Exists(s string) bool {
	_, exists := d.Sensor[s]
	if ! exists {
		return false
	}
	return true
}

func (d DatabaseRRD) Last(s string) (v string, t time.Time) {
	// dbfile := sensorDataDir + "/" + s
        // inf, err := rrd.Info(dbfile)
        // if err != nil {
        //         log.Println(err)
        //}
        // end := time.Unix(int64(inf["last_update"].(uint)), 0)
	return "unknown", time.Now() // TODO: return last value
}

func (d DatabaseRRD) Close() {
	for s, _ := range d.Open {
		d.Open[s].Update()
		delete(d.Open, s)
	}
}

func (d DatabaseRRD) Graph(s string) {
	dbfile := sensorDataDir + "/" + s
	g := rrd.NewGrapher()
        g.SetTitle(s)
        g.SetVLabel("some variable")
        g.SetSize(1600, 800)
        g.SetWatermark("some watermark")
        g.Def("v1", dbfile, "g", "AVERAGE")
        g.VDef("avg1", "v1,AVERAGE")
        g.Line(1, "v1", "ff0000", "var 1")
        g.GPrint("avg1", "avg1=%lf")
        g.Print("avg1", "avg1=%lf")

        now := time.Now()

        _, err := g.SaveGraph(serverRootDir + "/assets/temp/" + s + ".png", now.Add(-3600*time.Second), now)
        if err != nil {
                log.Println(err)
        }
}

// Old database implementation, in memory
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

func (d Database) AddT(s string, t time.Time) {
	d.Add(s)
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

func (d Database) Load(s string) interface{} {
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

func (d Database) Close() {
}

func (d Database) Graph(){
}















