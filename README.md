Datanomics implements a server to log sensor values through simple GET and/or POST requests and let various clients to retrieve them.

---
__Disclaimer__

Usually I upload my projects to Github to share them with the world. The primary motive for adding Datanomics to Github though is to help me with deployment.
For this reason I haven't clean up the project, neither I tried to make it easy for everyone to understand. If it suits you feel free to use it (AGPL 3.0 license applies) but I can't guarantee updates, an always consistent state, help or documentation.

There is a second part to the project, the web interface, which has html templates and javascript libraries as well as third party javascript and css libraries. It will be released at a later time.

---


To compile, place (or link) datanomics to your GOPATH src directory and run build.
Example:

    $ ln -s datanomics $GOPATH/src/datanomics
    $ go build datanomics


# Sensor requests.

Simple GET request will log with server's time:

    $ curl http://datanomics/log/<SENSOR-UUID>/<SENSOR-FLOAT-VALUE>

You can set your own timestamp:

    $ curl http://datanomics/log/<SENSOR-UUID>/<SENSOR-FLOAT-VALUE>/t/<UNIX-TIME-IN-SECONDS>

You can set how many seconds passed since your measurement and server will subtract them from its time:

    $ curl http://datanomics/log/<SENSOR-UUID>/<SENSOR-FLOAT-VALUE>/s/<SECONDS-SINCE-LOGGING>
