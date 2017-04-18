# BuffBusTracker

Server that pulls from eta api, formats the data to the JSON needed by our IOS/Android app and creates an api for the access of that data.

## Documentation

### Getting Started

``` bash
./control start

```

  TODO:
  * Clean-up code
  * Add persistent logging for public site(docker volumes)
  * Add a way to notify by email when the server fails (bash script + docker logging?)
  * Rewrite stop names based on the direction the bus is going
  * Redo config organization to allow URL encoding (TransitTime), per-source auth

  Future:
  * Send polylines from the server and code the app to read them in on app load?
  * Intelligent way of aligning parse times to minimize data stailness (exponential backoff)?
