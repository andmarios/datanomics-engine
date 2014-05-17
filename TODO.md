# Short Term

- _Accept sensor values only in historical order._
- _Home page show live updates_


# For production

- Oauth has something set to false, should set to true.
- _Export to CSV: breaks for firefox when units are % (humidity)._
- Websockets connection accepted only from our server. Example code for home/sensorTickerHandler:
  `log.Println(s.Ws.RemoteAddr().String())`
  This always returns the URL of the server without paths at the end, eg http://datanomics.andmarios.com (whether we are at home or sensor view).
- /iq/<sensor> leaks private data. Needs a change in implementation.
- **DEMO** Binary Log of requests to re-play in case of db loss.
- **DEMO** Log from POST.

# For OpenSourcing

- Google Client Keys and first Google Public APIs have been leaked to git. Revoke them and use new Client Keys. (Public API isn't use anywhere anymore)


# Long Term

- Users: normal database (eg MariaDB), use custom UUID as providers' ID varies a lot
- db.json: save only when needed and with every flush
- Share public sensor
- Aggregate sensors (many sources -> one sensor)
- Log to be: /log/UID/SID/[st]/timestamp. UID instead of username to prevent guessing **No! Maybe exposes personal data. Ties UID to SID maybe.**
- Manually _add_/delete sensors
- Cache/expire static items
- **DEMO** On replication part, store values when you can't send them and retry.
- Embed graph
