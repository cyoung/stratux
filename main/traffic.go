/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New"" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	traffic.go: Target management, UAT downlink message processing, 1090ES source input, GDL90 traffic reports.
*/

package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
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

	Tail        string
	Timestamp   time.Time
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
	var msg []byte
	for _, ti := range traffic { // TO-DO: Limit number of aircraft in traffic message. ForeFlight chokes at ~1000-2000 messages depending on iDevice RAM. Practical limit likely around 500-1000 aircraft
		if ti.Position_valid {
			msg = append(msg, makeTrafficReportMsg(ti)...)
		}
	}

	if len(msg) > 0 {
		sendGDL90(msg, false)
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

func makeTrafficReportMsg(ti TrafficInfo) []byte {
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

	return prepareMessage(msg)
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

	// TO-DO: Parse time from downlink message
	if !isGPSClockValid() {
		// this is where we'd update the RPi clock if GPS isn't connected
	}

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

	ti.Timestamp = time.Now() // TODO: Parse from downlink message.

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

			ti.Timestamp = time.Now() // TODO: Parse from x[8] and x[9]

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

/*
updateDemoTraffic creates / updates a simulated traffic target for demonstration / debugging
purpose. Target will circle clockwise around the current GPS position (if valid) or around
KOSH, once every five minutes.

Inputs are ICAO 24-bit hex code, tail number (8 chars max), relative altitude in feet,
groundspeed in knots, and bearing offset from 0 deg initial position.
*/
func updateDemoTraffic(icao uint32, tail string, relAlt float32, gs float64, offset int32) {
	var ti TrafficInfo

	hdg := float64((int32(stratuxClock.Milliseconds/1000) + offset) % 360)
	// gs := float64(220) // knots
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
	if hdg > 180 && hdg < 195 {
		ti.Speed_valid = false
	} else {
		ti.Speed_valid = true
	}
	ti.Vvel = 0
	ti.Tail = tail // "DEMO1234"
	ti.Timestamp = time.Now()
	ti.Last_seen = stratuxClock.Time
	ti.Last_source = 1
	if icao%7 == 1 { // make some of the traffic UAT sourced
		ti.Last_source = 2
	}

	// now insert this into the traffic map...
	trafficMutex.Lock()
	defer trafficMutex.Unlock()
	traffic[ti.Icao_addr] = ti
	registerTrafficUpdate(ti)
	seenTraffic[ti.Icao_addr] = true
}

func initTraffic() {
	traffic = make(map[uint32]TrafficInfo)
	seenTraffic = make(map[uint32]bool)
	trafficMutex = &sync.Mutex{}
	go esListen()
}
