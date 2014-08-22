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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ziutek/rrd"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"
	"sync"
	"time"
)

type Query interface {
	Add(string)
	AddT(string, time.Time) error
	AddM(string, sensorMetadata) error
	Delete(string)
	List() []string
	Store(string, string)
	StoreT(string, string, time.Time)
	LoadR(string) graphPoint
	LoadMR(string, int64, int64) []rawGraphPoint
	Load(string) string
	Exists(string) bool
	Last(string) (string, time.Time)
	Close()
	Graph(string)
	FlushDatabases()
	Info(string) sensorMetadata
	Count() int
	OpenCount() int
}

type graphPoint struct {
	Time  int64
	Value float64
}

// This will be used as fast input for sensors' websocket.
type rawGraphPoint struct {
	C string
	T int64
	V interface{}
}

// RRD database implementation

var (
	step      = uint(30) // seconds
	heartbeat = 2 * step
)

type DatabaseRRD struct {
	Sensor   map[string]string
	Open     map[string]*rrd.Updater
	Metadata map[string]sensorMetadata
}

type sensorMetadata struct {
	Name  string
	Owner string
	Unit  string
	Info  string
	Lat   float64
	Lon   float64
}

var mutexRRD = &sync.Mutex{}

func (d DatabaseRRD) helperCheckFlushBeforeRead(s string) bool {
	_, exists := d.Sensor[s]
	if exists {
		_, opened := d.Open[s]
		if opened {
			d.FlushDatabase(s)
		}
	} else {
		return false
	}
	return true
}

func (d DatabaseRRD) Add(s string) {
	_ = d.AddT(s, time.Now())
}

func (d DatabaseRRD) AddM(s string, m sensorMetadata) error {
	if d.Exists(s) {
		log.Println("Attempt to add sensor that exists from web interface!")
		return errors.New("Sensor exists")
	}
	err := d.AddT(s, time.Now().Add(-24*365*time.Hour))
	if err != nil {
		return err
	}
	d.Metadata[s] = m
	return nil
}

func (d DatabaseRRD) AddT(s string, t time.Time) error {
	mutexRRD.Lock()
	dbfile := sensorDataDir + "/" + s
	c := rrd.NewCreator(dbfile, t.Add(-time.Second), step)
	c.DS("g", "GAUGE", heartbeat, -1000000000000, 1000000000000)
	c.RRA("AVERAGE", 0.5, 1, 1440) // Hold 720 datapoints at fine resolution
	c.RRA("AVERAGE", 0.5, 4, 1440) // Hold 720 datapoints at resolution/4
	c.RRA("AVERAGE", 0.5, 10, 8640)
	c.RRA("AVERAGE", 0.5, 30, 35064)
	err := c.Create(true)
	if err != nil {
		log.Println(err)
		return errors.New("Error creating RRD")
	}
	d.Sensor[s] = s
	d.Open[s] = rrd.NewUpdater(dbfile)
	err = d.Open[s].Update()
	d.Metadata[s] = sensorMetadata{s, "unknown", "raw", "", 35.5312752, 24.0676485}
	mutexRRD.Unlock()
	if err != nil {
		log.Println(err)
		return errors.New("Error writing RRD")
	}
	return nil
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
		delete(d.Metadata, s)
		mutexRRD.Unlock()
	}
}

func (d DatabaseRRD) List() []string {
	var s []string
	for key := range d.Sensor {
		s = append(s, key)
	}
	return s
}

func (d DatabaseRRD) Store(s string, v string) {
	d.StoreT(s, v, time.Now())
}

func (d DatabaseRRD) StoreT(s string, v string, t time.Time) {
	_, open := d.Open[s]
	if !open {
		dbfile := sensorDataDir + "/" + s
		mutexRRD.Lock()
		d.Open[s] = rrd.NewUpdater(dbfile)
		mutexRRD.Unlock()
	}
	f, _ := strconv.ParseFloat(v, 0)
	mutexRRD.Lock()
	d.Open[s].Cache(t, f)
	mutexRRD.Unlock()
	//------------------------------------------
	//DATABASE MIGRATION, REMOVE AFTER MIGRATION
	_, exists := d.Metadata[s]
	if !exists {
		d.Metadata[s] = sensorMetadata{s, "unknown", "raw", "", -1, -1}
	}
	//------------------------------------------
}

func (d DatabaseRRD) Load(s string) string {
	if !d.helperCheckFlushBeforeRead(s) {
		return "oops"
	}
	dbfile := sensorDataDir + "/" + s
	inf, err := rrd.Info(dbfile)
	if err != nil {
		log.Println(err)
	}
	end := time.Unix(int64(inf["last_update"].(uint)), 0)
	start := end.Add(-60 * 60 * 3 * time.Second)
	data, err := rrd.Fetch(dbfile, "AVERAGE", start, end, time.Duration(step)*time.Second)
	defer data.FreeValues()
	if err != nil {
		log.Println(err)
	}

	var buffer bytes.Buffer
	row := 0
	buffer.WriteString("[")
	for ti := data.Start.Add(data.Step); ti.Before(end) || ti.Equal(end); ti = ti.Add(data.Step) {
		//                for i := 0; i < len(data.DsNames); i++ {
		v := data.ValueAt(0, row)
		buffer.WriteString(fmt.Sprintf("[%d000, %f],", ti.Unix(), v))
		//              }
		row++
	}
	buffer.Truncate(buffer.Len() - 1)
	buffer.WriteString("]")

	return buffer.String()
}

