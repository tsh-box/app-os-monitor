package main

import "sync"

//A type to represent a dataset with locking and a max length
type DataSet struct {
	data      []float64
	timestamp []float64
	sync.RWMutex
	maxLength int
}

func NewDataSet() DataSet {
	return DataSet{}
}

//add data to the data set
func (s *DataSet) Add(dataPoint float64, timestamp int64) {
	s.Lock()
	ts := timestamp / 1000
	s.data = append(s.data, dataPoint)
	s.timestamp = append(s.timestamp, float64(ts))
	if len(s.data) > s.maxLength {
		s.data = s.data[len(s.data)-s.maxLength:]
	}
	if len(s.timestamp) > s.maxLength {
		s.timestamp = s.timestamp[len(s.timestamp)-s.maxLength:]
	}
	s.Unlock()
}
