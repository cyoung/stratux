/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New"" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	gen_gdl90.go: Input demodulated UAT and 1090ES information, output GDL90. Heartbeat,
	 ownship, status messages, stats collection.
*/

package main

import (
	"bufio"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	humanize "github.com/dustin/go-humanize"

	"../uatparse"
)

// http://www.faa.gov/nextgen/programs/adsb/wsa/media/GDL90_Public_ICD_RevA.PDF

const (
	configLocation      = "/etc/stratux.conf"
	indexFilename       = "/var/log/stratux/LOGINDEX"
	managementAddr      = ":80"
	debugLog            = "/var/log/stratux.log"
	maxDatagramSize     = 8192
	maxUserMsgQueueSize = 25000 // About 10MB per port per connected client.
	logDirectory        = "/var/log/stratux"
	dataLogFile         = "/var/log/stratux.sqlite"

	UPLINK_BLOCK_DATA_BITS  = 576
	UPLINK_BLOCK_BITS       = (UPLINK_BLOCK_DATA_BITS + 160)
	UPLINK_BLOCK_DATA_BYTES = (UPLINK_BLOCK_DATA_BITS / 8)
	UPLINK_BLOCK_BYTES      = (UPLINK_BLOCK_BITS / 8)

	UPLINK_FRAME_BLOCKS     = 6
	UPLINK_FRAME_DATA_BITS  = (UPLINK_FRAME_BLOCKS * UPLINK_BLOCK_DATA_BITS)
	UPLINK_FRAME_BITS       = (UPLINK_FRAME_BLOCKS * UPLINK_BLOCK_BITS)
	UPLINK_FRAME_DATA_BYTES = (UPLINK_FRAME_DATA_BITS / 8)
	UPLINK_FRAME_BYTES      = (UPLINK_FRAME_BITS / 8)

	// assume 6 byte frames: 2 header bytes, 4 byte payload
	// (TIS-B heartbeat with one address, or empty FIS-B APDU)
	UPLINK_MAX_INFO_FRAMES = (424 / 6)

	MSGTYPE_UPLINK       = 0x07
	MSGTYPE_BASIC_REPORT = 0x1E
	MSGTYPE_LONG_REPORT  = 0x1F

	MSGCLASS_UAT      = 0
	MSGCLASS_ES       = 1
	MSGCLASS_GPS      = 3
	MSGCLASS_AHRS     = 4
	MSGCLASS_DUMP1090 = 5

	LON_LAT_RESOLUTION = float32(180.0 / 8388608.0)
	TRACK_RESOLUTION   = float32(360.0 / 256.0)
)

var maxSignalStrength int

var uatReplayLog string
var esReplayLog string
var gpsReplayLog string
var ahrsReplayLog string
var dump1090ReplayLog string

var stratuxBuild string
var stratuxVersion string

// CRC16 table generated to use to work with GDL90 messages.
var Crc16Table [256]uint16

// Current AHRS, pressure altitude, etc.
var mySituation SituationData

type WriteCloser interface {
	io.Writer
	io.Closer
}

type ReadCloser interface {
	io.Reader
	io.Closer
}

// File handles for replay logging.
var uatReplayWriter WriteCloser
var esReplayWriter WriteCloser
var gpsReplayWriter WriteCloser
var ahrsReplayWriter WriteCloser
var dump1090ReplayWriter WriteCloser

var developerMode bool

type msg struct {
	MessageClass     uint
	TimeReceived     time.Time
	Data             []byte
	Products         []uint32
	Signal_amplitude int
	Signal_strength  float64
	ADSBTowerID      string // Index in the 'ADSBTowers' map, if this is a parseable uplink message.
}

// Raw inputs.
var MsgLog []msg

// Time gen_gdl90 was started.
var timeStarted time.Time

type ADSBTower struct {
	Lat                         float64
	Lng                         float64
	Signal_strength_now         float64 // Current RSSI (dB)
	Signal_strength_max         float64 // all-time peak RSSI (dB) observed for this tower
	Energy_last_minute          uint64  // Summation of power observed for this tower across all messages last minute
	Signal_strength_last_minute float64 // Average RSSI (dB) observed for this tower last minute
	Messages_last_minute        uint64
	Messages_total              uint64
}

var ADSBTowers map[string]ADSBTower // Running list of all towers seen. (lat,lng) -> ADSBTower

func constructFilenames() {
	var fileIndexNumber uint

	// First, create the log file directory if it does not exist
	os.Mkdir(logDirectory, 0755)

	f, err := os.Open(indexFilename)
	if err != nil {
		log.Printf("Unable to open index file %s using index of 0\n", indexFilename)
		fileIndexNumber = 0
	} else {
		_, err := fmt.Fscanf(f, "%d\n", &fileIndexNumber)
		if err != nil {
			log.Printf("Unable to read index file %s using index of 0\n", indexFilename)
		}
		f.Close()
		fileIndexNumber++
	}
	fo, err := os.Create(indexFilename)
	if err != nil {
		log.Printf("Error creating index file %s\n", indexFilename)
	}
	_, err2 := fmt.Fprintf(fo, "%d\n", fileIndexNumber)
	if err2 != nil {
		log.Printf("Error writing to index file %s\n", indexFilename)
	}
	fo.Sync()
	fo.Close()
	if developerMode == true {
		uatReplayLog = fmt.Sprintf("%s/%04d-uat.log", logDirectory, fileIndexNumber)
		esReplayLog = fmt.Sprintf("%s/%04d-es.log", logDirectory, fileIndexNumber)
		gpsReplayLog = fmt.Sprintf("%s/%04d-gps.log", logDirectory, fileIndexNumber)
		ahrsReplayLog = fmt.Sprintf("%s/%04d-ahrs.log", logDirectory, fileIndexNumber)
		dump1090ReplayLog = fmt.Sprintf("%s/%04d-dump1090.log", logDirectory, fileIndexNumber)
	} else {
		uatReplayLog = fmt.Sprintf("%s/%04d-uat.log.gz", logDirectory, fileIndexNumber)
		esReplayLog = fmt.Sprintf("%s/%04d-es.log.gz", logDirectory, fileIndexNumber)
		gpsReplayLog = fmt.Sprintf("%s/%04d-gps.log.gz", logDirectory, fileIndexNumber)
		ahrsReplayLog = fmt.Sprintf("%s/%04d-ahrs.log.gz", logDirectory, fileIndexNumber)
		dump1090ReplayLog = fmt.Sprintf("%s/%04d-dump1090.log.gz", logDirectory, fileIndexNumber)
	}
}

