package main

import (
//	"net"
	"net/http"
	"log"
//	"io/ioutil"
	"flag"
	"os"
	"fmt"
	"path/filepath"
)

var (
	rootdir string
	port string
	address string
)

func init() {
	flag.StringVar(&rootdir, "root", "current directory", "webroot directory")
	flag.StringVar(&port,"port", "8080", "listen port")
	flag.StringVar(&address, "address", "*", "listen address")
	if rootdir == "current directory" {
		rootdir, err :=  filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Webroot set to \"" + rootdir + "\".")
	}
}

func main() {
	flag.Parse()
	http.Handle("/", http.FileServer(http.Dir(rootdir)))

	err := http.ListenAndServe(address + ":" + port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
		fmt.Println("Couldn't start server, possibly wrong address and/or port?")
	}
}

