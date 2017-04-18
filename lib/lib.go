package lib

import (
	"encoding/json"
	"github.com/empatica/csvparser"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	pb "BuffBusTracker/lib/proto"
	"github.com/golang/protobuf/proto"
)

const (
	ROUTES_URL             = "http://buffbus.etaspot.net/service.php?service=get_routes"
	STOPS_URL              = "http://buffbus.etaspot.net/service.php?service=get_stops"
	BUSES_URL              = "http://buffbus.etaspot.net/service.php?service=get_vehicles&includeETAData=1&orderedETAArray=1"
	ANNOUNCEMENTS_URL      = "http://buffbus.etaspot.net/service.php?service=get_service_announcements"
	RTD_ROUTES_URL         = "http://www.rtd-denver.com/google_sync/TripUpdate.pb"
	RTD_BUSES_URL          = "http://www.rtd-denver.com/google_sync/VehiclePosition.pb"
	TRANSITTIME_ROUTES_URL = "http://www.transitime.org/api/v1/key/SECRET/agency/rtd-denver/command/gtfs-rt/tripUpdates"
	TRANSITTIME_BUSES_URL  = "http://www.transitime.org/api/v1/key/SECRET/agency/rtd-denver/command/gtfs-rt/vehiclePositions"
	USER_AGENT             = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_1) AppleWebKit/600.1.25 (KHTML, like Gecko) Version/8.0 Safari/600.1.25"
	RTD_STOPS_FILE         = "RTDstops.txt"
)

var (
	// Holds processed list of stops from RTD schedule
	rtd_stops = []StopInfo{}
	// Hold the previous requests for sources that were not updated
	PreviousSources = []Source{}
)

/* Holds information parsed from config.json (used for RTD routes) */
type Config struct {
	Name     string         `json:"Name"`
	Username string         `json:"Username"`
	Password string         `json:"Password"`
	Key      string         `json:"Key"`
	Buses    map[string]int `json:"Buses"`
}
type Configs struct {
	Sources []Config `json:"Sources"`
}

/* List of sources requested by the server for updates */
type RequestedSources struct {
	ETA         bool
	RTD         bool
	TransitTime bool
}

/* Struct definitions for the json coming in from ETA */
type ETA_Routes struct {
	GetRoutes []struct {
		ID                 int    `json:"id"`
		Name               string `json:"name"`
		Stops              []int  `json:"stops"`
		Color              string `json:"color"`
		EncLine            string `json:"encLine"`
		Order              int    `json:"order"`
		ShowDirection      bool   `json:"showDirection"`
		ShowPlatform       bool   `json:"showPlatform"`
		ShowScheduleNumber int    `json:"showScheduleNumber"`
		Type               string `json:"type"`
		VType              string `json:"vType"`
	} `json:"get_routes"`
}

type ETA_Stops struct {
	GetStops []struct {
		ID   int     `json:"id"`
		Name string  `json:"name"`
		Lat  float64 `json:"lat"`
		Lng  float64 `json:"lng"`
	} `json:"get_stops"`
}

type ETA_Buses struct {
	GetBuses []struct {
		Equipmentid        string      `json:"equipmentID"`
		Lat                float64     `json:"lat"`
		Lng                float64     `json:"lng"`
		Routeid            int         `json:"routeID"`
		Nextstopid         interface{} `json:"nextStopID"`
		Schedulenumber     string      `json:"scheduleNumber"`
		Inservice          int         `json:"inService"`
		Minutestonextstops []Minutestonextstops
		Onschedule         interface{} `json:"onSchedule"`
		Receivetime        int64       `json:"receiveTime"`
	} `json:"get_vehicles"`
}
type Minutestonextstops struct {
	StopID  string `json:"stopID"`
	Minutes int    `json:"minutes"`
}