// Construct the CRC table. Adapted from FAA ref above.
func crcInit() {
	var i uint16
	var bitctr uint16
	var crc uint16
	for i = 0; i < 256; i++ {
		crc = (i << 8)
		for bitctr = 0; bitctr < 8; bitctr++ {
			z := uint16(0)
			if (crc & 0x8000) != 0 {
				z = 0x1021
			}
			crc = (crc << 1) ^ z
		}
		Crc16Table[i] = crc
	}
}

// Compute CRC. Adapted from FAA ref above.
func crcCompute(data []byte) uint16 {
	ret := uint16(0)
	for i := 0; i < len(data); i++ {
		ret = Crc16Table[ret>>8] ^ (ret << 8) ^ uint16(data[i])
	}
	return ret
}

func prepareMessage(data []byte) []byte {
	// Compute CRC before modifying the message.
	crc := crcCompute(data)
	// Add the two CRC16 bytes before replacing control characters.
	data = append(data, byte(crc&0xFF))
	data = append(data, byte(crc>>8))

	tmp := []byte{0x7E} // Flag start.

	// Copy the message over, escaping 0x7E (Flag Byte) and 0x7D (Control-Escape).
	for i := 0; i < len(data); i++ {
		mv := data[i]
		if (mv == 0x7E) || (mv == 0x7D) {
			mv = mv ^ 0x20
			tmp = append(tmp, 0x7D)
		}
		tmp = append(tmp, mv)
	}

	tmp = append(tmp, 0x7E) // Flag end.

	return tmp
}

func makeLatLng(v float32) []byte {
	ret := make([]byte, 3)

	v = v / LON_LAT_RESOLUTION
	wk := int32(v)

	ret[0] = byte((wk & 0xFF0000) >> 16)
	ret[1] = byte((wk & 0x00FF00) >> 8)
	ret[2] = byte((wk & 0x0000FF))

	return ret
}

func makeOwnshipReport() bool {
	if !isGPSValid() {
		return false
	}
	msg := make([]byte, 28)
	// See p.16.
	msg[0] = 0x0A // Message type "Ownship".

	msg[1] = 0x01 // Alert status, address type.

	code, _ := hex.DecodeString(globalSettings.OwnshipModeS)
	if len(code) != 3 {
		// Reserved dummy code.
		msg[2] = 0xF0
		msg[3] = 0x00
		msg[4] = 0x00
	} else {
		msg[2] = code[0] // Mode S address.
		msg[3] = code[1] // Mode S address.
		msg[4] = code[2] // Mode S address.
	}

	tmp := makeLatLng(mySituation.Lat)
	msg[5] = tmp[0] // Latitude.
	msg[6] = tmp[1] // Latitude.
	msg[7] = tmp[2] // Latitude.

	tmp = makeLatLng(mySituation.Lng)
	msg[8] = tmp[0]  // Longitude.
	msg[9] = tmp[1]  // Longitude.
	msg[10] = tmp[2] // Longitude.

	// This is **PRESSURE ALTITUDE**
	//FIXME: Temporarily removing "invalid altitude" when pressure altitude not available - using GPS altitude instead.
	//	alt := uint16(0xFFF) // 0xFFF "invalid altitude."

	var alt uint16
	var altf float64

	if isTempPressValid() {
		altf = float64(mySituation.Pressure_alt)
	} else {
		altf = float64(mySituation.Alt) //FIXME: Pass GPS altitude if PA not available. **WORKAROUND FOR FF**
	}
	altf = (altf + 1000) / 25

	alt = uint16(altf) & 0xFFF // Should fit in 12 bits.

	msg[11] = byte((alt & 0xFF0) >> 4) // Altitude.
	msg[12] = byte((alt & 0x00F) << 4)
	if isGPSGroundTrackValid() {
		msg[12] = msg[12] | 0x0B // "Airborne" + "True Heading"
	}

	msg[13] = byte(0x80 | (mySituation.NACp & 0x0F)) //Set NIC = 8 and use NACp from ry835ai.go.

	gdSpeed := uint16(0) // 1kt resolution.
	if isGPSGroundTrackValid() {
		gdSpeed = mySituation.GroundSpeed
	}

	// gdSpeed should fit in 12 bits.
	msg[14] = byte((gdSpeed & 0xFF0) >> 4)
	msg[15] = byte((gdSpeed & 0x00F) << 4)

	verticalVelocity := int16(0x800) // ft/min. 64 ft/min resolution.
	//TODO: 0x800 = no information available.
	// verticalVelocity should fit in 12 bits.
	msg[15] = msg[15] | byte((verticalVelocity&0x0F00)>>8)
	msg[16] = byte(verticalVelocity & 0xFF)

	// Showing magnetic (corrected) on ForeFlight. Needs to be True Heading.
	groundTrack := uint16(0)
	if isGPSGroundTrackValid() {
		groundTrack = mySituation.TrueCourse
	}
	trk := uint8(float32(groundTrack) / TRACK_RESOLUTION) // Resolution is ~1.4 degrees.

	msg[17] = byte(trk)

	msg[18] = 0x01 // "Light (ICAO) < 15,500 lbs"

	// Create callsign "Stratux".
	msg[19] = 0x53
	msg[20] = 0x74
	msg[21] = 0x72
	msg[22] = 0x61
	msg[23] = 0x74
	msg[24] = 0x75
	msg[25] = 0x78

	sendGDL90(prepareMessage(msg), false)
	return true
}

func makeOwnshipGeometricAltitudeReport() bool {
	if !isGPSValid() {
		return false
	}
	msg := make([]byte, 5)
	// See p.28.
	msg[0] = 0x0B                 // Message type "Ownship Geo Alt".
	alt := int16(mySituation.Alt) // GPS Altitude.
	alt = alt / 5
	msg[1] = byte(alt >> 8)     // Altitude.
	msg[2] = byte(alt & 0x00FF) // Altitude.

	//TODO: "Figure of Merit". 0x7FFF "Not available".
	msg[3] = 0x00
	msg[4] = 0x0A

	sendGDL90(prepareMessage(msg), false)
	return true
}

/*

	"SX" Stratux GDL90 message.
	http://hiltonsoftware.com/stratux/ for latest version (currently using V104)

*/

