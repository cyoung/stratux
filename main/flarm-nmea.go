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
	//"bufio"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"time"
)

/*
	sendNetFLARM() is a shortcut to network.go 'sendMsg()', and will send the referenced byte slice to the UDP network port
		defined by NETWORK_FLARM_NMEA in gen_gdl90.go as a non-queueable message to be used in XCSoar. It will also queue
		the message into a channel so it can be	sent out to a TCP server.
*/

func sendNetFLARM(msg string) {
	sendMsg([]byte(msg), NETWORK_FLARM_NMEA, false) // UDP (and possibly future serial) output. Traffic messages are always non-queuable.
	msgchan <- msg // TCP output.

}

func makeFlarmPFLAUString(ti TrafficInfo) (msg string) {
	// syntax: PFLAU,<RX>,<TX>,<GPS>,<Power>,<AlarmLevel>,<RelativeBearing>,<AlarmType>,<RelativeVertical>,<RelativeDistance>,<ID>
	gpsStatus := 0
	if isGPSValid() {
		gpsStatus = 2
	}

	dist, bearing, _, _ := distRect(float64(mySituation.GPSLatitude), float64(mySituation.GPSLongitude), float64(ti.Lat), float64(ti.Lng))
	relativeVertical := computeRelativeVertical(ti)
	alarmLevel := computeAlarmLevel(dist, relativeVertical)
	if bearing > 180 {
		bearing = -(360 - bearing)
	}
	alarmType := 0
	if alarmLevel > 0 {
		alarmType = 2
	}
	// TODO: we are always airbourne for now
	if alarmLevel > 0 {
		msg = fmt.Sprintf("PFLAU,%d,1,%d,1,%d,%d,%d,%d,%d", len(traffic), gpsStatus, alarmLevel, int32(bearing), alarmType, relativeVertical, int32(math.Abs(dist)))
	} else {
		msg = fmt.Sprintf("PFLAU,%d,1,%d,1,0,,0,,", len(traffic), gpsStatus)
	}

	checksumPFLAU := byte(0x00)
	for i := range msg {
		checksumPFLAU = checksumPFLAU ^ byte(msg[i])
	}
	msg = (fmt.Sprintf("$%s*%02X\r\n", msg, checksumPFLAU))
	return
}

// TODO: only very simplistic implementation
func computeAlarmLevel(dist float64, relativeVertical int32) (alarmLevel uint8) {
	if (dist < 926) && (relativeVertical < 152) && (relativeVertical > -152) { // 926 m = 0.5 NM; 152m = 500'
		alarmLevel = 3
	} else if (dist < 1852) && (relativeVertical < 304) && (relativeVertical > -304) { // 1852 m = 1.0 NM ; 304 m = 1000'
		alarmLevel = 2
	} else {
		alarmLevel = 0
	}
	return
}

func computeRelativeVertical(ti TrafficInfo) (relativeVertical int32) {
	altf := mySituation.BaroPressureAltitude
	if !isTempPressValid() && isGPSValid() { // if no pressure altitude available, use GPS altitude
		altf = mySituation.GPSAltitudeMSL
	}
	if ti.AltIsGNSS && isGPSValid() {
		// Altitude coming from OGN. We set the geoid separation to 0 in the OGN config, so OGN reports ellipsoid alt - we need to compare to that
		altf = mySituation.GPSHeightAboveEllipsoid
	}
	relativeVertical = int32(float32(ti.Alt)*0.3048 - altf*0.3048) // convert to meters
	return
}

/*
	makeFlarmPFLAAString() creates a NMEA-formatted PFLAA string (FLARM traffic format) with checksum from the referenced
		traffic object.
*/

