package main

import (
	"BuffBusTracker/lib"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	ROUTES_URL   = "http://buffbus.etaspot.net/service.php?service=get_routes"
	STOPS_URL    = "http://buffbus.etaspot.net/service.php?service=get_stops"
	BUSES_URL    = "http://buffbus.etaspot.net/service.php?service=get_vehicles&includeETAData=1&orderedETAArray=1"
	PORT         = "8080"
	REQ_INTERVAL = 10
)

var (
	JsonToSend             []byte
	RouteJsonToSend        []byte
	StopJsonToSend         []byte
	BusJsonToSend          []byte
	AnnouncementJsonToSend []byte
)

type Config struct {
	Username  string
	Passwords []string
}

func analyticsRequest(s string, i string) {
	// Strip off the ip of the client and send it with the analytics
	resp, err := http.Get("http://www.google-analytics.com/collect?v=1&t=pageview&tid=UA-68940534-1&cid=555&dh=cherishapps.me&dp=%2F" + s + "&uip=" + strings.Split(i, ":")[0])
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()
}

/* Define the different functions to handle the routes */
func routehandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(RouteJsonToSend)
}

func stophandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(StopJsonToSend)
	log.Println(r.RemoteAddr)
	//go analyticsRequest("stops",r.RemoteAddr)
}

func bushandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(BusJsonToSend)
}

func announcementhandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(AnnouncementJsonToSend)
}

func publichandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./src/BuffBusTracker/"+r.URL.Path)
}

// Sets the global variables to the json that will be sent
// Waits on the channel for a certain amount of time to then make the get to ETA's api
func SetJson() {
	for {
		StartTime := time.Now()
		Creator := lib.CreateParsedObjects()
		FinalObjects, err := Creator.CreateFinalJson()
		//BusJson, StopJson, RouteJson, AnnouncementJson, err := Creator.CreateFinalJson()
		if err != nil {
			// panic(err)
		}
		RouteJsonToSend = FinalObjects.Routes
		StopJsonToSend = FinalObjects.Stops
		BusJsonToSend = FinalObjects.Buses
		AnnouncementJsonToSend = FinalObjects.Announcements

		TimeElapsed := time.Since(StartTime)
		// Sleep remaining time
		time.Sleep((REQ_INTERVAL * time.Second) - TimeElapsed)
	}
}

/* Reads info from config file */
func ReadConfig() Config {
	configfile := "config.json"
	file, err := os.Open(configfile)
	if err != nil {
		log.Fatal("Config file is missing: ", configfile)
	}
	decoder := json.NewDecoder(file)
	config := Config{}
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal("Unable to parse config: ", configfile)
	}
	return config
}

/* Setup HTTP handlers oncreate */
func init() {
	http.HandleFunc("/buses", bushandler)
	http.HandleFunc("/stops", stophandler)
	http.HandleFunc("/routes", routehandler)
	http.HandleFunc("/announcements", announcementhandler)
	http.HandleFunc("/public/", publichandler)

	var conf = ReadConfig()
	log.Print(conf.Passwords[0])
}

func main() {
	// Create a go routine for this so it will run concurrently with the server
	go SetJson()

	// Listen on specified port and fail if anything bad happens
	log.Fatal(http.ListenAndServe(":"+PORT, nil))
}
