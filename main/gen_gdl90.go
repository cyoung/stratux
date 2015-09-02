package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
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
	stratuxVersion      = "v0.2"
	configLocation      = "/etc/stratux.conf"
	managementAddr      = ":80"
	maxDatagramSize     = 8192
	maxUserMsgQueueSize = 2500 // About 1MB per port per connected client.

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

var Crc16Table [256]uint16

var mySituation SituationData

type msg struct {
	MessageClass uint
	TimeReceived time.Time
	Data         []byte
}

var MsgLog []msg
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

func makeInitializationMessage() []byte {
	msg := make([]byte, 3)
	// See p.13.
	msg[0] = 0x02 // Message type "Initialization".
	msg[1] = 0x00 //TODO
	msg[2] = 0x00 //TODO
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
	for {
		<-timer.C
		sendGDL90(makeHeartbeat(), false)
		//		sendGDL90(makeTrafficReport())
		makeOwnshipReport()
		makeOwnshipGeometricAltitudeReport()
		sendGDL90(makeInitializationMessage(), false)
		sendTrafficUpdates()
		updateStatus()
	}
}

func updateStatus() {
	t := make([]msg, 0)
	m := len(MsgLog)
	UAT_messages_last_minute := uint(0)
	ES_messages_last_minute := uint(0)
	for i := 0; i < m; i++ {
		if time.Now().Sub(MsgLog[i].TimeReceived).Minutes() < 1 {
			t = append(t, MsgLog[i])
			if MsgLog[i].MessageClass == MSGCLASS_UAT {
				UAT_messages_last_minute++
			} else if MsgLog[i].MessageClass == MSGCLASS_ES {
				ES_messages_last_minute++
			}
		}
	}
	MsgLog = t
	globalStatus.UAT_messages_last_minute = UAT_messages_last_minute
	globalStatus.ES_messages_last_minute = ES_messages_last_minute

	// Update "max messages/min" counters.
	if globalStatus.UAT_messages_max < UAT_messages_last_minute {
		globalStatus.UAT_messages_max = UAT_messages_last_minute
	}
	if globalStatus.ES_messages_max < ES_messages_last_minute {
		globalStatus.ES_messages_max = ES_messages_last_minute
	}

	if isGPSValid() {
		globalStatus.GPS_satellites_locked = mySituation.satellites
	}

	// Update Uptime.
	globalStatus.Uptime = time.Since(timeStarted).String()

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

func parseInput(buf string) ([]byte, uint16) {
	x := strings.Split(buf, ";") // Discard everything after the first ';'.
	if len(x) == 0 {
		return nil, 0
	}
	s := x[0]
	if len(s) == 0 {
		return nil, 0
	}
	msgtype := uint16(0)

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
	MsgLog = append(MsgLog, thisMsg)

	return frame, msgtype
}

type settings struct {
	UAT_Enabled    bool
	ES_Enabled     bool
	GPS_Enabled    bool
	NetworkOutputs []networkConnection
	AHRS_Enabled   bool
}

type status struct {
	Version                  string
	Devices                  uint
	Connected_Users          uint
	UAT_messages_last_minute uint
	UAT_messages_max         uint
	ES_messages_last_minute  uint
	ES_messages_max          uint
	GPS_satellites_locked    uint16
	GPS_connected            bool
	RY835AI_connected        bool
	Uptime                   string
	CPUTemp                  float32
}

var globalSettings settings
var globalStatus status

func defaultSettings() {
	globalSettings.UAT_Enabled = true  //TODO
	globalSettings.ES_Enabled = false  //TODO
	globalSettings.GPS_Enabled = false //TODO
	//FIXME: Need to change format below.
	globalSettings.NetworkOutputs = []networkConnection{{nil, "", 4000, NETWORK_GDL90_STANDARD, false, nil}, {nil, "", 43211, NETWORK_GDL90_STANDARD | NETWORK_AHRS_GDL90, false, nil}, {nil, "", 49002, NETWORK_AHRS_FFSIM, false, nil}}
	globalSettings.AHRS_Enabled = false
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
	MsgLog = make([]msg, 0)

	crcInit() // Initialize CRC16 table.
	initTraffic()

	globalStatus.Version = stratuxVersion
	globalStatus.Devices = 0 //TODO

	readSettings()

	initRY835AI()

	//TODO: network stuff

	// Start the heartbeat message loop in the background, once per second.
	go heartBeatSender()
	// Start the management interface.
	go managementInterface()

	// Initialize the (out) network handler.
	initNetwork()

	reader := bufio.NewReader(os.Stdin)

	for {
		buf, _ := reader.ReadString('\n')
		o, msgtype := parseInput(buf)
		if o != nil && msgtype != 0 {
			relayMessage(msgtype, o)
		}
	}

}
