/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New" License
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
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/b3nn0/stratux/common"
	"github.com/b3nn0/stratux/uatparse"
	humanize "github.com/dustin/go-humanize"
	"github.com/ricochet2200/go-disk-usage/du"
)

// https://www.faa.gov/nextgen/programs/adsb/Archival/
// https://www.faa.gov/nextgen/programs/adsb/Archival/media/GDL90_Public_ICD_RevA.PDF

var logDirf string      // Directory for all logging
var debugLogf string    // Set according to OS config.
var dataLogFilef string // Set according to OS config.

const (
	STRATUX_HOME   = "/opt/stratux/"
	configLocation = "/boot/stratux.conf"
	managementAddr = ":80"
	logDir         = "/var/log/"
	debugLogFile   = "stratux.log"
	dataLogFile    = "stratux.sqlite"
	//FlightBox: log to /root.
	logDir_FB           = "/root/"
	maxDatagramSize     = 8192
	maxUserMsgQueueSize = 25000 // About 10MB per port per connected client.

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

	MSGCLASS_UAT   = 0
	MSGCLASS_ES    = 1
	MSGCLASS_OGN   = 2

	LON_LAT_RESOLUTION = float32(180.0 / 8388608.0)
	TRACK_RESOLUTION   = float32(360.0 / 256.0)

	/*
		GPS_TYPE_NMEA     = 0x01
		GPS_TYPE_UBX      = 0x02
		GPS_TYPE_SIRF     = 0x03
		GPS_TYPE_MEDIATEK = 0x04
		GPS_TYPE_FLARM    = 0x05
		GPS_TYPE_GARMIN   = 0x06
	*/

	GPS_TYPE_UBX9     = 0x09
	GPS_TYPE_UBX8     = 0x08
	GPS_TYPE_UBX7     = 0x07
	GPS_TYPE_UBX6     = 0x06
	GPS_TYPE_PROLIFIC = 0x02
	GPS_TYPE_UART     = 0x01
	GPS_TYPE_SERIAL   = 0x0A
	GPS_TYPE_OGNTRACKER = 0x03
	GPS_TYPE_SOFTRF_DONGLE = 0x0B
	GPS_TYPE_NETWORK  = 0x0C
	GPS_PROTOCOL_NMEA = 0x10
	// other GPS types to be defined as needed

)

var logFileHandle *os.File
var usage *du.DiskUsage

var maxSignalStrength int

var stratuxBuild string
var stratuxVersion string

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

type msg struct {
	MessageClass     uint
	TimeReceived     time.Time
	Data             string
	Products         []uint32
	Signal_amplitude int
	Signal_strength  float64
	ADSBTowerID      string // Index in the 'ADSBTowers' map, if this is a parseable uplink message.
	uatMsg           *uatparse.UATMsg
}

// Raw inputs.
var msgLog []msg
var msgLogMutex sync.Mutex

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
}

var ADSBTowers map[string]ADSBTower // Running list of all towers seen. (lat,lng) -> ADSBTower
var ADSBTowerMutex *sync.Mutex

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

func isDetectedOwnshipValid() bool {
	return stratuxClock.Since(OwnshipTrafficInfo.Last_seen).Seconds() < 10
}

