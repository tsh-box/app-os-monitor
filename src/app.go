package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	databox "github.com/me-box/lib-go-databox"
)

var dataSourceLoadavg1, _ = databox.JsonUnmarshal(os.Getenv("DATASOURCE_loadavg1"))
var dataSourceLoadavg5, _ = databox.JsonUnmarshal(os.Getenv("DATASOURCE_loadavg5"))
var dataSourceLoadavg15, _ = databox.JsonUnmarshal(os.Getenv("DATASOURCE_loadavg15"))
var dataSourceFreemem, _ = databox.JsonUnmarshal(os.Getenv("DATASOURCE_freemem"))
var storeURL, _ = databox.GetStoreURLFromDsHref(dataSourceFreemem["href"].(string))

func getStatusEndpoint(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("active\n"))
}

var latestData = map[string]float64{"loadavg1": 0.0, "loadavg5": 0.0, "loadavg15": 0.0, "freemem": 0.0}

func getDataEndpoint(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	res, _ := json.Marshal(latestData)

	w.Write(res)
}

type reading struct {
	DataSourceID string  `json:"datasource_id, string"`
	Data         float64 `json:"data, int, float"`
	Timestamp    int64   `json:"timestamp, int"`
	ID           string  `json:"_id, string"`
}

func main() {

	fmt.Printf(storeURL + "\n")

	databox.WaitForStoreStatus(storeURL)

	dChan, conError := databox.WSConnect(dataSourceLoadavg1["href"].(string))
	if conError != nil {
		fmt.Println("WSConnect:: ", conError)
	}

	res, err := databox.WSSubscribe(dataSourceLoadavg1["href"].(string), "ts")
	fmt.Println("WSSubscribe dataSourceLoadavg1", res, err)
	databox.WSSubscribe(dataSourceLoadavg5["href"].(string), "ts")
	databox.WSSubscribe(dataSourceLoadavg15["href"].(string), "ts")
	databox.WSSubscribe(dataSourceFreemem["href"].(string), "ts")

	go func() {
		for {
			msg := <-dChan
			fmt.Println("DATA:: ", string(msg[:]))
			var data reading
			err := json.Unmarshal(msg, &data)
			if err != nil {
				fmt.Println("json.Unmarshal error ", err)
			} else {
				latestData[data.DataSourceID] = data.Data
				if data.DataSourceID == "freemem" {
					jsonString, _ := json.Marshal(string(msg[:]))
					res, expError := databox.ExportLongpoll("https://export.amar.io/", string(jsonString))
					fmt.Println("Export result::", res, expError)
				}
			}
		}
	}()

	router := mux.NewRouter()

	ui := http.StripPrefix("/ui", http.FileServer(http.Dir("./www/")))
	router.HandleFunc("/ui/data", getDataEndpoint).Methods("GET")
	router.HandleFunc("/status", getStatusEndpoint).Methods("GET")
	router.PathPrefix("/ui").Handler(ui)

	log.Fatal(http.ListenAndServeTLS(":8080", databox.GetHttpsCredentials(), databox.GetHttpsCredentials(), router))
}