func makeFlarmPFLAAString(ti TrafficInfo) (msg string, valid bool, alarmLevel uint8) {

	/*	Format: $PFLAA,<AlarmLevel>,<RelativeNorth>,<RelativeEast>,<RelativeVertical>,<IDType>,<ID>,<Track>,<TurnRate>,<GroundSpeed>, <ClimbRate>,<AcftType>*<checksum>
		            $PFLAA,0,-10687,-22561,-10283,1,A4F2EE,136,0,269,0.0,0*4E
			<AlarmLevel>  Decimal integer value. Range: from 0 to 3.
							Alarm level as assessed by FLARM:
							0 = no alarm (also used for no-alarm traffic information)
							1 = alarm, 13-18 seconds to impact
							2 = alarm, 9-12 seconds to impact
							3 = alarm, 0-8 seconds to impact
			<RelativeNorth>,<RelativeEast>,<RelativeVertical> are distances in meters. Decimal integer value. Range: from -32768 to 32767.
				For traffic without known bearing, assign estimated distance to <RelativeNorth> and leave <RelativeEast> empty
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

	var idType, checksum uint8
	var relativeNorth, relativeEast, relativeVertical, groundSpeed int32
	var msg2 string

	idType = 1

	// determine distance and bearing to target
	dist, bearing, distN, distE := distRect(float64(mySituation.GPSLatitude), float64(mySituation.GPSLongitude), float64(ti.Lat), float64(ti.Lng))
	if !ti.Position_valid {
		dist = ti.DistanceEstimated
		distN = ti.DistanceEstimated
	}
	if globalSettings.DEBUG {
		log.Printf("FLARM - ICAO target %X (%s) is %.1f meters away at %.1f degrees\n", ti.Icao_addr, ti.Tail, dist, bearing)
	}

	// TODO: Estimate distance for bearingless / distanceless Mode S (1090) aircraft targets

	//if distN > 200000 || distN < -200000 || distE > 200000 || distE < -200000 {
	//	msg = ""
	//	valid = false
	//	return
	//} else {
	relativeNorth = int32(distN)
	relativeEast = int32(distE)
	//}

	relativeVertical = computeRelativeVertical(ti)
	alarmLevel = computeAlarmLevel(dist, relativeVertical)

	if ti.Speed_valid {
		groundSpeed = int32(float32(ti.Speed) * 0.5144) // convert to m/s
	}

	acType := 0
	switch ti.Emitter_category {
	case 9:
		acType = 1 // glider
	case 7:
		acType = 3 // rotorcraft
	case 1:
		acType = 8 // assume all light aircraft are piston
	case 2, 3, 4, 5, 6:
		acType = 9 // assume all heavier aircraft are jets
	default:
		acType = 0
	}

	climbRate := float32(ti.Vvel) * 0.3048 / 60 // convert to m/s
	if ti.Position_valid {
		msg = fmt.Sprintf("PFLAA,%d,%d,%d,%d,%d,%X!%s,%d,,%d,%0.1f,%d", alarmLevel, relativeNorth, relativeEast, relativeVertical, idType, ti.Icao_addr, ti.Tail, ti.Track, groundSpeed, climbRate, acType)
	} else {
		msg = fmt.Sprintf("PFLAA,%d,%d,,%d,%d,%X!%s,,,,%0.1f,%d", alarmLevel, int32(math.Abs(dist)), relativeVertical, idType, ti.Icao_addr, ti.Tail, climbRate, acType) // prototype for bearingless traffic
	}
	//msg = fmt.Sprintf("PFLAA,%d,%d,%d,%d,%d,%X!%s,%d,,%d,%0.1f,%d", alarmLevel, relativeNorth, relativeEast, relativeVertical, idType, ti.Icao_addr, ti.Tail, ti.Track, groundSpeed, climbRate, acType)
	
	for i := range msg {
		checksum = checksum ^ byte(msg[i])
	}
	msg = (fmt.Sprintf("$%s*%02X\r\n", msg, checksum))

	checksum = 0 // reset for next message
	for i := range msg2 {
		checksum = checksum ^ byte(msg2[i])
	}
	msg = msg
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

	lastFix := float64(mySituation.GPSLastFixSinceMidnightUTC)
	hr := math.Floor(lastFix / 3600)
	lastFix -= 3600 * hr
	mins := math.Floor(lastFix / 60)
	sec := lastFix - mins*60

	status := "V"
	if isGPSValid() && mySituation.GPSFixQuality > 0 {
		status = "A"
	}

	lat := float64(mySituation.GPSLatitude)
	ns := "N"
	if lat < 0 {
		lat = -lat
		ns = "S"
	}

	deg := math.Floor(lat)
	min := (lat - deg) * 60
	lat = deg*100 + min

	ew := "E"
	lng := float64(mySituation.GPSLongitude)
	if lng < 0 {
		lng = -lng
		ew = "W"
	}

	deg = math.Floor(lng)
	min = (lng - deg) * 60
	lng = deg*100 + min

	gs := float32(mySituation.GPSGroundSpeed)
	trueCourse := float32(mySituation.GPSTrueCourse)
	yy, mm, dd := time.Now().UTC().Date()
	yy = yy % 100
	var magVar, mvEW string
	mode := "N"
	if mySituation.GPSFixQuality == 1 {
		mode = "A"
	} else if mySituation.GPSFixQuality == 2 {
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

func makeGPGGAString() string {
	/*
	 xxGGA
	 time
	 lat (degmin.mmm)
	 NS
	 long (degmin.mmm)
	 EW
	 quality
	 numSV
	 HDOP
	 alt
	 ualt
	 sep
	 uSep
	 diffAge
	 diffStation
	*/

	thisSituation := mySituation
	lastFix := float64(thisSituation.GPSLastFixSinceMidnightUTC)
	hr := math.Floor(lastFix / 3600)
	lastFix -= 3600 * hr
	mins := math.Floor(lastFix / 60)
	sec := lastFix - mins*60

	lat := float64(mySituation.GPSLatitude)
	ns := "N"
	if lat < 0 {
		lat = -lat
		ns = "S"
	}

	deg := math.Floor(lat)
	min := (lat - deg) * 60
	lat = deg*100 + min

	ew := "E"
	lng := float64(mySituation.GPSLongitude)
	if lng < 0 {
		lng = -lng
		ew = "W"
	}

	deg = math.Floor(lng)
	min = (lng - deg) * 60
	lng = deg*100 + min

	numSV := thisSituation.GPSSatellites
	//if numSV > 12 {
	//	numSV = 12
	//}

	//hdop := float32(thisSituation.Accuracy / 4.0)
	//if hdop < 0.7 {hdop = 0.7}
	hdop := 1.0

	alt := thisSituation.GPSAltitudeMSL / 3.28084
	geoidSep := thisSituation.GPSGeoidSep / 3.28084

	var msg string

	if isGPSValid() {
		msg = fmt.Sprintf("GPGGA,%02.f%02.f%05.2f,%010.5f,%s,%011.5f,%s,%d,%d,%.2f,%.1f,M,%.1f,M,,", hr, mins, sec, lat, ns, lng, ew, thisSituation.GPSFixQuality, numSV, hdop, alt, geoidSep)
	} else {
		msg = fmt.Sprintf("GPGGA,,,,,,0,%d,,,,,,,", numSV)
	}

	var checksum byte
	for i := range msg {
		checksum = checksum ^ byte(msg[i])
	}
	msg = fmt.Sprintf("$%s*%X\r\n", msg, checksum)
	return msg

}

