# Short Term

- Accept sensor values only in historical order.
- Home page show live updates


# For production

- Oauth has something set to false, should set to true.

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
