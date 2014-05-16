package main

import (
	"fmt"
	"strconv"
	"net/http"
	"log"
	"time"
	"html/template"
	"encoding/json"
	"bytes"
	"os"
	"github.com/bradrydzewski/go.auth"
	"io/ioutil"
	"regexp"
)

var templates *template.Template
var homeTemplate *template.Template

func logHandler(w http.ResponseWriter, r *http.Request) {
	m := validLog.FindStringSubmatch(r.URL.Path)
	if len(m) == 0 {
		http.Error(w, "Sensor not found", http.StatusNotFound)
		return
	}
	debug("Sensor " + m[1] + " sent value " + m[2])
	tnew := time.Now()
	if m[4] != "" {
		t, _ := strconv.ParseInt(m[5], 10, 64)
		if m[4] == "t" {
			tnew = time.Unix(t, 0)
		} else { // m[4] == "s"
			tnew = time.Unix(time.Now().Unix() - t, 0)
		}
	}
	// From down here there is a bit of duplication with sendRemoteReading(). Remember to change both if needed.
	if ! d.Exists(m[1]) { // Remove when you add code to add/delete sensors instead of adding them automatically.
		h.Pipe <- Hometicker{"Unknown sensor: " + m[1], "fa-times-circle", "danger",
			"Sensor <em>" + m[1] + "</em> isn't registered. Ignored."}
		// h.Pipe <- Hometicker{"New sensor: " + m[1], "fa-check-circle", "success",
		//	"Sensor <em>" + m[1] + "</em> succesfully added."}
		// d.AddT(m[1], tnew) // This is not needed. Sensors are added automatically upon first reading. It is here only to make the next command to work.
		// sensorList()
		// latlonList()
		http.Error(w, "Sensor not found", http.StatusNotFound)
                return
	}
	// We can't check this with rrd cache. We do it though on database flush.
	// _, told := d.Last(m[1])
	// if ! tnew.After(told) {
	// 	http.Error(w, "Sensor send out of order timestamp", http.StatusNotFound)
	// 	h.Pipe <- Hometicker{m[1] + ": out of order reading", "fa-times-circle", "danger",
	// 		m[1] + "</em> sent out of order value <em>" + m[2] + "</em> at <em>" + tnew.String() + "</em>. Ignored."}
	// 	return
	// }

	d.StoreT(m[1], m[2], tnew)
	srC.Pipe <- remoteReading{m[1], m[2], tnew}
	t := d.Info(m[1]).Name
	h.Pipe <- Hometicker{"<a href='/view/" + m[1] + "'>" + t + "</a>: new reading", "fa-plus-circle", "info",
		t + "</em> sent value <em>" + m[2] + "</em> at <em>" + tnew.String() + "</em>"}
	sh.Pipe <- m[1]
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
	Title string
        LoginInfo template.HTML
	SensorList template.HTML
	CustomScript template.HTML
	LatLonList template.JS
}

const homeCustomScript = template.HTML(`
     <script src="/assets/cjs/hometicker.js"></script>
     <script src="http://maps.google.com/maps/api/js?sensor=false&libraries=geometry&v=3.7"></script>
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
            <li><a href="` + u.Link() + `"><img class="img-rounded" height="50px" src="`+ u.Picture() +`" /> ` + u.Name() + `</a></li>
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
		buffer.WriteString("{lat:"+ strconv.FormatFloat(v.Lat, 'f', -1, 64) + ", lon:" + strconv.FormatFloat(v.Lon, 'f', -1, 64) + ", title:'" + v.Name + "'},")
	}
	buffer.Truncate(buffer.Len() - 1)
	buffer.WriteString("];")
	LatLonList = template.JS(buffer.String())
}

type ViewPage struct {
	Title string
	LoginInfo template.HTML
	Sensor string
	Data template.JS
	SensorList template.HTML
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
        if ! d.Exists(m[1]) {
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
	if ! d.Exists(m[1]) {
		http.Error(w, "Sensor not found", http.StatusNotFound)
                return
        }
	debugln("Query for sensor " + m[1])
	a, _ := json.Marshal(d.Info(m[1]))
	fmt.Fprintf(w, string(a))
}

type ServerStats struct {
	Sensors int
	OpenSensors int
	RemoteServers []string
	WebsocketClientsHome int
	WebsocketClientsSensors int
	DatabaseSize int64
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
	if ! udb.Exists(u.Id()) {
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
        Title string
        LoginInfo template.HTML
	SensorList template.HTML
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
	lat, err := strconv.ParseFloat(r.FormValue("fsenLat"), 64)
	if err != nil {
		log.Println("lat not number")
	} else	if lat > 90 || lat < -90 {
		log.Println("wrong lat")
	}
	lon, err := strconv.ParseFloat(r.FormValue("fsenLon"), 64)
	if err != nil {
		log.Println("lon not number")
	} else if lon > 180 || lon < -180 {
		log.Println("wrong lon")
	}
	matched, _ := regexp.Compile("^[a-zA-Z0-9\\s-]+$")
	if ! matched.MatchString(r.FormValue("fsenName")) {
		log.Println("Name contains not permitted characters.")
	}
	matched, _ = regexp.Compile("^[a-zA-Z0-9]+$")
	if ! matched.MatchString(r.FormValue("fsenUUID")) {
		log.Println("UUID contains not permitted characters.")
	}
	if r.FormValue("fsenUnits") == "" {
		log.Println("Setting units to raw.")
	}
	if r.FormValue("fsenAgree") != "on" {
		log.Println("Not agreed to TOS")
	}


//	log.Println(r)
	http.Redirect(w, r, "/", http.StatusFound)
//	fmt.Fprintf(w, "ok")
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








