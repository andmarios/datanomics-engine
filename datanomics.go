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
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"flag"
	"github.com/bradrydzewski/go.auth"
	"github.com/ziutek/rrd"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime/pprof"
	"syscall"
	"time"
)

var version = "Datanomics 266625e+"

var (
	serverRootDir   string
	port            string
	address         string
	verbose         bool
	database        string
	userDatabase    string
	sensorDataDir   string
	journalDir      string
	configFile      string
	scPort          string
	remoteServers   []string
	googleAccessKey string
	googleSecretKey string
	googleRedirect  string
	githubAccessKey string
	githubSecretKey string
	emailUser       string
	emailPass       string
	emailServer     string
	emailSender     string
	emailServerPort string
)

const RemoteBuffer = 1000000 /* size of remote readings channel buffer */

type configVars struct {
	ServerRootDir    string
	Port             string
	Address          string
	Verbose          bool
	Database         string
	UserDatabase     string
	SensorDataDir    string
	JournalDir       string
	ScPort           string
	RemoteServers    []string
	FlushPeriod      int
	SendRemotePeriod int
	GoogleAccessKey  string
	GoogleSecretKey  string
	GoogleRedirect   string
	GithubAccessKey  string
	GithubSecretKey  string
	EmailUser        string
	EmailPass        string
	EmailServer      string
	EmailSender      string
	EmailServerPort  string
}

var (
	d   Query
	h   Hub
	sh  SensorHub
	srC SendReadingsCache
	udb Users
	j   Journal
)

var flushPeriod = 300     // seconds
var sendRemotePeriod = 10 // seconds

func init() {
	flag.StringVar(&serverRootDir, "root", "current directory", "webroot directory")
	flag.StringVar(&serverRootDir, "d", "current directory", "webroot directory"+" (shorthand)")
	flag.StringVar(&port, "port", "8080", "listen port")
	flag.StringVar(&port, "p", "8080", "listen port"+" (shorthand)")
	flag.StringVar(&address, "address", "*", "listen address")
	flag.StringVar(&address, "l", "*", "listen address"+" (shorthand)")
	flag.BoolVar(&verbose, "verbose", false, "be verbose")
	flag.BoolVar(&verbose, "v", false, "be verbose"+" (shorthand)")
	flag.StringVar(&database, "database", "db.json", "database file")
	flag.StringVar(&userDatabase, "usersdb", "usersdb.json", "users database file")
	flag.StringVar(&sensorDataDir, "storage", "sensors", "directory to store sensor data")
	flag.StringVar(&sensorDataDir, "s", "sensors", "directory to store sensor data"+" (shorthand)")
	flag.StringVar(&configFile, "config", "", "configuration file")
	flag.StringVar(&scPort, "scport", "12127", "port to listen for remote readings")
	flag.StringVar(&journalDir, "journal", "journal", "directory to store journal")

	// For pacakage auth
	flag.StringVar(&googleAccessKey, "googlecid", "[client id]", "your google client ID")
	flag.StringVar(&googleSecretKey, "googlecs", "[secret]", "your google client secret")
	flag.StringVar(&googleRedirect, "googlecb", "http://localhost:8080/oauth2callback", "your google redirect URI")
	flag.StringVar(&githubAccessKey, "githubcid", "[client id]", "your github client ID")
	flag.StringVar(&githubSecretKey, "githubcs", "[secret]", "your github client secret")

	emailUser = ""
	emailPass = ""
	emailServer = ""
	emailSender = ""
	emailServerPort = "25"
}

func debug(s string) {
	if verbose {
		log.Print(s)
	}
}

func debugln(v ...interface{}) {
	if verbose {
		log.Println(v...)
	}
}

