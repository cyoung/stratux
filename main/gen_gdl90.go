package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"

	"../uatparse"
)

// http://www.faa.gov/nextgen/programs/adsb/wsa/media/GDL90_Public_ICD_RevA.PDF

const (
	configLocation      = "/etc/stratux.conf"
	managementAddr      = ":80"
	debugLog            = "/var/log/stratux.log"
	maxDatagramSize     = 8192
	maxUserMsgQueueSize = 25000 // About 10MB per port per connected client.
	uatReplayLog        = "/var/log/stratux-uat.log"
	esReplayLog         = "/var/log/stratux-es.log"
	gpsReplayLog        = "/var/log/stratux-gps.log"
	ahrsReplayLog       = "/var/log/stratux-ahrs.log"

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

	MSGCLASS_UAT  = 0
	MSGCLASS_ES   = 1
	MSGCLASS_GPS  = 3
	MSGCLASS_AHRS = 4

	LON_LAT_RESOLUTION = float32(180.0 / 8388608.0)
	TRACK_RESOLUTION   = float32(360.0 / 256.0)
)

var stratuxBuild string
var stratuxVersion string

// CRC16 table generated to use to work with GDL90 messages.
var Crc16Table [256]uint16

// Current AHRS, pressure altitude, etc.
var mySituation SituationData

// File handles for replay logging.
var uatReplayfp *os.File
var esReplayfp *os.File
var gpsReplayfp *os.File
var ahrsReplayfp *os.File

type msg struct {
	MessageClass    uint
	TimeReceived    time.Time
	Data            []byte
	Products        []uint32
	Signal_strength int
	ADSBTowerID     string // Index in the 'ADSBTowers' map, if this is a parseable uplink message.
}

// Raw inputs.
var MsgLog []msg

// Time gen_gdl90 was started.
var timeStarted time.Time

type ADSBTower struct {
	Lat                         float64
	Lng                         float64
	Signal_strength_last_minute int
	signal_power_last_minute    int64 // Over total messages.
	Signal_strength_max         int
	Messages_last_minute        uint64
	Messages_total              uint64
}

var ADSBTowers map[string]ADSBTower // Running list of all towers seen. (lat,lng) -> ADSBTower

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

//TODO
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
	if isTempPressValid() {
		alt = uint16(mySituation.Pressure_alt)
	} else {
		alt = uint16(mySituation.Alt) //FIXME: This should not be here.
	}
	alt = (alt + 1000) / 25

	alt = alt & 0xFFF // Should fit in 12 bits.

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

	verticalVelocity := int16(1000 / 64) // ft/min. 64 ft/min resolution.
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