func (d DatabaseRRD) LoadR(s string) graphPoint {
	if !d.helperCheckFlushBeforeRead(s) {
		return graphPoint{0, 0}
	}
	dbfile := sensorDataDir + "/" + s
	inf, err := rrd.Info(dbfile)
	if err != nil {
		log.Println(err)
	}
	end := time.Unix(int64(inf["last_update"].(uint)), 0)
	start := end.Add(-60 * 60 * 6 * time.Second)
	data, err := rrd.Fetch(dbfile, "AVERAGE", start, end, time.Duration(step)*time.Second)
	defer data.FreeValues()
	if err != nil {
		log.Println(err)
	}

	row := 0
	for ti := data.Start.Add(data.Step); ti.Before(end) || ti.Equal(end); ti = ti.Add(data.Step) {
		row++
	}
	return graphPoint{end.Unix(), data.ValueAt(0, row-1)}
}

// Last value is last_update, so javascript can calculate max value and enable live updates.
func (d DatabaseRRD) LoadMR(s string, st int64, en int64) []rawGraphPoint {
	if !d.helperCheckFlushBeforeRead(s) {
		return []rawGraphPoint{{"d", 0, 0}}
	}
	dbfile := sensorDataDir + "/" + s
	inf, err := rrd.Info(dbfile)
	if err != nil {
		log.Println(err)
	}
	//end := time.Unix(int64(inf["last_update"].(uint)), 0)
	//start := end.Add(-60 * 60 * 12 * time.Second)
	end := time.Unix(en, 0)
	start := time.Unix(st, 0)
	data, err := rrd.Fetch(dbfile, "AVERAGE", start, end, time.Duration(step/2)*time.Second)
	defer data.FreeValues()
	if err != nil {
		log.Println(err)
	}

	var r []rawGraphPoint
	row := 0
	for ti := data.Start.Add(data.Step); ti.Before(end) || ti.Equal(end); ti = ti.Add(data.Step) {
		v := data.ValueAt(0, row)
		if math.IsNaN(v) {
			if ti.After(time.Now().Add(-24 * 36 * time.Hour)) {
				r = append(r, rawGraphPoint{"a", ti.Unix(), nil})
			}
		} else {
			r = append(r, rawGraphPoint{"a", ti.Unix(), v})
		}
		row++
	}
	r = append(r, rawGraphPoint{"g", int64(inf["last_update"].(uint)), 0})
	return r
}

func (d DatabaseRRD) Exists(s string) bool {
	_, exists := d.Sensor[s]
	if !exists {
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

func (d DatabaseRRD) Info(s string) sensorMetadata {
	return d.Metadata[s]
}

func (d DatabaseRRD) Count() int {
	return len(d.Sensor)
}

func (d DatabaseRRD) OpenCount() int {
	return len(d.Open)
}

func (d DatabaseRRD) Close() {
	for s := range d.Open {
		d.FlushDatabase(s)
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

	_, err := g.SaveGraph(serverRootDir+"/assets/temp/"+s+".png", now.Add(-3600*time.Second), now)
	if err != nil {
		log.Println(err)
	}
}

type Journal struct {
	Entries []string
	Pipe    chan string
}

func (j *Journal) DatabaseLog() {
	ticker := time.NewTicker(time.Duration(sendRemotePeriod) * time.Second)
	tickerD := time.NewTicker(3600 * time.Second)
	journalFile := journalDir + "/" + time.Now().Format("200601021504")
	_, err := os.Create(journalFile)
	if err != nil {
		log.Println("Could not create journal file.")
	}
	go func() {
		for {
			select {
			case r := <-j.Pipe:
				j.Entries = append(j.Entries, r)
			case <-ticker.C:
				file, err := os.OpenFile(journalFile, os.O_RDWR|os.O_APPEND, 0600)
				if err != nil {
					log.Println("Could not open journal file. Entries dropped!")
				}
				for _, s := range j.Entries {
					_, err = file.WriteString(s + "\n")
					if err != nil {
						log.Println("Could not write entry to journal file. Entry dropped!")
					}
				}
				file.Close()
				j.Entries = make([]string, 0, 0)
			case <-tickerD.C:
				journalFile = journalDir + "/" + time.Now().Format("200601021504")
				_, err := os.Create(journalFile)
				if err != nil {
					log.Println("Could not create journal file.")
				}
			}
		}
	}()
}

func (d DatabaseRRD) FlushDatabases() {
	ticker := time.NewTicker(time.Duration(flushPeriod) * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				for s := range d.Open {
					d.FlushDatabase(s)
				}
				dbs, _ := json.Marshal(d)
				err := ioutil.WriteFile(database, dbs, 0600)
				if err != nil {
					log.Println("Error saving database info.")
				}

			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (d DatabaseRRD) FlushDatabase(s string) {
	mutexRRD.Lock()
	err := d.Open[s].Update()
	mutexRRD.Unlock()
	if err != nil {
		log.Println(err)
		h.Pipe <- Hometicker{d.Metadata[s].Name + ": out of order reading", "fa-times-circle", "danger",
			d.Metadata[s].Name + "</em> sent some out of order values in the last " + strconv.FormatInt(int64(flushPeriod), 10) + " seconds. Ignoring."}
	}
}

// Old database implementation, in memory
type sensorlog struct {
	Data      []string
	Timestamp []time.Time
	Info      map[string]string
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
	for key := range d.Db {
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

func (d Database) Graph() {
}
