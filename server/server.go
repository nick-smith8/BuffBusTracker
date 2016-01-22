package main

import (
	//"buffbus/lib"
	"BuffBusTracker/lib"
	"log"
	"net/http"
	"time"
	"strings"
)

const (
	ROUTES_URL = "http://buffbus.etaspot.net/service.php?service=get_routes"
	STOPS_URL  = "http://buffbus.etaspot.net/service.php?service=get_stops"
	BUSES_URL  = "http://buffbus.etaspot.net/service.php?service=get_vehicles&includeETAData=1&orderedETAArray=1"
)

var (
	JsonToSend      []byte
	BusJsonToSend   []byte
	StopJsonToSend  []byte
	RouteJsonToSend []byte
	AnnouncementJsonToSend []byte
)

type myHandler struct{}

func analyticsRequest(s string, i string) {

	// Strip off the ip of the client and send it with the analytics
	resp, err := http.Get("http://www.google-analytics.com/collect?v=1&t=pageview&tid=UA-68940534-1&cid=555&dh=cherishapps.me&dp=%2F" + s + "&uip=" + strings.Split(i,":")[0])
	if err != nil {
		log.Println(err)
	}	
	defer	resp.Body.Close()
}

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
	log.Println(r.RemoteAddr)
	//go analyticsRequest("stops",r.RemoteAddr)

}
func routehandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(RouteJsonToSend)
}
func announcementhandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(AnnouncementJsonToSend)
}
func publichandler(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w,r,"./src/BuffBusTracker/"+r.URL.Path)
}
// Sets the global variables to the json that will be sent
// Waits on the channel for a certain amount of time to then make the get to ETA's api
func SetJson() {
	for {
		//t := time.Now()
		FinalCreator := lib.CreateFinalCreator()
		//t1 := time.Now()
		BusJson, StopJson, RouteJson, AnnouncementJson, err := FinalCreator.CreateFinalJson()
		if err != nil {
			// panic(err)
		}
		BusJsonToSend = BusJson
		StopJsonToSend = StopJson
		RouteJsonToSend = RouteJson
		AnnouncementJsonToSend = AnnouncementJson 
		<-time.After(10 * time.Second)
		//t2 := time.Now()
	}
}

func init() {
	http.HandleFunc("/buses", bushandler)
	http.HandleFunc("/stops", stophandler)
	http.HandleFunc("/routes", routehandler)
	http.HandleFunc("/announcements",announcementhandler)
	http.HandleFunc("/public/", publichandler)
}

func main() {
	// Create a go routine for this so it will run concurrently with the server
	go SetJson()

	// Listen on port 8080 and fail if anything bad happens
	log.Fatal(http.ListenAndServe(":8080", nil))
}
