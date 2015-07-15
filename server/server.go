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

		<-time.After(1000000 * time.Second)
	}
}
func init() {

	http.HandleFunc("/buses", bushandler)
	http.HandleFunc("/stops", stophandler)
	http.HandleFunc("/routes", routehandler)
}
func main() {
	go SetJson()

	// s := &http.Server{
	// 	Addr:           ":8080",
	// 	Handler:        &myHandler{},
	// 	ReadTimeout:    10 * time.Second,
	// 	WriteTimeout:   10 * time.Second,
	// 	MaxHeaderBytes: 1 << 20,
	// }
	// http.HandleFunc("/buses", func(w http.ResponseWriter, r *http.Request) {
	// 	w.Header().Set("Content-Type", "application/json")
	// 	w.WriteHeader(200)
	// 	w.Write(BusJsonToSend)
	// })
	// http.HandleFunc("/stops", func(w http.ResponseWriter, r *http.Request) {
	// 	w.Header().Set("Content-Type", "application/json")
	// 	w.WriteHeader(200)
	// 	w.Write(StopJsonToSend)
	// })
	// http.Handle("/routes",
	// })

	go log.Fatal(http.ListenAndServe(":8080", nil))

	// //go log.Fatal(r.ListenAndServe())
	// log.Fatal(s.ListenAndServe())
}
