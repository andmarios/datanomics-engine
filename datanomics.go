package main

import (
//	"net"
	"net/http"
	"log"
//	"io/ioutil"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"fmt"
//	"reflect"
	"strconv"
	"time"
//	"strings"
	"html/template"
)

var (
	rootdir string
	port string
	address string
	verbose bool
)

var (
	d Query
)

var templates *template.Template


func init() {
	flag.StringVar(&rootdir, "root", "current directory", "webroot directory")
	flag.StringVar(&rootdir, "d", "current directory", "webroot directory" + " (shorthand)")
	flag.StringVar(&port,"port", "8080", "listen port")
	flag.StringVar(&port,"p", "8080", "listen port" + " (shorthand)")
	flag.StringVar(&address, "address", "*", "listen address")
	flag.StringVar(&address, "l", "*", "listen address" + " (shorthand)")
	flag.BoolVar(&verbose, "verbose", false, "be verbose")
	flag.BoolVar(&verbose, "v", false, "be verbose" + " (shorthand)")
}

func debug(s string) {
	if verbose {
		log.Print(s)
	}
}

func debugln(v ...interface{}) {
	if verbose {
		log.Println(v ...)
	}
}

var validLog = regexp.MustCompile("^/log/([a-zA-Z0-9-]+)/([0-9]+[.]{0,1}[0-9]*)(/([ts])/([0-9]+))?/?$")
var validURLs = regexp.MustCompile("^/")

func logHandler(w http.ResponseWriter, r *http.Request) {
	m := validLog.FindStringSubmatch(r.URL.Path)
	if len(m) == 0 {
		http.Error(w, "Sensor not found", http.StatusNotFound)
		return
	}
	debug("Sensor " + m[1] + " sent value " + m[2])
	//if ! d.Exists(m[1]) {
	//	d.Add(m[1])
	//}
	v, _ := strconv.ParseFloat(m[2], 64)
	if m[4] != "" {
		t, _ := strconv.ParseInt(m[5], 10, 64)
		if m[4] == "t" {
			d.StoreT(m[1], v, time.Unix(t, 0))
		} else {
			d.StoreT(m[1], v, time.Unix(time.Now().Unix() - t, 0))
		}
	} else {
		d.Store(m[1], v)
	}
	debugln("Sensor " + m[1] + " now contains:", d.Load(m[1]))
	fmt.Fprintf(w, "ok")
}

type HomePage struct {
	SensorList template.HTML
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	var sl string
	for _, s := range d.List() {
		sl += "<div>" + s + "</div>"
	}
	templates = template.Must(template.ParseFiles(rootdir + "/templates/home.html")) // Remove when finish frontend
	err := templates.ExecuteTemplate(w, "home.html", HomePage{template.HTML(sl)})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func makeHandler(fn func (http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validURLs.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r)
	}
}

func main() {
	flag.Parse()
	var err error
	if rootdir == "current directory" {
		rootdir, err =  filepath.Abs(filepath.Dir(os.Args[0]))
	} else {
		rootdir, err = filepath.Abs(filepath.Dir(rootdir))
	}
	if err != nil {
		log.Fatal(err)
	}

	if address == "*" {
		address = ""
	}

	t := Database{ make(map[string] sensorlog) }
	d = t
	templates = template.Must(template.ParseFiles(rootdir + "/templates/home.html"))

	http.HandleFunc("/log/", logHandler)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(rootdir + "/assets"))))
//	http.Handle("/", http.FileServer(http.Dir(rootdir)))
	http.HandleFunc("/", makeHandler(homeHandler))

	log.Print("Starting webserver. Listening on " + address + ":" + port)
	log.Print("Webroot set to \"" + rootdir + "\".")
	err = http.ListenAndServe(address + ":" + port, nil)
	if err != nil {
		log.Fatal("Couldn't start server. ListenAndServe: ", err)
	}
}




