/*
	Copyright (c) 2020 Adrian Batzill
	Distributable under the terms of The "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	ais.go: Routines for reading traffic from ais-rx-eu
*/

package main

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
//	"io/ioutil"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/b3nn0/stratux/common"
	ais "github.com/BertoldVdb/go-ais"
	"github.com/BertoldVdb/go-ais/aisnmea"
)


// {"sys":"OGN","addr":"395F39","addr_type":3,"acft_type":"1","lat_deg":51.7657533,"lon_deg":-1.1918533,"alt_msl_m":124,"alt_std_m":63,"track_deg":0.0,"speed_mps":0.3,"climb_mps":-0.5,"turn_dps":0.0,"DOP":1.5}
type AISMessage struct {
	Sys string
	Time int64
	Addr string
	Addr_type int32
	Acft_type string
	Acft_cat string
	Reg string
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
	Tx_enabled   bool
}

func aisPublishNmea(nmea string) {
	if globalStatus.AIS_connected {
		if !strings.HasSuffix(nmea, "\r\n") {
			nmea += "\r\n"
		}
		aisOutgoingMsgChan <- nmea
	}
}

var aisOutgoingMsgChan chan string = make(chan string, 100)
var aisIncomingMsgChan chan string = make(chan string, 100)
var aisExitChan chan bool = make(chan bool, 1)
var aisTailNumberCache = make(map[string]string)

func aisListen() {
	//go predTest()
	for {
		if !globalSettings.AIS_Enabled || AISDev == nil {
			// wait until AIS is enabled
			time.Sleep(1 * time.Second)
			continue
		}
		// log.Printf("ais connecting...")
		aisAddr := "127.0.0.1:10111"
		conn, err := net.Dial("tcp", aisAddr)
		if err != nil { // Local connection failed.
			time.Sleep(3 * time.Second)
			continue
		}
		log.Printf("ais successfully connected")
		aisReadWriter := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
		globalStatus.AIS_connected = true

		// Make sure the exit channel is empty, so we don't exit immediately
		for len(aisExitChan) > 0 {
			<- aisExitChan
		}


		go func() {
			scanner := bufio.NewScanner(aisReadWriter.Reader)
			for scanner.Scan() {
				aisIncomingMsgChan <- scanner.Text()
			}
			if scanner.Err() != nil {
				log.Printf("ais-rx-eu connection lost: " + scanner.Err().Error())
			}
			aisExitChan <- true
		}()

		nm := aisnmea.NMEACodecNew(ais.CodecNew(false, false))
		

		loop: for globalSettings.AIS_Enabled {
			select {
			case data := <- aisOutgoingMsgChan:
				//fmt.Printf(data)
				aisReadWriter.Write([]byte(data))
				aisReadWriter.Flush()
			case data := <- aisIncomingMsgChan:
				log.Printf("AIS: " + data)
				result, _ := nm.ParseSentence(data)
				out, _ := json.MarshalIndent(result, "", "  ")
				log.Printf("AIS: " + string(out))
//				var thisMsg msg
//				thisMsg.MessageClass = MSGCLASS_AIS
//				thisMsg.TimeReceived = stratuxClock.Time
//				thisMsg.Data = data
	
//				var msg AISMessage
//				err = json.Unmarshal([]byte(data), &msg)
//				if err != nil {
//					log.Printf("Invalid Data from AIS: " + data)
//					continue
//				}
	
//				if msg.Sys == "status" {
//					importAISStatusMessage(msg)
//				} else {
					// TODO: RVT Renable
					//msgLogAppend(thisMsg)
					// TODO: RVT Renable
					//logMsg(thisMsg) // writes to replay logs
//					importAISTrafficMessage(msg, data)
//				}
			case <- aisExitChan:
				break loop

			}
		}
		globalStatus.AIS_connected = false
		conn.Close()
		time.Sleep(3*time.Second)
	}
}

// Update something....
// func importAISStatusMessage(msg AISMessage) {
// 	if msg.Tx_enabled {
// 		aisPublishNmea(getOgnTrackerConfigString())
// 	}
// }