var validLog = regexp.MustCompile("^/log(/([a-zA-Z0-9-]+)/(-?[0-9]+[.]{0,1}[0-9]*)(/([ts])/([0-9]+))?)?/?$")
var validQuery = regexp.MustCompile("^/q/([a-zA-Z0-9-]+)/?$")
var validInfoQuery = regexp.MustCompile("^/iq/([a-zA-Z0-9-]+)/?$")
var validRoot = regexp.MustCompile("^/(robots.txt/?)?$")
var validView = regexp.MustCompile("^/view/([a-zA-Z0-9-]+)/?$")
var validStats = regexp.MustCompile("^/_stats/?$")
var validLogin = regexp.MustCompile("^/login/?$")
var validLogged = regexp.MustCompile("^/login/success/?$")

func makeHandler(fn func(http.ResponseWriter, *http.Request), rexp regexp.Regexp) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := rexp.FindStringSubmatch(r.URL.Path)
		logRequest(r)
		w.Header().Add("Server", version)
		w.Header().Add("Vary", "Accept-Encoding")
		if m == nil {
			serve404(w, nil)
			return
		}
		fn(w, r)
	}
}

func makeSecureHandler(fn func(http.ResponseWriter, *http.Request, auth.User), rexp regexp.Regexp) auth.SecureHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, u auth.User) {
		m := rexp.FindStringSubmatch(r.URL.Path)
		logRequest(r)
		w.Header().Add("Server", version)
		w.Header().Add("Vary", "Accept-Encoding")
		if m == nil {
			serve404(w, u)
			return
		}
		fn(w, r, u)
	}
}

func makeNoLogHandler(fn func(http.ResponseWriter, *http.Request), rexp regexp.Regexp) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := rexp.FindStringSubmatch(r.URL.Path)
		w.Header().Add("Server", version)
		w.Header().Add("Vary", "Accept-Encoding")
		if m == nil {
			serve404(w, nil)
			return
		}
		fn(w, r)
	}
}

