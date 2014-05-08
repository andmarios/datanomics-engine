package main

import (
//	"net"
	"net/http"
	"log"
	"flag"
	"os"
	"path/filepath"
	"regexp"
//	"reflect"
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

var version = "Datanomics 742cc3a+"

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
	templates = template.Must(template.ParseFiles(rootdir + "/templates/header.html",
		rootdir + "/templates/menu.html",
		rootdir + "/templates/footer.html",
		rootdir + "/templates/home.html",
		rootdir + "/templates/sensor.html"))
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
	sensorList()
	loadTemplates()

	h.Connections = make(map[*Socket]bool)
	h.Pipe = make(chan Hometicker, 1)
	go h.Broadcast()

	go cleanup()

	http.HandleFunc("/log/", logHandler)
	http.HandleFunc("/q/", makeHandler(queryHandler, *validQuery))
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(rootdir + "/assets"))))
	http.HandleFunc("/reload/", reloadHandler)
	http.Handle("/_hometicker", websocket.Handler(homeTickerHandler))
	http.HandleFunc("/view/", makeHandler(viewHandler, *validView))
	http.HandleFunc("/", makeHandler(homeHandler, *validRoot))

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