//TODO
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
	timerMessageStats := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-timer.C:
			sendGDL90(makeHeartbeat(), false)
			sendGDL90(makeStratuxHeartbeat(), false)
			//		sendGDL90(makeTrafficReport())
			makeOwnshipReport()
			makeOwnshipGeometricAltitudeReport()
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
		tinf.Signal_strength_last_minute = 0
		ADSBTowers[t] = tinf
	}

	for i := 0; i < m; i++ {
		if time.Now().Sub(MsgLog[i].TimeReceived).Minutes() < 1 {
			t = append(t, MsgLog[i])
			if MsgLog[i].MessageClass == MSGCLASS_UAT {
				UAT_messages_last_minute++
				for _, p := range MsgLog[i].Products {
					products_last_minute[getProductNameFromId(int(p))]++
				}
				if len(MsgLog[i].ADSBTowerID) > 0 { // Update tower stats.
					tid := MsgLog[i].ADSBTowerID
					twr := ADSBTowers[tid]
					twr.Messages_last_minute++
					twr.signal_power_last_minute += int64(MsgLog[i].Signal_strength)
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
	globalStatus.uat_products_last_minute = products_last_minute

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
			tinf.Signal_strength_last_minute = 0
		} else {
			tinf.Signal_strength_last_minute = int(tinf.signal_power_last_minute / int64(tinf.Messages_last_minute))
		}
		ADSBTowers[t] = tinf
	}

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
	if isGPSValid() {
		globalStatus.GPS_satellites_locked = mySituation.Satellites
	}

	// Update Uptime value
	globalStatus.Uptime = time.Since(timeStarted).Nanoseconds() / 1000000
}

func replayLog(msg string, msgclass int) {
	if !globalSettings.ReplayLog { // Logging disabled.
		return
	}
	msg = strings.Trim(msg, " \r\n")
	if len(msg) == 0 { // Blank message.
		return
	}
	var fp *os.File
	switch msgclass {
	case MSGCLASS_UAT:
		fp = uatReplayfp
	case MSGCLASS_ES:
		fp = esReplayfp
	case MSGCLASS_GPS:
		fp = gpsReplayfp
	case MSGCLASS_AHRS:
		fp = ahrsReplayfp
	}
	if fp != nil {
		fmt.Fprintf(fp, "%d,%s\n", time.Since(timeStarted).Nanoseconds(), msg)
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
	wm.LocaltimeReceived = time.Now()

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

	if s[0] == '-' {
		parseDownlinkReport(s)
	}

	var thisSignalStrength int

	if isUplink && len(x) >= 3 {
		// See if we can parse out the signal strength.
		ss := x[2]
		if strings.HasPrefix(ss, "ss=") {
			ssStr := ss[3:]
			if ssInt, err := strconv.Atoi(ssStr); err == nil {
				thisSignalStrength = ssInt
				if ssInt > maxSignalStrength {
					maxSignalStrength = ssInt
				}
			}
		}
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
	thisMsg.TimeReceived = time.Now()
	thisMsg.Data = frame
	thisMsg.Signal_strength = thisSignalStrength
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
				ADSBTowers[towerid] = newTower
			}
			twr := ADSBTowers[towerid]
			twr.Messages_total++
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
	Version                  string
	Devices                  uint32
	Connected_Users          uint
	UAT_messages_last_minute uint
	uat_products_last_minute map[string]uint32
	UAT_messages_max         uint
	ES_messages_last_minute  uint
	ES_messages_max          uint
	GPS_satellites_locked    uint16
	GPS_connected            bool
	RY835AI_connected        bool
	Uptime                   int64
	CPUTemp                  float32
}

var globalSettings settings
var globalStatus status

func defaultSettings() {
	globalSettings.UAT_Enabled = true  //TODO
	globalSettings.ES_Enabled = false  //TODO
	globalSettings.GPS_Enabled = false //TODO
	//FIXME: Need to change format below.
	globalSettings.NetworkOutputs = []networkConnection{{nil, "", 4000, NETWORK_GDL90_STANDARD | NETWORK_AHRS_GDL90, nil, time.Time{}, time.Time{}, 0}, {nil, "", 49002, NETWORK_AHRS_FFSIM, nil, time.Time{}, time.Time{}, 0}}
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

func saveSettings() {
	fd, err := os.OpenFile(configLocation, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0644))
	if err != nil {
		log.Printf("can't save settings %s: %s\n", configLocation, err.Error())
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

	if uatReplayfp != nil {
		uatReplayfp.Write([]byte(t))
	}

	if esReplayfp != nil {
		esReplayfp.Write([]byte(t))
	}

	if gpsReplayfp != nil {
		gpsReplayfp.Write([]byte(t))
	}

	if ahrsReplayfp != nil {
		ahrsReplayfp.Write([]byte(t))
	}

}

func openReplay(fn string) (*os.File, error) {
	ret, err := os.OpenFile(fn, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Failed to open log file '%s': %s\n", fn, err.Error())
	} else {
		timeFmt := "Mon Jan 2 15:04:05 -0700 MST 2006"
		fmt.Fprintf(ret, "START,%s,%s\n", timeStarted.Format(timeFmt), time.Now().Format(timeFmt)) // Start time marker.
	}
	return ret, err
}

func printStats() {
	statTimer := time.NewTicker(30 * time.Second)
	for {
		<-statTimer.C
		var memstats runtime.MemStats
		runtime.ReadMemStats(&memstats)
		log.Printf("stats [up since: %s]\n", humanize.Time(timeStarted))
		log.Printf(" - CPUTemp=%.02f deg C, MemStats.Alloc=%s, MemStats.Sys=%s, totalNetworkMessagesSent=%s\n", globalStatus.CPUTemp, humanize.Bytes(uint64(memstats.Alloc)), humanize.Bytes(uint64(memstats.Sys)), humanize.Comma(int64(totalNetworkMessagesSent)))
		log.Printf(" - UAT/min %s/%s [maxSS=%.02f%%], ES/min %s/%s\n, Total traffic targets tracked=%s", humanize.Comma(int64(globalStatus.UAT_messages_last_minute)), humanize.Comma(int64(globalStatus.UAT_messages_max)), float64(maxSignalStrength)/10.0, humanize.Comma(int64(globalStatus.ES_messages_last_minute)), humanize.Comma(int64(globalStatus.ES_messages_max)), humanize.Comma(int64(len(seenTraffic))))
		if globalSettings.GPS_Enabled {
			log.Printf(" - Last GPS fix: %s, GPS solution type: %d, NACp: %d, est accuracy %.02f m\n", humanize.Time(mySituation.LastFixLocalTime), mySituation.quality, mySituation.NACp, mySituation.Accuracy)
		}
	}
}

func main() {
	timeStarted = time.Now()
	runtime.GOMAXPROCS(runtime.NumCPU()) // redundant with Go v1.5+ compiler

	// Duplicate log.* output to debugLog.
	fp, err := os.OpenFile(debugLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Failed to open '%s': %s\n", debugLog, err.Error())
	} else {
		defer fp.Close()
		mfp := io.MultiWriter(fp, os.Stdout)
		log.SetOutput(mfp)
	}

	log.Printf("Stratux %s (%s) starting.\n", stratuxVersion, stratuxBuild)

	ADSBTowers = make(map[string]ADSBTower)
	MsgLog = make([]msg, 0)

	crcInit() // Initialize CRC16 table.
	sdrInit()
	initTraffic()

	globalStatus.Version = stratuxVersion

	readSettings()

	// Set up the replay logs. Keep these files open in any case, even if replay logging is disabled.

	// UAT replay log.
	if uatfp, err := openReplay(uatReplayLog); err != nil {
		globalSettings.ReplayLog = false
	} else {
		uatReplayfp = uatfp
		defer uatReplayfp.Close()
	}
	// 1090ES replay log.
	if esfp, err := openReplay(esReplayLog); err != nil {
		globalSettings.ReplayLog = false
	} else {
		esReplayfp = esfp
		defer esReplayfp.Close()
	}
	// GPS replay log.
	if gpsfp, err := openReplay(gpsReplayLog); err != nil {
		globalSettings.ReplayLog = false
	} else {
		gpsReplayfp = gpsfp
		defer gpsReplayfp.Close()
	}
	// AHRS replay log.
	if ahrsfp, err := openReplay(ahrsReplayLog); err != nil {
		globalSettings.ReplayLog = false
	} else {
		ahrsReplayfp = ahrsfp
		defer ahrsReplayfp.Close()
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

}
