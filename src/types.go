package main

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

//FreeMemArray a type to parse json load freemem data from store
type freeMemArray []struct {
	TimestampMS int64 `json:"timestamp"`
	Data        struct {
		Data float64 `json:"data"`
	} `json:"data"`
}
