package main

import (
	"fmt"
	"github.com/stratoberry/go-gpsd"
	"log"
	"math"
	"sync"
)

func getNMEAName(sv int) string {
	// GPS NMEA = PRN. GLONASS NMEA = PRN + 65. SBAS is PRN; needs to be converted to NMEA for compatiblity with GSV messages.
	if sv < 33 { // indicates GPS
		return fmt.Sprintf("G%d", sv)
	} else if sv < 65 { // indicates SBAS: WAAS, EGNOS, MSAS, etc.
		return fmt.Sprintf("S%d", sv+87) // add 87 to convert from NMEA to PRN.
	} else if sv < 97 { // GLONASS
		return fmt.Sprintf("R%d", sv-64) // subtract 64 to convert from NMEA to PRN.
	} else if (sv >= 120) && (sv < 162) { // indicates SBAS: WAAS, EGNOS, MSAS, etc.
		return fmt.Sprintf("S%d", sv)
	} else { // TO-DO: Galileo
		return fmt.Sprintf("U%d", sv)
	}
}

func processDEVICES(r interface{}) {
	devices := r.(*gpsd.DEVICESReport)
	log.Printf("DEVICES (%d)", len(devices.Devices))
	for _, dev := range devices.Devices {
		log.Printf("  %s %s %x %s %s %i %s %s %i %s %s %i %d %d",
			dev.Path,
			dev.Activated,
			dev.Flags,
			dev.Driver,
			dev.Subtype,
			dev.Bps,
			dev.Parity,
			dev.Stopbits,
			dev.Native,
			dev.Cycle,
			dev.Mincycle)
	}

	if len(devices.Devices) > 0 {
		globalStatus.GPS_connected = true
	} else {
		globalStatus.GPS_connected = false
	}
}

func processTPV(r interface{}) {
	tpv := r.(*gpsd.TPVReport)
	log.Printf("TPV", tpv.Device, tpv.Mode, tpv.Time, tpv.Tag)

	mySituation.mu_GPS.Lock()

	defer func() {
		if globalSettings.DEBUG {
			logSituation()
		}
		mySituation.mu_GPS.Unlock()
	}()

	switch tpv.Mode {
	case 0:
		mySituation.Quality = 0
		return
	case 1:
		mySituation.Quality = 0
		return
	case 2: // 2D gps
		mySituation.Quality = 1
	case 3: // 3D gps
		mySituation.Quality = 1
		mySituation.Alt = float32(tpv.Alt) * 3.28084 // meters to feet
		mySituation.AccuracyVert = float32(tpv.Epv)
		mySituation.GPSVertVel = float32(tpv.Climb)
	}

	mySituation.Lat = float32(tpv.Lat)
	mySituation.Lng = float32(tpv.Lon)
	mySituation.Accuracy = float32(math.Sqrt(tpv.Epx*tpv.Epx + tpv.Epy*tpv.Epy))
	mySituation.LastFixLocalTime = stratuxClock.Time
	mySituation.TrueCourse = float32(tpv.Track)
	mySituation.GroundSpeed = uint16(tpv.Speed)
	mySituation.LastGroundTrackTime = tpv.Time
	mySituation.LastValidNMEAMessageTime = stratuxClock.Time

	globalStatus.GPS_connected = true
}

func processSKY(r interface{}) {
	sky := r.(*gpsd.SKYReport)
	log.Printf("SKY", sky.Device, sky.Tag)

	mySituation.mu_GPS.Lock()
	satelliteMutex.Lock()

	defer func() {
		satelliteMutex.Unlock()
		mySituation.mu_GPS.Unlock()
	}()

	var inSolution uint16 = 0

	for _, satellite := range sky.Satellites {
		var thisSatellite SatelliteInfo
		thisSatellite.SatelliteID = getNMEAName(int(satellite.PRN)) // fmt.Sprintf("%v", satellite.PRN)
		thisSatellite.Azimuth = int16(satellite.Az)
		thisSatellite.Elevation = int16(satellite.El)
		thisSatellite.Signal = int8(satellite.Ss)
		thisSatellite.InSolution = satellite.Used
		thisSatellite.TimeLastTracked = stratuxClock.Time
		thisSatellite.TimeLastSeen = stratuxClock.Time

		if thisSatellite.InSolution {
			thisSatellite.TimeLastSolution = stratuxClock.Time
		}

		Satellites[thisSatellite.SatelliteID] = thisSatellite

		if satellite.Used {
			inSolution++
		}
	}

	globalStatus.GPS_connected = true
	mySituation.LastValidNMEAMessageTime = stratuxClock.Time
	mySituation.Satellites = inSolution
	updateConstellation()
}

func processATT(r interface{}) {
	att := r.(*gpsd.ATTReport)
	log.Printf("ATT", att.Device, att.Tag, att.Pitch, att.Roll, att.Heading)

	mySituation.mu_GPS.Lock()

	defer func() {
		if globalSettings.DEBUG {
			logSituation()
		}
		mySituation.mu_GPS.Unlock()
	}()

	mySituation.Pitch = att.Pitch
	mySituation.Roll = att.Roll
	mySituation.Gyro_heading = att.Heading
	mySituation.LastAttitudeTime = stratuxClock.Time
}

func initGpsd() {
	log.Printf("Initializing gpsd\n")

	mySituation.mu_GPS = &sync.Mutex{}
	mySituation.mu_Attitude = &sync.Mutex{}
	satelliteMutex = &sync.Mutex{}
	Satellites = make(map[string]SatelliteInfo)

	var gps *gpsd.Session
	var err error

	if gps, err = gpsd.Dial(gpsd.DefaultAddress); err != nil {
		log.Printf("Failed to connect to gpsd: %s", err)
	}

	gps.AddFilter("DEVICES", processDEVICES)
	gps.AddFilter("TPV", processTPV)
	gps.AddFilter("SKY", processSKY)
	gps.AddFilter("ATT", processATT)

	gps.SendCommand("DEVICES")
	gps.Watch()
}