func makeOwnshipReport() bool {
	gpsValid := isGPSValid()
	selfOwnshipValid := isDetectedOwnshipValid()
	if !gpsValid && !selfOwnshipValid {
		return false
	}
	curOwnship := OwnshipTrafficInfo

	msg := make([]byte, 28)
	// See p.16.
	msg[0] = 0x0A // Message type "Ownship".

	// Retrieve ICAO code from settings
	code, _ := hex.DecodeString(globalSettings.OwnshipModeS)

	// Ownship Target Identify (see 3.5.1.2 of GDL-90 Specifications)
	// First half of byte is 0 for Alert type of 'No Traffic Alert'
	// Second half of byte is 0 for traffic type 'ADS-B with ICAO'
	// Send 0x01 by default, unless ICAO is set, send 0x00
	if (len(code) == 3 && code[0] != 0xF0 && code[0] != 0x00) {
		msg[1] = 0x00 // ADS-B Out with ICAO
		msg[2] = code[0] // Mode S address.
		msg[3] = code[1] // Mode S address.
		msg[4] = code[2] // Mode S address.
	} else {
		msg[1] = 0x01 // ADS-B Out with self-assigned code
		// Reserved dummy code.
		msg[2] = 0xF0
		msg[3] = 0x00
		msg[4] = 0x00
	}

	var tmp []byte
	var lat, lon float32
	if selfOwnshipValid {
		lat = curOwnship.Lat
		lon = curOwnship.Lng
	} else {
		lat = mySituation.GPSLatitude
		lon = mySituation.GPSLongitude
	}

	tmp = makeLatLng(lat)
	msg[5] = tmp[0] // Latitude.
	msg[6] = tmp[1] // Latitude.
	msg[7] = tmp[2] // Latitude.
	tmp = makeLatLng(lon)
	msg[8] = tmp[0]  // Longitude.
	msg[9] = tmp[1]  // Longitude.
	msg[10] = tmp[2] // Longitude.

	// This is **PRESSURE ALTITUDE**
	alt := uint16(0xFFF) // 0xFFF "invalid altitude."
	validAltf := false

	var altf float64

	if selfOwnshipValid {
		altf = float64(curOwnship.Alt)
		validAltf = true
	} else if isTempPressValid() {
		altf = float64(mySituation.BaroPressureAltitude)
		validAltf = true
	} else if isGPSValid() {
		altf = float64(mySituation.GPSAltitudeMSL)
		validAltf = true
	}

	if validAltf {
		altf = (altf + 1000) / 25
		alt = uint16(altf) & 0xFFF // Should fit in 12 bits.
	}

	msg[11] = byte((alt & 0xFF0) >> 4) // Altitude.
	msg[12] = byte((alt & 0x00F) << 4)
	if selfOwnshipValid || isGPSGroundTrackValid() {
		msg[12] = msg[12] | 0x09 // "Airborne" + "True Track"
	}

	msg[13] = byte(0x80 | (mySituation.GPSNACp & 0x0F)) //Set NIC = 8 and use NACp from gps.go.

	gdSpeed := uint16(0) // 1kt resolution.
	if selfOwnshipValid && curOwnship.Speed_valid {
		gdSpeed = curOwnship.Speed
	} else if isGPSGroundTrackValid() {
		gdSpeed = uint16(mySituation.GPSGroundSpeed + 0.5)
	}

	// gdSpeed should fit in 12 bits.
	msg[14] = byte((gdSpeed & 0xFF0) >> 4)
	msg[15] = byte((gdSpeed & 0x00F) << 4)

	verticalVelocity := int16(0x800) // ft/min. 64 ft/min resolution.
	//TODO: 0x800 = no information available.
	// verticalVelocity should fit in 12 bits.
	msg[15] = msg[15] | byte((verticalVelocity&0x0F00)>>8)
	msg[16] = byte(verticalVelocity & 0xFF)

	// Track is degrees true, set from GPS true course.
	groundTrack := float32(0)
	if selfOwnshipValid {
		groundTrack = float32(curOwnship.Track)
	} else if isGPSGroundTrackValid() {
		groundTrack = mySituation.GPSTrueCourse
	}

	tempTrack := groundTrack + TRACK_RESOLUTION/2 // offset by half the 8-bit resolution to minimize binning error

	for tempTrack > 360 {
		tempTrack -= 360
	}
	for tempTrack < 0 {
		tempTrack += 360
	}

	trk := uint8(tempTrack / TRACK_RESOLUTION) // Resolution is ~1.4 degrees.

	//log.Printf("For groundTrack = %.2f°, tempTrack= %.2f, trk = %d (%f°)\n",groundTrack,tempTrack,trk,float32(trk)*TRACK_RESOLUTION)

	msg[17] = byte(trk)

	msg[18] = 0x01 // "Light (ICAO) < 15,500 lbs"

	if selfOwnshipValid {
		// Limit tail number to 7 characters.
		tail := curOwnship.Tail
		if len(tail) > 7 {
			tail = tail[:7]
		}
		// Copy tail number into message.
		for i := 0; i < len(tail); i++ {
			msg[19+i] = tail[i]
		}
	}

	myReg := "Stratux" // Default callsign.
	// Use icao2reg() results for ownship tail number, if available.
	if len(code) == 3 {
		uintIcao := uint32(code[0])<<16 | uint32(code[1])<<8 | uint32(code[2])
		regFromIcao, regFromIcaoValid := icao2reg(uintIcao)
		if regFromIcaoValid {
			// Valid "decoded" registration. Use this for the reg.
			myReg = regFromIcao
		}
	}

	// Truncate registration to 8 characters.
	if len(myReg) > 8 {
		myReg = myReg[:8]
	}

	// Write the callsign.
	for i := 0; i < len(myReg); i++ {
		msg[19+i] = myReg[i]
	}

	sendGDL90(prepareMessage(msg), false)
	sendXPlane(createXPlaneGpsMsg(lat, lon, mySituation.GPSAltitudeMSL, groundTrack, float32(gdSpeed)), false)

	return true
}

