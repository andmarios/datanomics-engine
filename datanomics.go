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
)

var (
	rootdir string
	port string
	address string
	verbose bool
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

	if rootdir == "current directory" {
		rootdir, err :=  filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Fatal(err)
		}
		log.Print("Webroot set to \"" + rootdir + "\".")
	}
	if address == "*" {
		address = ""
	}
}

func debug(s string) {
	if verbose {
		log.Print(s)
	}
}

var validLog = regexp.MustCompile("^/log/([a-zA-Z0-9-]+)/([0-9]+)/?$")

func logHandler(w http.ResponseWriter, r *http.Request) {
	m := validLog.FindStringSubmatch(r.URL.Path)
	if len(m) == 0 {
		http.Error(w, "Sensor not found", http.StatusNotFound)
		return
	}
	debug("Sensor " + m[1] + " sent value " + m[2])
	fmt.Fprintf(w, "ok")
}

func main() {
	flag.Parse()
	http.HandleFunc("/log/", logHandler)
	http.Handle("/", http.FileServer(http.Dir(rootdir)))

	log.Print("Starting webserver. Listening on " + address + ":" + port)
	err := http.ListenAndServe(address + ":" + port, nil)
	if err != nil {
		log.Fatal("Couldn't start server. ListenAndServe: ", err)
	}
}

