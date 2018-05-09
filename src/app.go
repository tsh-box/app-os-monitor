package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	databox "github.com/tsh2/lib-go-databox"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
)

//A singel reading from the observe endpoints
type reading struct {
	Data float64 `json:"data, int, float"`
}

//type to parse json load average data from store
type loadReadings []struct {
	TimestampMS int64 `json:"timestamp"`
	Data        struct {
		Data float64 `json:"data"`
	} `json:"data"`
}

//Load historical load average data
func loadStats(dataset *dataSet, dataSourceID string, tsc databox.JSONTimeSeries_0_2_0) {
	data, loadStatsLastNerr := tsc.LastN(dataSourceID, 500)
	if loadStatsLastNerr != nil {
		fmt.Println("Error getting last N ", dataSourceID, loadStatsLastNerr)
	}
	fmt.Println(data)
	readings := loadReadings{}
	json.Unmarshal(data, &readings)
	for i := len(readings) - 1; i >= 0; i-- {
		dataset.add(readings[i].Data.Data, readings[i].TimestampMS/1000)
	}
}

//A type to represent a dataset with locking and a max length
type dataSet struct {
	data      []float64
	timestamp []float64
	sync.RWMutex
	maxLength int
}

//add data to the data set
func (s *dataSet) add(dataPoint float64, timestamp int64) {
	s.Lock()
	s.data = append(s.data, dataPoint)
	s.timestamp = append(s.timestamp, float64(timestamp))
	if len(s.data) > s.maxLength {
		s.data = s.data[len(s.data)-s.maxLength:]
	}
	if len(s.timestamp) > s.maxLength {
		s.timestamp = s.timestamp[len(s.timestamp)-s.maxLength:]
	}
	s.Unlock()
}

//Global datasets for holding the data
var loadAverage1Stats dataSet
var loadAverage5Stats dataSet
var loadAverage15Stats dataSet
var memStats dataSet

//Get the data source information from the environment variables
var dataSourceLoadavg1, _, _ = databox.HypercatToDataSourceMetadata(os.Getenv("DATASOURCE_loadavg1"))
var dataSourceLoadavg5, _, _ = databox.HypercatToDataSourceMetadata(os.Getenv("DATASOURCE_loadavg5"))
var dataSourceLoadavg15, _, _ = databox.HypercatToDataSourceMetadata(os.Getenv("DATASOURCE_loadavg15"))
var dataSourceFreemem, DATABOX_ZMQ_ENDPOINT, _ = databox.HypercatToDataSourceMetadata(os.Getenv("DATASOURCE_freemem"))

//
// HTPPS end points
//
func getStatusEndpoint(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("active\n"))
}

func getLoadPlot(w http.ResponseWriter, req *http.Request) {

	loadAverage1Stats.RLock()
	loadAverage5Stats.RLock()
	loadAverage15Stats.RLock()
	defer loadAverage1Stats.RUnlock()
	defer loadAverage5Stats.RUnlock()
	defer loadAverage15Stats.RUnlock()

	xys1 := make(plotter.XYs, len(loadAverage1Stats.data))
	xys5 := make(plotter.XYs, len(loadAverage5Stats.data))
	xys15 := make(plotter.XYs, len(loadAverage15Stats.data))
	for i, d := range loadAverage1Stats.data {
		xys1[i].X = loadAverage1Stats.timestamp[i]
		xys1[i].Y = d
	}
	for i, d := range loadAverage5Stats.data {
		xys5[i].X = loadAverage5Stats.timestamp[i]
		xys5[i].Y = d
	}
	for i, d := range loadAverage15Stats.data {
		xys15[i].X = loadAverage15Stats.timestamp[i]
		xys15[i].Y = d
	}

	g := plotter.NewGrid()
	g.Horizontal.Color = color.RGBA{R: 220, B: 220, G: 220, A: 255}
	g.Vertical.Width = 0

	p, err := plot.New()
	if err != nil {
		fmt.Println(err)
		w.Write([]byte("Error\n"))
		return
	}

	l1, _ := plotter.NewLine(xys1)
	l1.Color = color.RGBA{R: 255, A: 255}
	l5, _ := plotter.NewLine(xys5)
	l5.Color = color.RGBA{B: 255, A: 255}
	l15, _ := plotter.NewLine(xys15)
	l15.Color = color.RGBA{G: 255, A: 255}

	p.Add(g, l1, l5, l15)
	p.Title.Text = "Load Average"
	p.Y.Label.Text = "Load"
	p.X.Label.Text = "Time"
	p.X.Tick.Marker = plot.TimeTicks{Format: "15:04:05"}
	p.Legend.Add("1 min", l1)
	p.Legend.Add("5 min", l5)
	p.Legend.Add("15 min", l15)

	wt, err := p.WriterTo(800, 400, "png")
	if err != nil {
		fmt.Println(err)
		w.Write([]byte("Error\n"))
		return
	}
	_, err = wt.WriteTo(w)
	if err != nil {
		fmt.Println(err)
		w.Write([]byte("Error\n"))
		return
	}
}