func loadTemplates() {
	templates = template.Must(template.ParseFiles(serverRootDir+"/templates/header.html",
		serverRootDir+"/templates/menu.html",
		serverRootDir+"/templates/footer.html",
		serverRootDir+"/templates/home.html",
		serverRootDir+"/templates/sensor.html",
		serverRootDir+"/templates/login.html",
		serverRootDir+"/templates/404.html"))
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	var err error
	if serverRootDir == "current directory" {
		serverRootDir, err = filepath.Abs(filepath.Dir(os.Args[0]))
	} else {
		serverRootDir, err = filepath.Abs(filepath.Dir(serverRootDir))
	}
	if err != nil {
		log.Fatal(err)
	}

	if sensorDataDir == "sensors" {
		sensorDataDir, err = filepath.Abs(filepath.Dir(os.Args[0] + "/sensors"))
	} else {
		sensorDataDir, err = filepath.Abs(filepath.Dir(sensorDataDir))
	}

	if journalDir == "journal" {
		journalDir, err = filepath.Abs(filepath.Dir(os.Args[0] + "/journal"))
	} else {
		journalDir, err = filepath.Abs(filepath.Dir(journalDir))
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

	if configFile != "" {
		var confR configVars
		file, err := ioutil.ReadFile(configFile)
		if err != nil {
			log.Println("Creating new configuration file.")
		} else if err = json.Unmarshal(file, &confR); err != nil {
			log.Println("Couldn't parse configuration file. Ignoring.", err)
		} else {
			serverRootDir = confR.ServerRootDir
			port = confR.Port
			address = confR.Address
			verbose = confR.Verbose
			database = confR.Database
			userDatabase = confR.UserDatabase
			sensorDataDir = confR.SensorDataDir
			journalDir = confR.JournalDir
			scPort = confR.ScPort
			remoteServers = confR.RemoteServers
			flushPeriod = confR.FlushPeriod
			sendRemotePeriod = confR.SendRemotePeriod
			googleAccessKey = confR.GoogleAccessKey
			googleSecretKey = confR.GoogleSecretKey
			googleRedirect = confR.GoogleRedirect
			githubAccessKey = confR.GithubAccessKey
			githubSecretKey = confR.GithubSecretKey
			emailUser = confR.EmailUser
			emailPass = confR.EmailPass
			emailServer = confR.EmailServer
			emailSender = confR.EmailSender
			emailServerPort = confR.EmailServerPort
			log.Println("Loaded configuration. Command line options will be ignored.")
		}

		//		confR = configVars{serverRootDir, port, address, verbose, database, userDatabase, sensorDataDir,
		//			journalDir, scPort, remoteServers, flushPeriod, sendRemotePeriod, googleAccessKey,
		//			googleSecretKey, googleRedirect, githubAccessKey, githubSecretKey}
		//		confJ, _ := json.Marshal(confR)
		//		err = ioutil.WriteFile(configFile, confJ, 0600)
		//		if err != nil {
		//			log.Println("Error saving config info.")
		//		} else {
		//			log.Println("Saved config file.")
		//		}
	}

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

	ut := UserDB{make(map[string]User)}
	file, err = ioutil.ReadFile(userDatabase)
	if err != nil {
		log.Println("Using new user database.")
	} else if err = json.Unmarshal(file, &ut); err != nil {
		log.Println("User database corrupt. Creating new.", err)
	} else {
		log.Println("Loaded user database.")
	}
	udb = &ut

	sensorList()
	latlonList()
	loadTemplates()

	h.Connections = make(map[*Socket]bool)
	h.Pipe = make(chan Hometicker, 1)
	go h.Broadcast()

	sh.Connections = make(map[string]map[*Socket]bool)
	sh.Pipe = make(chan string)
	go sh.Broadcast()

	srC.Readings = make([]remoteReading, 0, 10)
	srC.Pipe = make(chan remoteReading, RemoteBuffer)
	go srC.SendReadingsCron()

	j.Entries = make([]string, 0, 0)
	j.Pipe = make(chan string)
	go j.DatabaseLog()

	go listenForRemoteReadings()

	go cleanup()

	if emailServer != "" && emailSender != "" {
		log.Println("Emails will be sent for sensor status changes.")
		go checkSensorStatus(&t, flushPeriod)
	} else {
		log.Println("Email notification system disabled.")
	}

	auth.Config.CookieSecret = []byte("82f6e00c-9053-4305-8662-aa163daca490")
	auth.Config.LoginSuccessRedirect = "/login/success"
	auth.Config.CookieSecure = false
	auth.Config.LoginRedirect = "/login/"

	googleHandler := auth.Google(googleAccessKey, googleSecretKey, googleRedirect)
	http.Handle("/login/google", googleHandler)

	// "" is for scope (which user data we need)
	githubHandler := auth.Github(githubAccessKey, githubSecretKey, "")
	http.Handle("/login/github", githubHandler)
	http.Handle("/login/success", auth.SecureUser(makeSecureHandler(userLoggedHandler, *validLogged)))

	http.HandleFunc("/log", logHandler)  /* catches exact log (user for POST */
	http.HandleFunc("/log/", logHandler) /* catches everything after log/ */
	http.HandleFunc("/q/", makeHandler(queryHandler, *validQuery))
	http.HandleFunc("/iq/", makeHandler(queryInfoHandler, *validInfoQuery))
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(serverRootDir+"/assets"))))
	http.HandleFunc("/reload/", reloadHandler)
	http.Handle("/_hometicker", websocket.Handler(homeTickerHandler))
	http.Handle("/_sensorticker", websocket.Handler(sensorTickerHandler))
	http.HandleFunc("/_stats/", makeNoLogHandler(statsHandler, *validStats))
	http.HandleFunc("/view/", auth.SecureGuest(makeSecureHandler(viewHandler, *validView)))
	http.HandleFunc("/login/", makeHandler(loginHandler, *validLogin))
	http.HandleFunc("/logout", logOutHandler)
	http.HandleFunc("/post/addsensor", auth.SecureUser(addSensorHandler))
	http.HandleFunc("/", auth.SecureGuest(makeSecureHandler(homeHandler, *validRoot)))

	log.Print("Starting webserver. Listening on " + address + ":" + port)
	log.Print("Webroot set to \"" + serverRootDir + "\".")
	err = http.ListenAndServe(address+":"+port, nil)
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
