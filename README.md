Datanomics implements a server to log sensor values through simple GET requests and let various clients to retrieve them.

To compile, place (or link) datanomics to your GOPATH src directory and run build.
Example:

    $ ln -s datanomics $GOPATH/src/datanomics
    $ go build datanomics


Sensor requests.

Simple log:

    $ curl http://127.0.0.1:8080/log/<SENSOR-UUID>/<SENSOR-FLOAT-VALUE>

Custom time:

    $ curl http://127.0.0.1:8080/log/<SENSOR-UUID>/<SENSOR-FLOAT-VALUE>/t/<UNIX-TIME-IN-SECONDS>

Seconds since time:

    $ curl http://127.0.0.1:8080/log/<SENSOR-UUID>/<SENSOR-FLOAT-VALUE>/s/<SECONDS-SINCE-LOGGING>