/*
Basic TCP server for sending NMEA messages to TCP-based (i.e. AIR Connect compatible)
software: SkyDemon, RunwayHD, etc.
Based on Andreas Krennmair's "Let's build a network application!" chat server demo
http://synflood.at/tmp/golang-slides/mrmcd2012.html#2
*/

type tcpClient struct {
	conn net.Conn
	ch   chan string
}

var msgchan chan string

func tcpNMEAListener() {
	ln, err := net.Listen("tcp", ":2000")
	if err != nil {
		fmt.Println(err)
		return
	}

	msgchan = make(chan string, 1024) // buffered channel n = 1024
	addchan := make(chan tcpClient)
	rmchan := make(chan tcpClient)

	go handleMessages(msgchan, addchan, rmchan)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		go handleConnection(conn, msgchan, addchan, rmchan)
	}
}

/*
func (c tcpClient) ReadLinesInto(ch chan<- string) {
	bufc := bufio.NewReader(c.conn)
	for {
		line, err := bufc.ReadString('\n')
		if err != nil {
			break
		}
		ch <- fmt.Sprintf("%s: %s", c.nickname, line)
	}
}
*/

func (c tcpClient) WriteLinesFrom(ch <-chan string) {
	for msg := range ch {
		_, err := io.WriteString(c.conn, msg)
		if err != nil {
			return
		}
	}
}

func handleConnection(c net.Conn, msgchan chan<- string, addchan chan<- tcpClient, rmchan chan<- tcpClient) {
	//bufc := bufio.NewReader(c)
	defer c.Close()
	client := tcpClient{
		conn: c,
		ch:   make(chan string),
	}
	io.WriteString(c, "PASS?")

	// disabling passcode checks. RunwayHD and SkyDemon don't send CR / LF, and PIN check is something else that can go wrong.
	//time.Sleep(100 * time.Millisecond)

	//code, _, _ := bufc.ReadLine()
	//log.Printf("Passcode entry was %v\n",code)

	//passcode := ""
	/*for passcode != "6000" {
		io.WriteString(c, "PASS?")
		code, _, err := bufc.ReadLine()
		if err != nil {
			log.Printf("Error scanning passcode from client %s: %s\n",c.RemoteAddr(), err)
			continue
		}
		passcode = string(code)
		log.Printf("Received passcode %s from client %s\n", passcode, c.RemoteAddr())
	}
	*/
	io.WriteString(c, "AOK") // correct passcode received; continue to writes
	log.Printf("Correct passcode on client %s. Unlocking.\n", c.RemoteAddr())
	// Register user
	addchan <- client
	defer func() {
		log.Printf("Connection from %s closed.\n", c.RemoteAddr())
		rmchan <- client
	}()

	// I/O
	//go client.ReadLinesInto(msgchan)  //treating the port as read-only once it's opened
	client.WriteLinesFrom(client.ch)
}

func handleMessages(msgchan <-chan string, addchan <-chan tcpClient, rmchan <-chan tcpClient) {
	clients := make(map[net.Conn]chan<- string)

	for {
		select {
		case msg := <-msgchan:
			if globalSettings.DEBUG {
				log.Printf("New message: %s", msg)
			}
			for _, ch := range clients {
				go func(mch chan<- string) { mch <- msg }(ch)
			}
		case client := <-addchan:
			log.Printf("New client: %v\n", client.conn.RemoteAddr().String())
			clients[client.conn] = client.ch
		case client := <-rmchan:
			log.Printf("Client disconnects: %v\n", client.conn.RemoteAddr().String())
			delete(clients, client.conn)
		}
	}
}