func makeStratuxStatus() []byte {
	msg := make([]byte, 29)
	msg[0] = 'S'
	msg[1] = 'X'
	msg[2] = 1

	msg[3] = 1 // "message version".

	// Version code. Messy parsing to fit into four bytes.
	thisVers := stratuxVersion[1:]                       // Skip first character, should be 'v'.
	m_str := thisVers[0:strings.Index(thisVers, ".")]    // Major version.
	mib_str := thisVers[strings.Index(thisVers, ".")+1:] // Minor and build version.

	tp := 0 // Build "type".
	mi_str := ""
	b_str := ""
	if strings.Index(mib_str, "rc") != -1 {
		tp = 3
		mi_str = mib_str[0:strings.Index(mib_str, "rc")]
		b_str = mib_str[strings.Index(mib_str, "rc")+2:]
	} else if strings.Index(mib_str, "r") != -1 {
		tp = 2
		mi_str = mib_str[0:strings.Index(mib_str, "r")]
		b_str = mib_str[strings.Index(mib_str, "r")+1:]
	} else if strings.Index(mib_str, "b") != -1 {
		tp = 1
		mi_str = mib_str[0:strings.Index(mib_str, "b")]
		b_str = mib_str[strings.Index(mib_str, "b")+1:]
	}

	// Convert to strings.
	m, _ := strconv.Atoi(m_str)
	mi, _ := strconv.Atoi(mi_str)
	b, _ := strconv.Atoi(b_str)

	msg[4] = byte(m)
	msg[5] = byte(mi)
	msg[6] = byte(tp)
	msg[7] = byte(b)

	//TODO: Hardware revision.
	msg[8] = 0xFF
	msg[9] = 0xFF
	msg[10] = 0xFF
	msg[11] = 0xFF

	// Valid and enabled flags.
	// Valid/Enabled: GPS portion.
	if isGPSValid() {
		switch mySituation.Quality {
		case 1: // 1 = 3D GPS.
			msg[13] = 1
		case 2: // 2 = DGPS (SBAS /WAAS).
			msg[13] = 2
		default: // Zero.
		}
	}

	// Valid/Enabled: AHRS portion.
	if isAHRSValid() {
		msg[13] = msg[13] | (1 << 2)
	}

	// Valid/Enabled: Pressure altitude portion.
	if isTempPressValid() {
		msg[13] = msg[13] | (1 << 3)
	}

	// Valid/Enabled: CPU temperature portion.
	if isCPUTempValid() {
		msg[13] = msg[13] | (1 << 4)
	}

	// Valid/Enabled: UAT portion.
	if globalSettings.UAT_Enabled {
		msg[13] = msg[13] | (1 << 5)
	}

	// Valid/Enabled: ES portion.
	if globalSettings.ES_Enabled {
		msg[13] = msg[13] | (1 << 6)
	}

	// Valid/Enabled: GPS Enabled portion.
	if globalSettings.GPS_Enabled {
		msg[13] = msg[13] | (1 << 7)
	}

	// Valid/Enabled: AHRS Enabled portion.
	if globalSettings.AHRS_Enabled {
		msg[12] = 1 << 0
	}

	// Valid/Enabled: last bit unused.

	// Connected hardware: number of radios.
	msg[15] = msg[15] | (byte(globalStatus.Devices) & 0x3)
	// Connected hardware: RY835AI.
	if globalStatus.RY835AI_connected {
		msg[15] = msg[15] | (1 << 2)
	}

	// Number of GPS satellites locked.
	msg[16] = byte(globalStatus.GPS_satellites_locked)

	// Number of satellites tracked
	msg[17] = byte(globalStatus.GPS_satellites_tracked)

	// Summarize number of UAT and 1090ES traffic targets for reports that follow.
	var uat_traffic_targets uint16
	var es_traffic_targets uint16
	for _, traf := range traffic {
		switch traf.Last_source {
		case TRAFFIC_SOURCE_1090ES:
			es_traffic_targets++
		case TRAFFIC_SOURCE_UAT:
			uat_traffic_targets++
		}
	}

	// Number of UAT traffic targets.
	msg[18] = byte((uat_traffic_targets & 0xFF00) >> 8)
	msg[19] = byte(uat_traffic_targets & 0xFF)
	// Number of 1090ES traffic targets.
	msg[20] = byte((es_traffic_targets & 0xFF00) >> 8)
	msg[21] = byte(es_traffic_targets & 0xFF)

	// Number of UAT messages per minute.
	msg[22] = byte((globalStatus.UAT_messages_last_minute & 0xFF00) >> 8)
	msg[23] = byte(globalStatus.UAT_messages_last_minute & 0xFF)
	// Number of 1090ES messages per minute.
	msg[24] = byte((globalStatus.ES_messages_last_minute & 0xFF00) >> 8)
	msg[25] = byte(globalStatus.ES_messages_last_minute & 0xFF)

	// CPU temperature.
	v := uint16(float32(10.0) * globalStatus.CPUTemp)

	msg[26] = byte((v & 0xFF00) >> 8)
	msg[27] = byte(v & 0xFF)

	// Number of ADS-B towers.
	num_towers := uint8(len(ADSBTowers))

	msg[28] = byte(num_towers)

	// List of ADS-B towers (lat, lng).
	for _, tower := range ADSBTowers {
		tmp := makeLatLng(float32(tower.Lat))
		msg = append(msg, tmp[0]) // Latitude.
		msg = append(msg, tmp[1]) // Latitude.
		msg = append(msg, tmp[2]) // Latitude.

		tmp = makeLatLng(float32(tower.Lng))
		msg = append(msg, tmp[0]) // Longitude.
		msg = append(msg, tmp[1]) // Longitude.
		msg = append(msg, tmp[2]) // Longitude.
	}

	return prepareMessage(msg)
}

/*

	"Stratux" GDL90 message.

	Message ID 0xCC.
	Byte1: p p p p p p GPS AHRS
	First 6 bytes are protocol version codes.
	Protocol 1: GPS on/off | AHRS on/off.
*/

func makeStratuxHeartbeat() []byte {
	msg := make([]byte, 2)
	msg[0] = 0xCC // Message type "Stratux".
	msg[1] = 0
	if isGPSValid() {
		msg[1] = 0x02
	}
	if isAHRSValid() {
		msg[1] = msg[1] | 0x01
	}

	protocolVers := int8(1)
	msg[1] = msg[1] | byte(protocolVers<<2)

	return prepareMessage(msg)
}

