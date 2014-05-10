package main

import (
	"net/http"
	"log"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"time"
	"html/template"
	"encoding/json"
	"runtime/pprof"
	"code.google.com/p/go.net/websocket"
	"io/ioutil"
	"os/signal"
	"syscall"
	"github.com/ziutek/rrd"
)

var version = "Datanomics 2c02ec7+"

var (
	serverRootDir string
	port string
	address string
	verbose bool
	database string
	sensorDataDir string
)

var (
	d Query
	h Hub
	sh SensorHub
)

var flushPeriod = 300 // seconds

func init() {
	flag.StringVar(&serverRootDir, "root", "current directory", "webroot directory")
	flag.StringVar(&serverRootDir, "d", "current directory", "webroot directory" + " (shorthand)")
	flag.StringVar(&port,"port", "8080", "listen port")
	flag.StringVar(&port,"p", "8080", "listen port" + " (shorthand)")
	flag.StringVar(&address, "address", "*", "listen address")
	flag.StringVar(&address, "l", "*", "listen address" + " (shorthand)")
	flag.BoolVar(&verbose, "verbose", false, "be verbose")
	flag.BoolVar(&verbose, "v", false, "be verbose" + " (shorthand)")
	flag.StringVar(&database, "database", "db.json", "database file")
	flag.StringVar(&sensorDataDir, "storage", "sensors", "directory to store sensor data")
	flag.StringVar(&sensorDataDir, "s", "sensors", "directory to store sensor data" + " (shorthand)")
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
var validInfoQuery = regexp.MustCompile("^/iq/([a-zA-Z0-9-]+)/?$")
var validRoot = regexp.MustCompile("^/$")
var validView = regexp.MustCompile("^/view/([a-zA-Z0-9-]+)/?$")

func makeHandler(fn func (http.ResponseWriter, *http.Request), rexp regexp.Regexp) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := rexp.FindStringSubmatch(r.URL.Path)
		logRequest(r)
		w.Header().Add("Server", version)
		w.Header().Add("Vary", "Accept-Encoding")
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r)
	}
}

func loadTemplates() {
	templates = template.Must(template.ParseFiles(serverRootDir + "/templates/header.html",
		serverRootDir + "/templates/menu.html",
		serverRootDir + "/templates/footer.html",
		serverRootDir + "/templates/home.html",
		serverRootDir + "/templates/sensor.html"))
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	var err error
	if serverRootDir == "current directory" {
		serverRootDir, err =  filepath.Abs(filepath.Dir(os.Args[0]))
	} else {
		serverRootDir, err = filepath.Abs(filepath.Dir(serverRootDir))
	}
	if err != nil {
		log.Fatal(err)
	}

	if sensorDataDir == "sensors" {
		sensorDataDir, err =  filepath.Abs(filepath.Dir(os.Args[0] + "/sensors"))
	} else {
		sensorDataDir, err = filepath.Abs(filepath.Dir(sensorDataDir))
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
	// t := Database{ make(map[string] sensorlog) }
	t := DatabaseRRD{make(map[string]string), make(map[string]*rrd.Updater), make(map[string]sensorMetadata)}
	file, err := ioutil.ReadFile(database)
	if err != nil {
		log.Println("Using new database.")
	} else if err = json.Unmarshal(file, &t); err != nil {
		log.Println("Database corrupt. Creating new.", err)
	} else {
		log.Println("Loaded database.")
	}

	d = &t
	go d.FlushDatabases()

	sensorList()
	loadTemplates()

	h.Connections = make(map[*Socket]bool)
	h.Pipe = make(chan Hometicker, 1)
	go h.Broadcast()

	sh.Connections = make(map[string]map[*Socket]bool)
	sh.Pipe = make(chan string)
	go sh.Broadcast()

	go cleanup()

	http.HandleFunc("/log/", logHandler)
	http.HandleFunc("/q/", makeHandler(queryHandler, *validQuery))
	http.HandleFunc("/iq/", makeHandler(queryInfoHandler, *validInfoQuery))
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(serverRootDir + "/assets"))))
	http.HandleFunc("/reload/", reloadHandler)
	http.Handle("/_hometicker", websocket.Handler(homeTickerHandler))
	http.Handle("/_sensorticker", websocket.Handler(sensorTickerHandler))
	http.HandleFunc("/view/", makeHandler(viewHandler, *validView))
	http.HandleFunc("/", makeHandler(homeHandler, *validRoot))

	log.Print("Starting webserver. Listening on " + address + ":" + port)
	log.Print("Webroot set to \"" + serverRootDir + "\".")
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
	d.Close()
	dbs, _ := json.Marshal(d)
	err := ioutil.WriteFile(database, dbs, 0600)
	if err != nil {
		log.Println("Error saving database.")
	}
	log.Println("Exiting. Goodbye.")
	os.Exit(1)
}

// This function was copied from https://github.com/mkaz/lanyon/blob/master/src/main.go
func logRequest(r *http.Request) {
	now := time.Now()
	log.Printf("%s - [%s] \"%s %s %s\" ",
		r.RemoteAddr,
		now.Format("02/Jan/2006:15:04:05 -0700"),
		r.Method,
		r.URL.RequestURI(),
		r.Proto)
}
