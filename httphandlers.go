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
	if ! d.Exists(m[1]) { // Remove when you add code to add/delete sensors instead of adding them automatically.
		h.Pipe <- Hometicker{"New sensor: " + m[1], "fa-check-circle", "success",
			"Sensor <em>" + m[1] + "</em> succesfully added."}
		d.AddT(m[1], tnew) // This is not needed. Sensors are added automatically upon first reading. It is here only to make the next command to work.
		sensorList()
	}
	// TODO: Fix it.
	// _, told := d.Last(m[1])
	// if ! tnew.After(told) {
	// 	http.Error(w, "Sensor send out of order timestamp", http.StatusNotFound)
	// 	h.Pipe <- Hometicker{m[1] + ": out of order reading", "fa-times-circle", "danger",
	// 		m[1] + "</em> sent out of order value <em>" + m[2] + "</em> at <em>" + tnew.String() + "</em>. Ignored."}
	// 	return
	// }

	d.StoreT(m[1], m[2], tnew)
	h.Pipe <- Hometicker{m[1] +": new reading", "fa-plus-circle", "info",
		m[1] + "</em> sent value <em>" + m[2] + "</em> at <em>" + tnew.String() + "</em>"}
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
                   <a href="/view/` + s + `">` + s + `</a>
                 </li>`)
	}
	SensorList = template.HTML(buffer.String())
}

type HomePage struct {
	Title string
	SensorList template.HTML
	CustomScript template.HTML
}

const homeCustomScript = template.HTML(`<script src="/assets/cjs/hometicker.js"></script>`)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w,
		"home.html",
		HomePage{"Datanomics alpha", SensorList, homeCustomScript})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
	}
}

type ViewPage struct {
	Title string
	Sensor string
	Data template.JS
	SensorList template.HTML
	CustomScript template.HTML
}
const viewCustomScript = template.HTML(`    <!--[if lte IE 8]><script src="js/excanvas.min.js"></script><![endif]-->
    <script src="/assets/js/plugins/flot/jquery.flot.js"></script>
    <script src="/assets/js/plugins/flot/jquery.flot.tooltip.min.js"></script>
    <script src="/assets/js/plugins/flot/jquery.flot.resize.js"></script>
    <script>
      var options = {
        xaxes: [{
            mode: 'time'
        }],
        yaxes: [{
            min: null,
            max: null
        }, {
            // align if we are to the right
            alignTicksWithAxis: 1,
            position: 'right'
        }],
        series: {
            lines: {
                 show: true,
                 fill: true
            },
            points: { show: false }
        },
        legend: {
            position: 'sw'
        },
        grid: {
            hoverable: true //IMPORTANT! this is needed for tooltip to work
        },
        tooltip: true,
        tooltipOpts: {
            content: "%s for %x was %y.2",
            xDateFormat: "%y-%0m-%0d %0H:%0M",
            yDateFormat: null,
            onHover: function(flotItem, $tooltipEl) {
              // console.log(flotItem, $tooltipEl);
            }
        }
      };
      $(document).ready(function() {
         $.plot($("#sensorGraph"), [{ data: sData, label: "Sensor AVG Data"}], options );
      });
    </script>
`)

func viewHandler(w http.ResponseWriter, r *http.Request) {
	m := validView.FindStringSubmatch(r.URL.Path)
        if ! d.Exists(m[1]) {
		err := templates.ExecuteTemplate(w,
			"sensor.html",
			ViewPage{"Datanomics alpha | Sensor not found", "Error", "Sensor not found", SensorList, viewCustomScript})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		//d.Graph(m[1])
		//a, _ := json.Marshal(d.Load(m[1]))
		err := templates.ExecuteTemplate(w,
                        "sensor.html",
                        ViewPage{"Datanomics alpha | " + m[1], m[1], template.JS(d.Load(m[1])), SensorList, viewCustomScript})
                if err != nil {
                        http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}