func makeHeartbeat() []byte {
	msg := make([]byte, 7)
	// See p.10.
	msg[0] = 0x00 // Message type "Heartbeat".
	msg[1] = 0x01 // "UAT Initialized".
	if isGPSValid() {
		msg[1] = msg[1] | 0x80
	}
	msg[1] = msg[1] | 0x10 //FIXME: Addr talkback.

	nowUTC := time.Now().UTC()
	// Seconds since 0000Z.
	midnightUTC := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC)
	secondsSinceMidnightUTC := uint32(nowUTC.Sub(midnightUTC).Seconds())

	msg[2] = byte(((secondsSinceMidnightUTC >> 16) << 7) | 0x1) // UTC OK.
	msg[3] = byte((secondsSinceMidnightUTC & 0xFF))
	msg[4] = byte((secondsSinceMidnightUTC & 0xFFFF) >> 8)

	// TODO. Number of uplink messages. See p.12.
	// msg[5]
	// msg[6]

	return prepareMessage(msg)
}

func relayMessage(msgtype uint16, msg []byte) {
	ret := make([]byte, len(msg)+4)
	// See p.15.
	ret[0] = byte(msgtype) // Uplink message ID.
	ret[1] = 0x00          //TODO: Time.
	ret[2] = 0x00          //TODO: Time.
	ret[3] = 0x00          //TODO: Time.

	for i := 0; i < len(msg); i++ {
		ret[i+4] = msg[i]
	}

	sendGDL90(prepareMessage(ret), true)
}

func heartBeatSender() {
	timer := time.NewTicker(1 * time.Second)
	timerMessageStats := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-timer.C:
			sendGDL90(makeHeartbeat(), false)
			sendGDL90(makeStratuxHeartbeat(), false)
			sendGDL90(makeStratuxStatus(), false)
			makeOwnshipReport()
			makeOwnshipGeometricAltitudeReport()

			// --- debug code: traffic demo ---
			// Uncomment and compile to display large number of artificial traffic targets
			/*
				numTargets := uint32(36)
				hexCode := uint32(0xFF0000)

				for i := uint32(0); i < numTargets; i++ {
					tail := fmt.Sprintf("DEMO%d", i)
					alt := float32((i*117%2000)*25 + 2000)
					hdg := int32((i * 149) % 360)
					spd := float64(50 + ((i*23)%13)*37)

					updateDemoTraffic(i|hexCode, tail, alt, spd, hdg)

				}
			*/

			// ---end traffic demo code ---
			sendTrafficUpdates()
			updateStatus()
		case <-timerMessageStats.C:
			// Save a bit of CPU by not pruning the message log every 1 second.
			updateMessageStats()
		}
	}
}

func updateMessageStats() {
	t := make([]msg, 0)
	m := len(MsgLog)
	UAT_messages_last_minute := uint(0)
	ES_messages_last_minute := uint(0)
	products_last_minute := make(map[string]uint32)

	// Clear out ADSBTowers stats.
	for t, tinf := range ADSBTowers {
		tinf.Messages_last_minute = 0
		tinf.Energy_last_minute = 0
		ADSBTowers[t] = tinf
	}

	for i := 0; i < m; i++ {
		if stratuxClock.Since(MsgLog[i].TimeReceived) < 1*time.Minute {
			t = append(t, MsgLog[i])
			if MsgLog[i].MessageClass == MSGCLASS_UAT {
				UAT_messages_last_minute++
				for _, p := range MsgLog[i].Products {
					products_last_minute[getProductNameFromId(int(p))]++
				}
				if len(MsgLog[i].ADSBTowerID) > 0 { // Update tower stats.
					tid := MsgLog[i].ADSBTowerID
					twr := ADSBTowers[tid]
					twr.Energy_last_minute += uint64((MsgLog[i].Signal_amplitude) * (MsgLog[i].Signal_amplitude))
					twr.Messages_last_minute++
					if MsgLog[i].Signal_strength > twr.Signal_strength_max { // Update alltime max signal strength.
						twr.Signal_strength_max = MsgLog[i].Signal_strength
					}
					ADSBTowers[tid] = twr
				}
			} else if MsgLog[i].MessageClass == MSGCLASS_ES {
				ES_messages_last_minute++
			}
		}
	}
	MsgLog = t
	globalStatus.UAT_messages_last_minute = UAT_messages_last_minute
	globalStatus.ES_messages_last_minute = ES_messages_last_minute
	globalStatus.UAT_products_last_minute = products_last_minute

	// Update "max messages/min" counters.
	if globalStatus.UAT_messages_max < UAT_messages_last_minute {
		globalStatus.UAT_messages_max = UAT_messages_last_minute
	}
	if globalStatus.ES_messages_max < ES_messages_last_minute {
		globalStatus.ES_messages_max = ES_messages_last_minute
	}

	// Update average signal strength over last minute for all ADSB towers.
	for t, tinf := range ADSBTowers {
		if tinf.Messages_last_minute == 0 {
			tinf.Signal_strength_last_minute = -99
		} else {
			tinf.Signal_strength_last_minute = 10 * (math.Log10(float64((tinf.Energy_last_minute / tinf.Messages_last_minute))) - 6)
		}
		ADSBTowers[t] = tinf
	}

}

// Check if CPU temperature is valid. Assume <= 0 is invalid.
func isCPUTempValid() bool {
	return globalStatus.CPUTemp > 0
}

/*
	cpuTempMonitor() reads the RPi board temperature every second and updates it in globalStatus.
	This is broken out into its own function (run as its own goroutine) because the RPi temperature
	 monitor code is buggy, and often times reading this file hangs quite some time.
*/
func cpuTempMonitor() {
	timer := time.NewTicker(1 * time.Second)
	for {
		<-timer.C

		// Update CPUTemp.
		globalStatus.CPUTemp = float32(-99.0) // Default value - in case code below hangs.

		temp, err := ioutil.ReadFile("/sys/class/thermal/thermal_zone0/temp")
		tempStr := strings.Trim(string(temp), "\n")
		if err == nil {
			tInt, err := strconv.Atoi(tempStr)
			if err == nil {
				if tInt > 1000 {
					globalStatus.CPUTemp = float32(tInt) / float32(1000.0)
				} else {
					globalStatus.CPUTemp = float32(tInt) // case where Temp is returned as simple integer
				}
			}
		}

	}
}