func makeOwnshipGeometricAltitudeReport() bool {
	if !isGPSValid() {
		return false
	}
	msg := make([]byte, 5)
	// See p.28.
	msg[0] = 0x0B // Message type "Ownship Geo Alt".

	var GPSalt float32
	GPSalt = mySituation.GPSHeightAboveEllipsoid
	encodedAlt := int16(GPSalt / 5)    // GPS Altitude, encoded to 16-bit int using 5-foot resolution
	msg[1] = byte(encodedAlt >> 8)     // Altitude.
	msg[2] = byte(encodedAlt & 0x00FF) // Altitude.

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
		switch mySituation.GPSFixQuality {
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
	if common.IsCPUTempValid(globalStatus.CPUTemp) {
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

	// Ping provides ES and UAT
	if globalSettings.Ping_Enabled {
		msg[13] = msg[13] | (1 << 5) | (1 << 6)
	}

	// Valid/Enabled: GPS Enabled portion.
	if globalSettings.GPS_Enabled {
		msg[13] = msg[13] | (1 << 7)
	}

	// Valid/Enabled: AHRS Enabled portion.
	if globalSettings.IMU_Sensor_Enabled {
		msg[12] = 1 << 0
	}

	// Valid/Enabled: last bit unused.

	// Connected hardware: number of radios.
	msg[15] = msg[15] | (byte(globalStatus.Devices) & 0x3)

	// Connected hardware: IMU (spec really says just RY835AI, could revise for other IMUs
	if globalStatus.IMUConnected {
		msg[15] = msg[15] | (1 << 2)
	}

	// Number of GPS satellites locked.
	msg[16] = byte(globalStatus.GPS_satellites_locked)

	// Number of satellites tracked
	msg[17] = byte(globalStatus.GPS_satellites_tracked)

	// Number of UAT traffic targets.
	msg[18] = byte((globalStatus.UAT_traffic_targets_tracking & 0xFF00) >> 8)
	msg[19] = byte(globalStatus.UAT_traffic_targets_tracking & 0xFF)
	// Number of 1090ES traffic targets.
	msg[20] = byte((globalStatus.ES_traffic_targets_tracking & 0xFF00) >> 8)
	msg[21] = byte(globalStatus.ES_traffic_targets_tracking & 0xFF)

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

	// Number of ADS-B towers. Map structure is protected by ADSBTowerMutex.
	ADSBTowerMutex.Lock()
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
	ADSBTowerMutex.Unlock()
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

/*

	ForeFlight "ID Message".

	Sends device information to ForeFlight.

*/
func makeFFIDMessage() []byte {
	msg := make([]byte, 39)
	msg[0] = 0x65 // Message type "ForeFlight".
	msg[1] = 0    // ID message identifier.
	msg[2] = 1    // Message version.
	// Serial number. Set to "invalid" for now.
	for i := 3; i <= 10; i++ {
		msg[i] = 0xFF
	}
	devShortName := "Stratux" // Temporary. Will be populated in the future with other names.
	if len(devShortName) > 8 {
		devShortName = devShortName[:8] // 8 chars.
	}
	copy(msg[11:], devShortName)

	devLongName := fmt.Sprintf("%s-%s", stratuxVersion, stratuxBuild)
	if len(devLongName) > 16 {
		devLongName = devLongName[:16] // 16 chars.
	}
	copy(msg[19:], devLongName)

	//if globalSettings.GDL90MSLAlt_Enabled {
		msg[38] = 0x00 // Capabilities mask. MSL altitude for Ownship Geometric report. We only support HAE as in spec.
	//}

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

	// "Maintenance Req'd". Add flag if there are any current critical system errors.
	if len(globalStatus.Errors) > 0 {
		msg[1] = msg[1] | 0x40
	}

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

func blinkStatusLED() {
	timer := time.NewTicker(100 * time.Millisecond)
	ledON := false
	for {
		<-timer.C
		if ledON {
			ioutil.WriteFile("/sys/class/leds/led0/brightness", []byte("0\n"), 0644)
		} else {
			ioutil.WriteFile("/sys/class/leds/led0/brightness", []byte("1\n"), 0644)
		}
		ledON = !ledON
	}
}

func sendAllOwnshipInfo() {
	//log.Printf("Sending ownship info")
	sendGDL90(makeHeartbeat(), false)
	if !globalSettings.SkyDemonAndroidHack {
		// Skydemon ignores these anyway - reduce data rate a bit
		sendGDL90(makeStratuxHeartbeat(), false)
		sendGDL90(makeStratuxStatus(), false)
		sendGDL90(makeFFIDMessage(), false)
	}
	makeOwnshipReport()
	makeOwnshipGeometricAltitudeReport()
}

func heartBeatSender() {
	timerFast := time.NewTicker(150 * time.Millisecond)
	timer := time.NewTicker(1 * time.Second)
	timerMessageStats := time.NewTicker(2 * time.Second)
	ledBlinking := false
	for {
		select {
		case <-timerFast.C:
			// Skydemon Android socket bug workaround: send ownship info every 200ms
			if globalSettings.SkyDemonAndroidHack {
				sendAllOwnshipInfo()
			}
		case <-timer.C:
			// Green LED - always on during normal operation.
			//  Blinking when there is a critical system error (and Stratux is still running).

			if len(globalStatus.Errors) == 0 { // Any system errors?
				if !globalStatus.NightMode { // LED is off by default (/boot/config.txt.)
					// Turn on green ACT LED on the Pi.
					ioutil.WriteFile("/sys/class/leds/led0/brightness", []byte("1\n"), 0644)
				}
			} else if !ledBlinking {
				// This assumes that system errors do not disappear until restart.
				go blinkStatusLED()
				ledBlinking = true
			}

			// Normal behaviour: Send ownship info once per secopnd
			if !globalSettings.SkyDemonAndroidHack {
				sendAllOwnshipInfo()
			}

			sendNetFLARM(makeGPRMCString())
			sendNetFLARM(makeGPGGAString())
			if isTempPressValid() && mySituation.BaroSourceType != BARO_TYPE_NONE && mySituation.BaroSourceType != BARO_TYPE_ADSBESTIMATE {
				sendNetFLARM(makePGRMZString())
			}
			sendNetFLARM("$GPGSA,A,3,,,,,,,,,,,,,1.0,1.0,1.0*33\r\n")

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

func msgLogAppend(m msg) {
	msgLogMutex.Lock()
	defer msgLogMutex.Unlock()
	msgLog = append(msgLog, m)
}

func updateMessageStats() {
	ADSBTowerMutex.Lock()
	defer ADSBTowerMutex.Unlock()

	msgLogMutex.Lock()
	defer msgLogMutex.Unlock()

	m := len(msgLog)
	t := make([]msg, 0)
	UAT_messages_last_minute := uint(0)
	ES_messages_last_minute := uint(0)
	OGN_messages_last_minute := uint(0)

	// Clear out ADSBTowers stats.
	for t, tinf := range ADSBTowers {
		tinf.Messages_last_minute = 0
		tinf.Energy_last_minute = 0
		ADSBTowers[t] = tinf
	}

	for i := 0; i < m; i++ {
		if stratuxClock.Since(msgLog[i].TimeReceived).Minutes() < 1 {
			t = append(t, msgLog[i])
			if msgLog[i].MessageClass == MSGCLASS_UAT {
				UAT_messages_last_minute++
				if len(msgLog[i].ADSBTowerID) > 0 { // Update tower stats.
					tid := msgLog[i].ADSBTowerID

					if _, ok := ADSBTowers[tid]; !ok { // First time we've seen the tower? Start tracking.
						var newTower ADSBTower
						newTower.Lat = msgLog[i].uatMsg.Lat
						newTower.Lng = msgLog[i].uatMsg.Lon
						newTower.Signal_strength_max = -999 // dBmax = 0, so this needs to initialize below scale ( << -48 dB)
						ADSBTowers[tid] = newTower
					}

					twr := ADSBTowers[tid]
					twr.Signal_strength_now = msgLog[i].Signal_strength

					twr.Energy_last_minute += uint64((msgLog[i].Signal_amplitude) * (msgLog[i].Signal_amplitude))
					twr.Messages_last_minute++
					if msgLog[i].Signal_strength > twr.Signal_strength_max { // Update alltime max signal strength.
						twr.Signal_strength_max = msgLog[i].Signal_strength
					}
					ADSBTowers[tid] = twr
				}
			} else if msgLog[i].MessageClass == MSGCLASS_ES {
				ES_messages_last_minute++
			} else if msgLog[i].MessageClass == MSGCLASS_OGN {
				OGN_messages_last_minute++
			}
		}
	}
	msgLog = t
	globalStatus.UAT_messages_last_minute = UAT_messages_last_minute
	globalStatus.ES_messages_last_minute = ES_messages_last_minute
	globalStatus.OGN_messages_last_minute = OGN_messages_last_minute

	// Update "max messages/min" counters.
	if globalStatus.UAT_messages_max < UAT_messages_last_minute {
		globalStatus.UAT_messages_max = UAT_messages_last_minute
	}
	if globalStatus.ES_messages_max < ES_messages_last_minute {
		globalStatus.ES_messages_max = ES_messages_last_minute
	}
	if globalStatus.OGN_messages_max < OGN_messages_last_minute {
		globalStatus.OGN_messages_max = OGN_messages_last_minute
	}

	// Update average signal strength over last minute for all ADSB towers.
	for t, tinf := range ADSBTowers {
		if tinf.Messages_last_minute == 0 || tinf.Energy_last_minute == 0 {
			tinf.Signal_strength_last_minute = -999
		} else {
			tinf.Signal_strength_last_minute = 10 * (math.Log10(float64((tinf.Energy_last_minute / tinf.Messages_last_minute))) - 6)
		}
		ADSBTowers[t] = tinf
	}

}

func updateStatus() {
	if mySituation.GPSFixQuality == 2 {
		globalStatus.GPS_solution = "3D GPS + SBAS"
	} else if mySituation.GPSFixQuality == 1 {
		globalStatus.GPS_solution = "3D GPS"
	} else if mySituation.GPSFixQuality == 6 {
		globalStatus.GPS_solution = "Dead Reckoning"
	} else if mySituation.GPSFixQuality == 0 {
		globalStatus.GPS_solution = "No Fix"
	} else {
		globalStatus.GPS_solution = "Unknown"
	}

	if !(globalStatus.GPS_connected) || !(isGPSConnected()) { // isGPSConnected looks for valid NMEA messages. GPS_connected is set by gpsSerialReader and will immediately fail on disconnected USB devices, or in a few seconds after "blocked" comms on ttyAMA0.

		mySituation.muSatellite.Lock()
		Satellites = make(map[string]SatelliteInfo)
		mySituation.muSatellite.Unlock()

		mySituation.GPSSatellites = 0
		mySituation.GPSSatellitesSeen = 0
		mySituation.GPSSatellitesTracked = 0
		mySituation.GPSFixQuality = 0
		globalStatus.GPS_solution = "Disconnected"
		globalStatus.GPS_connected = false
	}

	globalStatus.GPS_satellites_locked = mySituation.GPSSatellites
	globalStatus.GPS_satellites_seen = mySituation.GPSSatellitesSeen
	globalStatus.GPS_satellites_tracked = mySituation.GPSSatellitesTracked
	globalStatus.GPS_position_accuracy = mySituation.GPSHorizontalAccuracy

	// Update Uptime value
	globalStatus.Uptime = int64(stratuxClock.Milliseconds)
	globalStatus.UptimeClock = stratuxClock.Time

	usage = du.NewDiskUsage("/")
	globalStatus.DiskBytesFree = usage.Free()
	fileInfo, err := logFileHandle.Stat()
	if err == nil {
		globalStatus.Logfile_Size = fileInfo.Size()
	}

	var ahrsLogSize int64
	ahrsLogFiles, _ := ioutil.ReadDir("/var/log")
	for _, f := range ahrsLogFiles {
		if v, _ := filepath.Match("sensors_*.csv", f.Name()); v {
			ahrsLogSize += f.Size()
		}
	}
	globalStatus.AHRS_LogFiles_Size = ahrsLogSize
}

type WeatherMessage struct {
	Type              string
	Location          string
	Time              string
	Data              string
	LocaltimeReceived time.Time
}

// Send update to connected websockets.
func registerADSBTextMessageReceived(msg string, uatMsg *uatparse.UATMsg) {
	x := strings.Split(msg, " ")
	if len(x) < 5 {
		return
	}

	var wm WeatherMessage

	if (x[0] == "METAR") || (x[0] == "SPECI") {
		globalStatus.UAT_METAR_total++
	}
	if (x[0] == "TAF") || (x[0] == "TAF.AMD") {
		globalStatus.UAT_TAF_total++
	}
	if x[0] == "WINDS" {
		globalStatus.UAT_TAF_total++
	}
	if x[0] == "PIREP" {
		globalStatus.UAT_PIREP_total++
	}
	wm.Type = x[0]
	wm.Location = x[1]
	wm.Time = x[2]
	wm.Data = strings.Join(x[3:], " ")
	wm.LocaltimeReceived = stratuxClock.Time

	// Send to weatherUpdate channel for any connected clients.
	weatherUpdate.SendJSON(wm)
}

func UpdateUATStats(ProductID uint32) {
	switch ProductID {
	case 0, 20:
		globalStatus.UAT_METAR_total++
	case 1, 21:
		globalStatus.UAT_TAF_total++
	case 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 81, 82, 83:
		globalStatus.UAT_NEXRAD_total++
	// AIRMET and SIGMETS
	case 2, 3, 4, 6, 11, 12, 22, 23, 24, 26, 254:
		globalStatus.UAT_SIGMET_total++
	case 5, 25:
		globalStatus.UAT_PIREP_total++
	case 8:
		globalStatus.UAT_NOTAM_total++
	case 413:
		// Do nothing in the case since text is recorded elsewhere
		return
	default:
		globalStatus.UAT_OTHER_total++
	}
}

func parseInput(buf string) ([]byte, uint16) {
	//FIXME: We're ignoring all invalid format UAT messages (not sending to datalog).
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
	} else if msglen == 48 {
		// With Reed Solomon appended
		msgtype = MSGTYPE_LONG_REPORT
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
	thisMsg.Data = buf
	thisMsg.Signal_amplitude = thisSignalStrength
	if thisSignalStrength > 0 {
		thisMsg.Signal_strength = 20 * math.Log10((float64(thisSignalStrength))/1000)
	} else {
		thisMsg.Signal_strength = -999
	}
	thisMsg.Products = make([]uint32, 0)
	if msgtype == MSGTYPE_UPLINK {
		// Parse the UAT message.
		uatMsg, err := uatparse.New(buf)
		if err == nil {
			uatMsg.DecodeUplink()
			towerid := fmt.Sprintf("(%f,%f)", uatMsg.Lat, uatMsg.Lon)
			thisMsg.ADSBTowerID = towerid
			// Get all of the "product ids".
			for _, f := range uatMsg.Frames {
				thisMsg.Products = append(thisMsg.Products, f.Product_id)
				UpdateUATStats(f.Product_id)
				weatherRawUpdate.SendJSON(f)
			}
			// Get all of the text reports.
			textReports, _ := uatMsg.GetTextReports()
			for _, r := range textReports {
				registerADSBTextMessageReceived(r, uatMsg)
			}
			thisMsg.uatMsg = uatMsg
		}
	}

	msgLogAppend(thisMsg)
	logMsg(thisMsg)

	return frame, msgtype
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
	DarkMode             bool
	UAT_Enabled          bool
	ES_Enabled           bool
	OGN_Enabled        bool
	Ping_Enabled         bool
	GPS_Enabled          bool
	BMP_Sensor_Enabled   bool
	IMU_Sensor_Enabled   bool
	NetworkOutputs       []networkConnection
	SerialOutputs        map[string]serialConnection
	DisplayTrafficSource bool
	DEBUG                bool
	ReplayLog            bool
	AHRSLog              bool
	PersistentLogging    bool
	IMUMapping           [2]int     // Map from aircraft axis to sensor axis: accelerometer
	SensorQuaternion     [4]float64 // Quaternion mapping from sensor frame to aircraft frame
	C, D                 [3]float64 // IMU Accel, Gyro zero bias
	PPM                  int
	AltitudeOffset       int
	OwnshipModeS         string
	WatchList            string
	DeveloperMode        bool
	GLimits              string
	StaticIps            []string
	WiFiSSID             string
	WiFiChannel          int
	WiFiSecurityEnabled  bool
	WiFiPassphrase       string
	WiFiSmartEnabled     bool // "Smart WiFi" - disables the default gateway for iOS.
	NoSleep              bool

	WiFiMode             int
	WiFiDirectPin        string
	WiFiIPAddress        string
	SkyDemonAndroidHack  bool
	EstimateBearinglessDist bool
	RadarLimits          int
	RadarRange           int

	OGNAddr              string
	OGNAddrType          int
	OGNAcftType          int
	OGNPilot             string
	OGNReg               string
	OGNTxPower           int

	PWMDutyMin           int
}

type status struct {
	Version                                    string
	Build                                      string
	HardwareBuild                              string
	Devices                                    uint32
	Connected_Users                            uint
	DiskBytesFree                              uint64
	UAT_messages_last_minute                   uint
	UAT_messages_max                           uint
	ES_messages_last_minute                    uint
	ES_messages_max                            uint
	OGN_messages_last_minute                 uint
	OGN_messages_max                         uint
	OGN_connected                            bool
	UAT_traffic_targets_tracking               uint16
	ES_traffic_targets_tracking                uint16
	Ping_connected                             bool
	UATRadio_connected                         bool
	GPS_satellites_locked                      uint16
	GPS_satellites_seen                        uint16
	GPS_satellites_tracked                     uint16
	GPS_position_accuracy                      float32
	GPS_connected                              bool
	GPS_solution                               string
	GPS_detected_type                          uint
	GPS_NetworkRemoteIp                        string // for NMEA via TCP from OGN tracker: display remote IP to configure the OGN tracker
	Uptime                                     int64
	UptimeClock                                time.Time
	CPUTemp                                    float32
	CPUTempMin                                 float32
	CPUTempMax                                 float32
	NetworkDataMessagesSent                    uint64
	NetworkDataMessagesSentNonqueueable        uint64
	NetworkDataBytesSent                       uint64
	NetworkDataBytesSentNonqueueable           uint64
	NetworkDataMessagesSentLastSec             uint64
	NetworkDataMessagesSentNonqueueableLastSec uint64
	NetworkDataBytesSentLastSec                uint64
	NetworkDataBytesSentNonqueueableLastSec    uint64
	UAT_METAR_total                            uint32
	UAT_TAF_total                              uint32
	UAT_NEXRAD_total                           uint32
	UAT_SIGMET_total                           uint32
	UAT_PIREP_total                            uint32
	UAT_NOTAM_total                            uint32
	UAT_OTHER_total                            uint32
	Errors                                     []string
	Logfile_Size                               int64
	AHRS_LogFiles_Size                         int64
	BMPConnected                               bool
	IMUConnected                               bool
	NightMode                                  bool // For turning off LEDs.
	OGN_noise_db                               float32
	OGN_gain_db                                float32
	OGN_tx_enabled                             bool // If ogn-rx-eu uses a local tx module for transmission
}

var globalSettings settings
var globalStatus status

func defaultSettings() {
	globalSettings.DarkMode = false
	globalSettings.UAT_Enabled = false
	globalSettings.ES_Enabled = true
	globalSettings.OGN_Enabled = true
	globalSettings.GPS_Enabled = true
	globalSettings.IMU_Sensor_Enabled = true
	globalSettings.BMP_Sensor_Enabled = true
	//FIXME: Need to change format below.
	globalSettings.NetworkOutputs = []networkConnection{
		{Conn: nil, Ip: "", Port: 4000, Capability: NETWORK_GDL90_STANDARD | NETWORK_AHRS_GDL90},
		{Conn: nil, Ip: "", Port: 2000, Capability: NETWORK_FLARM_NMEA},
		{Conn: nil, Ip: "", Port: 49002, Capability: NETWORK_POSITION_FFSIM | NETWORK_AHRS_FFSIM},
	}
	globalSettings.DEBUG = false
	globalSettings.DisplayTrafficSource = false
	globalSettings.ReplayLog = false //TODO: 'true' for debug builds.
	globalSettings.AHRSLog = false
	globalSettings.IMUMapping = [2]int{-1, 0}
	globalSettings.OwnshipModeS = "F00000"
	globalSettings.DeveloperMode = true
	globalSettings.StaticIps = make([]string, 0)
	globalSettings.NoSleep = false
	globalSettings.SkyDemonAndroidHack = false
	globalSettings.EstimateBearinglessDist = false

	globalSettings.WiFiChannel = 1
	globalSettings.WiFiIPAddress = "192.168.10.1"
	globalSettings.WiFiPassphrase = ""
	globalSettings.WiFiSSID = "stratux"
	globalSettings.WiFiSecurityEnabled = false

	globalSettings.RadarLimits = 2000
	globalSettings.RadarRange = 10
	globalSettings.AltitudeOffset = 0

	globalSettings.PWMDutyMin = 0
}

func readSettings() {
	defaultSettings()

	fd, err := os.Open(configLocation)
	if err != nil {
		log.Printf("can't read settings %s: %s\n", configLocation, err.Error())
		return
	}
	defer fd.Close()
	buf := make([]byte, 10000)
	count, err := fd.Read(buf)
	if err != nil {
		log.Printf("can't read settings %s: %s\n", configLocation, err.Error())
		return
	}
	err = json.Unmarshal(buf[0:count], &globalSettings)
	if err != nil {
		log.Printf("can't read settings %s: %s\n", configLocation, err.Error())
		return
	}
	log.Printf("read in settings.\n")
}

func addSystemError(err error) {
	globalStatus.Errors = append(globalStatus.Errors, err.Error())
}

var systemErrsMutex *sync.Mutex
var systemErrs map[string]string

func addSingleSystemErrorf(ident string, format string, a ...interface{}) {
	systemErrsMutex.Lock()
	if _, ok := systemErrs[ident]; !ok {
		// Error hasn't been thrown yet.
		systemErrs[ident] = fmt.Sprintf(format, a...)
		globalStatus.Errors = append(globalStatus.Errors, systemErrs[ident])
		log.Printf("Added critical system error: %s\n", systemErrs[ident])
	}
	// Do nothing on this call if the error has already been thrown.
	systemErrsMutex.Unlock()
}

func overlayctl(cmd string) {
	out, err := exec.Command("/bin/sh", "/sbin/overlayctl", cmd).Output()
	if err != nil {
		log.Printf("overlayctl error: %s\n%s", err.Error(), out)
	} else {
		log.Printf("overlayctl: %s\n", out)
	}
}

func saveSettings() {
	fd, err := os.OpenFile(configLocation, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0644))
	if err != nil {
		addSingleSystemErrorf("save-settings", "can't save settings %s: %s", configLocation, err.Error())
		return
	}
	defer fd.Close()
	jsonSettings, _ := json.Marshal(&globalSettings)
	fd.Write(jsonSettings)
	fd.Sync()
	log.Printf("wrote settings.\n")
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

/*
	fsWriteTest().
	 Makes a temporary file in 'dir', checks for error. Deletes the file.
*/

func fsWriteTest(dir string) error {
	fn := dir + "/.write_test"
	err := ioutil.WriteFile(fn, []byte("test\n"), 0644)
	if err != nil {
		return err
	}
	err = os.Remove(fn)
	return err
}

func printStats() {
	statTimer := time.NewTicker(30 * time.Second)
	for {
		<-statTimer.C
		var memstats runtime.MemStats
		runtime.ReadMemStats(&memstats)
		log.Printf("stats [started: %s]\n", humanize.RelTime(time.Time{}, stratuxClock.Time, "ago", "from now"))
		log.Printf(" - Disk bytes used = %s (%.1f %%), Disk bytes free = %s (%.1f %%)\n", humanize.Bytes(usage.Used()), 100*usage.Usage(), humanize.Bytes(usage.Free()), 100*(1-usage.Usage()))
		log.Printf(" - CPUTemp=%.02f [%.02f - %.02f] deg C, MemStats.Alloc=%s, MemStats.Sys=%s, totalNetworkMessagesSent=%s\n", globalStatus.CPUTemp, globalStatus.CPUTempMin, globalStatus.CPUTempMax, humanize.Bytes(uint64(memstats.Alloc)), humanize.Bytes(uint64(memstats.Sys)), humanize.Comma(int64(totalNetworkMessagesSent)))
		log.Printf(" - UAT/min %s/%s [maxSS=%.02f%%], ES/min %s/%s, Total traffic targets tracked=%s", humanize.Comma(int64(globalStatus.UAT_messages_last_minute)), humanize.Comma(int64(globalStatus.UAT_messages_max)), float64(maxSignalStrength)/10.0, humanize.Comma(int64(globalStatus.ES_messages_last_minute)), humanize.Comma(int64(globalStatus.ES_messages_max)), humanize.Comma(int64(len(seenTraffic))))
		log.Printf(" - Network data messages sent: %d total, %d nonqueueable.  Network data bytes sent: %d total, %d nonqueueable.\n", globalStatus.NetworkDataMessagesSent, globalStatus.NetworkDataMessagesSentNonqueueable, globalStatus.NetworkDataBytesSent, globalStatus.NetworkDataBytesSentNonqueueable)
		if globalSettings.GPS_Enabled {
			log.Printf(" - Last GPS fix: %s, GPS solution type: %d using %d satellites (%d/%d seen/tracked), NACp: %d, est accuracy %.02f m\n", stratuxClock.HumanizeTime(mySituation.GPSLastFixLocalTime), mySituation.GPSFixQuality, mySituation.GPSSatellites, mySituation.GPSSatellitesSeen, mySituation.GPSSatellitesTracked, mySituation.GPSNACp, mySituation.GPSHorizontalAccuracy)
			log.Printf(" - GPS vertical velocity: %.02f ft/sec; GPS vertical accuracy: %v m\n", mySituation.GPSVerticalSpeed, mySituation.GPSVerticalAccuracy)
		}
		log.Printf(" - Mode-S Distance factors (<5000, <10000, >10000): %f, %f, %f", estimatedDistFactors[0], estimatedDistFactors[1], estimatedDistFactors[2])
		sensorsOutput := make([]string, 0)
		if globalSettings.IMU_Sensor_Enabled {
			sensorsOutput = append(sensorsOutput, fmt.Sprintf("Last IMU read: %s", stratuxClock.HumanizeTime(mySituation.AHRSLastAttitudeTime)))
		}
		if globalSettings.BMP_Sensor_Enabled {
			sensorsOutput = append(sensorsOutput, fmt.Sprintf("Last BMP read: %s", stratuxClock.HumanizeTime(mySituation.BaroLastMeasurementTime)))
		}
		if len(sensorsOutput) > 0 {
			log.Printf("- " + strings.Join(sensorsOutput, ", ") + "\n")
		}
		// Check if we're using more than 95% of the free space. If so, throw a warning (only once).
		if usage.Usage() > 0.95 {
			addSingleSystemErrorf("disk-space", "Disk bytes used = %s (%.1f %%), Disk bytes free = %s (%.1f %%)", humanize.Bytes(usage.Used()), 100*usage.Usage(), humanize.Bytes(usage.Free()), 100*(1-usage.Usage()))
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

// Graceful shutdown. Do everything except for kill the process.
func gracefulShutdown() {
	// Shut down SDRs.
	sdrKill()
	pingKill()

	// Shut down data logging.
	if dataLogStarted {
		closeDataLog()
	}

	pprof.StopCPUProfile()

	//TODO: Any other graceful shutdown functions.

	// Turn off green ACT LED on the Pi.
	ioutil.WriteFile("/sys/class/leds/led0/brightness", []byte("0\n"), 0644)
}

// Close log file handle, open new one.
func handleSIGHUP() {
	logFileHandle.Close()
	fp, err := os.OpenFile(debugLogf, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		addSingleSystemErrorf(debugLogf, "Failed to open '%s': %s", debugLogf, err.Error())
	} else {
		// Keep the logfile handle for later use
		logFileHandle = fp
		mfp := io.MultiWriter(fp, os.Stdout)
		log.SetOutput(mfp)
	}
	log.Printf("signal caught: SIGHUP, handled.\n")
}

func signalWatcher() {
	for {
		sig := <-sigs
		if sig == syscall.SIGHUP {
			handleSIGHUP()
		} else {
			log.Printf("signal caught: %s - shutting down.\n", sig.String())
			gracefulShutdown()
			os.Exit(1)
		}
	}
}

func clearDebugLogFile() {
	if logFileHandle != nil {
		_, err := logFileHandle.Seek(0, 0)
		if err != nil {
			log.Printf("Could not seek to the beginning of the logfile\n")
			return
		} else {
			err2 := logFileHandle.Truncate(0)
			if err2 != nil {
				log.Printf("Could not truncate the logfile\n")
				return
			}
			log.Printf("Logfile truncated\n")
		}
	}
}

func isX86DebugMode() bool {
	return runtime.GOARCH == "i386" || runtime.GOARCH == "amd64"
}

func main() {
	// Catch signals for graceful shutdown.
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go signalWatcher()

	stratuxClock = NewMonotonic() // Start our "stratux clock".

	// Set up mySituation, do it here so logging JSON doesn't panic
	mySituation.muGPS = &sync.Mutex{}
	mySituation.muGPSPerformance = &sync.Mutex{}
	mySituation.muAttitude = &sync.Mutex{}
	mySituation.muBaro = &sync.Mutex{}
	mySituation.muSatellite = &sync.Mutex{}

	// Set up system error tracking.
	systemErrsMutex = &sync.Mutex{}
	systemErrs = make(map[string]string)

	// Set up status.
	globalStatus.Version = stratuxVersion
	globalStatus.Build = stratuxBuild
	globalStatus.Errors = make([]string, 0)
	//FlightBox: detect via presence of /etc/FlightBox file.
	if _, err := os.Stat("/etc/FlightBox"); !os.IsNotExist(err) {
		globalStatus.HardwareBuild = "FlightBox"
		logDirf = logDir_FB
	} else { // if not using the FlightBox config, use "normal" log file locations
		logDirf = logDir
	}
	//Merlin: detect presence of /etc/Merlin file.
	if _, err := os.Stat("/etc/Merlin"); !os.IsNotExist(err) {
		globalStatus.HardwareBuild = "Merlin"
	}
	debugLogf = filepath.Join(logDirf, debugLogFile)
	dataLogFilef = filepath.Join(logDirf, dataLogFile)

	//	replayESFilename := flag.String("eslog", "none", "ES Log filename")
	replayUATFilename := flag.String("uatlog", "none", "UAT Log filename")
	replayFlag := flag.Bool("replay", false, "Replay file flag")
	replaySpeed := flag.Int("speed", 1, "Replay speed multiplier")
	stdinFlag := flag.Bool("uatin", false, "Process UAT messages piped to stdin")

	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")

	flag.Parse()

	timeStarted = time.Now()
	runtime.GOMAXPROCS(runtime.NumCPU()) // redundant with Go v1.5+ compiler

	// Start CPU profile, if requested.
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Writing CPU profile to: %s\n", *cpuprofile)
		pprof.StartCPUProfile(f)
	}

	// Duplicate log.* output to debugLog.
	fp, err := os.OpenFile(debugLogf, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		addSingleSystemErrorf(debugLogf, "Failed to open '%s': %s", debugLogf, err.Error())
	} else {
		defer fp.Close()
		// Keep the logfile handle for later use
		logFileHandle = fp
		mfp := io.MultiWriter(fp, os.Stdout)
		log.SetOutput(mfp)

		// Make sure crash dumps are written to the log as well
		syscall.Dup3(int(fp.Fd()), 2, 0)
	}

	log.Printf("Stratux %s (%s) starting.\n", stratuxVersion, stratuxBuild)

	ADSBTowers = make(map[string]ADSBTower)
	ADSBTowerMutex = &sync.Mutex{}
	msgLog = make([]msg, 0)

	// Start the management interface.
	go managementInterface()

	crcInit() // Initialize CRC16 table.

	sdrInit()
	pingInit()
	initTraffic()

	// Read settings.
	readSettings()

	// Disable replay logs when replaying - so that messages replay data isn't copied into the logs.
	// Override after reading in the settings.
	if *replayFlag == true {
		log.Printf("Replay file %s\n", *replayUATFilename)
		globalSettings.ReplayLog = false
	}

	if globalSettings.DeveloperMode == true {
		log.Printf("Developer mode set\n")
	}

	//FIXME: Only do this if data logging is enabled.
	initDataLog()

	// Start the AHRS sensor monitoring.
	initI2CSensors()

	// Start the GPS external sensor monitoring.
	initGPS()

	// Start the heartbeat message loop in the background, once per second.
	go heartBeatSender()

	// Initialize the (out) network handler.
	initNetwork()

	// Start printing stats periodically to the logfiles.
	go printStats()

	// Extrapolate traffic when no signal is received.
	go trafficInfoExtrapolator()

	// Guesses barometric altitude if we don't have our own baro source by using GnssBaroDiff from other traffic at similar altitude
	go baroAltGuesser()

	// Monitor RPi CPU temp.
	globalStatus.CPUTempMin = common.InvalidCpuTemp
	globalStatus.CPUTempMax = common.InvalidCpuTemp
	go common.CpuTempMonitor(func(cpuTemp float32) {
		globalStatus.CPUTemp = cpuTemp
		if common.IsCPUTempValid(cpuTemp) && ((cpuTemp < globalStatus.CPUTempMin) || !common.IsCPUTempValid(globalStatus.CPUTempMin)) {
			globalStatus.CPUTempMin = cpuTemp
		}
		if common.IsCPUTempValid(cpuTemp) && ((cpuTemp > globalStatus.CPUTempMax) || !common.IsCPUTempValid(globalStatus.CPUTempMax)) {
			globalStatus.CPUTempMax = cpuTemp
		}
	})

	// Start reading from serial UAT radio.
	initUATRadioSerial()

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

