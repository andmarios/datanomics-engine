package main

import (
//	"net"
	"net/http"
	"log"
//	"io/ioutil"
	"flag"
	"os"
	"path/filepath"
)

var (
	rootdir string
	port string
	address string
)

func init() {
	flag.StringVar(&rootdir, "root", "current directory", "webroot directory")
	flag.StringVar(&rootdir, "d", "current directory", "webroot directory" + " (shorthand)")
	flag.StringVar(&port,"port", "8080", "listen port")
	flag.StringVar(&port,"p", "8080", "listen port" + " (shorthand)")
	flag.StringVar(&address, "address", "*", "listen address")
	flag.StringVar(&address, "l", "*", "listen address" + " (shorthand)")

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

func main() {
	flag.Parse()
	http.Handle("/", http.FileServer(http.Dir(rootdir)))

	log.Print("Starting webserver. Listening on " + address + ":" + port)
	err := http.ListenAndServe(address + ":" + port, nil)
	if err != nil {
		log.Fatal("Couldn't start server. ListenAndServe: ", err)
	}
}