func updateStatus() {
	if mySituation.Quality == 2 {
		globalStatus.GPS_solution = "DGPS (SBAS / WAAS)"
	} else if mySituation.Quality == 1 {
		globalStatus.GPS_solution = "3D GPS"
	} else if mySituation.Quality == 6 {
		globalStatus.GPS_solution = "Dead Reckoning"
	} else if mySituation.Quality == 0 {
		globalStatus.GPS_solution = "No Fix"
	} else {
		globalStatus.GPS_solution = "Unknown"
	}

	if !(globalStatus.GPS_connected) || !(isGPSConnected()) { // isGPSConnected looks for valid NMEA messages. GPS_connected is set by gpsSerialReader and will immediately fail on disconnected USB devices, or in a few seconds after "blocked" comms on ttyAMA0.
		mySituation.Satellites = 0
		mySituation.SatellitesSeen = 0
		mySituation.SatellitesTracked = 0
		mySituation.Quality = 0
		globalStatus.GPS_solution = "Disconnected"
		globalStatus.GPS_connected = false
	}

	globalStatus.GPS_satellites_locked = mySituation.Satellites
	globalStatus.GPS_satellites_seen = mySituation.SatellitesSeen
	globalStatus.GPS_satellites_tracked = mySituation.SatellitesTracked

	// Update Uptime value
	globalStatus.Uptime = int64(stratuxClock.Milliseconds)
	globalStatus.UptimeClock = stratuxClock.Time
	globalStatus.Clock = time.Now()
}

type ReplayWriter struct {
	fp *os.File
}

func (r ReplayWriter) Write(p []byte) (n int, err error) {
	return r.fp.Write(p)
}

func (r ReplayWriter) Close() error {
	return r.fp.Close()
}

func makeReplayLogEntry(msg string) string {
	return fmt.Sprintf("%d,%s\n", time.Since(timeStarted).Nanoseconds(), msg)
}

func replayLog(msg string, msgclass int) {
	if !globalSettings.ReplayLog { // Logging disabled.
		return
	}
	msg = strings.Trim(msg, " \r\n")
	if len(msg) == 0 { // Blank message.
		return
	}
	var fp WriteCloser

	switch msgclass {
	case MSGCLASS_UAT:
		fp = uatReplayWriter
	case MSGCLASS_ES:
		fp = esReplayWriter
	case MSGCLASS_GPS:
		fp = gpsReplayWriter
	case MSGCLASS_AHRS:
		fp = ahrsReplayWriter
	case MSGCLASS_DUMP1090:
		fp = dump1090ReplayWriter
	}

	if fp != nil {
		s := makeReplayLogEntry(msg)
		fp.Write([]byte(s))
	}
}

type WeatherMessage struct {
	Type              string
	Location          string
	Time              string
	Data              string
	LocaltimeReceived time.Time
}

// Send update to connected websockets.
func registerADSBTextMessageReceived(msg string) {
	x := strings.Split(msg, " ")
	if len(x) < 5 {
		return
	}

	var wm WeatherMessage

	wm.Type = x[0]
	wm.Location = x[1]
	wm.Time = x[2]
	wm.Data = strings.Join(x[3:], " ")
	wm.LocaltimeReceived = stratuxClock.Time

	wmJSON, _ := json.Marshal(&wm)

	// Send to weatherUpdate channel for any connected clients.
	weatherUpdate.Send(wmJSON)
}

func parseInput(buf string) ([]byte, uint16) {
	replayLog(buf, MSGCLASS_UAT) // Log the raw message.

	x := strings.Split(buf, ";") // Discard everything after the first ';'.
	s := x[0]
	if len(s) == 0 {
		return nil, 0
	}
	msgtype := uint16(0)
	isUplink := false

	if s[0] == '+' {
		isUplink = true
	}

	var thisSignalStrength int

	if /*isUplink &&*/ len(x) >= 3 {
		// See if we can parse out the signal strength.
		ss := x[2]
		//log.Printf("x[2] = %s\n",ss)
		if strings.HasPrefix(ss, "ss=") {
			ssStr := ss[3:]
			if ssInt, err := strconv.Atoi(ssStr); err == nil {
				thisSignalStrength = ssInt
				if isUplink && (ssInt > maxSignalStrength) { // only look at uplinks; ignore ADS-B and TIS-B/ADS-R messages
					maxSignalStrength = ssInt
				}
			} else {
				//log.Printf("Error was %s\n",err.Error())
			}
		}
	}

	if s[0] == '-' {
		parseDownlinkReport(s, int(thisSignalStrength))
	}

	s = s[1:]
	msglen := len(s) / 2

	if len(s)%2 != 0 { // Bad format.
		return nil, 0
	}

	if isUplink && msglen == UPLINK_FRAME_DATA_BYTES {
		msgtype = MSGTYPE_UPLINK
	} else if msglen == 34 {
		msgtype = MSGTYPE_LONG_REPORT
	} else if msglen == 18 {
		msgtype = MSGTYPE_BASIC_REPORT
	} else {
		msgtype = 0
	}

	if msgtype == 0 {
		log.Printf("UNKNOWN MESSAGE TYPE: %s - msglen=%d\n", s, msglen)
	}

	// Now, begin converting the string into a byte array.
	frame := make([]byte, UPLINK_FRAME_DATA_BYTES)
	hex.Decode(frame, []byte(s))

	var thisMsg msg
	thisMsg.MessageClass = MSGCLASS_UAT
	thisMsg.TimeReceived = stratuxClock.Time
	thisMsg.Data = frame
	thisMsg.Signal_amplitude = thisSignalStrength
	thisMsg.Signal_strength = 20 * math.Log10((float64(thisSignalStrength))/1000)
	thisMsg.Products = make([]uint32, 0)
	if msgtype == MSGTYPE_UPLINK {
		// Parse the UAT message.
		uatMsg, err := uatparse.New(buf)
		if err == nil {
			uatMsg.DecodeUplink()
			towerid := fmt.Sprintf("(%f,%f)", uatMsg.Lat, uatMsg.Lon)
			thisMsg.ADSBTowerID = towerid
			if _, ok := ADSBTowers[towerid]; !ok { // First time we've seen the tower. Start tracking.
				var newTower ADSBTower
				newTower.Lat = uatMsg.Lat
				newTower.Lng = uatMsg.Lon
				newTower.Signal_strength_now = thisMsg.Signal_strength
				newTower.Signal_strength_max = -999 // dBmax = 0, so this needs to initialize below scale ( << -48 dB)
				ADSBTowers[towerid] = newTower
			}
			twr := ADSBTowers[towerid]
			twr.Messages_total++
			twr.Signal_strength_now = thisMsg.Signal_strength
			ADSBTowers[towerid] = twr
			// Get all of the "product ids".
			for _, f := range uatMsg.Frames {
				thisMsg.Products = append(thisMsg.Products, f.Product_id)
			}
			// Get all of the text reports.
			textReports, _ := uatMsg.GetTextReports()
			for _, r := range textReports {
				registerADSBTextMessageReceived(r)
			}
		}
	}

	MsgLog = append(MsgLog, thisMsg)

	return frame, msgtype
}

