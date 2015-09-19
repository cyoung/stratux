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

	MSGCLASS_UAT = 0
	MSGCLASS_ES  = 1

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

type msg struct {
	MessageClass uint
	TimeReceived time.Time
	Data         []byte
	Product      uint32
}

// Raw inputs.
var MsgLog []msg

// Time gen_gdl90 was started.
var timeStarted time.Time

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

	msg[2] = 1 // Address.
	msg[3] = 1 // Address.
	msg[4] = 1 // Address.

	tmp := makeLatLng(mySituation.lat)
	msg[5] = tmp[0] // Latitude.
	msg[6] = tmp[1] // Latitude.
	msg[7] = tmp[2] // Latitude.

	tmp = makeLatLng(mySituation.lng)
	msg[8] = tmp[0]  // Longitude.
	msg[9] = tmp[1]  // Longitude.
	msg[10] = tmp[2] // Longitude.

	// This is **PRESSURE ALTITUDE**
	alt := uint16(0xFFF) // 0xFFF "invalid altitude."

	if isTempPressValid() {
		alt = uint16(mySituation.pressure_alt)
	}
	alt = (alt + 1000) / 25
	alt = alt & 0xFFF // Should fit in 12 bits.

	msg[11] = byte((alt & 0xFF0) >> 4) // Altitude.
	msg[12] = byte((alt & 0x00F) << 4)

	if isGPSGroundTrackValid() {
		msg[12] = byte(((alt & 0x00F) << 4) | 0xB) // "Airborne" + "True Heading"
	} else {
		msg[12] = byte((alt & 0x00F) << 4)
	}
	msg[13] = 0xBB // NIC and NACp.

	gdSpeed := uint16(0) // 1kt resolution.
	if isGPSGroundTrackValid() {
		gdSpeed = mySituation.groundSpeed
	}
	gdSpeed = gdSpeed & 0x0FFF // Should fit in 12 bits.

	msg[14] = byte((gdSpeed & 0xFF0) >> 4)
	msg[15] = byte((gdSpeed & 0x00F) << 4)

	verticalVelocity := int16(1000 / 64) // ft/min. 64 ft/min resolution.
	//TODO: 0x800 = no information available.
	verticalVelocity = verticalVelocity & 0x0FFF // Should fit in 12 bits.
	msg[15] = msg[15] | byte((verticalVelocity&0x0F00)>>8)
	msg[16] = byte(verticalVelocity & 0xFF)

	// Showing magnetic (corrected) on ForeFlight. Needs to be True Heading.
	groundTrack := uint16(0)
	if isGPSGroundTrackValid() {
		groundTrack = mySituation.trueCourse
	}
	trk := uint8(float32(groundTrack) / TRACK_RESOLUTION) // Resolution is ~1.4 degrees.

	msg[17] = byte(trk)

	msg[18] = 0x01 // "Light (ICAO) < 15,500 lbs"

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
	alt := int16(mySituation.alt) // GPS Altitude.
	alt = alt / 5
	msg[1] = byte(alt >> 8)     // Altitude.
	msg[2] = byte(alt & 0x00FF) // Altitude.

	//TODO: "Figure of Merit". 0x7FFF "Not available".
	msg[3] = 0x00
	msg[4] = 0x0A

	sendGDL90(prepareMessage(msg), false)
	return true
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
	for i := 0; i < m; i++ {
		if time.Now().Sub(MsgLog[i].TimeReceived).Minutes() < 1 {
			t = append(t, MsgLog[i])
			if MsgLog[i].MessageClass == MSGCLASS_UAT {
				UAT_messages_last_minute++
				products_last_minute[getProductNameFromId(int(MsgLog[i].Product))]++
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

}

func updateStatus() {
	if isGPSValid() {
		globalStatus.GPS_satellites_locked = mySituation.satellites
	}

	// Update Uptime value
	globalStatus.Uptime = time.Since(timeStarted).Nanoseconds() / 1000000

	// Update CPUTemp.
	temp, err := ioutil.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	tempStr := strings.Trim(string(temp), "\n")
	globalStatus.CPUTemp = float32(-99.0)
	if err == nil {
		tInt, err := strconv.Atoi(tempStr)
		if err == nil {
			globalStatus.CPUTemp = float32(tInt) / float32(1000.0)
		}
	}
}

func replayLog(msg string, msgclass int) {
	if !globalSettings.ReplayLog { // Logging disabled.
		return
	}
	msg = strings.Trim(msg, " \r\n")
	if len(msg) == 0 { // Blank message.
		return
	}
	if msgclass == MSGCLASS_UAT {
		fmt.Fprintf(uatReplayfp, "%d,%s\n", time.Since(timeStarted).Nanoseconds(), msg)
	} else if msgclass == MSGCLASS_ES {
		fmt.Fprintf(esReplayfp, "%d,%s\n", time.Since(timeStarted).Nanoseconds(), msg)
	}
}

func parseInput(buf string) ([]byte, uint16) {
	replayLog(buf, MSGCLASS_UAT) // Log the raw message.

	x := strings.Split(buf, ";") // Discard everything after the first ';'.
	if len(x) == 0 {
		return nil, 0
	}
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

	s = s[1:]
	msglen := len(s) / 2

	if len(s)%2 != 0 { // Bad format.
		return nil, 0
	}

	if msglen == UPLINK_FRAME_DATA_BYTES {
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
	thisMsg.Product = 9999
	if isUplink && msgtype == MSGTYPE_UPLINK && len(x) > 11 { //FIXME: Need to pull out FIS-B frames from within the uplink packet.
		thisMsg.Product = ((uint32(frame[10]) & 0x1f) << 6) | (uint32(frame[11]) >> 2)
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
	ReplayLog      bool // Startup only option. Cannot be changed during runtime.
	PPM            int
}

type status struct {
	Version                  string
	Devices                  uint
	Connected_Users          uint
	UAT_messages_last_minute uint
	UAT_products_last_minute map[string]uint32
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
	globalSettings.NetworkOutputs = []networkConnection{{nil, "", 4000, NETWORK_GDL90_STANDARD, nil, time.Time{}, time.Time{}}, {nil, "", 43211, NETWORK_GDL90_STANDARD | NETWORK_AHRS_GDL90, nil, time.Time{}, time.Time{}}, {nil, "", 49002, NETWORK_AHRS_FFSIM, nil, time.Time{}, time.Time{}}}
	globalSettings.AHRS_Enabled = false
	globalSettings.DEBUG = false
	globalSettings.ReplayLog = false //TODO: 'true' for debug builds.
}

func readSettings() {
	fd, err := os.Open(configLocation)
	defer fd.Close()
	if err != nil {
		log.Printf("can't read settings %s: %s\n", configLocation, err.Error())
		defaultSettings()
		return
	}
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
	defer fd.Close()
	if err != nil {
		log.Printf("can't save settings %s: %s\n", configLocation, err.Error())
		return
	}
	jsonSettings, _ := json.Marshal(&globalSettings)
	fd.Write(jsonSettings)
	log.Printf("wrote settings.\n")
}

func main() {
	timeStarted = time.Now()
	runtime.GOMAXPROCS(runtime.NumCPU()) // redundant with Go v1.5+ compiler

	// Duplicate log.* output to debugLog.
	fp, err := os.OpenFile(debugLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	defer fp.Close()
	if err != nil {
		log.Printf("Failed to open log file '%s': %s\n", debugLog, err.Error())
	}
	mfp := io.MultiWriter(fp, os.Stdout)
	log.SetOutput(mfp)

	log.Printf("Stratux %s (%s) starting.\n", stratuxVersion, stratuxBuild)

	MsgLog = make([]msg, 0)

	crcInit() // Initialize CRC16 table.
	sdrInit()
	initTraffic()

	globalStatus.Version = stratuxVersion

	readSettings()

	// Log inputs.
	if globalSettings.ReplayLog {
		uatfp, err := os.OpenFile(uatReplayLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Printf("Failed to open log file '%s': %s\n", uatReplayLog, err.Error())
			globalSettings.ReplayLog = false
		} else {
			uatReplayfp = uatfp
			fmt.Fprintf(uatReplayfp, "START,%s\n", timeStarted.Format("Mon Jan 2 15:04:05 -0700 MST 2006"))
			defer uatReplayfp.Close()
		}
		esfp, err := os.OpenFile(esReplayLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Printf("Failed to open log file '%s': %s\n", esReplayLog, err.Error())
			globalSettings.ReplayLog = false
		} else {
			esReplayfp = esfp
			fmt.Fprintf(esReplayfp, "START,%s\n", timeStarted.Format("Mon Jan 2 15:04:05 -0700 MST 2006"))
			defer esReplayfp.Close()
		}
	}

	initRY835AI()

	// Start the heartbeat message loop in the background, once per second.
	go heartBeatSender()
	// Start the management interface.
	go managementInterface()

	// Initialize the (out) network handler.
	initNetwork()

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
