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
	JsonToSend []byte
)

type myHandler struct {
	Json []byte
}

func (*myHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(JsonToSend)
}

func SetJson() {
	for {

		FinalCreator := lib.CreateFinalCreator()
		FinalJson, err := FinalCreator.CreateFinalJson()
		if err != nil {
			panic(err)
		}
		JsonToSend = *FinalJson
		<-time.After(1000000 * time.Second)
	}
}

func main() {
	go SetJson()

	s := &http.Server{
		Addr:           ":8080",
		Handler:        &myHandler{},
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Fatal(s.ListenAndServe())
}
