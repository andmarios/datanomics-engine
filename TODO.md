# Short Term

- Accept sensor values only in historical order.
- Home page show live updates


# For production

- Oauth has something set to false, should set to true.
- Export to CSV: breaks for firefox when units are % (humidity).
- Websockets connection accepted only from our server. Example code for home/sensorTickerHandler:
  `log.Println(s.Ws.RemoteAddr().String())`
  This always returns the URL of the server without paths at the end, eg http://datanomics.andmarios.com (whether we are at home or sensor view).
- /iq/<sensor> leaks private data. Needs a change in implementation.


# For OpenSourcing

- Google Client Keys and first Google Public APIs have been leaked to git. Revoke them and use new Client Keys. (Public API isn't use anywhere anymore)


# Long Term

- Users: normal database (eg MariaDB), use custom UUID as providers' ID varies a lot
- db.json: save only when needed and with every flush
- Share public sensor
- Aggregate sensors (many sources -> one sensor)
- Log to be: /log/UID/SID/[st]/timestamp. UID instead of username to prevent guessing
- Manually add/delete sensors
- Cache/expire static items
- On replication part, store values when you can't send them and retry.
- Embed graph
