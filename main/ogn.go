
/*
	Copyright (c) 2020 Adrian Batzill
	Distributable under the terms of The "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	ogn.go: Routines for reading traffic from ogn-rx-eu
*/

package main

import (
	"encoding/json"
	"encoding/hex"
	"encoding/binary"
	"net"
	"bufio"
	"strconv"
	"strings"
	"time"
	"log"
	"io/ioutil"
)

// {"sys":"OGN","addr":"395F39","addr_type":3,"acft_type":"1","lat_deg":51.7657533,"lon_deg":-1.1918533,"alt_msl_m":124,"alt_std_m":63,"track_deg":0.0,"speed_mps":0.3,"climb_mps":-0.5,"turn_dps":0.0,"DOP":1.5}
type OgnMessage struct {
	Sys string
	Time int64
	Addr string
	Addr_type int32
	Acft_type string
	Acft_cat string
	Lat_deg float32
	Lon_deg float32
	Alt_msl_m float32
	Alt_hae_m float32
	Alt_std_m float32
	Track_deg float64
	Speed_mps float64
	Climb_mps float64
	Turn_dps float64
	DOP float64
	SNR_dB float64
	Rx_err int32

	// Status message (Sys=status):
	Bkg_noise_db float32
	Gain_db      float32
}


var ognReadWriter *bufio.ReadWriter



func ognPublishNmea(nmea string) {
	if ognReadWriter != nil {
		// TODO: we could filter a bit more to only send RMC/GGA, but for now it's just everything
		if len(nmea) > 5 && nmea[3:6] == "GGA" || nmea[3:6] == "RMC" {
			//log.Printf(nmea)
			ognReadWriter.Write([]byte(nmea + "\r\n"))
			ognReadWriter.Flush()
		}
	}
}

func ognListen() {
	//go predTest()
	for {
		if !globalSettings.OGN_Enabled || OGNDev == nil {
			// wait until OGN is enabled
			time.Sleep(1 * time.Second)
			continue
		}
		log.Printf("ogn-rx-eu connecting...")
		ognAddr := "127.0.0.1:30010"
		conn, err := net.Dial("tcp", ognAddr)
		if err != nil { // Local connection failed.
			time.Sleep(3 * time.Second)
			continue
		}
		log.Printf("ogn-rx-eu successfully connected")
		globalStatus.OGN_connected = true
		ognReadWriter = bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
		for globalSettings.OGN_Enabled {
			buf, err := ognReadWriter.ReadBytes('\n')
			if err != nil {
				log.Printf("ogn-rx-eu connection lost.")
				break
			}

			var thisMsg msg
			thisMsg.MessageClass = MSGCLASS_OGN
			thisMsg.TimeReceived = stratuxClock.Time
			thisMsg.Data = string(buf)

			var msg OgnMessage
			err = json.Unmarshal(buf, &msg)
			if err != nil {
				log.Printf("Invalid Data from OGN: " + string(buf))
				continue
			}

			if msg.Sys == "status" {
				importOgnStatusMessage(msg)
			} else {
				msgLogAppend(thisMsg)
				logMsg(thisMsg) // writes to replay logs
				importOgnTrafficMessage(msg, buf)
			}
		}
		globalStatus.OGN_connected = false
		ognReadWriter = nil
		conn.Close()
		time.Sleep(3*time.Second)
	}
}

func importOgnStatusMessage(msg OgnMessage) {
	globalStatus.OGN_noise_db = msg.Bkg_noise_db
	globalStatus.OGN_gain_db = msg.Gain_db
}

