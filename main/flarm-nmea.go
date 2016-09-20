/*
	Copyright (c) 2016 Keith Tschohl
	Distributable under the terms of The "BSD New"" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	flarm-nmea.go: Functions for generating FLARM-related NMEA sentences
		to communicate traffic bearing / distance to glider computers
		and UK / EU oriented EFBs.
*/

package main

import (
	"fmt"
	"log"
	"math"
	"time"
)

/*
	sendNetFLARM() is a shortcut to network.go 'sendMsg()', and will send the referenced byte slice to the network port
		defined by NETWORK_FLARM_NMEA in gen_gdl90.go as a non-queueable message.
*/

func sendNetFLARM(msg []byte) {
	sendMsg(msg, NETWORK_FLARM_NMEA, false) // UDP output. Traffic messages are always non-queuable.
	// TO-DO: add call to TCP server for SkyDemon and RunwayHD
}

/*
	makeFlarmPFLAAString() creates a NMEA-formatted PFLAA string (FLARM traffic format) with checksum from the referenced
		traffic object.
*/

func makeFlarmPFLAAString(ti TrafficInfo) (msg string, valid bool) {

	/*	Format: $PFLAA,<AlarmLevel>,<RelativeNorth>,<RelativeEast>,<RelativeVertical>,<IDType>,<ID>,<Track>,<TurnRate>,<GroundSpeed>, <ClimbRate>,<AcftType>*<checksum>
		            $PFLAA,0,-10687,-22561,-10283,1,A4F2EE,136,0,269,0.0,0*4E
			<AlarmLevel>  Decimal integer value. Range: from 0 to 3.
							Alarm level as assessed by FLARM:
							0 = no alarm (also used for no-alarm traffic information)
							1 = alarm, 13-18 seconds to impact
							2 = alarm, 9-12 seconds to impact
							3 = alarm, 0-8 seconds to impact
			<RelativeNorth>,<RelativeEast>,<RelativeVertical> are distances in meters. Decimal integer value. Range: from -32768 to 32767.
			<IDType>: 1 = official ICAO 24-bit aircraft address; 2 = stable FLARM ID (chosen by FLARM) 3 = anonymous ID, used if stealth mode is activated.
			For ADS-B traffic, we'll always pick 1.
			<ID>: 6-digit hexadecimal value (e.g. “5A77B1”) as configured in the target’s PFLAC,,ID sentence. For ADS-B targets always use reported 24-bit ICAO address.
					NOTE: Appending "!CALLSIGN" may allow certain
			<Track>: Decimal integer value. Range: from 0 to 359. The target’s true ground track in degrees.
			<TurnRate>: Not used. Empty field.
			<GroundSpeed>: Decimal integer value. Range: from 0 to 32767. The target’s ground speed in m/s
			<ClimbRate>: Decimal fixed point number with one digit after the radix point (dot). Range: from -32.7 to 32.7. The target’s climb rate in m/s.
			Positive values indicate a climbing aircraft.
			<AcftType>: Hexadecimal value. Range: from 0 to F.
							Aircraft types:
							0 = unknown
							1 = glider / motor glider
							2 = tow / tug plane
							3 = helicopter / rotorcraft
							4 = skydiver
							5 = drop plane for skydivers
							6 = hang glider (hard)
							7 = paraglider (soft)
							8 = aircraft with reciprocating engine(s)
							9 = aircraft with jet/turboprop engine(s)
							A = unknown
							B = balloon
							C = airship
							D = unmanned aerial vehicle (UAV)
							E = unknown
							F = static object
	*/

	var alarmLevel, idType, checksum uint8
	var relativeNorth, relativeEast, relativeVertical, groundSpeed int16
	var climbRate float32

	idType = 1

	// determine distance and bearing to target
	dist, bearing, distN, distE := distRect(float64(mySituation.Lat), float64(mySituation.Lng), float64(ti.Lat), float64(ti.Lng))
	if globalSettings.DEBUG {
		log.Printf("FLARM - ICAO target %X (%s) is %.1f meters away at %.1f degrees\n", ti.Icao_addr, ti.Tail, dist, bearing)
	}

	// TODO: Estimate distance for bearingless / distanceless Mode S (1090) aircraft targets

	if distN > 32767 || distN < -32767 || distE > 32767 || distE < -32767 {
		msg = ""
		valid = false
		return
	} else {
		relativeNorth = int16(distN)
		relativeEast = int16(distE)
	}

	altf := mySituation.Pressure_alt
	if !isTempPressValid() { // if no pressure altitude available, use GPS altitude
		altf = float64(mySituation.Alt)
	}
	relativeVertical = int16(float64(ti.Alt)*0.3048 - altf*0.3048) // convert to meters

	// demo of alarm levels... may remove for final release.
	if (dist < 926) && (relativeVertical < 152) && (relativeVertical > -152) { // 926 m = 0.5 NM; 152m = 500'
		alarmLevel = 2
	} else if (dist < 1852) && (relativeVertical < 304) && (relativeVertical > -304) { // 1852 m = 1.0 NM ; 304 m = 1000'
		alarmLevel = 1
	}

	if ti.Speed_valid {
		groundSpeed = int16(float32(ti.Speed) * 0.5144) // convert to m/s
	}

	climbRate = float32(ti.Vvel) * 0.3048 / 60 // convert to m/s
	msg = fmt.Sprintf("PFLAA,%d,%d,%d,%d,%d,%X!%s,%d,0,%d,%0.1f,0", alarmLevel, relativeNorth, relativeEast, relativeVertical, idType, ti.Icao_addr, ti.Tail, ti.Track, groundSpeed, climbRate)
	for i := range msg {
		checksum = checksum ^ byte(msg[i])
	}
	msg = (fmt.Sprintf("$%s*%X\r\n", msg, checksum))
	valid = true
	return
}