var product_name_map = map[int]string{
	0:   "METAR",
	1:   "TAF",
	2:   "SIGMET",
	3:   "Conv SIGMET",
	4:   "AIRMET",
	5:   "PIREP",
	6:   "Severe Wx",
	7:   "Winds Aloft",
	8:   "NOTAM",           //"NOTAM (Including TFRs) and Service Status";
	9:   "D-ATIS",          //"Aerodrome and Airspace – D-ATIS";
	10:  "Terminal Wx",     //"Aerodrome and Airspace - TWIP";
	11:  "AIRMET",          //"Aerodrome and Airspace - AIRMET";
	12:  "SIGMET",          //"Aerodrome and Airspace - SIGMET/Convective SIGMET";
	13:  "SUA",             //"Aerodrome and Airspace - SUA Status";
	20:  "METAR",           //"METAR and SPECI";
	21:  "TAF",             //"TAF and Amended TAF";
	22:  "SIGMET",          //"SIGMET";
	23:  "Conv SIGMET",     //"Convective SIGMET";
	24:  "AIRMET",          //"AIRMET";
	25:  "PIREP",           //"PIREP";
	26:  "Severe Wx",       //"AWW";
	27:  "Winds Aloft",     //"Winds and Temperatures Aloft";
	51:  "NEXRAD",          //"National NEXRAD, Type 0 - 4 level";
	52:  "NEXRAD",          //"National NEXRAD, Type 1 - 8 level (quasi 6-level VIP)";
	53:  "NEXRAD",          //"National NEXRAD, Type 2 - 8 level";
	54:  "NEXRAD",          //"National NEXRAD, Type 3 - 16 level";
	55:  "NEXRAD",          //"Regional NEXRAD, Type 0 - low dynamic range";
	56:  "NEXRAD",          //"Regional NEXRAD, Type 1 - 8 level (quasi 6-level VIP)";
	57:  "NEXRAD",          //"Regional NEXRAD, Type 2 - 8 level";
	58:  "NEXRAD",          //"Regional NEXRAD, Type 3 - 16 level";
	59:  "NEXRAD",          //"Individual NEXRAD, Type 0 - low dynamic range";
	60:  "NEXRAD",          //"Individual NEXRAD, Type 1 - 8 level (quasi 6-level VIP)";
	61:  "NEXRAD",          //"Individual NEXRAD, Type 2 - 8 level";
	62:  "NEXRAD",          //"Individual NEXRAD, Type 3 - 16 level";
	63:  "NEXRAD Regional", //"Global Block Representation - Regional NEXRAD, Type 4 – 8 level";
	64:  "NEXRAD CONUS",    //"Global Block Representation - CONUS NEXRAD, Type 4 - 8 level";
	81:  "Tops",            //"Radar echo tops graphic, scheme 1: 16-level";
	82:  "Tops",            //"Radar echo tops graphic, scheme 2: 8-level";
	83:  "Tops",            //"Storm tops and velocity";
	101: "Lightning",       //"Lightning strike type 1 (pixel level)";
	102: "Lightning",       //"Lightning strike type 2 (grid element level)";
	151: "Lightning",       //"Point phenomena, vector format";
	201: "Surface",         //"Surface conditions/winter precipitation graphic";
	202: "Surface",         //"Surface weather systems";
	254: "G-AIRMET",        //"AIRMET, SIGMET: Bitmap encoding";
	351: "Time",            //"System Time";
	352: "Status",          //"Operational Status";
	353: "Status",          //"Ground Station Status";
	401: "Imagery",         //"Generic Raster Scan Data Product APDU Payload Format Type 1";
	402: "Text",
	403: "Vector Imagery", //"Generic Vector Data Product APDU Payload Format Type 1";
	404: "Symbols",
	405: "Text",
	411: "Text",    //"Generic Textual Data Product APDU Payload Format Type 1";
	412: "Symbols", //"Generic Symbolic Product APDU Payload Format Type 1";
	413: "Text",    //"Generic Textual Data Product APDU Payload Format Type 2";
}

func getProductNameFromId(product_id int) string {
	name, present := product_name_map[product_id]
	if present {
		return name
	}

	if product_id == 600 || (product_id >= 2000 && product_id <= 2005) {
		return "Custom/Test"
	}

	return fmt.Sprintf("Unknown (%d)", product_id)
}

type settings struct {
	UAT_Enabled    bool
	ES_Enabled     bool
	GPS_Enabled    bool
	NetworkOutputs []networkConnection
	AHRS_Enabled   bool
	DEBUG          bool
	ReplayLog      bool
	PPM            int
	OwnshipModeS   string
	WatchList      string
}

type status struct {
	Version                                    string
	Build                                      string
	HardwareBuild                              string
	Devices                                    uint32
	Connected_Users                            uint
	UAT_messages_last_minute                   uint
	UAT_products_last_minute                   map[string]uint32
	UAT_messages_max                           uint
	ES_messages_last_minute                    uint
	ES_messages_max                            uint
	GPS_satellites_locked                      uint16
	GPS_satellites_seen                        uint16
	GPS_satellites_tracked                     uint16
	GPS_connected                              bool
	GPS_solution                               string
	RY835AI_connected                          bool
	Uptime                                     int64
	Clock                                      time.Time
	UptimeClock                                time.Time
	CPUTemp                                    float32
	NetworkDataMessagesSent                    uint64
	NetworkDataMessagesSentNonqueueable        uint64
	NetworkDataBytesSent                       uint64
	NetworkDataBytesSentNonqueueable           uint64
	NetworkDataMessagesSentLastSec             uint64
	NetworkDataMessagesSentNonqueueableLastSec uint64
	NetworkDataBytesSentLastSec                uint64
	NetworkDataBytesSentNonqueueableLastSec    uint64
	Errors                                     []string
}

var globalSettings settings
var globalStatus status

func defaultSettings() {
	globalSettings.UAT_Enabled = true
	globalSettings.ES_Enabled = true
	globalSettings.GPS_Enabled = true
	//FIXME: Need to change format below.
	globalSettings.NetworkOutputs = []networkConnection{
		{Conn: nil, Ip: "", Port: 4000, Capability: NETWORK_GDL90_STANDARD | NETWORK_AHRS_GDL90},
		//		{Conn: nil, Ip: "", Port: 49002, Capability: NETWORK_AHRS_FFSIM},
	}
	globalSettings.AHRS_Enabled = false
	globalSettings.DEBUG = false
	globalSettings.ReplayLog = false //TODO: 'true' for debug builds.
	globalSettings.OwnshipModeS = "F00000"
}

