/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New"" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	Support for ForeFlight simulator mode, ownship details parsing,
	and FLARM NMEA message generation (c) 2016 AvSquirrel
	(https://github.com/AvSquirrel)

	traffic.go: Target management, UAT downlink message processing, 1090ES source input, GDL90 traffic reports.
*/

package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

//-0b2b48fe3aef1f88621a0856110a31c01105c4e6c4e6c40a9a820300000000000000;rs=7;

/*

HDR:
 MDB Type:          1
 Address:           2B48FE (TIS-B track file address)
SV:
 NIC:               6
 Latitude:          +41.4380
 Longitude:         -84.1056
 Altitude:          2300 ft (barometric)
 N/S velocity:      -65 kt
 E/W velocity:      -98 kt
 Track:             236
 Speed:             117 kt
 Vertical rate:     0 ft/min (from barometric altitude)
 UTC coupling:      no
 TIS-B site ID:     1
MS:
 Emitter category:  No information
 Callsign:          unavailable
 Emergency status:  No emergency
 UAT version:       2
 SIL:               2
 Transmit MSO:      38
 NACp:              8
 NACv:              1
 NICbaro:           0
 Capabilities:
 Active modes:
 Target track type: true heading
AUXSV:
 Sec. altitude:     unavailable

*/

const (
	TRAFFIC_SOURCE_1090ES = 1
	TRAFFIC_SOURCE_UAT    = 2
)

type TrafficInfo struct {
	Icao_addr        uint32
	OnGround         bool
	addr_type        uint8
	emitter_category uint8

	Lat float32
	Lng float32

	Position_valid bool

	Alt int32

	Track       uint16
	Speed       uint16
	Speed_valid bool

	Vvel int16

	Tail string

	Last_seen   time.Time
	Last_source uint8
}

var traffic map[uint32]TrafficInfo
var trafficMutex *sync.Mutex
var seenTraffic map[uint32]bool // Historical list of all ICAO addresses seen.

func cleanupOldEntries() {
	for icao_addr, ti := range traffic {
		if stratuxClock.Since(ti.Last_seen) > 60*time.Second { //FIXME: 60 seconds with no update on this address - stop displaying.
			delete(traffic, icao_addr)
		}
	}
}

func sendTrafficUpdates() {
	trafficMutex.Lock()
	defer trafficMutex.Unlock()
	cleanupOldEntries()
	for _, ti := range traffic {
		if ti.Position_valid {
			makeTrafficReport(ti)
		}
	}
}

// Send update to attached client.
func registerTrafficUpdate(ti TrafficInfo) {
	if !ti.Position_valid { // Don't send unless a valid position exists.
		return
	}

	tiJSON, _ := json.Marshal(&ti)
	trafficUpdate.Send(tiJSON)
}

func makeTrafficReport(ti TrafficInfo) {
	msg := make([]byte, 28)
	// See p.16.
	msg[0] = 0x14 // Message type "Traffic Report".

	msg[1] = 0x10 | ti.addr_type // Alert status, address type.

	// ICAO Address.
	msg[2] = byte((ti.Icao_addr & 0x00FF0000) >> 16)
	msg[3] = byte((ti.Icao_addr & 0x0000FF00) >> 8)
	msg[4] = byte((ti.Icao_addr & 0x000000FF))

	lat := float32(ti.Lat)
	tmp := makeLatLng(lat)

	msg[5] = tmp[0] // Latitude.
	msg[6] = tmp[1] // Latitude.
	msg[7] = tmp[2] // Latitude.

	lng := float32(ti.Lng)
	tmp = makeLatLng(lng)

	msg[8] = tmp[0]  // Longitude.
	msg[9] = tmp[1]  // Longitude.
	msg[10] = tmp[2] // Longitude.

	// Altitude: OK
	// GDL 90 Data Interface Specification examples:
	// where 1,000 foot offset and 25 foot resolution (1,000 / 25 = 40)
	//    -1,000 feet               0x000
	//    0 feet                    0x028
	//    +1000 feet                0x050
	//    +101,350 feet             0xFFE
	//    Invalid or unavailable    0xFFF
	//
	// Algo example at: https://play.golang.org/p/VXCckSdsvT
	//
	var alt int16
	if ti.Alt < -1000 || ti.Alt > 101350 {
		alt = 0x0FFF
	} else {
		// output guaranteed to be between 0x0000 and 0x0FFE
		alt = int16((ti.Alt / 25) + 40)
	}
	msg[11] = byte((alt & 0xFF0) >> 4) // Altitude.
	msg[12] = byte((alt & 0x00F) << 4)

	msg[12] = byte(((alt & 0x00F) << 4) | 0x3) // True heading.
	if !ti.OnGround {
		msg[12] = msg[12] | 0x08 // Airborne.
	}

	msg[13] = 0x11 //FIXME.

	// Horizontal velocity (speed).

	msg[14] = byte((ti.Speed & 0x0FF0) >> 4)
	msg[15] = byte((ti.Speed & 0x000F) << 4)

	// Vertical velocity.
	vvel := ti.Vvel / 64 // 64fpm resolution.
	msg[15] = msg[15] | byte((vvel&0x0F00)>>8)
	msg[16] = byte(vvel & 0x00FF)

	// Track.
	trk := uint8(float32(ti.Track) / TRACK_RESOLUTION) // Resolution is ~1.4 degrees.
	msg[17] = byte(trk)

	msg[18] = ti.emitter_category

	// msg[19] to msg[26] are "call sign" (tail).
	for i := 0; i < len(ti.Tail) && i < 8; i++ {
		c := byte(ti.Tail[i])
		if c != 20 && !((c >= 48) && (c <= 57)) && !((c >= 65) && (c <= 90)) && c != 'e' && c != 'u' { // See p.24, FAA ref.
			c = byte(20)
		}
		msg[19+i] = c
	}

	if globalSettings.ForeFlightSimMode == false {
		sendGDL90(prepareMessage(msg), false)
		//fmt.Printf("Sending GDL traffic message\n")
	} else {
		airborne := 0
		if !ti.OnGround {
			airborne = 1
		}

		ffmsg := fmt.Sprintf("XTRAFFICStratux,%v,%.4f,%.4f,%.f,%.f,%b,%.f,%.f,%s %03d", ti.Icao_addr, ti.Lat, ti.Lng, float32(ti.Alt), float32(ti.Vvel), airborne, float32(ti.Track), float32(ti.Speed), ti.Tail, int16(ti.Alt/100))

		sendMsg([]byte(ffmsg), NETWORK_AHRS_FFSIM, false)
		fmt.Printf("Sending FF traffic message %s\n", ffmsg)

		/*  Documentation of FF Flight Sim format from https://www.foreflight.com/support/network-gps/
		For traffic data, the simulator will need to send packets in the form of a string message like this:

		XTRAFFICMy Sim,,33.85397339,-118.32486725,3749.9,-213.0,1,68.2,126.0,KS6

		The "words" are separated by a comma (no word may contain a comma). The required words are:

		XTRAFFIC followed by a name/ID of the simulator type sending the data (that might be "My Sim" without quotes)
		ICAO address, an integer ID
		Traffic latitude - float
		Traffic longitude - float
		Traffic geometric altitude - float (feet)
		Traffic vertical speed - float (ft/min)
		Airborne boolean flag - 1 or 0: 1=airborne; 0=surface
		Heading - float, degrees true
		Velocity knots - float
		Callsign - string
		*/
	}

	/*
		Prototype for FLARM-formatted traffic output.
	*/

	if globalSettings.FLARMTraffic == true && isGPSValid() {
		flarmmsg, valid := makeFlarmNMEAString(ti)
		fmt.Printf("FLARM string valid: %t\n", valid)
		// if valid {
		log.Printf("FLARM String: %s\n", flarmmsg) // TO-DO: Send this to /dev/ttyAMA0
		//}

	}
}

/*
parseOwnshipADSBMessage scans the traffic map for the ownship ICAO address. If found, it
will feed that data into mySituation.x

Return value is a byte indicating status of the message

 Bit 0: 0 if code not found; 1 if code found
 Bit
 Bit 1-6: Age of message

 7 6 5 4 3 2 1 0

*/

func parseOwnshipADSBMessage() uint8 {
	code, _ := strconv.ParseInt(globalSettings.OwnshipModeS, 16, 32)
	ti, present := traffic[uint32(code)]
	if !present { // address isn't in the map
		fmt.Printf("Address %X not seen in the traffic map.\n", code)
		return 0
	}
	mySituation.OwnshipTail = ti.Tail
	mySituation.OwnshipPressureAlt = ti.Alt
	mySituation.OwnshipLat = ti.Lat
	mySituation.OwnshipLng = ti.Lng
	mySituation.OwnshipLastSeen = ti.Last_seen

	fmt.Printf("Address %X found in the traffic map at %.3f° LAT and %.3f° LNG with tail number %s at %d' MSL.\n", code, mySituation.OwnshipLat, mySituation.OwnshipLng, mySituation.OwnshipTail, mySituation.OwnshipPressureAlt)
	// do all the things!
	return 1
}

/*
makeFlarmNMEAString creates a NMEA-formatted PFLAA string (FLARM traffic format) with checksum.
*/

func makeFlarmNMEAString(ti TrafficInfo) (msg string, valid bool) {

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
	fmt.Printf("ICAO target %X (%s) is %.1f meters away at %.1f degrees\n", ti.Icao_addr, ti.Tail, dist, bearing)

	if distN > 32767 || distN < -32767 || distE > 32767 || distE < -32767 {
		msg = ""
		valid = false
		return
	} else {
		relativeNorth = int16(distN)
		relativeEast = int16(distE)
	}

	relativeVertical = int16(float64(ti.Alt)*0.3048 - mySituation.Pressure_alt*0.3048) // convert to meters

	if (dist < 926) && (relativeVertical < 152) && (relativeVertical > -152) { // 926 m = 0.5 NM; 152m = 500'
		alarmLevel = 2
	} else if (dist < 1852) && (relativeVertical < 304) && (relativeVertical > -304) { // 1852 m = 1.0 NM ; 304 m = 1000'
		alarmLevel = 1
	}

	if ti.Speed_valid {
		groundSpeed = int16(float32(ti.Speed) * 0.5144) // convert to m/s
	}

	climbRate = float32(ti.Vvel) * 0.3048 / 60 // convert to m/s
	msg = fmt.Sprintf("PFLAA,%d,%d,%d,%d,%d,%X,%d,0,%d,%0.1f,0", alarmLevel, relativeNorth, relativeEast, relativeVertical, idType, ti.Icao_addr, ti.Track, groundSpeed, climbRate)
	for i := range msg {
		checksum = checksum ^ byte(msg[i])
	}
	msg = fmt.Sprintf("$%s*%X", msg, checksum)
	valid = true
	return
}

// Create fake traffic targets for demonstration purpose. Targets will circle clockwise about current GPS position once every 5 minutes
// Parameters are ICAO 24-bit address, tail number (8-byte max), speed in knots, and degree offset from initial position.

func updateDemoTraffic(icao uint32, tail string, relAlt float32, gs float64, offset int32) {
	var ti TrafficInfo

	hdg := float64((int32(stratuxClock.Milliseconds/1000) + offset) % 360)
	radius := gs * 0.1 / (2 * math.Pi)
	x := radius * math.Cos(hdg*math.Pi/180.0)
	y := radius * math.Sin(hdg*math.Pi/180.0)
	// default traffic location is Oshkosh if GPS not detected
	lat := 43.99
	lng := -88.56
	if isGPSValid() {
		lat = float64(mySituation.Lat)
		lng = float64(mySituation.Lng)
	}
	traffRelLat := y / 60
	traffRelLng := -x / (60 * math.Cos(lat*math.Pi/180.0))

	ti.Icao_addr = icao
	ti.OnGround = false
	ti.addr_type = 0
	ti.emitter_category = 1
	ti.Lat = float32(lat + traffRelLat)
	ti.Lng = float32(lng + traffRelLng)
	ti.Position_valid = true
	ti.Alt = int32(mySituation.Alt + relAlt)
	ti.Track = uint16(hdg)
	ti.Speed = uint16(gs)
	ti.Speed_valid = true
	ti.Vvel = 0
	ti.Tail = tail // "DEMO1234"
	ti.Last_seen = stratuxClock.Time
	ti.Last_source = 1

	// now insert this into the traffic map...
	trafficMutex.Lock()
	defer trafficMutex.Unlock()
	traffic[ti.Icao_addr] = ti
	registerTrafficUpdate(ti)
	seenTraffic[ti.Icao_addr] = true
}

func parseDownlinkReport(s string) {
	var ti TrafficInfo
	s = s[1:]
	frame := make([]byte, len(s)/2)
	hex.Decode(frame, []byte(s))

	// Header.
	msg_type := (uint8(frame[0]) >> 3) & 0x1f

	// Extract emitter category.
	if msg_type == 1 || msg_type == 3 {
		v := (uint16(frame[17]) << 8) | (uint16(frame[18]))
		ti.emitter_category = uint8((v / 1600) % 40)
	}

	icao_addr := (uint32(frame[1]) << 16) | (uint32(frame[2]) << 8) | uint32(frame[3])

	trafficMutex.Lock()
	defer trafficMutex.Unlock()
	if curTi, ok := traffic[icao_addr]; ok { // Retrieve the current entry, as it may contain some useful information like "tail" from 1090ES.
		ti = curTi
	}
	ti.Icao_addr = icao_addr

	ti.addr_type = uint8(frame[0]) & 0x07

	// OK.
	//	fmt.Printf("%d, %d, %06X\n", msg_type, ti.addr_type, ti.Icao_addr)

	nic := uint8(frame[11]) & 15 //TODO: Meaning?

	raw_lat := (uint32(frame[4]) << 15) | (uint32(frame[5]) << 7) | (uint32(frame[6]) >> 1)
	raw_lon := ((uint32(frame[6]) & 0x01) << 23) | (uint32(frame[7]) << 15) | (uint32(frame[8]) << 7) | (uint32(frame[9]) >> 1)

	lat := float32(0.0)
	lng := float32(0.0)

	position_valid := false
	if nic != 0 || raw_lat != 0 || raw_lon != 0 {
		position_valid = true
		lat = float32(raw_lat) * 360.0 / 16777216.0
		if lat > 90 {
			lat = lat - 180
		}
		lng = float32(raw_lon) * 360.0 / 16777216.0
		if lng > 180 {
			lng = lng - 360
		}
	}
	ti.Lat = lat
	ti.Lng = lng
	ti.Position_valid = position_valid

	raw_alt := (int32(frame[10]) << 4) | ((int32(frame[11]) & 0xf0) >> 4)
	//	alt_geo := false // Barometric if not geometric.
	alt := int32(0)
	if raw_alt != 0 {
		//		alt_geo = (uint8(frame[9]) & 1) != 0
		alt = ((raw_alt - 1) * 25) - 1000
	}
	ti.Alt = alt

	//OK.
	//	fmt.Printf("%d, %t, %f, %f, %t, %d\n", nic, position_valid, lat, lng, alt_geo, alt)

	airground_state := (uint8(frame[12]) >> 6) & 0x03
	//OK.
	//	fmt.Printf("%d\n", airground_state)

	ns_vel := int16(0)
	ew_vel := int16(0)
	track := uint16(0)
	speed_valid := false
	speed := uint16(0)
	vvel := int16(0)
	//	vvel_geo := false
	if airground_state == 0 || airground_state == 1 { // Subsonic. Supersonic.
		// N/S velocity.
		ns_vel_valid := false
		ew_vel_valid := false
		raw_ns := ((int16(frame[12]) & 0x1f) << 6) | ((int16(frame[13]) & 0xfc) >> 2)
		if (raw_ns & 0x3ff) != 0 {
			ns_vel_valid = true
			ns_vel = ((raw_ns & 0x3ff) - 1)
			if (raw_ns & 0x400) != 0 {
				ns_vel = 0 - ns_vel
			}
			if airground_state == 1 { // Supersonic.
				ns_vel = ns_vel * 4
			}
		}
		// E/W velocity.
		raw_ew := ((int16(frame[13]) & 0x03) << 9) | (int16(frame[14]) << 1) | ((int16(frame[15] & 0x80)) >> 7)
		if (raw_ew & 0x3ff) != 0 {
			ew_vel_valid = true
			ew_vel = (raw_ew & 0x3ff) - 1
			if (raw_ew & 0x400) != 0 {
				ew_vel = 0 - ew_vel
			}
			if airground_state == 1 { // Supersonic.
				ew_vel = ew_vel * 4
			}
		}
		if ns_vel_valid && ew_vel_valid {
			if ns_vel != 0 && ew_vel != 0 {
				//TODO: Track type
				track = uint16((360 + 90 - (int16(math.Atan2(float64(ns_vel), float64(ew_vel)) * 180 / math.Pi))) % 360)
			}
			speed_valid = true
			speed = uint16(math.Sqrt(float64((ns_vel * ns_vel) + (ew_vel * ew_vel))))
		}

		// Vertical velocity.
		raw_vvel := ((int16(frame[15]) & 0x7f) << 4) | ((int16(frame[16]) & 0xf0) >> 4)
		if (raw_vvel & 0x1ff) != 0 {
			//			vvel_geo = (raw_vvel & 0x400) == 0
			vvel = ((raw_vvel & 0x1ff) - 1) * 64
			if (raw_vvel & 0x200) != 0 {
				vvel = 0 - vvel
			}
		}
	} else if airground_state == 2 { // Ground vehicle.
		ti.OnGround = true
		raw_gs := ((uint16(frame[12]) & 0x1f) << 6) | ((uint16(frame[13]) & 0xfc) >> 2)
		if raw_gs != 0 {
			speed_valid = true
			speed = ((raw_gs & 0x3ff) - 1)
		}

		raw_track := ((uint16(frame[13]) & 0x03) << 9) | (uint16(frame[14]) << 1) | ((uint16(frame[15]) & 0x80) >> 7)
		//tt := ((raw_track & 0x0600) >> 9)
		//FIXME: tt == 1 TT_TRACK. tt == 2 TT_MAG_HEADING. tt == 3 TT_TRUE_HEADING.
		track = uint16((raw_track & 0x1ff) * 360 / 512)

		// Dimensions of vehicle - skip.
	}

	ti.Track = track
	ti.Speed = speed
	ti.Vvel = vvel
	ti.Speed_valid = speed_valid

	//OK.
	//	fmt.Printf("ns_vel %d, ew_vel %d, track %d, speed_valid %t, speed %d, vvel_geo %t, vvel %d\n", ns_vel, ew_vel, track, speed_valid, speed, vvel_geo, vvel)

	/*
		utc_coupled := false
		tisb_site_id := uint8(0)

		if (uint8(frame[0]) & 7) == 2 || (uint8(frame[0]) & 7) == 3 { //TODO: Meaning?
			tisb_site_id = uint8(frame[16]) & 0x0f
		} else {
			utc_coupled = (uint8(frame[16]) & 0x08) != 0
		}
	*/

	//OK.
	//	fmt.Printf("tisb_site_id %d, utc_coupled %t\n", tisb_site_id, utc_coupled)

	ti.Last_source = TRAFFIC_SOURCE_UAT
	ti.Last_seen = stratuxClock.Time

	// Parse tail number, if available.
	if msg_type == 1 || msg_type == 3 { // Need "MS" portion of message.
		base40_alphabet := string("0123456789ABCDEFGHIJKLMNOPQRTSUVWXYZ  ..")
		tail := ""

		v := (uint16(frame[17]) << 8) | uint16(frame[18])
		tail += string(base40_alphabet[(v/40)%40])
		tail += string(base40_alphabet[v%40])
		v = (uint16(frame[19]) << 8) | uint16(frame[20])
		tail += string(base40_alphabet[(v/1600)%40])
		tail += string(base40_alphabet[(v/40)%40])
		tail += string(base40_alphabet[v%40])
		v = (uint16(frame[21]) << 8) | uint16(frame[22])
		tail += string(base40_alphabet[(v/1600)%40])
		tail += string(base40_alphabet[(v/40)%40])
		tail += string(base40_alphabet[v%40])

		tail = strings.Trim(tail, " ")
		ti.Tail = tail
	}

	if globalSettings.DEBUG {
		// This is a hack to show the source of the traffic in ForeFlight.
		if len(ti.Tail) == 0 || (len(ti.Tail) != 0 && len(ti.Tail) < 8 && ti.Tail[0] != 'U') {
			ti.Tail = "u" + ti.Tail
		}
	}

	traffic[ti.Icao_addr] = ti
	registerTrafficUpdate(ti)
	seenTraffic[ti.Icao_addr] = true // Mark as seen.
}

func esListen() {
	for {
		if !globalSettings.ES_Enabled {
			time.Sleep(1 * time.Second) // Don't do much unless ES is actually enabled.
			continue
		}
		dump1090Addr := "127.0.0.1:30003"
		inConn, err := net.Dial("tcp", dump1090Addr)
		if err != nil { // Local connection failed.
			time.Sleep(1 * time.Second)
			continue
		}
		rdr := bufio.NewReader(inConn)
		for globalSettings.ES_Enabled {
			buf, err := rdr.ReadString('\n')
			if err != nil { // Must have disconnected?
				break
			}
			buf = strings.Trim(buf, "\r\n")
			//log.Printf("%s\n", buf)
			replayLog(buf, MSGCLASS_ES) // Log the raw message.
			x := strings.Split(buf, ",")
			if len(x) < 22 {
				continue
			}
			icao := x[4]
			icaoDecf, err := strconv.ParseInt(icao, 16, 32)
			if err != nil {
				continue
			}

			// Log the message after we've determined that it at least meets some requirements on the fields.
			var thisMsg msg
			thisMsg.MessageClass = MSGCLASS_ES
			thisMsg.TimeReceived = stratuxClock.Time
			thisMsg.Data = []byte(buf)
			MsgLog = append(MsgLog, thisMsg)

			// Begin to parse the message.
			icaoDec := uint32(icaoDecf)
			trafficMutex.Lock()
			// Retrieve previous information on this ICAO code.
			var ti TrafficInfo
			if val, ok := traffic[icaoDec]; ok {
				ti = val
			}

			ti.Icao_addr = icaoDec

			//FIXME: Some stale information will be renewed.
			valid_change := true
			if x[1] == "3" { // ES airborne position message. DF17 BDS 0,5.
				//MSG,3,111,11111,AC2BB7,111111,2015/07/28,03:59:12.363,2015/07/28,03:59:12.353,,5550,,,42.35847,-83.42212,,,,,,0
				//MSG,3,111,11111,A5D007,111111,          ,            ,          ,            ,,35000,,,42.47454,-82.57433,,,0,0,0,0
				alt := x[11]
				lat := x[14]
				lng := x[15]

				if len(alt) == 0 || len(lat) == 0 || len(lng) == 0 { //FIXME.
					valid_change = false
				}

				altFloat, err := strconv.ParseFloat(alt, 32)
				if err != nil {
					//					log.Printf("err parsing alt (%s): %s\n", alt, err.Error())
					valid_change = false
				}

				latFloat, err := strconv.ParseFloat(lat, 32)
				if err != nil {
					//					log.Printf("err parsing lat (%s): %s\n", lat, err.Error())
					valid_change = false
				}
				lngFloat, err := strconv.ParseFloat(lng, 32)
				if err != nil {
					//					log.Printf("err parsing lng (%s): %s\n", lng, err.Error())
					valid_change = false
				}

				//log.Printf("icao=%s, icaoDec=%d, alt=%s, lat=%s, lng=%s\n", icao, icaoDec, alt, lat, lng)
				if valid_change {
					ti.Alt = int32(altFloat)
					ti.Lat = float32(latFloat)
					ti.Lng = float32(lngFloat)
					ti.Position_valid = true
				}
			}
			if x[1] == "4" { // ES airborne velocity message. DF17 BDS 0,9.
				// MSG,4,111,11111,A3B557,111111,2015/07/28,06:13:36.417,2015/07/28,06:13:36.398,,,414,278,,,-64,,,,,0
				// MSG,4,111,11111,ABE287,111111,2016/01/03,19:44:43.440,2016/01/03,19:44:43.401,,,469,88,,,0,,,,,0
				speed := x[12]
				track := x[13]
				vvel := x[16]

				if len(speed) == 0 || len(track) == 0 || len(vvel) == 0 {
					valid_change = false
				}

				speedFloat, err := strconv.ParseFloat(speed, 32)
				if err != nil {
					//					log.Printf("err parsing speed (%s): %s\n", speed, err.Error())
					valid_change = false
				}

				trackFloat, err := strconv.ParseFloat(track, 32)
				if err != nil {
					//					log.Printf("err parsing track (%s): %s\n", track, err.Error())
					valid_change = false
				}
				vvelFloat, err := strconv.ParseFloat(vvel, 32)
				if err != nil {
					//					log.Printf("err parsing vvel (%s): %s\n", vvel, err.Error())
					valid_change = false
				}

				//log.Printf("icao=%s, icaoDec=%d, vel=%s, hdg=%s, vr=%s\n", icao, icaoDec, vel, hdg, vr)
				if valid_change {
					ti.Speed = uint16(speedFloat)
					ti.Track = uint16(trackFloat)
					ti.Vvel = int16(vvelFloat)
					ti.Speed_valid = true
				}
			}
			if x[1] == "1" { // ES identification and category. DF17 BDS 0,8.
				// MSG,1,,,%02X%02X%02X,,,,,,%s,,,,,,,,0,0,0,0
				tail := x[10]

				if len(tail) == 0 {
					valid_change = false
				}

				if valid_change {
					ti.Tail = tail
				}
			}
			if x[1] == "5" { // Surveillance alt message. DF4, DF20.
				// MSG,5,,,%02X%02X%02X,,,,,,,%d,,,,,,,%d,%d,%d,%d
				// MSG,5,111,11111,AB5F1B,111111,2016/01/03,04:43:52.028,2016/01/03,04:43:52.006,,13050,,,,,,,0,,0,0
				alt := x[11]

				altFloat, err := strconv.ParseFloat(alt, 32)
				if len(alt) != 0 && err == nil {
					ti.Alt = int32(altFloat)
				}
			}

			// Update "last seen" (any type of message, as long as the ICAO addr can be parsed).
			ti.Last_source = TRAFFIC_SOURCE_1090ES
			ti.Last_seen = stratuxClock.Time

			ti.addr_type = 0           //FIXME: ADS-B with ICAO address. Not recognized by ForeFlight.
			ti.emitter_category = 0x01 //FIXME. "Light"

			// This is a hack to show the source of the traffic in ForeFlight.
			ti.Tail = strings.Trim(ti.Tail, " ")
			if globalSettings.DEBUG {
				if len(ti.Tail) == 0 || (len(ti.Tail) != 0 && len(ti.Tail) < 8 && ti.Tail[0] != 'E') {
					ti.Tail = "e" + ti.Tail
				}
			}

			traffic[icaoDec] = ti // Update information on this ICAO code.
			registerTrafficUpdate(ti)
			seenTraffic[icaoDec] = true // Mark as seen.
			trafficMutex.Unlock()
		}
	}
}

func initTraffic() {
	traffic = make(map[uint32]TrafficInfo)
	seenTraffic = make(map[uint32]bool)
	trafficMutex = &sync.Mutex{}
	go esListen()
}
