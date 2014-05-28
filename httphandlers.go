package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bradrydzewski/go.auth"
	"github.com/nu7hatch/gouuid"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
)

var templates *template.Template
var homeTemplate *template.Template

func logHandler(w http.ResponseWriter, r *http.Request) {
	m := validLog.FindStringSubmatch(r.URL.Path)
	var err error

	if r.Method == "POST" { /* process POST request */
		if len(m) == 0 || m[2] != "" {
			http.Error(w, "Sensor not found", http.StatusNotFound)
			log.Println(len(m))
			log.Println(m)
			return
		}
		errf := r.ParseForm()
		if errf != nil {
			http.Error(w, errf.Error(), http.StatusInternalServerError)
			log.Println(errf)
			return
		}
		chid, _ := regexp.Compile("^[a-zA-Z0-9-]+$")
		chnum, _ := regexp.Compile("^-?[0-9]+[.]{0,1}[0-9]*$")
		chtyp, _ := regexp.Compile("^[ts]$")
		chtim, _ := regexp.Compile("^[0-9]+$")

		if !chid.MatchString(r.FormValue("id")) || !chnum.MatchString(r.FormValue("val")) {
			http.Error(w, "Sensor not found", http.StatusInternalServerError)
			log.Println(err)
			return
		}
		if chtyp.MatchString(r.FormValue("f")) && chtim.MatchString(r.FormValue("t")) {
			err = logReading(r.FormValue("id"), r.FormValue("val"),
				r.FormValue("f"), r.FormValue("t"))
		} else if r.FormValue("f") == "" && r.FormValue("t") == "" {
			err = logReading(r.FormValue("id"), r.FormValue("val"), "", "")
		} else {
			http.Error(w, "Sensor not found", http.StatusNotFound)
			return
		}
	} else { /* process GET request */
		if len(m) == 0 {
			http.Error(w, "Sensor not found", http.StatusNotFound)
			return
		}
		err = logReading(m[2], m[3], m[5], m[6])
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, "ok")
}

func logReading(sen string, val string, typ string, tim string) error {
	/* next 3 are not needed because they happen with the regexp */
	//	if sen == "" {return errors.New("Sensor not found")}
	//	if val == "" {return errors.New("Invalid value")}
	//	if typ != "" && tim == "" {return errors.New("Missing time field")}

	tnew := time.Now()
	if typ != "" {
		t, _ := strconv.ParseInt(tim, 10, 64)
		if typ == "t" {
			tnew = time.Unix(t, 0)
		} else { // m[4] == "s"
			tnew = time.Unix(time.Now().Unix()-t, 0)
		}
	}
	// From down here there is a bit of duplication with sendRemoteReading(). Remember to change both if needed.
	if !d.Exists(sen) { // Remove when you add code to add/delete sensors instead of adding them automatically.
		h.Pipe <- Hometicker{"Unknown sensor: " + sen, "fa-times-circle", "danger",
			"Sensor <em>" + sen + "</em> isn't registered. Ignored."}
		// For Benchmark puproses uncooment the next line and comment the http.Error and return lines below.
		// d.AddT(sen, tnew) // This is not needed. Sensors are added automatically upon first reading. It is here only to make the next command to work.
		//sensorList()
		//latlonList()
		return errors.New("Sensor not found")
	}
	// We can't check if value came out of order with rrd cache. We do it though on database flush.
	d.StoreT(sen, val, tnew)
	srC.Pipe <- remoteReading{sen, val, tnew}
	j.Pipe <- sen + "/" + val + "/t/" + strconv.FormatInt(tnew.Unix(), 10)
	t := d.Info(sen).Name
	h.Pipe <- Hometicker{"<a href='/view/" + sen + "'>" + t + "</a>: new reading", "fa-plus-circle", "info",
		t + "</em> sent value <em>" + val + "</em> at <em>" + tnew.String() + "</em>"}
	sh.Pipe <- sen
	return nil
}