type ETA_Announcements struct {
	GetServiceAnnouncements []struct {
		Type          string `json:"type"`
		Announcements []struct {
			End   string `json:"end"`
			Start string `json:"start"`
			Text  string `json:"text"`
		} `json:"announcements"`
	} `json:"get_service_announcements"`
}

/* Collection of parsed structs from clients */
type ParsedObjects struct {
	ETA_Routes        ETA_Routes
	ETA_Stops         ETA_Stops
	ETA_Buses         ETA_Buses
	ETA_Announcements ETA_Announcements
	RTD               interface{}
}

/* Final structs each source is parsed in to */
type RouteInfo struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Stops []int  `json:"stops"`
}

type StopInfo struct {
	ID                int              `json:"id" csv:"0"`
	Name              string           `json:"name" csv:"1"`
	NextBusTimesFinal map[string][]int `json:"nextBusTimes"`
	Lat               float64          `json:"lat" csv:"2"`
	Lng               float64          `json:"lng" csv:"3"`
}

type BusInfo struct {
	RouteID int     `json:"routeID"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
}

type AnnouncementInfo struct {
	Announcements []string `json:"announcements"`
}

/* Collection of fully parsed objects from a Client */
type FinalObjects struct {
	Routes        []RouteInfo
	Stops         []StopInfo
	Buses         []BusInfo
	Announcements []AnnouncementInfo
}

/* Collection of final data to be sent by the server (merged FinalObjects) */
type FinalJSONs struct {
	Routes        []byte
	Stops         []byte
	Buses         []byte
	Announcements []byte
}

/* Represents a source of information */
type Client struct {
	Url  string
	Type string
	Auth
}
type Auth struct {
	Username string
	Password string
}

/* Maintains association between Clients and structs to hold parsed response
 * Each client gets parsed to a GenericStructure or a ProtoStructure
 */
type Request struct {
	Client
	GenericStructure interface{}
	ProtoStructure   *pb.FeedMessage
}

/* Represents a list of requests from the same location (ETA or RTD) and their FinalObject */
type Source struct {
	Name     string
	Requests []Request
	Final    FinalObjects
	Config   Config
}

/* Create sorter for StopInfo objects */
type IDSorter []StopInfo

func (a IDSorter) Len() int           { return len(a) }
func (a IDSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a IDSorter) Less(i, j int) bool { return a[i].ID < a[j].ID }

func init() {
	/* Parse stop schema from RTD schedule data */
	rtd_stops = LoadStopData()
}

/* Make an HTTP call to a Client and return the raw output */
func (c Client) httpCall() ([]byte, error) {
	req, err := http.NewRequest("GET", c.Url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", USER_AGENT)

	// If auth is set
	if (Auth{}) != c.Auth {
		req.SetBasicAuth(c.Auth.Username, c.Auth.Password)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("Response did not return 200. Status received: ", resp.Status)
		log.Println("On this request: ", resp.Request)
		return nil, err
	}

	body, err1 := ioutil.ReadAll(resp.Body)
	if err1 != nil {
		return nil, err1
	}

	return body, nil
}

/* Create struct of parsed responses from servers */
func CreateFinalObjects(included RequestedSources, confs Configs) FinalJSONs {
	Sources := []Source{}
	Conf := Config{}
	SourceName := ""

	// Initialize ETA
	SourceName = "ETA"
	var ETARequests []Request
	if included.ETA {
		ETARequests = []Request{
			{Client: Client{Url: ROUTES_URL, Type: "json"},
				GenericStructure: &ETA_Routes{}},
			{Client: Client{Url: STOPS_URL, Type: "json"},
				GenericStructure: &ETA_Stops{}},
			{Client: Client{Url: BUSES_URL, Type: "json"},
				GenericStructure: &ETA_Buses{}},
			{Client: Client{Url: ANNOUNCEMENTS_URL, Type: "json"},
				GenericStructure: &ETA_Announcements{}},
		}
	}
	ETASource := Source{
		Name:     SourceName,
		Requests: ETARequests,
		Final:    FinalObjects{},
		Config:   Config{},
	}
	Sources = append(Sources, ETASource)

	// Initialize RTD
	SourceName = "RTD"
	for _, elem := range confs.Sources {
		if elem.Name == SourceName {
			Conf = elem
			break
		}
	}

	var RTDRequests []Request
	if included.RTD {
		RTDRequests = []Request{
			{Client: Client{
				Url:  RTD_ROUTES_URL,
				Type: "proto",
				Auth: Auth{Username: Conf.Username, Password: Conf.Password},
			}, ProtoStructure: &pb.FeedMessage{}},
			{Client: Client{
				Url:  RTD_BUSES_URL,
				Type: "proto",
				Auth: Auth{Username: Conf.Username, Password: Conf.Password},
			}, ProtoStructure: &pb.FeedMessage{}},
		}
	}
	RTDSource := Source{
		Name:     SourceName,
		Requests: RTDRequests,
		Final:    FinalObjects{},
		Config:   Conf,
	}
	Sources = append(Sources, RTDSource)

	// Initialize TransitTime
	SourceName = "TransitTime"
	for _, elem := range confs.Sources {
		if elem.Name == SourceName {
			Conf = elem
			break
		}
	}

	// Insert the authentication key
	TransitRoutesAuthUrl := strings.Replace(TRANSITTIME_ROUTES_URL, "SECRET", Conf.Key, 1)
	TransitBusesAuthUrl := strings.Replace(TRANSITTIME_BUSES_URL, "SECRET", Conf.Key, 1)

	var TransitTimeRequests []Request
	if included.TransitTime {
		TransitTimeRequests = []Request{
			{Client: Client{
				Url:  TransitRoutesAuthUrl,
				Type: "proto",
			}, ProtoStructure: &pb.FeedMessage{}},
			{Client: Client{
				Url:  TransitBusesAuthUrl,
				Type: "proto",
			}, ProtoStructure: &pb.FeedMessage{}},
		}
	}
	TransitTimeSource := Source{
		Name:     SourceName,
		Requests: TransitTimeRequests,
		Final:    FinalObjects{},
		Config:   Conf,
	}
	Sources = append(Sources, TransitTimeSource)

	for i, _ := range Sources {
		source := &Sources[i]
		// Send HTTP requests
		for j, _ := range source.Requests {
			request := &source.Requests[j]
			clientResp, err := request.Client.httpCall()
			if err != nil {
				log.Println(err)
			}

			// Unmarshall responses
			if request.Type == "json" {
				err = json.Unmarshal(clientResp, request.GenericStructure)
			} else if request.Type == "proto" {
				err = proto.Unmarshal(clientResp, request.ProtoStructure)
			}
			if err != nil {
				log.Println(err)
			}
		}

		// Parse responses
		var err error
		if source.Name == "ETA" {
			if !included.ETA && len(PreviousSources) >= 1 {
				// Add old source data for non-requested updates
				source.Final = PreviousSources[0].Final
			} else if included.ETA {
				// Parse data for requested updates
				source.Final, _ = ParseETAObjects(source.Requests)
			}
		} else if source.Name == "RTD" {
			if !included.RTD && len(PreviousSources) >= 2 {
				source.Final = PreviousSources[1].Final
			} else if included.RTD {

				source.Final = ParseRTDObjects(source.Requests, source.Config)
			}
		} else if source.Name == "TransitTime" {
			if !included.TransitTime && len(PreviousSources) >= 2 {
				source.Final = PreviousSources[2].Final
			} else if included.TransitTime {
				source.Final = ParseTransitTimeObjects(source.Requests, source.Config)
			}
		}

		if err != nil {
			log.Println(err)
		}
	}

	PreviousSources = Sources
	// Combine FinalObjects
	return CreateFinalJSON(Sources)
}

/* Parse RTD retrieved objects into an instance of FinalObject
   TODO: This is the exact same as RTD. There's a better way to do this.
*/
func ParseTransitTimeObjects(requests []Request, conf Config) FinalObjects {
	Final := FinalObjects{
		Routes: []RouteInfo{},
		Stops:  []StopInfo{},
		Buses:  []BusInfo{},
	}
	trips := requests[0].ProtoStructure
	vehicles := requests[1].ProtoStructure

	// Iterate through every active vehicle for stops, routes
	for _, entity := range trips.GetEntity() {
		trip := entity.GetTripUpdate().GetTrip()
		times := entity.GetTripUpdate().GetStopTimeUpdate()
		routeName := trip.GetRouteId()

		// Only take routes found in the config
		if _, ok := conf.Buses[routeName]; ok {
			routeId := conf.Buses[routeName]

			currentRoutePtr := &RouteInfo{}

			// Check if route is already recorded
			for i, _ := range Final.Routes {
				if Final.Routes[i].ID == routeId {
					currentRoutePtr = &Final.Routes[i]
					break
				}
			}

			// Route not seen yet
			if currentRoutePtr.ID == 0 {
				newRoute := RouteInfo{
					ID:    routeId,
					Name:  routeName,
					Stops: []int{},
				}
				// Add new route and record stops to it
				Final.Routes = append(Final.Routes, newRoute)
				currentRoutePtr = &Final.Routes[len(Final.Routes)-1]
			}

			// For every stop in current route
			for _, stopTimeUpdate := range times {
				stopId, err := strconv.Atoi(stopTimeUpdate.GetStopId())
				if err != nil {
					log.Println(err)
				}

				// Find the index of this stop in our stop list
				i := sort.Search(len(rtd_stops),
					func(i int) bool { return rtd_stops[i].ID >= stopId })
				if i < len(rtd_stops) && rtd_stops[i].ID == stopId {
					currentStopPtr := &StopInfo{}
					// Check if stop is already recorded
					for j, _ := range Final.Stops {
						if Final.Stops[j].ID == stopId {
							currentStopPtr = &Final.Stops[j]
							break
						}
					}

					// Stop not seen yet
					if currentStopPtr.ID == 0 {
						newStop := StopInfo{
							ID:                rtd_stops[i].ID,
							Name:              rtd_stops[i].Name,
							Lat:               rtd_stops[i].Lat,
							Lng:               rtd_stops[i].Lng,
							NextBusTimesFinal: map[string][]int{},
						}
						// Add new stop and record active buses to it
						Final.Stops = append(Final.Stops, newStop)
						currentStopPtr = &Final.Stops[len(Final.Stops)-1]
					}

					arrivalTime := time.Unix(stopTimeUpdate.GetArrival().GetTime(), 0)
					timeUntil := arrivalTime.Sub(time.Now())
					// Ceiling time estimate for plausible deniability
					minutesUntil := int((timeUntil + time.Minute) / time.Minute)

					if minutesUntil >= 0 && minutesUntil <= 300 {
						routeStr := strconv.Itoa(routeId)
						// Prepend next time value
						currentStopPtr.NextBusTimesFinal[routeStr] =
							append([]int{minutesUntil}, currentStopPtr.NextBusTimesFinal[routeStr]...)
						// Ensure earliest times are presented first
						if !sort.IntsAreSorted(currentStopPtr.NextBusTimesFinal[routeStr]) {
							sort.Ints(currentStopPtr.NextBusTimesFinal[routeStr])
						}

					}
				} // Find stop in list

				currentRoutePtr.Stops = append(currentRoutePtr.Stops, stopId)
			}
		} // Take route if defined in config

		// Ensure stops in routes are sorted and unique
		for i, _ := range Final.Routes {
			Final.Routes[i].Stops = RemoveDuplicates(Final.Routes[i].Stops)
			sort.Ints(Final.Routes[i].Stops)
		}

	} // Iterate through trips feed

	// Iterate through vehicles feed
	for _, entity := range vehicles.GetEntity() {
		bus := entity.GetVehicle()
		routeName := bus.GetTrip().GetRouteId()

		// Only take routes found in the config
		if _, ok := conf.Buses[routeName]; ok {
			routeId := conf.Buses[routeName]
			newBus := BusInfo{
				RouteID: routeId,
				Lat:     float64(bus.GetPosition().GetLatitude()),
				Lng:     float64(bus.GetPosition().GetLongitude()),
			}
			Final.Buses = append(Final.Buses, newBus)
		}
	}

	// Sort stops once all are recorded
	sort.Sort(IDSorter(Final.Stops))

	return Final
}

/* Parse RTD retrieved objects into an instance of FinalObject */
func ParseRTDObjects(requests []Request, conf Config) FinalObjects {
	Final := FinalObjects{
		Routes: []RouteInfo{},
		Stops:  []StopInfo{},
		Buses:  []BusInfo{},
	}
	trips := requests[0].ProtoStructure
	vehicles := requests[1].ProtoStructure

	// Iterate through every active vehicle for stops, routes
	for _, entity := range trips.GetEntity() {
		trip := entity.GetTripUpdate().GetTrip()
		times := entity.GetTripUpdate().GetStopTimeUpdate()
		routeName := trip.GetRouteId()

		// Only take routes found in the config
		if _, ok := conf.Buses[routeName]; ok {
			routeId := conf.Buses[routeName]
			// Rewrite stampede routename
			if routeName == "STMP" {
				routeName = "Stampede"
			}
			currentRoutePtr := &RouteInfo{}

			// Check if route is already recorded
			for i, _ := range Final.Routes {
				if Final.Routes[i].ID == routeId {
					currentRoutePtr = &Final.Routes[i]
					break
				}
			}

			// Route not seen yet
			if currentRoutePtr.ID == 0 {
				newRoute := RouteInfo{
					ID:    routeId,
					Name:  routeName,
					Stops: []int{},
				}
				// Add new route and record stops to it
				Final.Routes = append(Final.Routes, newRoute)
				currentRoutePtr = &Final.Routes[len(Final.Routes)-1]
			}

			// For every stop in current route
			for _, stopTimeUpdate := range times {
				stopId, err := strconv.Atoi(stopTimeUpdate.GetStopId())
				if err != nil {
					log.Println(err)
				}

				// Find the index of this stop in our stop list
				i := sort.Search(len(rtd_stops),
					func(i int) bool { return rtd_stops[i].ID >= stopId })
				if i < len(rtd_stops) && rtd_stops[i].ID == stopId {
					currentStopPtr := &StopInfo{}
					// Check if stop is already recorded
					for j, _ := range Final.Stops {
						if Final.Stops[j].ID == stopId {
							currentStopPtr = &Final.Stops[j]
							break
						}
					}

					// Stop not seen yet
					if currentStopPtr.ID == 0 {
						newStop := StopInfo{
							ID:                rtd_stops[i].ID,
							Name:              rtd_stops[i].Name,
							Lat:               rtd_stops[i].Lat,
							Lng:               rtd_stops[i].Lng,
							NextBusTimesFinal: map[string][]int{},
						}
						// Add new stop and record active buses to it
						Final.Stops = append(Final.Stops, newStop)
						currentStopPtr = &Final.Stops[len(Final.Stops)-1]
					}

					arrivalTime := time.Unix(stopTimeUpdate.GetArrival().GetTime(), 0)
					timeUntil := arrivalTime.Sub(time.Now())
					// Ceiling time estimate for plausible deniability
					minutesUntil := int((timeUntil + time.Minute) / time.Minute)

					if minutesUntil >= 0 && minutesUntil <= 300 {
						routeStr := strconv.Itoa(routeId)
						// Prepend next time value
						currentStopPtr.NextBusTimesFinal[routeStr] =
							append([]int{minutesUntil}, currentStopPtr.NextBusTimesFinal[routeStr]...)
						// Ensure earliest times are presented first
						if !sort.IntsAreSorted(currentStopPtr.NextBusTimesFinal[routeStr]) {
							sort.Ints(currentStopPtr.NextBusTimesFinal[routeStr])
						}

					}
				} // Find stop in list

				currentRoutePtr.Stops = append(currentRoutePtr.Stops, stopId)
			}
		} // Take route if defined in config

		// Ensure stops in routes are sorted and unique
		for i, _ := range Final.Routes {
			Final.Routes[i].Stops = RemoveDuplicates(Final.Routes[i].Stops)
			sort.Ints(Final.Routes[i].Stops)
		}

	} // Iterate through trips feed

	// Iterate through vehicles feed
	for _, entity := range vehicles.GetEntity() {
		bus := entity.GetVehicle()
		routeName := bus.GetTrip().GetRouteId()

		// Only take routes found in the config
		if _, ok := conf.Buses[routeName]; ok {
			routeId := conf.Buses[routeName]
			// Rewrite stampede routename
			if routeName == "STMP" {
				routeName = "Stampede"
			}
			newBus := BusInfo{
				RouteID: routeId,
				Lat:     float64(bus.GetPosition().GetLatitude()),
				Lng:     float64(bus.GetPosition().GetLongitude()),
			}
			Final.Buses = append(Final.Buses, newBus)
		}
	}

	// Sort stops once all are recorded
	sort.Sort(IDSorter(Final.Stops))

	return Final
}

//TODO actually refactor this method
/* Parse ETA retrieves objects into an instance of FinalObject */
func ParseETAObjects(requests []Request) (FinalObjects, error) {
	var nextBusTimesStart []int

	Final := FinalObjects{
		Routes: []RouteInfo{},
		Stops:  []StopInfo{},
		Buses:  []BusInfo{},
	}

	ETARoutes := requests[0].GenericStructure.(*ETA_Routes)
	ETAStops := requests[1].GenericStructure.(*ETA_Stops)
	ETABuses := requests[2].GenericStructure.(*ETA_Buses)
	ETAAnnouncements := requests[3].GenericStructure.(*ETA_Announcements)

	var err error

	for _, route := range ETARoutes.GetRoutes {
		var stopToInt []int
		for _, stop := range route.Stops {
			stopToInt = append(stopToInt, stop)
		}
		if strings.EqualFold(route.Name, "Will Vill - Brown Line") {
			route.Name = "Buff Bus"
		}

		newRoute := RouteInfo{
			ID:    route.ID,
			Name:  route.Name,
			Stops: stopToInt,
		}
		Final.Routes = append(Final.Routes, newRoute)
	}

	//NEED to optomize this....  :(
	for _, stop := range ETAStops.GetStops {
		mapIt := map[string][]int{}
		for _, bus := range ETABuses.GetBuses {

			if len(bus.Minutestonextstops) != 0 {
				for _, minute := range bus.Minutestonextstops {
					str := strconv.Itoa(bus.Routeid)
					if minute.StopID == strconv.Itoa(stop.ID) && minute.Minutes >= 0 {
						if _, ok := mapIt[str]; ok {
							mapIt[str] = append(mapIt[str], minute.Minutes)
						} else {
							nextBusTimesStart = append(nextBusTimesStart, minute.Minutes)
							mapIt[str] = nextBusTimesStart
							nextBusTimesStart = nil
						}
					}
				}
			}
		}

		for k := range mapIt {
			sort.Ints(mapIt[k])
		}
		// Manually rewrite names
		if strings.EqualFold(stop.Name, "Discovery Learning Center") || strings.EqualFold(stop.Name, "Public Safety") {
			stop.Name = "Engineering Center"
		}
		if strings.EqualFold(stop.Name, "Euclid") {
			stop.Name = "UMC"
		}

		if !strings.EqualFold(stop.Name, "30th and Colorado E Bound") && !strings.EqualFold(stop.Name, "30th and Colorado WB") {
			newStop := StopInfo{
				ID:                stop.ID,
				Name:              stop.Name,
				Lat:               stop.Lat,
				Lng:               stop.Lng,
				NextBusTimesFinal: mapIt,
			}
			isDuplicate := 0
			for _, v := range Final.Stops {
				if v.ID == newStop.ID {
					isDuplicate = 1
				}
			}
			if isDuplicate == 0 {
				Final.Stops = append(Final.Stops, newStop)
			}
		}
	}

	// Parse buses
	for _, bus := range ETABuses.GetBuses {
		if bus.Routeid != 777 && bus.Inservice != 0 {
			newBus := BusInfo{
				RouteID: bus.Routeid,
				Lat:     bus.Lat,
				Lng:     bus.Lng,
			}
			Final.Buses = append(Final.Buses, newBus)
		}
	}

	// Parse announcements
	var announcementString []string
	for _, announcement := range ETAAnnouncements.GetServiceAnnouncements {
		for _, message := range announcement.Announcements {
			if message.Text != "" {
				announcementString = append(announcementString, message.Text)
			}
		}
	}

	newAnnouncement := AnnouncementInfo{
		Announcements: announcementString,
	}
	Final.Announcements = append(Final.Announcements, newAnnouncement)

	return Final, err
}

/* Creates the final json to be served by the server */
func CreateFinalJSON(Sources []Source) FinalJSONs {
	JSONs := FinalJSONs{}
	Final := FinalObjects{
		Routes:        []RouteInfo{},
		Stops:         []StopInfo{},
		Buses:         []BusInfo{},
		Announcements: []AnnouncementInfo{},
	}

	for _, source := range Sources {
		Final.Routes = append(Final.Routes, source.Final.Routes...)
		Final.Stops = append(Final.Stops, source.Final.Stops...)
		Final.Buses = append(Final.Buses, source.Final.Buses...)
		Final.Announcements = append(Final.Announcements,
			source.Final.Announcements...)
	}

	var err error
	JSONs.Stops, err = json.Marshal(Final.Stops)
	if err != nil {
		log.Println("Error Marshalling the JSON for the Audit", err)
	}

	JSONs.Routes, err = json.Marshal(Final.Routes)
	if err != nil {
		log.Println("Error Marshalling the JSON for the Audit", err)
	}

	JSONs.Buses, err = json.Marshal(Final.Buses)
	if err != nil {
		log.Println("Error Marshalling the JSON for the Audit", err)
	}

	JSONs.Announcements, err = json.Marshal(Final.Announcements)
	if err != nil {
		log.Println("Error marshalling the JSOM for the Audit", err)
	}

	return JSONs
}

/* Helper functions */
func LoadStopData() []StopInfo {
	data := []StopInfo{}
	csvParser := parser.CsvParser{
		CsvFile:         RTD_STOPS_FILE,
		CsvSeparator:    ',',
		SkipFirstLine:   true,
		SkipEmptyValues: true,
	}

	// Parse to general items array using specified struct
	parsedItems, err := csvParser.Parse(StopInfo{})
	if err != nil {
		log.Fatal("Error parsing file: ", err)
	}

	// Copy items to StopInfo array
	for _, item := range parsedItems {
		data = append(data, *item.(*StopInfo))
	}

	// Sort StopInfo based on IDs
	sort.Sort(IDSorter(data))

	return data
}

/* Not the most efficient solution, but the most straight-forward */
func RemoveDuplicates(elements []int) []int {
	seen := map[int]bool{}
	uniques := []int{}

	for i, _ := range elements {
		if seen[elements[i]] == true {
			// Do nothing
		} else {
			seen[elements[i]] = true
			uniques = append(uniques, elements[i])
		}
	}
	return uniques
}