/*
	makeGPRMCString() creates a NMEA-formatted GPRMC string (GPS recommended minimum data) with checksum from the current GPS position.
		If current position is invalid, the GPRMC string will indicate no-fix.

*/

func makeGPRMCString() string {
	/*
				 RMC          Recommended Minimum sentence C
			     123519       Fix taken at 12:35:19 UTC
			     A            Status A=active or V=Void.
			     4807.038,N   Latitude 48 deg 07.038' N
			     01131.000,E  Longitude 11 deg 31.000' E
			     022.4        Speed over the ground in knots
			     084.4        Track angle in degrees True
			     230394       Date - 23rd of March 1994
			     003.1,W      Magnetic Variation
			     D				mode field (nmea 2.3 and higher)
			     *6A          The checksum data, always begins with *
		LastFixSinceMidnightUTC uint32
		Lat                     float32
		Lng                     float32
		Quality                 uint8
		GeoidSep                float32 // geoid separation, ft, MSL minus HAE (used in altitude calculation)
		Satellites              uint16  // satellites used in solution
		SatellitesTracked       uint16  // satellites tracked (almanac data received)
		SatellitesSeen          uint16  // satellites seen (signal received)
		Accuracy                float32 // 95% confidence for horizontal position, meters.
		NACp                    uint8   // NACp categories are defined in AC 20-165A
		Alt                     float32 // Feet MSL
		AccuracyVert            float32 // 95% confidence for vertical position, meters
		GPSVertVel              float32 // GPS vertical velocity, feet per second
		LastFixLocalTime        time.Time
		TrueCourse              uint16
		GroundSpeed             uint16
		LastGroundTrackTime     time.Time
	*/

	lastFix := float64(mySituation.LastFixSinceMidnightUTC)
	hr := math.Floor(lastFix / 3600)
	lastFix -= 3600 * hr
	mins := math.Floor(lastFix / 60)
	sec := lastFix - mins*60

	status := "V"
	if isGPSValid() && mySituation.Quality > 0 {
		status = "A"
	}

	lat := float64(mySituation.Lat)
	ns := "N"
	if lat < 0 {
		lat = -lat
		ns = "S"
	}

	deg := math.Floor(lat)
	min := (lat - deg) * 60
	lat = deg*100 + min

	ew := "E"
	lng := float64(mySituation.Lng)
	if lng < 0 {
		lng = -lng
		ew = "W"
	}

	deg = math.Floor(lng)
	min = (lng - deg) * 60
	lng = deg*100 + min

	gs := float32(mySituation.GroundSpeed)
	trueCourse := float32(mySituation.TrueCourse)
	yy, mm, dd := time.Now().UTC().Date()
	yy = yy % 100
	var magVar, mvEW string
	mode := "N"
	if mySituation.Quality == 1 {
		mode = "A"
	} else if mySituation.Quality == 2 {
		mode = "D"
	}

	var msg string

	if isGPSValid() {
		msg = fmt.Sprintf("GPRMC,%02.f%02.f%05.2f,%s,%010.5f,%s,%011.5f,%s,%.1f,%.1f,%02d%02d%02d,%s,%s,%s", hr, mins, sec, status, lat, ns, lng, ew, gs, trueCourse, dd, mm, yy, magVar, mvEW, mode)
	} else {
		msg = fmt.Sprintf("GPRMC,,%s,,,,,,,%02d%02d%02d,%s,%s,%s", status, dd, mm, yy, magVar, mvEW, mode) // return null lat-lng and velocity if invalid GPS
	}

	var checksum byte
	for i := range msg {
		checksum = checksum ^ byte(msg[i])
	}
	msg = fmt.Sprintf("$%s*%X\r\n", msg, checksum)
	return msg
}
