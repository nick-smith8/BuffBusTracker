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