func importOgnTrafficMessage(msg OgnMessage, buf []byte) {
	var ti TrafficInfo
	addressBytes, _ := hex.DecodeString(msg.Addr)
	addressBytes = append([]byte{0}, addressBytes...)
	address := binary.BigEndian.Uint32(addressBytes)

	// GDL90 only knows 2 address types. ICAO and non-ICAO, so we map to those.
	// for OGN: 1=ICAO. For us: 0=ICAO, 1="ADS-B with Self-assigned address"
	addrType := uint8(1) // Non-ICAO Address
	if msg.Addr_type == 1 { // ICAO Address
		addrType = 0 
	}
	// Store in higher-order bytes in front of the 24 bit address so we can handle address collinsions gracefully.
	// For ICAO it will be null, so traffic is merged. For others it will be 1, so traffic is kept seperately
	key := uint32(addrType) << 24 | address 

	// Basic plausibility check:
	dist, _, _, _ := distRect(float64(mySituation.GPSLatitude), float64(mySituation.GPSLongitude), float64(msg.Lat_deg), float64(msg.Lon_deg))
	if dist >= 50000  || (msg.Lat_deg == 0 && msg.Lon_deg == 0) {
		// more than 50km away? Ignore. Most likely invalid data
		return
	}

	trafficMutex.Lock()
	defer trafficMutex.Unlock()

	if existingTi, ok := traffic[key]; ok {
		ti = existingTi
	}
	ti.Icao_addr = address
	ti.Addr_type = addrType
	if len(ti.Tail) == 0 {
		ti.Tail = getTailNumber(msg.Addr, msg.Sys)
	}
	ti.Last_source = TRAFFIC_SOURCE_OGN
	if msg.Time > 0 {
		ti.Timestamp = time.Unix(msg.Time, 0)
	} else {
		ti.Timestamp = time.Now().UTC()
	}
	ti.Age = time.Now().UTC().Sub(ti.Timestamp).Seconds()
	if ti.Age > 30 || ti.Age < -2 {
		log.Printf("Discarding likely invalid OGN target: %s", string(buf))
		return
	}

	// set altitude
	// To keep the rest of the system as simple as possible, we want to work with barometric altitude everywhere.
	// To do so, we use our own known geoid separation and pressure difference to compute the expected barometric altitude of the traffic.
	// Some OGN trackers are equiped with a baro sensor, but older firmwares send wrong data, so we usually can't rely on it.
	alt := msg.Alt_msl_m * 3.28084
	if alt == 0 {
		alt = msg.Alt_hae_m * 3.28084 - mySituation.GPSGeoidSep
	}
	if isGPSValid() && isTempPressValid() {
		ti.Alt = int32(alt - mySituation.GPSAltitudeMSL + mySituation.BaroPressureAltitude)
		ti.AltIsGNSS = false
	} else if msg.Alt_std_m != 0 {
		// Fall back to received baro alt
		ti.Alt = int32(msg.Alt_std_m * 3.28084)
		ti.AltIsGNSS = false
	} else {
		// Fall back to GNSS alt
		ti.Alt = int32(alt)
		ti.AltIsGNSS = true
	}

	// Maybe the sender has baro AND GNS altitude.. in that case we can use that to estimage GnssBaroDiff to guess our own baro altitude
	// TODO: don't do that because of invalid baro alts from old OGN trackers.
	/*if msg.Alt_msl_m != 0 && msg.Alt_std_m != 0 {
		ti.Last_GnssDiffAlt = ti.Alt
		hae := msg.Alt_msl_m + mySituation.GPSGeoidSep
		ti.GnssDiffFromBaroAlt = int32((hae - msg.Alt_std_m) * 3.28084)
		ti.Last_GnssDiff = stratuxClock.Time
	} else if msg.Alt_hae_m != 0 && msg.Alt_std_m != 0 {
		ti.Last_GnssDiffAlt = ti.Alt
		ti.GnssDiffFromBaroAlt = int32((msg.Alt_hae_m - msg.Alt_std_m) * 3.28084)
		ti.Last_GnssDiff = stratuxClock.Time
	}*/

	ti.TurnRate = float32(msg.Turn_dps)
	if ti.TurnRate > 360 || ti.TurnRate < -360 {
		ti.TurnRate = 0
	}
	ti.Vvel = int16(msg.Climb_mps * 196.85)
	ti.Lat = msg.Lat_deg
	ti.Lng = msg.Lon_deg
	ti.Track = float32(msg.Track_deg)
	ti.Speed = uint16(msg.Speed_mps * 1.94384)
	ti.Speed_valid = true
	ti.SignalLevel = msg.SNR_dB

	if isGPSValid() {
		ti.Distance, ti.Bearing = distance(float64(mySituation.GPSLatitude), float64(mySituation.GPSLongitude), float64(ti.Lat), float64(ti.Lng))
		ti.BearingDist_valid = true
	}
	ti.Position_valid = true
	ti.ExtrapolatedPosition = false
	ti.Last_seen = stratuxClock.Time
	ageMs := int64(ti.Age * 1000)
	ti.Last_seen = ti.Last_seen.Add(-time.Duration(ageMs) * time.Millisecond)
	ti.Last_alt = ti.Last_seen
	ti.Last_speed = ti.Last_seen

	emitter, err := strconv.ParseInt(msg.Acft_cat, 16, 8)
	if len(msg.Acft_cat) == 2 && err == nil {
		ti.Emitter_category = uint8(emitter)
	} else {
		switch(msg.Acft_type) {
			case "1": ti.Emitter_category = 9 // glider = glider
			case "2", "5", "8": ti.Emitter_category = 1 // tow, drop, piston = light
			case "3": ti.Emitter_category = 7 // helicopter = helicopter
			case "4": ti.Emitter_category = 11 // skydiver
			case "6", "7": ti.Emitter_category = 12 // hang glider / paraglider
			case "9": ti.Emitter_category = 3 // jet = large
			case "B", "C": ti.Emitter_category = 10 // Balloon, airship = lighter than air
		}
	}

	traffic[key] = ti
	registerTrafficUpdate(ti)
	seenTraffic[key] = true

	if globalSettings.DEBUG {
		txt, _ := json.Marshal(ti)
		log.Printf("OGN traffic imported: " + string(txt))
	}
}

var ognTailNumberCache = make(map[string]string)
func lookupOgnTailNumber(ognid string) string {
	if len(ognTailNumberCache) == 0 {
		log.Printf("Parsing OGN device db")
		ddb, err := ioutil.ReadFile("/etc/ddb.json")
		if err != nil {
			log.Printf("Failed to read OGN device db")
			return ognid
		}
		var data map[string]interface{}
		err = json.Unmarshal(ddb, &data)
		if err != nil {
			log.Printf("Failed to parse OGN device db")
			return ognid
		}
		devlist := data["devices"].([]interface{})
		for i := 0; i < len(devlist); i++ {
			dev := devlist[i].(map[string]interface{})
			ognid := dev["device_id"].(string)
			tail := dev["registration"].(string)
			ognTailNumberCache[ognid] = tail
		}
		log.Printf("Successfully parsed OGN device db")
	}
	if tail, ok := ognTailNumberCache[ognid]; ok {
		return tail
	}
	return ""
}

func getTailNumber(ognid string, sys string) string {
	tail := lookupOgnTailNumber(ognid)
	if globalSettings.DisplayTrafficSource {
		if sys == "" {
			sys = "un"
		}
		sys = strings.ToLower(sys)[0:2]
		tail = sys + tail
	}
	return tail
}