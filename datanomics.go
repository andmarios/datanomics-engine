package main

import (
//	"net"
	"net/http"
	"log"
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
	"encoding/json"
	"runtime/pprof"
	"code.google.com/p/go.net/websocket"
	"io/ioutil"
	"os/signal"
	"syscall"
)

var (
	rootdir string
	port string
	address string
	verbose bool
	database string
)

var (
	d Query
	h Hub
)

var templates *template.Template
var homeTemplate *template.Template

func init() {
	flag.StringVar(&rootdir, "root", "current directory", "webroot directory")
	flag.StringVar(&rootdir, "d", "current directory", "webroot directory" + " (shorthand)")
	flag.StringVar(&port,"port", "8080", "listen port")
	flag.StringVar(&port,"p", "8080", "listen port" + " (shorthand)")
	flag.StringVar(&address, "address", "*", "listen address")
	flag.StringVar(&address, "l", "*", "listen address" + " (shorthand)")
	flag.BoolVar(&verbose, "verbose", false, "be verbose")
	flag.BoolVar(&verbose, "v", false, "be verbose" + " (shorthand)")
	flag.StringVar(&database, "database", "db.json", "database file")
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

var validLog = regexp.MustCompile("^/log/([a-zA-Z0-9-]+)/(-?[0-9]+[.]{0,1}[0-9]*)(/([ts])/([0-9]+))?/?$")
var validQuery = regexp.MustCompile("^/q/([a-zA-Z0-9-]+)/?$")
var validURLs = regexp.MustCompile("^/")

func logHandler(w http.ResponseWriter, r *http.Request) {
	m := validLog.FindStringSubmatch(r.URL.Path)
	if len(m) == 0 {
		http.Error(w, "Sensor not found", http.StatusNotFound)
		return
	}
	debug("Sensor " + m[1] + " sent value " + m[2])
	if m[4] != "" {
		t, _ := strconv.ParseInt(m[5], 10, 64)
		var tnew time.Time
		if m[4] == "t" {
			tnew = time.Unix(t, 0)
		} else { // m[4] == "s"
			tnew = time.Unix(time.Now().Unix() - t, 0)
		}
		_, told := d.Last(m[1])

		if tnew.After(told) {
			d.StoreT(m[1], m[2], tnew)
		} else {
			http.Error(w, "Sensor send out of order timestamp", http.StatusNotFound)
			h.Pipe <- HometickerJson{"Out of Order Reading", "fa-times-circle", "danger", "Sensor " + m[1] + " out of order value " + m[2] + ". Ignored."}
			return
		}
	} else {
		if ! d.Exists(m[1]) { // Remove when you add code to add/delete sensors instead of adding them automatically.
			h.Pipe <- HometickerJson{"New Sensor", "fa-check-circle", "success", "Sensor " + m[1] + " succesfully added."}
		}
		d.Store(m[1], m[2])
		h.Pipe <- HometickerJson{"New Reading", "fa-plus-circle", "info", "Sensor " + m[1] + " sent value " + m[2] + "."}
	}
	debugln("Sensor " + m[1] + " now contains:", d.Load(m[1]))
	fmt.Fprintf(w, "ok")
}

func queryHandler(w http.ResponseWriter, r *http.Request) {
	m := validQuery.FindStringSubmatch(r.URL.Path)
        if len(m) == 0 {
                http.Error(w, "Sensor not found", http.StatusNotFound)
                return
        }
	if ! d.Exists(m[1]) {
		http.Error(w, "Sensor not found", http.StatusNotFound)
                return
        }
	debugln("Query for sensor " + m[1])
	a, _ := json.Marshal(d.Load(m[1]))
	fmt.Fprintf(w, string(a))
}

func reloadHandler(w http.ResponseWriter, r *http.Request) {
	loadTemplates()
	fmt.Fprintf(w, "templates reloaded")
}

type HomePage struct {
	Title string
	SensorList template.HTML
	CustomScript template.HTML
}

const homeCustomScript = template.HTML(`<script src="/assets/cjs/hometicker.js"></script>`)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	var sl string
	for _, s := range d.List() {
		sl += `
                 <li>
                   <a href="view/` + s + `">` + s + `</a>
                 </li>`
	}
	err := templates.ExecuteTemplate(w,
		"home.html",
		HomePage{"Datanomics alpha", template.HTML(sl),	homeCustomScript})
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

func loadTemplates() {
	templates = template.Must(template.ParseFiles(rootdir + "/templates/header.html",
		rootdir + "/templates/menu.html",
		rootdir + "/templates/footer.html",
		rootdir + "/templates/home.html"))
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

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

	// CPU profiling
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	t := Database{ make(map[string] sensorlog) }
	file, err := ioutil.ReadFile(database)
	if err != nil {
		log.Println("Using new database.")
	} else if err = json.Unmarshal(file, &t); err != nil {
		log.Println("Database corrupt. Creating new.", err)
	} else {
		log.Println("Loaded database.")
	}

	d = &t
	loadTemplates()

	h.Connections = make(map[*Socket]bool)
	h.Pipe = make(chan HometickerJson, 1)
	go h.Broadcast()

	go cleanup()

	http.HandleFunc("/log/", logHandler)
	http.HandleFunc("/q/", queryHandler)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(rootdir + "/assets"))))
	http.HandleFunc("/reload/", reloadHandler)
	http.Handle("/_hometicker", websocket.Handler(homeTickerHandler))
	http.HandleFunc("/", makeHandler(homeHandler))

	log.Print("Starting webserver. Listening on " + address + ":" + port)
	log.Print("Webroot set to \"" + rootdir + "\".")
	err = http.ListenAndServe(address + ":" + port, nil)
	if err != nil {
		log.Fatal("Couldn't start server. ListenAndServe: ", err)
	}
}

func cleanup() {
        ch := make(chan os.Signal)
        signal.Notify(ch, syscall.SIGINT)
        <-ch
	log.Println("Writing database to disk.")
	dbs, _ := json.Marshal(d)
	err := ioutil.WriteFile(database, dbs, 0600)
	if err != nil {
		log.Println("Error saving database.")
	}
	log.Println("Exiting. Goodbye.")
	os.Exit(1)
}