func readSettings() {
	fd, err := os.Open(configLocation)
	if err != nil {
		log.Printf("can't read settings %s: %s\n", configLocation, err.Error())
		defaultSettings()
		return
	}
	defer fd.Close()
	buf := make([]byte, 1024)
	count, err := fd.Read(buf)
	if err != nil {
		log.Printf("can't read settings %s: %s\n", configLocation, err.Error())
		defaultSettings()
		return
	}
	var newSettings settings
	err = json.Unmarshal(buf[0:count], &newSettings)
	if err != nil {
		log.Printf("can't read settings %s: %s\n", configLocation, err.Error())
		defaultSettings()
		return
	}
	globalSettings = newSettings
	log.Printf("read in settings.\n")
}

func addSystemError(err error) {
	globalStatus.Errors = append(globalStatus.Errors, err.Error())
}

func saveSettings() {
	fd, err := os.OpenFile(configLocation, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0644))
	if err != nil {
		err_ret := fmt.Errorf("can't save settings %s: %s", configLocation, err.Error())
		addSystemError(err_ret)
		log.Printf("%s\n", err_ret.Error())
		return
	}
	defer fd.Close()
	jsonSettings, _ := json.Marshal(&globalSettings)
	fd.Write(jsonSettings)
	log.Printf("wrote settings.\n")
}

func replayMark(active bool) {
	var t string
	if !active {
		t = fmt.Sprintf("PAUSE,%d\n", time.Since(timeStarted).Nanoseconds())
	} else {
		t = fmt.Sprintf("UNPAUSE,%d\n", time.Since(timeStarted).Nanoseconds())
	}

	if uatReplayWriter != nil {
		uatReplayWriter.Write([]byte(t))
	}

	if esReplayWriter != nil {
		esReplayWriter.Write([]byte(t))
	}

	if gpsReplayWriter != nil {
		gpsReplayWriter.Write([]byte(t))
	}

	if ahrsReplayWriter != nil {
		ahrsReplayWriter.Write([]byte(t))
	}

	if dump1090ReplayWriter != nil {
		dump1090ReplayWriter.Write([]byte(t))
	}

}