func importAISTrafficMessage(msg AISMessage, data string) {
	var ti TrafficInfo
	addressBytes, _ := hex.DecodeString(msg.Addr)
	addressBytes = append([]byte{0}, addressBytes...)
	address := binary.BigEndian.Uint32(addressBytes)

	// GDL90 only knows 2 address types. ICAO and non-ICAO, so we map to those.
	// for AIS: 1=ICAO. For us: 0=ICAO, 1="ADS-B with Self-assigned address"
	addrType := uint8(1) // Non-ICAO Address
	otherAddrType := uint8(0)
	if msg.Addr_type == 1 { // ICAO Address
		addrType = 0 
		otherAddrType = 1;
	}
	// Store in higher-order bytes in front of the 24 bit address so we can handle address collinsions gracefully.
	// For ICAO it will be 0, so traffic is merged. For others it will be 1, so traffic is kept seperately
	// Only issue: PAW and FANET don't know the concept of address types. So for those, we need to be more tolerant.
	// There are 2 cases here:
	// - If non-PAW/FNT is received first, we can simply merge PAW/FNT onto that
	// - If PAW/FNT is received first, we might need to update the traffic type later on
	// To make the code a bit simpler, we don't actually update the existing traffic in the second case, but just let it time out
	// and - from then on - only update the one with the correct AddrType

	key := uint32(addrType) << 24 | address
	otherKey := uint32(otherAddrType) << 24 | address

	trafficMutex.Lock()
	defer trafficMutex.Unlock()

	if msg.Sys == "PAW" || msg.Sys == "FNT" {
		// First, assume the AddrType guess is wrong and try to merge.. Only if that fails we use our guessed AddrType
		_, otherAddrTypeOk := traffic[otherKey]
		if otherAddrTypeOk {
			key = otherKey
			addrType = otherAddrType
		}
	}

	if existingTi, ok := traffic[key]; ok {
		ti = existingTi
		// ais-rx sends 2 types of messages.. normal ones with coords etc, and ones that only supply registration info. These usually don't have
		// coords, so we can't validate them easily. Therefore, this is handled before other validations and only if we already received the traffic earlier
		if len(msg.Reg) > 0 {
			ti.Tail = msg.Reg
			traffic[key] = ti
		}
		if msg.Time > 0 && !ti.Timestamp.IsZero() {
 			msgtime := time.Unix(msg.Time, 0)
			if ti.Position_valid && ti.Last_source == TRAFFIC_SOURCE_AIS && msgtime.Before(ti.Timestamp) {
				return // We already have a newer message for this target. This message was probably relayed by another tracker -- skip
			}
		}

	}

	// Sometimes there seems to be wildly invalid lat/lons, which can trip over distRect's normailization..
	if msg.Lat_deg > 360 || msg.Lat_deg < -360 || msg.Lon_deg > 360 || msg.Lon_deg < -360 {
		return
	}

	// Basic plausibility check:
	dist, _, _, _ := common.DistRect(float64(mySituation.GPSLatitude), float64(mySituation.GPSLongitude), float64(msg.Lat_deg), float64(msg.Lon_deg))
	if (isGPSValid() && dist >= 50000)  || (msg.Lat_deg == 0 && msg.Lon_deg == 0) {
		// more than 50km away? Ignore. Most likely invalid data
		return
	}

	ti.Icao_addr = address
	ti.Addr_type = addrType

	if len(ti.Tail) == 0 {
		ti.Tail = getTailNumber(msg.Addr, msg.Sys)
	}
	ti.Last_source = TRAFFIC_SOURCE_AIS
	if msg.Time > 0 {
		ti.Timestamp = time.Unix(msg.Time, 0)
	} else {
		ti.Timestamp = time.Now().UTC()
	}
	ti.Age = time.Now().UTC().Sub(ti.Timestamp).Seconds()
	if ti.Age > 30 || ti.Age < -2 {
		log.Printf("Discarding likely invalid AIS target: %s", data)
		return
	}

	// set altitude
	// To keep the rest of the system as simple as possible, we want to work with barometric altitude everywhere.
	// To do so, we use our own known geoid separation and pressure difference to compute the expected barometric altitude of the traffic.
	// Some AIS trackers are equiped with a baro sensor, but older firmwares send wrong data, so we usually can't rely on it.
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
	// TODO: don't do that because of invalid baro alts from old AIS trackers.
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
		ti.Distance, ti.Bearing = common.Distance(float64(mySituation.GPSLatitude), float64(mySituation.GPSLongitude), float64(ti.Lat), float64(ti.Lng))
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
		ti.Emitter_category = nmeaAircraftTypeToGdl90(msg.Acft_type)
	}

	traffic[key] = ti
	// postProcessTraffic(&ti) For AIS we do need to post process traffic, we always get positions
	registerTrafficUpdate(ti)
	seenTraffic[key] = true

	if globalSettings.DEBUG {
		txt, _ := json.Marshal(ti)
		log.Printf("AIS traffic imported: " + string(txt))
	}
}