func queryHandler(w http.ResponseWriter, r *http.Request) {
	m := validQuery.FindStringSubmatch(r.URL.Path)
	if len(m) == 0 {
		http.Error(w, "Sensor not found", http.StatusNotFound)
		return
	}
	if !d.Exists(m[1]) {
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

var SensorList template.HTML

func sensorList() { // When we add/remove sensors manually, make this run once and store its value for performance?
	// This is nice. Initially I used a string instead of a buffer. But concatenating huge strings is slow.
	// So, when testing with 1.000.000 sensors the code needed many minutes (maybe hour?) to enumerate the
	// sensors. Now it takes 1 second!!!
	var buffer bytes.Buffer
	for _, s := range d.List() {
		buffer.WriteString(`
                 <li>
                   <a href="/view/` + s + `">` + d.Info(s).Name + `</a>
                 </li>`)
	}
	SensorList = template.HTML(buffer.String())
}

type HomePage struct {
	Title        string
	LoginInfo    template.HTML
	SensorList   template.HTML
	CustomScript template.HTML
	LatLonList   template.JS
}

const homeCustomScript = template.HTML(`
     <script src="/assets/cjs/hometicker.js"></script>
     <script src="/assets/js/maplace.min.js"></script>
      <script>
       $(function() {
         new Maplace({
           locations: Locs,
           map_div: '#gmap',
           controls_type: 'dropdown',
           controls_on_map: false,
           controls_cssclass: "btn-default"
         }).Load();
       });
      </script>
`)

func userMenu(u auth.User) template.HTML {
	if u != nil {
		return template.HTML(`
            <li><a id="username" href="` + u.Link() + `"><img class="img-rounded" height="50px" src="` + u.Picture() + `" /> ` + u.Name() + `</a></li>
            <li class="divider"></li>
            <li><a href="/logout"><i class="fa fa-sign-out fa-fw"></i> Logout</a></li>
`)
	} else {
		return template.HTML(`
            <li><a href="/login/"><i class="fa fa-sign-in fa-fw"></i> Login</a></li>`)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request, u auth.User) {
	err := templates.ExecuteTemplate(w,
		"home.html",
		HomePage{"Datanomics™ alpha", userMenu(u), SensorList, homeCustomScript, LatLonList})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
	}
}

var LatLonList template.JS

func latlonList() {
	var buffer bytes.Buffer
	buffer.WriteString("var Locs = [")
	for _, s := range d.List() {
		v := d.Info(s)
		buffer.WriteString("{lat:" + strconv.FormatFloat(v.Lat, 'f', -1, 64) + ", lon:" + strconv.FormatFloat(v.Lon, 'f', -1, 64) + ", title:'" + v.Name + "'},")
	}
	buffer.Truncate(buffer.Len() - 1)
	buffer.WriteString("];")
	LatLonList = template.JS(buffer.String())
}

type ViewPage struct {
	Title        string
	LoginInfo    template.HTML
	Sensor       string
	Data         template.JS
	SensorList   template.HTML
	CustomScript template.HTML
}

const viewCustomScript = template.HTML(`    <!--[if lte IE 8]><script src="js/excanvas.min.js"></script><![endif]-->
    <script src="/assets/js/plugins/flot/jquery.flot.js"></script>
    <script src="/assets/js/plugins/flot/jquery.flot.tooltip.min.js"></script>
    <script src="/assets/js/plugins/flot/jquery.flot.resize.js"></script>
    <script src="/assets/js/plugins/flot/jquery.flot.time.js"></script>
    <link rel="stylesheet" id="themeCSS" href="/assets/js/plugins/jqrangeslider/css/classic-min.css">
    <script src="/assets/js/jquery-ui-1.10.4.custom.min.js"></script>
    <script src="/assets/js/plugins/jqrangeslider/lib/jquery.mousewheel.min.js"></script>
    <script src="/assets/js/plugins/jqrangeslider/jQRangeSlider-min.js"></script>
    <script src="/assets/cjs/sensorticker.js"></script>
`)

func viewHandler(w http.ResponseWriter, r *http.Request, u auth.User) {
	m := validView.FindStringSubmatch(r.URL.Path)
	if !d.Exists(m[1]) {
		err := templates.ExecuteTemplate(w,
			"sensor.html",
			ViewPage{"Datanomics™ alpha | Sensor not found", userMenu(u), "Error", "Sensor not found", SensorList, viewCustomScript})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		//d.Graph(m[1])
		//a, _ := json.Marshal(d.Load(m[1]))
		s := d.Load(m[1])
		s += "; var sensorID = '" + m[1] + "';"
		n := d.Info(m[1]).Name
		err := templates.ExecuteTemplate(w,
			"sensor.html",
			ViewPage{"Datanomics™ alpha | " + n, userMenu(u), n, template.JS(s), SensorList, viewCustomScript})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func queryInfoHandler(w http.ResponseWriter, r *http.Request) {
	m := validInfoQuery.FindStringSubmatch(r.URL.Path)
	if len(m) == 0 {
		http.Error(w, "Sensor not found", http.StatusNotFound)
		return
	}
	if !d.Exists(m[1]) {
		http.Error(w, "Sensor not found", http.StatusNotFound)
		return
	}
	debugln("Query for sensor " + m[1])
	t := d.Info(m[1])
	tt, _ := udb.Info(t.Owner)
	a, _ := json.Marshal(sensorMetadata{t.Name, tt.Name, t.Unit, t.Info, t.Lat, t.Lon})
	fmt.Fprintf(w, string(a))
}

type ServerStats struct {
	Sensors                 int
	OpenSensors             int
	RemoteServers           []string
	WebsocketClientsHome    int
	WebsocketClientsSensors int
	DatabaseSize            int64
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	var wsch = 0
	for i := range h.Connections {
		if h.Connections[i] == true {
			wsch++
		}
	}
	var wscc = 0
	for i := range sh.Connections {
		wscc += len(sh.Connections[i])
	}
	dbdir, _ := os.Open(sensorDataDir)
	dbfiles, _ := dbdir.Readdir(-1)
	var ds int64
	for _, i := range dbfiles {
		ds += i.Size()
	}
	t := ServerStats{d.Count(), d.OpenCount(), remoteServers, wsch, wscc, ds}
	a, _ := json.Marshal(t)
	fmt.Fprintf(w, string(a))
}

func logOutHandler(w http.ResponseWriter, r *http.Request) {
	auth.DeleteUserCookie(w, r)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "login.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
	}
}

func userLoggedHandler(w http.ResponseWriter, r *http.Request, u auth.User) {
	if !udb.Exists(u.Id()) {
		_ = udb.Add(User{u.Id(), u.Name(), u.Picture(), u.Email(), u.Link()})
		log.Println("Added new user: " + u.Id())
	}
	udbs, _ := json.Marshal(udb)
	err := ioutil.WriteFile(userDatabase, udbs, 0600)
	if err != nil {
		log.Println("Error saving user database.")
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

type serve404Page struct {
	Title        string
	LoginInfo    template.HTML
	SensorList   template.HTML
	CustomScript template.JS
}

func serve404(w http.ResponseWriter, u auth.User) {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	err := templates.ExecuteTemplate(w, "404.html", serve404Page{"Datanomics™ alpha | Page not found", userMenu(u), SensorList, template.JS("")})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
	}
}

func addSensorHandler(w http.ResponseWriter, r *http.Request, u auth.User) {
	if r.Method != "POST" {
		serve404(w, u)
		return
	}
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	accepted := true
	problems := ""

	matched, _ := regexp.Compile("^[a-zA-Z0-9\\s-]+$")
	name := ""
	if !matched.MatchString(r.FormValue("fsenName")) {
		accepted = false
		problems += "<li>name is wrong</li>"
	} else {
		name = r.FormValue("fsenName")
	}

	lat, err := strconv.ParseFloat(r.FormValue("fsenLat"), 64)
	if err != nil {
		accepted = false
		problems += "<li>latitude is wrong</li>"
	} else if lat > 90 || lat < -90 {
		accepted = false
		problems += "<li>latitude is wrong</li>"
	}

	lon, err := strconv.ParseFloat(r.FormValue("fsenLon"), 64)
	if err != nil {
		accepted = false
		problems += "<li>longitude is wrong</li>"
	} else if lon > 180 || lon < -180 {
		accepted = false
		problems += "<li>longtitude is wrong</li>"
	}

	units := "raw"
	if r.FormValue("fsenUnits") != "" {
		units = r.FormValue("fsenUnits")
	}
	if units == "%" { // TODO: extent this to search anywhere in units for percent symbol
		units = "\u0026#37;"
	}

	info := r.FormValue("fsenInfo")

	if r.FormValue("fsenAgree") != "on" {
		accepted = false
		problems += "<li>you did not agreed to the TOS</li>"
	}

	suuid, err := uuid.NewV4()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
	}

	if !accepted {
		fmt.Fprintf(w, `
         <div class="alert alert-danger fade in">
           <button type="button" class="close" data-dismiss="alert" aria-hidden="true">&times;</button>
           <h4>Sensor not added</h4>
           <p>We couldn't add your sensor. Here is a list of what went wrong:
           <ul>`+problems+`</ul>
           </p>
         </div>`)
		return
	}

	if accepted {
		err = d.AddM(suuid.String(), sensorMetadata{name, u.Id(), units, info, lat, lon})
		if err != nil {
			fmt.Fprintf(w, `
         <div class="alert alert-danger fade in">
           <button type="button" class="close" data-dismiss="alert" aria-hidden="true">&times;</button>
           <h4>Sensor not added</h4>
           <p>We couldn't add your sensor. An unknown error occured. Please try again and if it persists, contact support.</p>
         </div>`)
		} else {
			fmt.Fprintf(w, `
         <div class="alert alert-success fade in">
           <button type="button" class="close" data-dismiss="alert" aria-hidden="true">&times;</button>
           <h4>Sensor added</h4>
           <p>You can find your new sensor page <a href="/view/`+suuid.String()+`/">here</a>. </p>
           <p>You can send readings from your sensor to: <pre>http://datanomics.andmarios.com/log/`+suuid.String()+`</pre>. </p>
         </div>`)
		}
		log.Println("New sensor added: " + suuid.String())
		sensorList()
		latlonList()
	}
}

// type serve500Page struct {
//         Title string
//         LoginInfo template.HTML
//         SensorList template.HTML
//         CustomScript template.JS
// }

// func serve500(w http.ResponseWriter, u auth.User) {
// 	w.WriteHeader(http.StatusInternalServerError)
// 	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
// 	err := templates.ExecuteTemplate(w, "500.html", serve404Page{"Datanomics™ alpha | Internal Server Error", userMenu(u), SensorList, template.JS("")})
//         if err != nil {
//                 http.Error(w, err.Error(), http.StatusInternalServerError)
//                 log.Println(err)
//         }
//