func openReplay(fn string, compressed bool) (WriteCloser, error) {
	fp, err := os.OpenFile(fn, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

	if err != nil {
		log.Printf("Failed to open log file '%s': %s\n", fn, err.Error())
		return nil, err
	}

	var ret WriteCloser
	if compressed {
		ret = gzip.NewWriter(fp) //FIXME: Close() on the gzip.Writer will not close the underlying file.
	} else {
		ret = fp
	}

	timeFmt := "Mon Jan 2 15:04:05 -0700 MST 2006"
	s := fmt.Sprintf("START,%s,%s\n", timeStarted.Format(timeFmt), time.Now().Format(timeFmt)) // Start time marker.

	ret.Write([]byte(s))
	return ret, err
}

func printStats() {
	statTimer := time.NewTicker(30 * time.Second)
	for {
		<-statTimer.C
		var memstats runtime.MemStats
		runtime.ReadMemStats(&memstats)
		log.Printf("stats [started: %s]\n", humanize.RelTime(time.Time{}, stratuxClock.Time, "ago", "from now"))
		log.Printf(" - CPUTemp=%.02f deg C, MemStats.Alloc=%s, MemStats.Sys=%s, totalNetworkMessagesSent=%s\n", globalStatus.CPUTemp, humanize.Bytes(uint64(memstats.Alloc)), humanize.Bytes(uint64(memstats.Sys)), humanize.Comma(int64(totalNetworkMessagesSent)))
		log.Printf(" - UAT/min %s/%s [maxSS=%.02f%%], ES/min %s/%s\n, Total traffic targets tracked=%s", humanize.Comma(int64(globalStatus.UAT_messages_last_minute)), humanize.Comma(int64(globalStatus.UAT_messages_max)), float64(maxSignalStrength)/10.0, humanize.Comma(int64(globalStatus.ES_messages_last_minute)), humanize.Comma(int64(globalStatus.ES_messages_max)), humanize.Comma(int64(len(seenTraffic))))
		log.Printf(" - Network data messages sent: %d total, %d nonqueueable.  Network data bytes sent: %d total, %d nonqueueable.\n", globalStatus.NetworkDataMessagesSent, globalStatus.NetworkDataMessagesSentNonqueueable, globalStatus.NetworkDataBytesSent, globalStatus.NetworkDataBytesSentNonqueueable)
		if globalSettings.GPS_Enabled {
			log.Printf(" - Last GPS fix: %s, GPS solution type: %d using %d satellites (%d/%d seen/tracked), NACp: %d, est accuracy %.02f m\n", stratuxClock.HumanizeTime(mySituation.LastFixLocalTime), mySituation.Quality, mySituation.Satellites, mySituation.SatellitesSeen, mySituation.SatellitesTracked, mySituation.NACp, mySituation.Accuracy)
			log.Printf(" - GPS vertical velocity: %.02f ft/sec; GPS vertical accuracy: %v m\n", mySituation.GPSVertVel, mySituation.AccuracyVert)
		}
		logStatus()
	}
}

var uatReplayDone bool

func uatReplay(f ReadCloser, replaySpeed uint64) {
	defer f.Close()
	rdr := bufio.NewReader(f)
	curTick := int64(0)
	for {
		buf, err := rdr.ReadString('\n')
		if err != nil {
			break
		}
		linesplit := strings.Split(buf, ",")
		if len(linesplit) < 2 { // Blank line or invalid.
			continue
		}
		if linesplit[0] == "START" { // Reset ticker, new start.
			curTick = 0
		} else { // If it's not "START", then it's a tick count.
			i, err := strconv.ParseInt(linesplit[0], 10, 64)
			if err != nil {
				log.Printf("invalid tick: '%s'\n", linesplit[0])
				continue
			}
			thisWait := (i - curTick) / int64(replaySpeed)

			if thisWait >= 120000000000 { // More than 2 minutes wait, skip ahead.
				log.Printf("UAT skipahead - %d seconds.\n", thisWait/1000000000)
			} else {
				time.Sleep(time.Duration(thisWait) * time.Nanosecond) // Just in case the units change.
			}

			p := strings.Trim(linesplit[1], " ;\r\n")
			buf := fmt.Sprintf("%s;\n", p)
			o, msgtype := parseInput(buf)
			if o != nil && msgtype != 0 {
				relayMessage(msgtype, o)
			}
			curTick = i
		}
	}
	uatReplayDone = true
}

func openReplayFile(fn string) ReadCloser {
	fp, err := os.Open(fn)
	if err != nil {
		log.Printf("error opening '%s': %s\n", fn, err.Error())
		os.Exit(1)
		return nil
	}

	var ret ReadCloser
	if strings.HasSuffix(fn, ".gz") { // Open as a compressed replay log, depending on the suffix.
		ret, err = gzip.NewReader(fp)
		if err != nil {
			log.Printf("error opening compressed log '%s': %s\n", fn, err.Error())
			os.Exit(1)
			return nil
		}
	} else {
		ret = fp
	}

	return ret
}

var stratuxClock *monotonic
var sigs = make(chan os.Signal, 1) // Signal catch channel (shutdown).

// Close replay log file handles.
func closeReplayLogs() {
	if uatReplayWriter != nil {
		uatReplayWriter.Close()
	}
	if esReplayWriter != nil {
		esReplayWriter.Close()
	}
	if gpsReplayWriter != nil {
		gpsReplayWriter.Close()
	}
	if ahrsReplayWriter != nil {
		ahrsReplayWriter.Close()
	}
	if dump1090ReplayWriter != nil {
		dump1090ReplayWriter.Close()
	}

}

// Graceful shutdown.
func gracefulShutdown() {
	// Shut down SDRs.
	sdrKill()
	//TODO: Any other graceful shutdown functions.
	closeReplayLogs()
	os.Exit(1)
}

func signalWatcher() {
	sig := <-sigs
	log.Printf("signal caught: %s - shutting down.\n", sig.String())
	gracefulShutdown()
}

func main() {
	// Catch signals for graceful shutdown.
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go signalWatcher()

	stratuxClock = NewMonotonic() // Start our "stratux clock".

	// Set up status.
	globalStatus.Version = stratuxVersion
	globalStatus.Build = stratuxBuild
	globalStatus.Errors = make([]string, 0)
	if _, err := os.Stat("/etc/FlightBox"); !os.IsNotExist(err) {
		globalStatus.HardwareBuild = "FlightBox"
	}

	//	replayESFilename := flag.String("eslog", "none", "ES Log filename")
	replayUATFilename := flag.String("uatlog", "none", "UAT Log filename")
	develFlag := flag.Bool("developer", false, "Developer mode")
	replayFlag := flag.Bool("replay", false, "Replay file flag")
	replaySpeed := flag.Int("speed", 1, "Replay speed multiplier")
	stdinFlag := flag.Bool("uatin", false, "Process UAT messages piped to stdin")

	flag.Parse()

	timeStarted = time.Now()
	runtime.GOMAXPROCS(runtime.NumCPU()) // redundant with Go v1.5+ compiler

	if *develFlag == true {
		log.Printf("Developer mode flag true!\n")
		developerMode = true
	}

	// Duplicate log.* output to debugLog.
	fp, err := os.OpenFile(debugLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		err_log := fmt.Errorf("Failed to open '%s': %s", debugLog, err.Error())
		addSystemError(err_log)
		log.Printf("%s\n", err_log.Error())
	} else {
		defer fp.Close()
		mfp := io.MultiWriter(fp, os.Stdout)
		log.SetOutput(mfp)
	}

	log.Printf("Stratux %s (%s) starting.\n", stratuxVersion, stratuxBuild)
	constructFilenames()

	ADSBTowers = make(map[string]ADSBTower)
	MsgLog = make([]msg, 0)

	crcInit() // Initialize CRC16 table.

	sdrInit()
	initTraffic()

	// Read settings.
	readSettings()

	// Disable replay logs when replaying - so that messages replay data isn't copied into the logs.
	// Override after reading in the settings.
	if *replayFlag == true {
		log.Printf("Replay file %s\n", *replayUATFilename)
		globalSettings.ReplayLog = true
	}

	//FIXME: Only do this if data logging is enabled.
	initDataLog()
	// Set up the replay logs. Keep these files open in any case, even if replay logging is disabled.

	if uatwt, err := openReplay(uatReplayLog, !developerMode); err != nil {
		globalSettings.ReplayLog = false
	} else {
		uatReplayWriter = uatwt
		defer uatReplayWriter.Close()
	}
	// 1090ES replay log.
	if eswt, err := openReplay(esReplayLog, !developerMode); err != nil {
		globalSettings.ReplayLog = false
	} else {
		esReplayWriter = eswt
		defer esReplayWriter.Close()
	}
	// GPS replay log.
	if gpswt, err := openReplay(gpsReplayLog, !developerMode); err != nil {
		globalSettings.ReplayLog = false
	} else {
		gpsReplayWriter = gpswt
		defer gpsReplayWriter.Close()
	}
	// AHRS replay log.
	if ahrswt, err := openReplay(ahrsReplayLog, !developerMode); err != nil {
		globalSettings.ReplayLog = false
	} else {
		ahrsReplayWriter = ahrswt
		defer ahrsReplayWriter.Close()
	}
	// Dump1090 replay log.
	if dump1090wt, err := openReplay(dump1090ReplayLog, !developerMode); err != nil {
		globalSettings.ReplayLog = false
	} else {
		dump1090ReplayWriter = dump1090wt
		defer dump1090ReplayWriter.Close()
	}

	// Mark the files (whether we're logging or not).
	replayMark(globalSettings.ReplayLog)

	initRY835AI()

	// Start the heartbeat message loop in the background, once per second.
	go heartBeatSender()
	// Start the management interface.
	go managementInterface()

	// Initialize the (out) network handler.
	initNetwork()

	// Start printing stats periodically to the logfiles.
	go printStats()

	// Monitor RPi CPU temp.
	go cpuTempMonitor()

	reader := bufio.NewReader(os.Stdin)

	if *replayFlag == true {
		fp := openReplayFile(*replayUATFilename)

		playSpeed := uint64(*replaySpeed)
		log.Printf("Replay speed: %dx\n", playSpeed)
		go uatReplay(fp, playSpeed)

		for {
			time.Sleep(1 * time.Second)
			if uatReplayDone {
				//&& esDone {
				return
			}
		}

	} else if *stdinFlag == true {
		for {
			buf, err := reader.ReadString('\n')
			if err != nil {
				log.Printf("lost stdin.\n")
				break
			}
			o, msgtype := parseInput(buf)
			if o != nil && msgtype != 0 {
				relayMessage(msgtype, o)
			}
		}
	} else {
		// wait indefinitely
		select {}
	}
}
