package main

import (
	"github.com/stratoberry/go-gpsd"
	"log"
	"sync"
)

func processTPV(r interface{}) {
	tpv := r.(*gpsd.TPVReport)
	log.Printf("TPV", tpv.Mode, tpv.Time)

	mySituation.mu_GPS.Lock()

	defer func() {
		if globalSettings.DEBUG {
			logSituation()
		}
		mySituation.mu_GPS.Unlock()
	}()

	mySituation.Lat = float32(tpv.Lat)
	mySituation.Lng = float32(tpv.Lon)
	mySituation.Accuracy = float32((tpv.Epx + tpv.Epy) / 2)
	mySituation.Alt = float32(tpv.Alt)
	mySituation.AccuracyVert = float32(tpv.Epv)
	mySituation.GPSVertVel = float32(tpv.Climb)
	mySituation.LastFixLocalTime = tpv.Time
	mySituation.TrueCourse = float32(tpv.Track)
	mySituation.GroundSpeed = uint16(tpv.Speed)
	mySituation.LastGroundTrackTime = tpv.Time
}

func initGpsd() {
	log.Printf("Initializing GPS\n")

	mySituation.mu_GPS = &sync.Mutex{}
	mySituation.mu_Attitude = &sync.Mutex{}
	satelliteMutex = &sync.Mutex{}
	Satellites = make(map[string]SatelliteInfo)

	var gps *gpsd.Session
	var err error

	if gps, err = gpsd.Dial(gpsd.DefaultAddress); err != nil {
		log.Printf("Failed to connect to gpsd: %s", err)
	}

	gps.AddFilter("TPV", processTPV)

	gps.Watch()
}
