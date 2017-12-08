package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	databox "github.com/toshbrown/lib-go-databox"
)

type reading struct {
	DataSourceID string  `json:"datasource_id, string"`
	Data         float64 `json:"data, int, float"`
	Timestamp    int64   `json:"timestamp, int"`
	ID           string  `json:"_id, string"`
}

var dataSourceLoadavg1, _, _ = databox.HypercatToDataSourceMetadata(os.Getenv("DATASOURCE_loadavg1"))
var dataSourceLoadavg5, _, _ = databox.HypercatToDataSourceMetadata(os.Getenv("DATASOURCE_loadavg5"))
var dataSourceLoadavg15, _, _ = databox.HypercatToDataSourceMetadata(os.Getenv("DATASOURCE_loadavg15"))
var dataSourceFreemem, DATABOX_ZMQ_ENDPOINT, _ = databox.HypercatToDataSourceMetadata(os.Getenv("DATASOURCE_freemem"))

func getStatusEndpoint(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("active\n"))
}

var latestData = map[string]float64{"loadavg1": 0.0, "loadavg5": 0.0, "loadavg15": 0.0, "freemem": 0.0}

func getDataEndpoint(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	res, _ := json.Marshal(latestData)

	w.Write(res)
}

func main() {

	fmt.Println(DATABOX_ZMQ_ENDPOINT)

	tsc, err := databox.NewJSONTimeSeriesClient(DATABOX_ZMQ_ENDPOINT, false)
	if err != nil {
		panic("Cant connect to store: " + err.Error())
	}

	load1Chan, _ := tsc.Observe(dataSourceLoadavg1.DataSourceID)
	load5Chan, _ := tsc.Observe(dataSourceLoadavg5.DataSourceID)
	load15Chan, _ := tsc.Observe(dataSourceLoadavg15.DataSourceID)
	freememChan, _ := tsc.Observe(dataSourceFreemem.DataSourceID)

	go func(_load1Chan, _load5Chan, _load15Chan, _freememChan <-chan []byte) {
		for {
			select {
			case msg := <-_load1Chan:
				var data reading
				err := json.Unmarshal(msg, &data)
				if err != nil {
					fmt.Println("json.Unmarshal error ", err)
				} else {
					latestData[dataSourceLoadavg1.DataSourceID] = data.Data
				}
			case msg := <-_load5Chan:
				var data reading
				err := json.Unmarshal(msg, &data)
				if err != nil {
					fmt.Println("json.Unmarshal error ", err)
				} else {
					latestData[dataSourceLoadavg5.DataSourceID] = data.Data
				}
			case msg := <-_load15Chan:
				var data reading
				err := json.Unmarshal(msg, &data)
				if err != nil {
					fmt.Println("json.Unmarshal error ", err)
				} else {
					latestData[dataSourceLoadavg15.DataSourceID] = data.Data
				}
			case msg := <-_freememChan:
				var data reading
				err := json.Unmarshal(msg, &data)
				if err != nil {
					fmt.Println("json.Unmarshal error ", err)
				} else {
					latestData[dataSourceFreemem.DataSourceID] = data.Data
					jsonString, _ := json.Marshal(string(msg[:]))
					res, expError := databox.ExportLongpoll("https://export.amar.io/", string(jsonString))
					fmt.Println("Export result::", res, expError)
				}
			default:
				time.Sleep(time.Millisecond * 10)
			}
		}
	}(load1Chan, load5Chan, load15Chan, freememChan)

	router := mux.NewRouter()

	ui := http.StripPrefix("/ui", http.FileServer(http.Dir("./www/")))
	router.HandleFunc("/ui/data", getDataEndpoint).Methods("GET")
	router.HandleFunc("/status", getStatusEndpoint).Methods("GET")
	router.PathPrefix("/ui").Handler(ui)

	log.Fatal(http.ListenAndServeTLS(":8080", databox.GetHttpsCredentials(), databox.GetHttpsCredentials(), router))
}
