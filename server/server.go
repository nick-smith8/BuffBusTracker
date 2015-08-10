package main

import (
	"time"
	//"fmt"
	//"encoding/json"
	"buffbus/lib"
	"log"
	"net/http"
)

const (
	ROUTES_URL = "http://buffbus.etaspot.net/service.php?service=get_routes&token=TESTING"
	STOPS_URL  = "http://buffbus.etaspot.net/service.php?service=get_stops&token=TESTING"
	BUSES_URL  = "http://buffbus.etaspot.net/service.php?service=get_vehicles&includeETAData=1&orderedETAArray=1&token=TESTING"
)

var (
	JsonToSend      []byte
	BusJsonToSend   []byte
	StopJsonToSend  []byte
	RouteJsonToSend []byte
)

type myHandler struct{}

// Define the different functions to handle the routes
func bushandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(BusJsonToSend)
}
func stophandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(StopJsonToSend)
}
func routehandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(RouteJsonToSend)
}

// Sets the global variables to the json that will be sent
// Waits on the channel for a certain amount of time to then make the get to ETA's api
func SetJson() {
	for {
		FinalCreator := lib.CreateFinalCreator()
		BusJson, StopJson, RouteJson, err := FinalCreator.CreateFinalJson()
		if err != nil {
			panic(err)
		}
		BusJsonToSend = BusJson
		StopJsonToSend = StopJson
		RouteJsonToSend = RouteJson

		<-time.After(10 * time.Second)
	}
}

func init() {
	http.HandleFunc("/buses", bushandler)
	http.HandleFunc("/stops", stophandler)
	http.HandleFunc("/routes", routehandler)
}

func main() {
	// Create a go routine for this so it will run concurrently with the server
	go SetJson()

	// Listen on port 8080 and fail if anything bad happens
	go log.Fatal(http.ListenAndServe(":8080", nil))
}
