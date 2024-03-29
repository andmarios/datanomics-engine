# Short Term

- _DONE_ _Accept sensor values only in historical order._
- _DONE_ _Home page show live updates_

# For production

- Oauth has something set to false, should set to true.
- _Export to CSV: breaks for firefox when units are % (humidity)._
- Websockets connection accepted only from our server. Example code for home/sensorTickerHandler:
  `log.Println(s.Ws.RemoteAddr().String())`
  This always returns the URL of the server without paths at the end, eg http://datanomics.andmarios.com (whether we are at home or sensor view).
- /iq/<sensor> leaks private data. Needs a change in implementation.
- _DONE_ **DEMO** Binary Log of requests to re-play in case of db loss.
- _DONE_ **DEMO** Log from POST.
- _BUG_ Escape strings in units, name and sensor info. For example a % in sensor info breaks javascript.
- Instead of UUID use a 10-12 character [a-zA-Z0-9] identifier. First two characters keep the same, they may act as routing in the feature.
- Use a second UUID for logging, so that you can change it if needed (eg it leaked to bad party and user needs to change it)
- _DONE_ email when a sensor hasn't send a value for a user-defined period
-  **IMPORTANT** maps aren't concurrency safe, check if I always use mutex (maybe -race?)

# For Making Public

- Google Client Keys and first Google Public APIs have been leaked to git. Revoke them and use new Client Keys. (Public API isn't use anywhere anymore)


# Long Term

- Users: normal database (eg MariaDB), use custom UUID as providers' ID varies a lot
- db.json: save only when needed and with every flush
- Share public sensor
- Aggregate sensors (many sources -> one sensor)
- _DONE_ Log to be: /log/UID/SID/[st]/timestamp. UID instead of username to prevent guessing **No! Maybe exposes personal data. Ties UID to SID maybe.**
- _SEMI DONE_ Manually _add_/delete sensors
- Cache/expire static items
- _DONE_ **DEMO** On replication part, store values when you can't send them and retry.
- Embed graph
-  **TODO** _TEMP FIXED_ When we can't send to remote server, logging halts because we used a synchronous channel. I set the channel buffer to 1000000 instead of 1 to prevent temporarily this.
-  **TODO** _TEMP FIXED_ sendRemoteReadings() has a hard limit on how many entries it will keep in case of timeout. Also it resends all values to all servers on failure. Something more elegant IIO.
- a more elegant solution for robots.txt which now is part of homeHandler ?
