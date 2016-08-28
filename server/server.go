package main

import (
	"BuffBusTracker/lib"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	PORT = "8080"
	// How often to send requests
	REQ_INTERVAL = 10
	// Multiplier to REQ_INTERVAL for this source
	// eg 3 means request from this source every 3*10 seconds
	ETA_MULTIPLIER = 1
	RTD_MULTIPLIER = 3
)

var (
	JsonToSend             []byte
	RouteJsonToSend        []byte
	StopJsonToSend         []byte
	BusJsonToSend          []byte
	AnnouncementJsonToSend []byte
	RequestCount           uint64
)

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
	var conf = ReadConfig()
	for {
		RequestCount++
		log.Println("Request:", RequestCount)

		StartTime := time.Now()

		// Indentify what sources to include
		included := lib.RequestedSources{
			ETA: false,
			RTD: false,
		}
		if RequestCount%ETA_MULTIPLIER == 0 {
			included.ETA = true
		}
		if RequestCount%RTD_MULTIPLIER == 0 {
			included.RTD = true
		}

		JSONs := lib.CreateFinalObjects(included, conf)

		// Update JSONs being served
		RouteJsonToSend = JSONs.Routes
		StopJsonToSend = JSONs.Stops
		BusJsonToSend = JSONs.Buses
		AnnouncementJsonToSend = JSONs.Announcements

		TimeElapsed := time.Since(StartTime)
		// Sleep remaining time
		time.Sleep((REQ_INTERVAL * time.Second) - TimeElapsed)
	}
}

/* Reads info from config file */
func ReadConfig() lib.Config {
	configfile := "config.json"
	file, err := os.Open(configfile)
	if err != nil {
		log.Fatal("Config file is missing: ", configfile)
	}
	decoder := json.NewDecoder(file)
	config := lib.Config{}
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal("Unable to parse config: ", configfile)
	}
	// Just in case
	sort.Strings(config.Buses)
	return config
}

/* Setup HTTP handlers oncreate */
func init() {
	http.HandleFunc("/buses", bushandler)
	http.HandleFunc("/stops", stophandler)
	http.HandleFunc("/routes", routehandler)
	http.HandleFunc("/announcements", announcementhandler)
	http.HandleFunc("/public/", publichandler)

}

func main() {
	// Create a go routine for this so it will run concurrently with the server
	go SetJson()

	// Listen on specified port and fail if anything bad happens
	log.Fatal(http.ListenAndServe(":"+PORT, nil))
}
