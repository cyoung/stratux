package main

import (
	"fmt"
	"github.com/stratoberry/go-gpsd"
	"log"
	"math"
	"sync"
)

// Determine type of satellite based on PRN number.
func satelliteType(prnId int) uint8 {
	// TODO: Galileo
	switch {
	case 0 < prnId && prnId <= 32:
		return SAT_TYPE_GPS
	case 32 < prnId && prnId <= 64:
		// This is actually the NMEA id range for SBAS: WAAS, EGNOS, MSAS, etc.
		return SAT_TYPE_SBAS
	case 64 < prnId && prnId <= 96:
		return SAT_TYPE_GLONASS
	case 120 <= prnId && prnId <= 138:
		return SAT_TYPE_SBAS
	default:
		return SAT_TYPE_UNKNOWN
	}
}

func satelliteTypeCode(satType uint8) string {
	switch satType {
	case SAT_TYPE_GPS:
		return "G"
	case SAT_TYPE_SBAS:
		return "S"
	case SAT_TYPE_GLONASS:
		return "R"
	case SAT_TYPE_UNKNOWN:
		return "U"
	default:
		return "U"
	}
}

// Caller must protect Satellites with satelliteMutex.
func isSbasInSolution() bool {
	for _, satellite := range Satellites {
		if satellite.Type == SAT_TYPE_SBAS && satellite.InSolution {
			return true
		}
	}
	return false
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
	satelliteMutex.Lock()

	defer func() {
		if globalSettings.DEBUG {
			logSituation()
		}
		mySituation.mu_GPS.Unlock()
		satelliteMutex.Unlock()
	}()

	// 0 = No gps data, 1 = No fix, 2 = 2D fix, 3 = 3D fix
	// Without a way to report 2D coordinates to the client without
	// reporting bad altitude data, discard 2D fixes.
	switch tpv.Mode {
	case 0, 1, 2:
		mySituation.Quality = 0
		return
	case 3: // 3D gps
		// accept fix
	default:
		log.Printf("Unknown gpsd TPV mode %i received, ignoring TPV.", tpv.Mode)
		return
	}

	if isSbasInSolution() {
		mySituation.Quality = 2
	} else {
		mySituation.Quality = 1
	}

	mySituation.Alt = float32(tpv.Alt) * 3.28084 // meters to feet
	mySituation.AccuracyVert = float32(tpv.Epv)
	mySituation.GPSVertVel = float32(tpv.Climb)
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
		thisSatellite.Type = satelliteType(int(satellite.PRN))
		thisSatellite.SatelliteID = fmt.Sprint(satelliteTypeCode(thisSatellite.Type), int(satellite.PRN))
		thisSatellite.Prn = uint8(satellite.PRN)
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