func getMemPlot(w http.ResponseWriter, req *http.Request) {

	memStats.RLock()
	defer memStats.RUnlock()

	xys := make(plotter.XYs, len(memStats.data))
	for i, d := range memStats.data {
		xys[i].X = memStats.timestamp[i]
		xys[i].Y = d / float64(1048576)
	}

	g := plotter.NewGrid()
	g.Horizontal.Color = color.RGBA{R: 220, B: 220, G: 220, A: 255}
	g.Vertical.Width = 0

	p, err := plot.New()
	if err != nil {
		fmt.Println(err)
		w.Write([]byte("Error\n"))
		return
	}

	lmem, _ := plotter.NewLine(xys)
	lmem.Color = color.RGBA{R: 255, A: 255}

	p.Add(g, lmem)
	p.Title.Text = "Free Mem"
	p.Y.Label.Text = "Mem (MB)"
	p.X.Label.Text = "Time"
	p.X.Tick.Marker = plot.TimeTicks{Format: "15:04:05"}
	p.Legend.Add("Mem", lmem)

	wt, err := p.WriterTo(800, 400, "png")
	if err != nil {
		fmt.Println(err)
		w.Write([]byte("Error\n"))
		return
	}
	_, err = wt.WriteTo(w)
	if err != nil {
		fmt.Println(err)
		w.Write([]byte("Error\n"))
		return
	}
}

func getUI(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s", `
		<h1>Databox stats</h1>
		<center>
		<div style="width:93%">
		<img src="/app-os-monitor/ui/mem.png?rand=0" style="float:left; padding:1%; width:45%; min-width:500px">
		<img src="/app-os-monitor/ui/load.png?rand=0" style="float:left; padding:1%; width:45%; min-width:500px">
		</div>
		</center>
		<script>
		setInterval(function() {
			var imgs = document.getElementsByTagName("IMG");
			for (var i=0; i < imgs.length; i++) {
				var eqPos = imgs[i].src.lastIndexOf("=");
				var src = imgs[i].src.substr(0, eqPos+1);
				imgs[i].src = src + Math.random();
			}
		}, 1000);
		</script>
	`)
}

func main() {

	loadAverage1Stats = dataSet{maxLength: 500}
	loadAverage5Stats = dataSet{maxLength: 500}
	loadAverage15Stats = dataSet{maxLength: 500}
	memStats = dataSet{maxLength: 500}

	fmt.Println(DATABOX_ZMQ_ENDPOINT)

	tsc, err := databox.NewJSONTimeSeriesClient(DATABOX_ZMQ_ENDPOINT, false)
	if err != nil {
		panic("Cant connect to store: " + err.Error())
	}

	//Load in the last seen 500 points
	type FreeMemArray []struct {
		TimestampMS int64 `json:"timestamp"`
		Data        struct {
			Data float64 `json:"data"`
		} `json:"data"`
	}
	res, lastNerr := tsc.LastN(dataSourceFreemem.DataSourceID, 500)
	if lastNerr != nil {
		fmt.Println("Error getting last N ", dataSourceFreemem.DataSourceID, lastNerr)
	}
	fma := FreeMemArray{}
	json.Unmarshal(res, &fma)
	for i := len(fma) - 1; i >= 0; i-- {
		memStats.add(fma[i].Data.Data, fma[i].TimestampMS/1000)
	}

	loadStats(&loadAverage1Stats, dataSourceLoadavg1.DataSourceID, tsc)
	loadStats(&loadAverage5Stats, dataSourceLoadavg5.DataSourceID, tsc)
	loadStats(&loadAverage15Stats, dataSourceLoadavg15.DataSourceID, tsc)

	//listen for new data
	load1Chan, obsErr := tsc.Observe(dataSourceLoadavg1.DataSourceID)
	if obsErr != nil {
		fmt.Println("Error Observing ", dataSourceLoadavg1.DataSourceID)
	}
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
					loadAverage1Stats.add(data.Data, time.Now().Unix())
				}
			case msg := <-_load5Chan:
				var data reading
				err := json.Unmarshal(msg, &data)
				if err != nil {
					fmt.Println("json.Unmarshal error ", err)
				} else {
					loadAverage5Stats.add(data.Data, time.Now().Unix())
				}
			case msg := <-_load15Chan:
				var data reading
				err := json.Unmarshal(msg, &data)
				if err != nil {
					fmt.Println("json.Unmarshal error ", err)
				} else {
					loadAverage15Stats.add(data.Data, time.Now().Unix())
				}
			case msg := <-_freememChan:
				var data reading
				err := json.Unmarshal(msg, &data)
				if err != nil {
					fmt.Println("json.Unmarshal error ", err)
				} else {
					memStats.add(data.Data, time.Now().Unix())
					jsonString, _ := json.Marshal(string(msg[:]))
					fmt.Println("LLA added in app-os-monitor app.go data String is ", jsonString)
					databox.ExportLongpoll("https://export.amar.io/", string(jsonString))
				}
			default:
				time.Sleep(time.Millisecond * 10)
			}
		}
	}(load1Chan, load5Chan, load15Chan, freememChan)

	router := mux.NewRouter()

	router.HandleFunc("/ui", getUI).Methods("GET")
	router.HandleFunc("/status", getStatusEndpoint).Methods("GET")
	router.HandleFunc("/ui/load.png", getLoadPlot).Methods("GET")
	router.HandleFunc("/ui/mem.png", getMemPlot).Methods("GET")

	log.Fatal(http.ListenAndServeTLS(":8080", databox.GetHttpsCredentials(), databox.GetHttpsCredentials(), router))
}
