package main

import (
	"encoding/hex"
	"math"
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

type TrafficInfo struct {
	icao_addr uint32
	addr_type uint8

	lat float32
	lng float32

	position_valid bool

	alt uint32

	track       uint16
	speed       uint16
	speed_valid bool

	vvel int16

	tail string

	last_seen time.Time
}

var traffic map[uint32]TrafficInfo
var trafficMutex *sync.Mutex

func cleanupOldEntries() {
	trafficMutex.Lock()
	defer trafficMutex.Unlock()
	for icao_addr, ti := range traffic {
		if time.Since(ti.last_seen).Seconds() > float64(60) { //FIXME: 60 seconds with no update on this address - stop displaying.
			delete(traffic, icao_addr)
		}
	}
}

func sendTrafficUpdates() {
	trafficMutex.Lock()
	defer trafficMutex.Unlock()
	cleanupOldEntries()
	for _, ti := range traffic {
		makeTrafficReport(ti)
	}
}

func initTraffic() {
	traffic = make(map[uint32]TrafficInfo)
	trafficMutex = &sync.Mutex{}
}

func makeTrafficReport(ti TrafficInfo) {
	msg := make([]byte, 28)
	// See p.16.
	msg[0] = 0x14 // Message type "Traffic Report".

	msg[1] = 0x10 | ti.addr_type // Alert status, address type.

	// ICAO Address.
	msg[2] = byte((ti.icao_addr & 0x00FF0000) >> 16)
	msg[3] = byte((ti.icao_addr & 0x0000FF00) >> 8)
	msg[4] = byte((ti.icao_addr & 0x000000FF))

	lat := float32(ti.lat)
	tmp := makeLatLng(lat)

	msg[5] = tmp[0] // Latitude.
	msg[6] = tmp[1] // Latitude.
	msg[7] = tmp[2] // Latitude.

	lng := float32(ti.lng)
	tmp = makeLatLng(lng)

	msg[8] = tmp[0]  // Longitude.
	msg[9] = tmp[1]  // Longitude.
	msg[10] = tmp[2] // Longitude.

	//Altitude: OK
	//TODO: 0xFFF "invalid altitude."
	alt := uint16(ti.alt)
	alt = (alt + 1000) / 25
	alt = alt & 0xFFF // Should fit in 12 bits.

	msg[11] = byte((alt & 0xFF0) >> 4) // Altitude.
	msg[12] = byte((alt & 0x00F) << 4)

	msg[12] = byte(((alt & 0x00F) << 4) | 0x8) //FIXME. "Airborne".

	msg[13] = 0x11 //FIXME.

	// Horizontal velocity (speed).

	msg[14] = byte((ti.speed & 0x0FF0) >> 4)
	msg[15] = byte((ti.speed & 0x000F) << 4)

	// Vertical velocity.
	vvel := ti.vvel / 64 // 64fpm resolution.
	msg[15] = msg[15] | byte((vvel&0x0F00)>>8)
	msg[16] = byte(vvel & 0x00FF)

	// Track.
	trk := uint8(float32(ti.track) / TRACK_RESOLUTION) // Resolution is ~1.4 degrees.
	msg[17] = byte(trk)

	msg[18] = 0x01 // "light"

	sendMsg(prepareMessage(msg))
}

func parseDownlinkReport(s string) {
	var ti TrafficInfo
	s = s[1:]
	frame := make([]byte, len(s)/2)
	hex.Decode(frame, []byte(s))

	// Header.
	//	msg_type := (uint8(frame[0]) >> 3) & 0x1f
	ti.addr_type = uint8(frame[0]) & 0x07
	ti.icao_addr = (uint32(frame[1]) << 16) | (uint32(frame[2]) << 8) | uint32(frame[3])

	// OK.
	//	fmt.Printf("%d, %d, %06X\n", msg_type, ti.addr_type, ti.icao_addr)

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
	ti.lat = lat
	ti.lng = lng
	ti.position_valid = position_valid

	raw_alt := (uint32(frame[10]) << 4) | ((uint32(frame[11]) & 0xf0) >> 4)
	//	alt_geo := false // Barometric if not geometric.
	alt := uint32(0)
	if raw_alt != 0 {
		//		alt_geo = (uint8(frame[9]) & 1) != 0
		alt = ((raw_alt - 1) * 25) - 1000
	}
	ti.alt = alt

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
			if airground_state == 1 { // Supersonic
				ew_vel = ew_vel * 4
			}
		}
		if ns_vel_valid && ew_vel_valid {
			if ns_vel != 0 && ew_vel != 0 {
				//TODO: Track type
				track = (360 + 90 - uint16(math.Atan2(float64(ns_vel), float64(ew_vel))*180/math.Pi)) % 360
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
		//TODO.
		return
	}

	ti.track = track
	ti.speed = speed
	ti.vvel = vvel
	ti.speed_valid = speed_valid

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

	ti.last_seen = time.Now()

	trafficMutex.Lock()
	traffic[ti.icao_addr] = ti
	trafficMutex.Unlock()
}

//TODO
/*
//	dump1090Addr    = "127.0.0.1:30003"
inConn, err := net.Dial("tcp", dump1090Addr)
if err != nil {
	time.Sleep(1 * time.Second)
	continue
}
rdr := bufio.NewReader(inConn)
for {
	buf, err := rdr.ReadString('\n')
	if err != nil { // Must have disconnected?
		break
	}
	buf = strings.Trim(buf, "\r\n")
	//log.Printf("%s\n", buf)
	x := strings.Split(buf, ",")
	//TODO: Add more sophisticated stuff that combines heading/speed updates with the location.
	if len(x) < 22 {
		continue
	}
	icao := x[4]
	icaoDec, err := strconv.ParseInt(icao, 16, 32)
	if err != nil {
		continue
	}
	mutex.Lock()
	// Retrieve previous information on this ICAO code.
	var pi PositionInfo
	if _, ok := blips[icaoDec]; ok {
		pi = blips[icaoDec]
	}

	if x[1] == "3" {
		//MSG,3,111,11111,AC2BB7,111111,2015/07/28,03:59:12.363,2015/07/28,03:59:12.353,,5550,,,42.35847,-83.42212,,,,,,0
		alt := x[11]
		lat := x[14]
		lng := x[15]

		//log.Printf("icao=%s, icaoDec=%d, alt=%s, lat=%s, lng=%s\n", icao, icaoDec, alt, lat, lng)
		pi.alt = alt
		pi.lat = lat
		pi.lng = lng
	}
	if x[1] == "4" {
		// MSG,4,111,11111,A3B557,111111,2015/07/28,06:13:36.417,2015/07/28,06:13:36.398,,,414,278,,,-64,,,,,0
		vel := x[12]
		hdg := x[13]
		vr := x[16]

		//log.Printf("icao=%s, icaoDec=%d, vel=%s, hdg=%s, vr=%s\n", icao, icaoDec, vel, hdg, vr)
		pi.vel = vel
		pi.hdg = hdg
		pi.vr = vr
	}
	if x[1] == "1" {
		// MSG,1,,,%02X%02X%02X,,,,,,%s,,,,,,,,0,0,0,0
		tail := x[10]
		pi.tail = tail
	}

	// Update "last seen" (any type of message).
	pi.last_seen = time.Now()

	blips[icaoDec] = pi // Update information on this ICAO code.
	mutex.Unlock()
*/
