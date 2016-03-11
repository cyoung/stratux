/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New"" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	ry835ai.go: GPS functions, GPS init, AHRS status messages, other external sensor monitoring.
*/

package main

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"bufio"

	"github.com/kidoman/embd"
	_ "github.com/kidoman/embd/host/all"
	"github.com/kidoman/embd/sensor/bmp180"
	"github.com/tarm/serial"

	"os"
	"os/exec"

	"../mpu6050"
)

type SituationData struct {
	mu_GPS     *sync.Mutex
	mu_GPSPerf *sync.Mutex
	// From GPS.
	lastFixSinceMidnightUTC float32 // Time of most recent GPS fix time, UTC seconds
	Lat                     float32 // decimal latitude DD.dddddd (North is positive)
	Lng                     float32 // decimal longitude DDD.dddddd (East is positive)
	quality                 uint8   // 0 = no fix; 1 = GPS/GLONASS; 2 = differential GPS (e.g. WAAS); 6 = dead reckoning
	HeightAboveEllipsoid    float32 // GPS height above WGS84 ellipsoid, ft. This is specified by the GDL90 protocol, but most EFBs use MSL altitude instead. HAE is about 70-100 ft below GPS MSL altitude over most of the US.
	GeoidSep                float32 // geoid separation, ft, MSL minus HAE (used in altitude calculation)
	Satellites              uint16  // satellites used in solution
	SatellitesTracked       uint16  // satellites tracked (almanac data received)
	SatellitesSeen          uint16  // satellites seen (signal received)
	Accuracy                float32 // 95% confidence for horizontal position, meters.
	NACp                    uint8   // NACp categories are defined in AC 20-165A
	Alt                     float32 // Feet MSL
	AccuracyVert            float32 // 95% confidence for vertical position, meters
	GPSVertVel              float32 // GPS vertical velocity, feet per second
	LastFixLocalTime        time.Time
	TrueCourse              uint16
	GPSTurnRate             float64 // calculated GPS rate of turn, degrees per second
	GroundSpeed             uint16  // knots
	LastGroundTrackTime     time.Time
	LastGPSTimeTime         time.Time
	LastNMEAMessage         time.Time // time valid NMEA message last seen

	mu_Attitude *sync.Mutex

	// From ownship Mode S or UAT broadcasts
	OwnshipPressureAlt int32
	OwnshipLat         float32
	OwnshipLng         float32
	OwnshipTail        string
	OwnshipLastSeen    time.Time

	// From BMP180 pressure sensor.
	Temp              float64
	Pressure_alt      float64
	lastTempPressTime time.Time

	// From MPU6050 accel/gyro or other AHRS methods
	Pitch            float64
	Roll             float64
	Gyro_heading     float64
	LastAttitudeTime time.Time
}

/*
myGPSPerfStats used to track short-term position / velocity trends, used to feed dynamic AHRS model. Use floats for better resolution of calculated data.
myGPSMsgStats used to evaluate receive health / ability to communicate
*/
type gpsPerfStats struct {
	stratuxTime   uint64  // time since Stratux start, msec
	nmeaTime      float32 // timestamp from NMEA message
	msgType       string  // NMEA message type
	gsf           float32 // knots
	coursef       float32 // true course [degrees]
	alt           float32 // gps altitude, ft msl
	vv            float32 // vertical velocity, ft/sec
	gpsTurnRate   float64 // calculated turn rate, deg/sec. Right turn is positive.
	gpsPitch      float64 // estimated pitch angle, deg. Calculated from gps ground speed and VV. Equal to flight path angle.
	gpsRoll       float64 // estimated roll angle from turn rate and groundspeed, deg. Assumes airplane in coordinated turns.
	gpsLoadFactor float64 // estimated load factor from turn rate and groundspeed, "gee". Assumes airplane in coordinated turns.
}

type gpsMsgStats struct {
	stratuxTime uint64 // time since Stratux start, msec
	msgType     string // NMEA message type
	msgValid    bool   // was message verified by NMEAChecksum()

}

var gpsPerf gpsPerfStats
var gpsMsg gpsMsgStats

var myGPSPerfStats []gpsPerfStats
var myGPSMsgStats []gpsMsgStats

var serialConfig *serial.Config
var serialPort *serial.Port

var readyToInitGPS bool // TO-DO: replace with channel control to terminate goroutine when complete

/*
chksumUBX returns the two-byte Fletcher algorithm checksum of byte array msg.
This is used in configuration messages for the u-blox GPS. See p. 97 of the
u-blox M8 Receiver Description.
*/

func chksumUBX(msg []byte) []byte {
	ret := make([]byte, 2)
	for i := 0; i < len(msg); i++ {
		ret[0] = ret[0] + msg[i]
		ret[1] = ret[1] + ret[0]
	}
	return ret
}

/*
makeUBXCFG creates a UBX-formatted package consisting of two sync characters,
class, ID, payload length in bytes (2-byte little endian), payload, and checksum.
See p. 95 of the u-blox M8 Receiver Description.
*/
func makeUBXCFG(class, id byte, msglen uint16, msg []byte) []byte {
	ret := make([]byte, 6)
	ret[0] = 0xB5
	ret[1] = 0x62
	ret[2] = class
	ret[3] = id
	ret[4] = byte(msglen & 0xFF)
	ret[5] = byte((msglen >> 8) & 0xFF)
	ret = append(ret, msg...)
	chk := chksumUBX(ret[2:])
	ret = append(ret, chk[0])
	ret = append(ret, chk[1])
	return ret
}

func makeNMEACmd(cmd string) []byte {
	chk_sum := byte(0)
	for i := range cmd {
		chk_sum = chk_sum ^ byte(cmd[i])
	}
	return []byte(fmt.Sprintf("$%s*%02x\x0d\x0a", cmd, chk_sum))
}

/*
initGPSSerial determines which tty port the GPS receiver is installed on,
opens the port at 9600 baud, performs UBX configuration for GNSS,
navigation, rate, message, and UART settings on the GPS receiver, and reopens
the port to begin receiving NMEA messages.
*/

// IN WORK: Detection routines for UBX, and SIRF iv (BU-353), and MTK3339 (Adafruit Ultimate GPS).

func initGPSSerial() bool {
	globalStatus.GPS_detected_type = 0 // reset detection criteria each time we initialize.
	var device string
	baudrate := int(9600)
	isSirfIV := bool(false)

	log.Printf("Configuring GPS\n")

	if _, err := os.Stat("/dev/ttyUSB0"); err == nil { // note -- method is not robust; ttyUSB could be used by any number of USB-to-serial bridges, not just GPS devices.
		isSirfIV = true
		baudrate = 4800
		device = "/dev/ttyUSB0"
	} else if _, err := os.Stat("/dev/ttyACM0"); err == nil {
		device = "/dev/ttyACM0"
	} else if _, err := os.Stat("/dev/ttyAMA0"); err == nil {
		device = "/dev/ttyAMA0"
	} else {
		log.Printf("No suitable device found.\n")
		return false
	}
	log.Printf("Using %s for GPS\n", device)

	// Developer option -- uncomment to allow "hot" configuration of U-blox serial GPS (assuming 38.4 kpbs on warm start)
	/*
		serialConfig = &serial.Config{Name: device, Baud: 38400}
		p, err := serial.OpenPort(serialConfig)
		if err != nil {
			log.Printf("serial port err: %s\n", err.Error())
			return false
		} else { // reset port to 9600 baud for configuration
			cfg1 := make([]byte, 20)
			cfg1[0] = 0x01 // portID.
			cfg1[1] = 0x00 // res0.
			cfg1[2] = 0x00 // res1.
			cfg1[3] = 0x00 // res1.

			//      [   7   ] [   6   ] [   5   ] [   4   ]
			//      0000 0000 0000 0000 1000 0000 1100 0000
			// UART mode. 0 stop bits, no parity, 8 data bits. Little endian order.
			cfg1[4] = 0xC0
			cfg1[5] = 0x08
			cfg1[6] = 0x00
			cfg1[7] = 0x00

			// Baud rate. Little endian order.
			bdrt1 := uint32(9600)
			cfg1[11] = byte((bdrt1 >> 24) & 0xFF)
			cfg1[10] = byte((bdrt1 >> 16) & 0xFF)
			cfg1[9] = byte((bdrt1 >> 8) & 0xFF)
			cfg1[8] = byte(bdrt1 & 0xFF)

			// inProtoMask. NMEA and UBX. Little endian.
			cfg1[12] = 0x03
			cfg1[13] = 0x00

			// outProtoMask. NMEA. Little endian.
			cfg1[14] = 0x02
			cfg1[15] = 0x00

			cfg1[16] = 0x00 // flags.
			cfg1[17] = 0x00 // flags.

			cfg1[18] = 0x00 //pad.
			cfg1[19] = 0x00 //pad.

			p.Write(makeUBXCFG(0x06, 0x00, 20, cfg1))
			p.Close()
		}
	*/
	//-- End developer option */

	// Open port at default baud for config.
	serialConfig = &serial.Config{Name: device, Baud: baudrate, ReadTimeout: time.Millisecond * 2500}
	p, err := serial.OpenPort(serialConfig)
	if err != nil {
		log.Printf("serial port err: %s\n", err.Error())
		return false
	}

	if isSirfIV {
		log.Printf("Using SiRFIV config on %s.\n", device)
		// Enable 38400 baud.
		p.Write(makeNMEACmd("PSRF100,1,38400,8,1,0"))

		baudrate = 38400
		p.Close()

		time.Sleep(250 * time.Millisecond)
		// Re-open port at newly configured baud so we can configure 5Hz messages.
		serialConfig = &serial.Config{Name: device, Baud: baudrate}
		p, err = serial.OpenPort(serialConfig)

		// Enable 5Hz. (To switch back to 1Hz: $PSRF103,00,7,00,0*22)
		p.Write(makeNMEACmd("PSRF103,00,6,00,0"))

		// Enable GGA.
		p.Write(makeNMEACmd("PSRF103,00,00,01,01"))
		// Enable GSA.
		p.Write(makeNMEACmd("PSRF103,02,00,01,01"))
		// Enable RMC.
		p.Write(makeNMEACmd("PSRF103,04,00,01,01"))
		// Enable VTG.
		p.Write(makeNMEACmd("PSRF103,05,00,01,01"))
		// Disable GSV.
		p.Write(makeNMEACmd("PSRF103,03,00,00,01"))

		log.Printf("Finished writing SiRF GPS config to %s. Opening port to test connection.\n", device)
	} else {
		log.Printf("Sent UBX and MTK ident commands at %d\n", stratuxClock.Milliseconds)
		p.Write(makeNMEACmd("PUBX,00")) // probe for u-blox
		p.Write(makeNMEACmd("PMTK605")) // probe for Mediatek
		serialPort = p
		scanner := bufio.NewScanner(serialPort)
		timeout := stratuxClock.Time

		for (globalStatus.GPS_detected_type < 2) && stratuxClock.Since(timeout) < 3*time.Second && scanner.Scan() {
			s := scanner.Text()
			log.Printf("[%d] Serial reader: %s\n", stratuxClock.Milliseconds, s)

			l_valid, validNMEAcs := validateNMEAChecksum(s)
			if !validNMEAcs {
				log.Printf("Data was seen on %s seen during GPS probing, but not NMEA message: %s\n", device, l_valid)
				continue
			}

			if globalStatus.GPS_detected_type == 0 {
				log.Printf("GPS detected: NMEA messages seen.\n")
				globalStatus.GPS_detected_type = GPS_TYPE_NMEA // If this is the first time we see a NMEA message, set our status flag
			}
			x := strings.Split(l_valid, ",")
			if len(x) > 0 {
				if x[0] == "PUBX" { // u-blox proprietary message
					globalStatus.GPS_detected_type = GPS_TYPE_UBX // Only UBX GPS receivers send UBX messages
					log.Printf("GPS detected: u-blox NMEA position message seen.\n")

				} else if x[0] == "PMTK705" { // MTK response to
					globalStatus.GPS_detected_type = GPS_TYPE_MEDIATEK
					pmtk705msg := ""
					if len(x) > 1 {
						pmtk705msg = x[1]
					}
					log.Printf("GPS detected: MediaTek NMEA firmware message (%s) seen.\n", pmtk705msg)
				} else if strings.Contains(x[0], "PMTK") { // any other 1st sting with MTK
					globalStatus.GPS_detected_type = GPS_TYPE_MEDIATEK
					log.Printf("GPS detected: MediaTek other NMEA message (%s) seen.\n", x[0])
				}
			}
		}

		if globalStatus.GPS_detected_type == GPS_TYPE_UBX {
			// Set 10Hz update. Little endian order.
			p.Write(makeUBXCFG(0x06, 0x08, 6, []byte{0x64, 0x00, 0x01, 0x00, 0x01, 0x00}))

			// Set navigation settings.
			nav := make([]byte, 36)
			nav[0] = 0x05 // Set dyn and fixMode only.
			nav[1] = 0x00
			// dyn.
			nav[2] = 0x07 // "Airborne with >2g Acceleration".
			nav[3] = 0x02 // 3D only.

			p.Write(makeUBXCFG(0x06, 0x24, 36, nav))

			// GNSS configuration CFG-GNSS for ublox 7 higher, p. 125 (v8)
			//
			// NOTE: Max position rate = 5 Hz if GPS+GLONASS used.
			// Disable GLONASS to enable 10 Hz solution rate. GLONASS is not used
			// for SBAS (WAAS), so little real-world impact.

			cfgGnss := []byte{0x00, 0x20, 0x20, 0x05}
			gps := []byte{0x00, 0x08, 0x10, 0x00, 0x01, 0x00, 0x01, 0x01}
			sbas := []byte{0x01, 0x02, 0x03, 0x00, 0x01, 0x00, 0x01, 0x01}
			beidou := []byte{0x03, 0x00, 0x10, 0x00, 0x00, 0x00, 0x01, 0x01}
			qzss := []byte{0x05, 0x00, 0x03, 0x00, 0x00, 0x00, 0x01, 0x01}
			glonass := []byte{0x06, 0x04, 0x0E, 0x00, 0x00, 0x00, 0x01, 0x01}
			cfgGnss = append(cfgGnss, gps...)
			cfgGnss = append(cfgGnss, sbas...)
			cfgGnss = append(cfgGnss, beidou...)
			cfgGnss = append(cfgGnss, qzss...)
			cfgGnss = append(cfgGnss, glonass...)
			p.Write(makeUBXCFG(0x06, 0x3E, uint16(len(cfgGnss)), cfgGnss))

			// SBAS configuration for ublox 6 and higher
			p.Write(makeUBXCFG(0x06, 0x16, 8, []byte{0x01, 0x07, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00}))

			// Message output configuration -- disable standard NMEA messages except 1Hz GGA
			//                                             Msg   DDC   UART1 UART2 USB   I2C   Res
			p.Write(makeUBXCFG(0x06, 0x01, 8, []byte{0xF0, 0x00, 0x00, 0x0A, 0x00, 0x0A, 0x00, 0x01})) // GGA
			p.Write(makeUBXCFG(0x06, 0x01, 8, []byte{0xF0, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01})) // GLL
			p.Write(makeUBXCFG(0x06, 0x01, 8, []byte{0xF0, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01})) // GSA
			p.Write(makeUBXCFG(0x06, 0x01, 8, []byte{0xF0, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01})) // GSV
			p.Write(makeUBXCFG(0x06, 0x01, 8, []byte{0xF0, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01})) // RMC
			p.Write(makeUBXCFG(0x06, 0x01, 8, []byte{0xF0, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01})) // VGT
			p.Write(makeUBXCFG(0x06, 0x01, 8, []byte{0xF0, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})) // GRS
			p.Write(makeUBXCFG(0x06, 0x01, 8, []byte{0xF0, 0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})) // GST
			p.Write(makeUBXCFG(0x06, 0x01, 8, []byte{0xF0, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})) // ZDA
			p.Write(makeUBXCFG(0x06, 0x01, 8, []byte{0xF0, 0x09, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})) // GBS
			p.Write(makeUBXCFG(0x06, 0x01, 8, []byte{0xF0, 0x0A, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})) // DTM
			p.Write(makeUBXCFG(0x06, 0x01, 8, []byte{0xF0, 0x0D, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})) // GNS
			p.Write(makeUBXCFG(0x06, 0x01, 8, []byte{0xF0, 0x0E, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})) // ???
			p.Write(makeUBXCFG(0x06, 0x01, 8, []byte{0xF0, 0x0F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})) // VLW

			p.Write(makeUBXCFG(0x06, 0x01, 8, []byte{0xF1, 0x00, 0x01, 0x01, 0x01, 0x01, 0x01, 0x00})) // Ublox,0
			p.Write(makeUBXCFG(0x06, 0x01, 8, []byte{0xF1, 0x03, 0x0A, 0x0A, 0x0A, 0x0A, 0x0A, 0x00})) // Ublox,3
			p.Write(makeUBXCFG(0x06, 0x01, 8, []byte{0xF1, 0x04, 0x0A, 0x0A, 0x0A, 0x0A, 0x0A, 0x00})) // Ublox,4

			// Reconfigure serial port.
			cfg := make([]byte, 20)
			cfg[0] = 0x01 // portID.
			cfg[1] = 0x00 // res0.
			cfg[2] = 0x00 // res1.
			cfg[3] = 0x00 // res1.

			//      [   7   ] [   6   ] [   5   ] [   4   ]
			//	0000 0000 0000 0000 0000 10x0 1100 0000
			// UART mode. 0 stop bits, no parity, 8 data bits. Little endian order.
			cfg[4] = 0xC0
			cfg[5] = 0x08
			cfg[6] = 0x00
			cfg[7] = 0x00

			// Baud rate. Little endian order.
			bdrt := uint32(38400)
			cfg[11] = byte((bdrt >> 24) & 0xFF)
			cfg[10] = byte((bdrt >> 16) & 0xFF)
			cfg[9] = byte((bdrt >> 8) & 0xFF)
			cfg[8] = byte(bdrt & 0xFF)

			// inProtoMask. NMEA and UBX. Little endian.
			cfg[12] = 0x03
			cfg[13] = 0x00

			// outProtoMask. NMEA. Little endian.
			cfg[14] = 0x02
			cfg[15] = 0x00

			cfg[16] = 0x00 // flags.
			cfg[17] = 0x00 // flags.

			cfg[18] = 0x00 //pad.
			cfg[19] = 0x00 //pad.

			p.Write(makeUBXCFG(0x06, 0x00, 20, cfg))
			baudrate = 38400

			log.Printf("Finished writing u-blox GPS config to %s. Opening port to test connection.\n", device)

		} else if globalStatus.GPS_detected_type == GPS_TYPE_MEDIATEK {
			// send GGA, VTG, RMC, GSA once per second. Send GSV once every five (?)
			p.Write(makeNMEACmd("PMTK314,0,1,1,1,1,5,0,0,0,0,0,0,0,0,0,0,0,0,0")) // GLL, RMC, VTG, GGA, GSA, GSV

			// set WAAS
			p.Write(makeNMEACmd("PMTK301,2"))
			p.Write(makeNMEACmd("PMTK513,1"))

			// set sample rate to 10 Hz
			p.Write(makeNMEACmd("PMTK220,100"))

			// set baud rate to 38400
			p.Write(makeNMEACmd("PMTK251,38400"))
			baudrate = 38400

			log.Printf("Finished writing MediaTek GPS config to %s. Opening port to test connection.\n", device)

		} else if globalStatus.GPS_detected_type == GPS_TYPE_NMEA {
			//baudrate = 9600
			log.Printf("Using generic NMEA GPS support at %d baud on %s. Opening port to test connection.\n", baudrate, device)
			// TO-DO: SIRF detection. Need hardware to run debug.
			// For now, do nothing. Keep baud rate at 9600.

		} else {
			// No messages detected. If a GPS is connected, this is usually seen on hot-starts.
			// Wait until the gpsSerialReader starts to do detection.
			baudrate = 38400
		}

	}
	p.Close()

	time.Sleep(250 * time.Millisecond)
	// Re-open port at newly configured baud so we can read messages. ReadTimeout is set to keep from blocking the gpsSerialReader() on misconfigures or ttyAMA disconnects
	serialConfig = &serial.Config{Name: device, Baud: baudrate, ReadTimeout: time.Millisecond * 2500}
	p, err = serial.OpenPort(serialConfig)
	if err != nil {
		log.Printf("serial port err: %s\n", err.Error())
		return false
	}

	// TO-DO: Verify port is sending correct messages.

	serialPort = p
	return true
}

// func validateNMEAChecksum determines if a string is a properly formatted NMEA sentence with a valid checksum.
//
// If the input string is valid, output is the input stripped of the "$" token and checksum, along with a boolean 'true'
// If the input string is the incorrect format, the checksum is missing/invalid, or checksum calculation fails, an error string and
// boolean 'false' are returned
//
// Checksum is calculated as XOR of all bytes between "$" and "*"

func validateNMEAChecksum(s string) (string, bool) {
	//validate format. NMEA sentences start with "$" and end in "*xx" where xx is the XOR value of all bytes between
	if !(strings.HasPrefix(s, "$") && strings.Contains(s, "*")) {
		return "Invalid NMEA message", false
	}

	// strip leading "$" and split at "*"
	s_split := strings.Split(strings.TrimPrefix(s, "$"), "*")
	s_out := s_split[0]
	s_cs := s_split[1]

	if len(s_cs) < 2 {
		return "Missing checksum. Fewer than two bytes after asterisk", false
	}

	cs, err := strconv.ParseUint(s_cs[:2], 16, 8)
	if err != nil {
		return "Invalid checksum", false
	}

	cs_calc := byte(0)
	for i := range s_out {
		cs_calc = cs_calc ^ byte(s_out[i])
	}

	if cs_calc != byte(cs) {
		return fmt.Sprintf("Checksum failed. Calculated %#X; expected %#X", cs_calc, cs), false
	}

	return s_out, true
}

//  Only count this heading if a "sustained" >7 kts is obtained. This filters out a lot of heading
//  changes while on the ground and "movement" is really only changes in GPS fix as it settles down.
//TODO: Some more robust checking above current and last speed.
//TODO: Dynamic adjust for gain based on groundspeed
func setTrueCourse(groundSpeed, trueCourse uint16) {
	if myMPU6050 != nil && globalStatus.RY835AI_connected && globalSettings.AHRS_Enabled {
		if mySituation.GroundSpeed >= 7 && groundSpeed >= 7 {
			myMPU6050.ResetHeading(float64(trueCourse), 0.10)
		}
	}
}

/*
calcGPSAttitude estimates turn rate, pitch, and roll based on recent GPS groundspeed, track, and altitude / vertical speed.

Method uses stored performance statistics from myGPSPerfStats[]. Calculation is based on most recent 1.5 seconds of data,
assuming 10 Hz sampling frequency.



(c) 2016 AvSquirrel (https://github.com/AvSquirrel) . All rights reserved.
Distributable under the terms of the "BSD-New" License that can be found in
the LICENSE file, herein included as part of this header.
*/

func calcGPSAttitude() bool {
	// check slice length. Return error if empty set or set zero values
	mySituation.mu_GPSPerf.Lock()
	defer mySituation.mu_GPSPerf.Unlock()
	length := len(myGPSPerfStats)
	index := length - 1

	if length == 0 {
		log.Printf("myGPSPerfStats is empty set. Not calculating attitude.\n")
		return false
	} else if length == 1 {
		//log.Printf("myGPSPerfStats has one data point. Setting statistics to zero.\n")
		myGPSPerfStats[index].gpsTurnRate = 0
		myGPSPerfStats[index].gpsPitch = 0
		myGPSPerfStats[index].gpsRoll = 0
		return false
	}

	// check if GPS data was put in the structure more than three seconds ago -- this shouldn't happen unless something is wrong.
	if (stratuxClock.Milliseconds - myGPSPerfStats[index].stratuxTime) > 3000 {
		myGPSPerfStats[index].gpsTurnRate = 0
		myGPSPerfStats[index].gpsPitch = 0
		myGPSPerfStats[index].gpsRoll = 0
		log.Printf("GPS attitude: GPS data is more than three seconds old. Setting attitude to zero.\n")
		return false
	}

	// check time interval between samples
	t1 := myGPSPerfStats[index].nmeaTime
	t0 := myGPSPerfStats[index-1].nmeaTime
	dt := t1 - t0

	// first time error case: index is more than three seconds ahead of index-1
	if dt > 3 {
		log.Printf("GPS attitude: Can't calculate GPS attitude. Reference data is old. dt = %v\n", dt)
		return false
	}

	// second case: index is behind index-1. This could be result of day rollover. If time is within n seconds of UTC,
	// we rebase to the previous day, and will re-rebase the entire slice forward to the current day once all values roll over.
	// TO-DO: Validate by testing at 0000Z
	if dt < 0 {
		log.Printf("GPS attitude: Current GPS time (%.2f) is older than last GPS time (%.2f). Checking for 0000Z rollover.\n", t1, t0)
		if myGPSPerfStats[index-1].nmeaTime > 86300 && myGPSPerfStats[index].nmeaTime < 100 { // be generous with the time window at rollover
			myGPSPerfStats[index].nmeaTime += 86400
		} else {
			// time decreased, but not due to a recent rollover. Something odd is going on.
			log.Printf("GPS attitude: Time isn't near 0000Z. Unknown reason for offset. Can't calculate GPS attitude.\n")
			return false
		}

		// check time array to see if all timestamps are > 86401 seconds since midnight
		var tempTime []float64
		tempTime = make([]float64, length, length)
		for i := 0; i < length; i++ {
			tempTime[i] = float64(myGPSPerfStats[i].nmeaTime)
		}
		minTime, _ := arrayMin(tempTime)
		if minTime > 86401.0 {
			log.Printf("GPS attitude: Rebasing GPS time since midnight to current day.\n")
			for i := 0; i < length; i++ {
				myGPSPerfStats[i].nmeaTime -= 86400
			}
		}

		// Verify adjustment
		dt = myGPSPerfStats[index].nmeaTime - myGPSPerfStats[index-1].nmeaTime
		log.Printf("GPS attitude: New dt = %f\n", dt)
		if dt > 3 {
			log.Printf("GPS attitude: Can't calculate GPS attitude. Reference data is old. dt = %v\n", dt)
			return false
		} else if dt < 0 {
			log.Printf("GPS attitude: Something went wrong rebasing the time.\n")
			return false
		}

	}

	// otherwise if all bounds checks are good, process the data.

	// temp vars
	var headingAvg, dh, v_x, v_z, a_c, omega, slope, intercept float64
	var tempHdg, tempHdgUnwrapped, tempHdgTime, tempSpeed, tempVV, tempSpeedTime, tempRegWeights []float64 // temporary arrays for regression calculation
	var valid bool
	var lengthHeading, lengthSpeed int

	center := float64(myGPSPerfStats[index].nmeaTime) // current time for calculating regression weights
	halfwidth := float64(1.5)                         // width of regression evaluation window. Default of 1.5 seconds for 10 Hz sampling; can increase to 2.0 sec @ 5 Hz or 5 sec @ 1 Hz
	//detectedFreq := float64(10.0)  // TO-DO

	if globalStatus.GPS_detected_type == GPS_TYPE_UBX { // UBX reports vertical speed, so we can just walk through all of the PUBX messages in order
		// Speed and VV. Use all values in myGPSPerfStats; perform regression.
		tempSpeedTime = make([]float64, length, length) // all are length of original slice
		tempSpeed = make([]float64, length, length)
		tempVV = make([]float64, length, length)
		tempRegWeights = make([]float64, length, length)

		for i := 0; i < length; i++ {
			tempSpeed[i] = float64(myGPSPerfStats[i].gsf)
			tempVV[i] = float64(myGPSPerfStats[i].vv)
			tempSpeedTime[i] = float64(myGPSPerfStats[i].nmeaTime)
			tempRegWeights[i] = triCubeWeight(center, halfwidth, tempSpeedTime[i])
		}

		// Groundspeed regression estimate.
		slope, intercept, valid = linRegWeighted(tempSpeedTime, tempSpeed, tempRegWeights)
		if !valid {
			log.Printf("GPS attitude: Error calculating speed regression from UBX position messages")
			return false
		} else {
			v_x = (slope*float64(myGPSPerfStats[index].nmeaTime) + intercept) * 1.687810 // units are knots, converted to feet/sec
		}

		// Vertical speed regression estimate.
		slope, intercept, valid = linRegWeighted(tempSpeedTime, tempVV, tempRegWeights)
		if !valid {
			log.Printf("GPS attitude: Error calculating vertical speed regression from UBX position messages")
			return false
		} else {
			v_z = (slope*float64(myGPSPerfStats[index].nmeaTime) + intercept) // units are feet per sec; no conversion needed
		}

	} else { // If we need to parse standard NMEA messages, determine if it's RMC or GGA, then fill the temporary slices accordingly. Need to pull from multiple message types since GGA doesn't do course or speed; VTG / RMC don't do altitude, etc. Grrr.
		halfwidth = 2.5 // SIRF configuration is 5 Hz, so extend the timebase a bit. This will also allow basic calculation to be done for 1 Hz generic NMEA.
		// TODO
		v_x = float64(myGPSPerfStats[index].gsf * 1.687810) //FIXME. Pull current value from RMC message and convert to ft/sec.
		v_z = 0                                             // FIXME

		// first, parse groundspeed from RMC messages.
		tempSpeedTime = make([]float64, 0)
		tempSpeed = make([]float64, 0)
		tempRegWeights = make([]float64, 0)

		for i := 0; i < length; i++ {
			if myGPSPerfStats[i].msgType == "GPRMC" || myGPSPerfStats[i].msgType == "GNRMC" {
				tempSpeed = append(tempSpeed, float64(myGPSPerfStats[i].gsf))
				tempSpeedTime = append(tempSpeedTime, float64(myGPSPerfStats[i].nmeaTime))
				tempRegWeights = append(tempRegWeights, triCubeWeight(center, halfwidth, float64(myGPSPerfStats[i].nmeaTime)))
			}
		}
		if lengthSpeed == 0 {
			log.Printf("GPS Attitude: No groundspeed data could be parsed from NMEA RMC messages\n")
			return false
		} else if lengthSpeed == 1 {
			v_x = tempSpeed[0] * 1.687810
		} else {
			slope, intercept, valid = linRegWeighted(tempSpeedTime, tempSpeed, tempRegWeights)
			if !valid {
				log.Printf("GPS attitude: Error calculating speed regression from NMEA RMC position messages")
				return false
			} else {
				v_x = (slope*float64(myGPSPerfStats[index].nmeaTime) + intercept) * 1.687810 // units are knots, converted to feet/sec
			}
		}

		// next, calculate vertical velocity from GGA altitude data.
		tempSpeedTime = make([]float64, 0)
		tempVV = make([]float64, 0)
		tempRegWeights = make([]float64, 0)

		for i := 0; i < length; i++ {
			if myGPSPerfStats[i].msgType == "GPGGA" || myGPSPerfStats[i].msgType == "GNGGA" {
				tempVV = append(tempVV, float64(myGPSPerfStats[i].alt))
				tempSpeedTime = append(tempSpeedTime, float64(myGPSPerfStats[i].nmeaTime))
				tempRegWeights = append(tempRegWeights, triCubeWeight(center, halfwidth, float64(myGPSPerfStats[i].nmeaTime)))
			}
		}
		if lengthSpeed < 2 {
			log.Printf("GPS Attitude: Not enough points to calculate vertical speed from NMEA GGA messages\n")
			return false
		} else {
			slope, _, valid = linRegWeighted(tempSpeedTime, tempVV, tempRegWeights)
			if !valid {
				log.Printf("GPS attitude: Error calculating vertical speed regression from NMEA GGA messages")
				return false
			} else {
				v_z = slope // units are feet/sec

			}
		}

	}

	// If we're going too slow for processNMEALine() to give us valid heading data, there's no sense in trying to parse it.
	// However, this function should still return a valid level attitude so we don't get the "red X of death" on our AHRS display.
	// This will also eliminate most of the nuisance error message from the turn rate calculation.
	if v_x < 6 { // ~3.55 knots

		myGPSPerfStats[index].gpsPitch = 0
		myGPSPerfStats[index].gpsRoll = 0
		myGPSPerfStats[index].gpsTurnRate = 0
		myGPSPerfStats[index].gpsLoadFactor = 1.0
		mySituation.GPSTurnRate = 0

		if globalSettings.VerboseLogs {
			// Output format:GPSAtttiude,nmeaTime,msg_type,GS,Course,Alt,VV,filtered_GS,filtered_course,turn rate,filtered_vv,pitch, roll,load_factor
			log.Printf(",GPSAttitude,%.2f,%s,%0.3f,%0.3f,%0.3f,%0.3f,%0.3f,%0.3f,%0.3f,%0.3f,%0.3f,%0.3f,%0.3f\n", myGPSPerfStats[index].nmeaTime, myGPSPerfStats[index].msgType, myGPSPerfStats[index].gsf, myGPSPerfStats[index].coursef, myGPSPerfStats[index].alt, myGPSPerfStats[index].vv, v_x/1.687810, headingAvg, myGPSPerfStats[index].gpsTurnRate, v_z, myGPSPerfStats[index].gpsPitch, myGPSPerfStats[index].gpsRoll, myGPSPerfStats[index].gpsLoadFactor)
		}

		return true
	}

	// Heading.  Same method used for UBX and generic.
	// First, walk through the PerfStats and parse only valid heading data.
	//log.Printf("Raw heading data:")
	for i := 0; i < length; i++ {
		//log.Printf("%.1f,",myGPSPerfStats[i].coursef)
		if myGPSPerfStats[i].coursef >= 0 { // negative values are used to flag invalid / unavailable course
			tempHdg = append(tempHdg, float64(myGPSPerfStats[i].coursef))
			tempHdgTime = append(tempHdgTime, float64(myGPSPerfStats[i].nmeaTime))
		}
	}
	//log.Printf("\n")
	//log.Printf("tempHdg: %v\n", tempHdg)

	// Next, unwrap the heading so we don't mess up the regression by fitting a line across the 0/360 deg discontinutiy
	lengthHeading = len(tempHdg)
	tempHdgUnwrapped = make([]float64, lengthHeading, lengthHeading)
	tempRegWeights = make([]float64, lengthHeading, lengthHeading)

	if lengthHeading > 1 {
		tempHdgUnwrapped[0] = tempHdg[0]
		tempRegWeights[0] = triCubeWeight(center, halfwidth, tempHdgTime[0])
		for i := 1; i < lengthHeading; i++ {
			tempRegWeights[i] = triCubeWeight(center, halfwidth, tempHdgTime[i])
			if math.Abs(tempHdg[i]-tempHdg[i-1]) < 180 { // case 1: if angle change is less than 180 degrees, use the same reference system
				tempHdgUnwrapped[i] = tempHdgUnwrapped[i-1] + tempHdg[i] - tempHdg[i-1]
			} else if tempHdg[i] > tempHdg[i-1] { // case 2: heading has wrapped around from NE to NW. Subtract 360 to keep consistent with previous data.
				tempHdgUnwrapped[i] = tempHdgUnwrapped[i-1] + tempHdg[i] - tempHdg[i-1] - 360
			} else { // case 3:  heading has wrapped around from NW to NE. Add 360 to keep consistent with previous data.
				tempHdgUnwrapped[i] = tempHdgUnwrapped[i-1] + tempHdg[i] - tempHdg[i-1] + 360
			}
		}
	} else { //
		if globalSettings.VerboseLogs {
			log.Printf("GPS attitude: Can't calculate turn rate with less than two points.\n")
		}
		return false
	}

	// Finally, calculate turn rate as the slope of the weighted linear regression of unwrapped heading.
	slope, intercept, valid = linRegWeighted(tempHdgTime, tempHdgUnwrapped, tempRegWeights)

	if !valid {
		log.Printf("GPS attitude: Regression error calculating turn rate")
		return false
	} else {
		headingAvg = slope*float64(myGPSPerfStats[index].nmeaTime) + intercept
		dh = slope // units are deg per sec; no conversion needed here
		//log.Printf("Calculated heading and turn rate: %.3f degrees, %.3f deg/sec\n",headingAvg,dh)
	}

	myGPSPerfStats[index].gpsTurnRate = dh
	mySituation.GPSTurnRate = dh

	// pitch angle -- or to be more pedantic, glide / climb angle, since we're just looking a rise-over-run.
	// roll angle, based on turn rate and ground speed. Only valid for coordinated flight. Differences between airspeed and groundspeed will trip this up.
	if v_x > 20 { // reduce nuisance 'bounce' at low speeds. 20 ft/sec = 11.9 knots.
		myGPSPerfStats[index].gpsPitch = math.Atan2(v_z, v_x) * 180.0 / math.Pi

		/*
			Governing equations for roll calculations

			Physics tells us that
				a_z = g     (in steady-state flight -- climbing, descending, or level -- this is gravity. 9.81 m/s^2 or 32.2 ft/s^2)
				a_c = v^2/r (centripetal acceleration)

			We don't know r. However, we do know the tangential velocity (v) and angular velocity (omega). Express omega in radians per unit time, and

				v = omega*r

			By substituting and rearranging terms:

				a_c = v^2 / (v / omega)
				a_c = v*omega

			Free body diagram time!

				   /|
			  a_r / |  a_z
				 /__|
			   X   a_c
				\_________________ [For the purpose of this comment, " X" is an airplane in a 20 degree bank. Use your imagination, mkay?)

			Resultant acceleration a_r is what the wings feel; a_r/a_z = load factor. Anyway, trig out the bank angle:

				bank angle = atan(a_c/a_z)
						   = atan(v*omega/g)

				wing loading = sqrt(a_c^2 + a_z^2) / g

		*/

		g := 32.174                                        // ft-s^-2
		omega = radians(myGPSPerfStats[index].gpsTurnRate) // need radians/sec
		a_c = v_x * omega
		myGPSPerfStats[index].gpsRoll = math.Atan2(a_c, g) * 180 / math.Pi // output is degrees
		myGPSPerfStats[index].gpsLoadFactor = math.Sqrt(a_c*a_c+g*g) / g
	} else {
		myGPSPerfStats[index].gpsPitch = 0
		myGPSPerfStats[index].gpsRoll = 0
		myGPSPerfStats[index].gpsLoadFactor = 1
	}

	if globalSettings.VerboseLogs {
		// Output format:,GPSAtttiude,nmeaTime,msg_type,GS,Course,Alt,VV,filtered_GS,filtered_course,turn rate,filtered_vv,pitch, roll,load_factor
		log.Printf(",GPSAttitude,%.2f,%s,%0.3f,%0.3f,%0.3f,%0.3f,%0.3f,%0.3f,%0.3f,%0.3f,%0.3f,%0.3f,%0.3f\n", myGPSPerfStats[index].nmeaTime, myGPSPerfStats[index].msgType, myGPSPerfStats[index].gsf, myGPSPerfStats[index].coursef, myGPSPerfStats[index].alt, myGPSPerfStats[index].vv, v_x/1.687810, headingAvg, myGPSPerfStats[index].gpsTurnRate, v_z, myGPSPerfStats[index].gpsPitch, myGPSPerfStats[index].gpsRoll, myGPSPerfStats[index].gpsLoadFactor)
	}
	return true
}

func calculateNACp(accuracy float32) uint8 {
	ret := uint8(0)

	if accuracy < 3 {
		ret = 11
	} else if accuracy < 10 {
		ret = 10
	} else if accuracy < 30 {
		ret = 9
	} else if accuracy < 92.6 {
		ret = 8
	} else if accuracy < 185.2 {
		ret = 7
	} else if accuracy < 555.6 {
		ret = 6
	}

	return ret
}

/*
processNMEALine parses NMEA-0183 formatted strings against several message types.

Standard messages supported: RMC GGA VTG GSA
U-blox proprietary messages: PUBX,00 PUBX,03 PUBX,04

return is false if errors occur during parse, or if GPS position is invalid
return is true if parse occurs correctly and position is valid.

*/

func processNMEALine(l string) bool {
	replayLog(l, MSGCLASS_GPS)

	// Local variables
	thisGpsPerf := gpsPerf                              // write to myGPSPerfStats at end of function IFF
	thisGpsPerf.coursef = -999.9                        // default value; indicates invalid heading to regression calculation
	thisGpsPerf.stratuxTime = stratuxClock.Milliseconds // Only needed for gross indexing
	updateGPSPerf := false                              // change to true when position or vector info is read

	// append to gps message stat
	lenGPSMsgStats := len(myGPSMsgStats)
	if lenGPSMsgStats > 999 {
		myGPSMsgStats = myGPSMsgStats[(lenGPSMsgStats - 999):]
	}
	myGPSMsgStats = append(myGPSMsgStats, gpsMsg)
	indexGPSMsgStats := len(myGPSMsgStats) - 1

	myGPSMsgStats[indexGPSMsgStats].stratuxTime = stratuxClock.Milliseconds
	l_valid, validNMEAcs := validateNMEAChecksum(l)
	if !validNMEAcs {
		log.Printf("GPS error. Invalid NMEA string: %s\n", l_valid) // remove log message once validation complete
		myGPSMsgStats[indexGPSMsgStats].msgValid = false
		return false
	}
	if globalStatus.GPS_detected_type == 0 {
		log.Printf("GPS detected: NMEA messages seen.\n")
		globalStatus.GPS_detected_type = GPS_TYPE_NMEA // If this is the first time we see a NMEA message, set our status flag
	}

	x := strings.Split(l_valid, ",")

	mySituation.LastNMEAMessage = stratuxClock.Time

	myGPSMsgStats[indexGPSMsgStats].msgValid = true
	myGPSMsgStats[indexGPSMsgStats].msgType = x[0]

	if x[0] == "PUBX" { // UBX proprietary message
		if globalStatus.GPS_detected_type != GPS_TYPE_UBX {
			globalStatus.GPS_detected_type = GPS_TYPE_UBX // Only UBX GPS receivers send UBX messages
			log.Printf("GPS detected: u-blox NMEA position message seen.\n")
		}

		myGPSMsgStats[indexGPSMsgStats].msgType = myGPSMsgStats[indexGPSMsgStats].msgType + x[1]
		if x[1] == "00" { // position message
			if len(x) < 20 {
				return false
			}

			thisGpsPerf.msgType = x[0] + x[1]

			//mySituation.mu_GPS.Lock()
			//defer mySituation.mu_GPS.Unlock()

			// Do the accuracy / quality fields first to prevent invalid position etc. from being sent downstream

			// field 8 = nav status
			// DR = dead reckoning, G2= 2D GPS, G3 = 3D GPS, D2= 2D diff, D3 = 3D diff, RK = GPS+DR, TT = time only

			okReturn := true

			if x[8] == "D2" || x[8] == "D3" {
				mySituation.quality = 2
			} else if x[8] == "G2" || x[8] == "G3" {
				mySituation.quality = 1
			} else if x[8] == "DR" || x[8] == "RK" {
				mySituation.quality = 6
			} else if x[8] == "NF" {
				mySituation.quality = 0
				okReturn = false //  better to have no data than wrong data
			} else {
				mySituation.quality = 0
				okReturn = false //  better to have no data than wrong data
			}

			// field 9 = horizontal accuracy, m
			hAcc, err := strconv.ParseFloat(x[9], 32)
			if err != nil {
				okReturn = false
			}
			mySituation.Accuracy = float32(hAcc * 2) // UBX reports 1-sigma variation; NACp is 95% confidence (2-sigma)

			// NACp estimate.
			mySituation.NACp = calculateNACp(mySituation.Accuracy)

			// field 10 = vertical accuracy, m
			vAcc, err := strconv.ParseFloat(x[10], 32)
			if err != nil {
				okReturn = false
			}
			mySituation.AccuracyVert = float32(vAcc * 2) // UBX reports 1-sigma variation; we want 95% confidence

			if !okReturn {
				return false
			}

			// field 2 = time
			if len(x[2]) < 8 {
				return false
			}
			hr, err1 := strconv.Atoi(x[2][0:2])
			min, err2 := strconv.Atoi(x[2][2:4])
			sec, err3 := strconv.ParseFloat(x[2][4:], 32)
			if err1 != nil || err2 != nil || err3 != nil {
				return false
			}

			mySituation.lastFixSinceMidnightUTC = float32((3600*hr)+(60*min)) + float32(sec)
			thisGpsPerf.nmeaTime = mySituation.lastFixSinceMidnightUTC

			//log.Printf("Latest GPS time is %02d:%02d:%06.3f UTC. Time since midnight UTC is %06.3f\n", hr, min, sec, mySituation.lastFixSinceMidnightUTC)

			// field 3-4 = lat

			if len(x[3]) < 4 {
				return false
			}

			hr, err1 = strconv.Atoi(x[3][0:2])
			minf, err2 := strconv.ParseFloat(x[3][2:], 32)
			if err1 != nil || err2 != nil {
				return false
			}

			mySituation.Lat = float32(hr) + float32(minf/60.0)
			if x[4] == "S" { // South = negative.
				mySituation.Lat = -mySituation.Lat
			}

			// field 5-6 = lon
			if len(x[5]) < 5 {
				return false
			}
			hr, err1 = strconv.Atoi(x[5][0:3])
			minf, err2 = strconv.ParseFloat(x[5][3:], 32)
			if err1 != nil || err2 != nil {
				return false
			}

			mySituation.Lng = float32(hr) + float32(minf/60.0)
			if x[6] == "W" { // West = negative.
				mySituation.Lng = -mySituation.Lng
			}

			// field 7 = height above ellipsoid, m

			hae, err1 := strconv.ParseFloat(x[7], 32)
			if err1 != nil {
				return false
			}
			alt := float32(hae*3.28084) - mySituation.GeoidSep        // convert to feet and offset by geoid separation
			mySituation.HeightAboveEllipsoid = float32(hae * 3.28084) // feet
			mySituation.Alt = alt
			thisGpsPerf.alt = alt

			mySituation.LastFixLocalTime = stratuxClock.Time

			// field 11 = groundspeed, km/h
			groundspeed, err := strconv.ParseFloat(x[11], 32)
			if err != nil {
				return false
			}
			groundspeed = groundspeed * 0.540003 // convert to knots
			mySituation.GroundSpeed = uint16(groundspeed)
			thisGpsPerf.gsf = float32(groundspeed)

			// field 12 = track, deg
			trueCourse := uint16(0)
			tc, err := strconv.ParseFloat(x[12], 32)
			if err != nil {
				return false
			}

			if groundspeed > 3 { // TO-DO: use average groundspeed over last n seconds to avoid random "jumps"
				trueCourse = uint16(tc)
				setTrueCourse(uint16(groundspeed), trueCourse)
				mySituation.TrueCourse = uint16(trueCourse)
				thisGpsPerf.coursef = float32(tc)
			} else {
				thisGpsPerf.coursef = -999.9 // regression will skip negative values
				// Negligible movement. Don't update course, but do use the slow speed.
				// TO-DO: use average course over last n seconds?
			}
			mySituation.LastGroundTrackTime = stratuxClock.Time

			// field 13 = vertical velocity, m/s
			vv, err := strconv.ParseFloat(x[13], 32)
			if err != nil {
				return false
			}
			mySituation.GPSVertVel = float32(vv * -3.28084) // convert to ft/sec and positive = up
			thisGpsPerf.vv = mySituation.GPSVertVel

			// field 14 = age of diff corrections

			// field 18 = number of satellites
			sat, err1 := strconv.Atoi(x[18])
			if err1 != nil {
				return false
			}
			mySituation.Satellites = uint16(sat)

			updateGPSPerf = true

		} else if x[1] == "03" { // satellite status message

			// field 2 = number of satellites tracked
			satSeen := 0 // satellites seen (signal present)
			satTracked, err := strconv.Atoi(x[2])
			if err != nil {
				return false
			}
			mySituation.SatellitesTracked = uint16(satTracked)

			// fields 3-8 are repeated block

			for i := 0; i < satTracked; i++ {
				j := 7 + 6*i
				if j < len(x) {
					if x[j] != "" {
						satSeen++
					}
				}
			}

			mySituation.SatellitesSeen = uint16(satSeen)
			// log.Printf("Satellites with signal: %v\n",mySituation.SatellitesSeen)

			/* Reference for future constellation tracking
						for i:= 0; i < satTracked; i++ {
							x[3+6*i] // sv number
							x[4+6*i] // status [ U | e | - ] for used / ephemeris / not used
			                                x[5+6*i] // azimuth, deg, 0-359
			                                x[6+6*i] // elevation, deg, 0-90
			                                x[7+6*i] // signal strength dB-Hz
			                                x[8+6*i] // lock time, sec, 0-64
			*/

		} else if x[1] == "04" { // clock message
			// field 5 is UTC week (epoch = 1980-JAN-06). If this is invalid, do not parse date / time
			utcWeek, err0 := strconv.Atoi(x[5])
			if err0 != nil {
				// log.Printf("Error reading GPS week\n")
				return false
			}
			if utcWeek < 1877 || utcWeek >= 32767 { // unless we're in a flying Delorean, UTC dates before 2016-JAN-01 are not valid. Check underflow condition as well.
				log.Printf("GPS week # %v out of scope; not setting time and date\n", utcWeek)
				return false
			} /* else {
				log.Printf("GPS week # %v valid; evaluate time and date\n", utcWeek) //debug option
			} */

			// field 2 is UTC time
			if len(x[2]) < 7 {
				return false
			}
			hr, err1 := strconv.Atoi(x[2][0:2])
			min, err2 := strconv.Atoi(x[2][2:4])
			sec, err3 := strconv.ParseFloat(x[2][4:], 32)
			if err1 != nil || err2 != nil || err3 != nil {
				return false
			}

			mySituation.lastFixSinceMidnightUTC = float32(3600*hr+60*min) + float32(sec)

			// field 3 is date

			if len(x[3]) == 6 {
				// Date of Fix, i.e 191115 =  19 November 2015 UTC  field 9
				gpsTimeStr := fmt.Sprintf("%s %02d:%02d:%06.3f", x[3], hr, min, sec)
				gpsTime, err := time.Parse("020106 15:04:05.000", gpsTimeStr)
				if err == nil {
					mySituation.LastGPSTimeTime = stratuxClock.Time
					// log.Printf("GPS time is: %s\n", gpsTime) //debug
					if time.Since(gpsTime) > 3*time.Second || time.Since(gpsTime) < -3*time.Second {
						setStr := gpsTime.Format("20060102 15:04:05.000") + " UTC"
						log.Printf("setting system time to: '%s'\n", setStr)
						if err := exec.Command("date", "-s", setStr).Run(); err != nil {
							log.Printf("Set Date failure: %s error\n", err)
						} else {
							log.Printf("Time set from GPS. Current time is %v\n", time.Now())
						}
					}
				}
			}

		}

		// otherwise parse the NMEA standard messages as a compatibility option for SIRF, generic NMEA, etc.
	} else if (x[0] == "GNVTG") || (x[0] == "GPVTG") { // Ground track information.
		if len(x) < 9 { // Reduce from 10 to 9 to allow parsing by devices pre-NMEA v2.3
			return false
		}
		//mySituation.mu_GPS.Lock()
		//defer mySituation.mu_GPS.Unlock()

		groundspeed, err := strconv.ParseFloat(x[5], 32) // Knots.
		if err != nil {
			return false
		}
		mySituation.GroundSpeed = uint16(groundspeed)

		trueCourse := uint16(0)
		tc, err := strconv.ParseFloat(x[1], 32)
		if err != nil {
			return false
		}

		if groundspeed > 3 { // TO-DO: use average groundspeed over last n seconds to avoid random "jumps"
			trueCourse = uint16(tc)
			setTrueCourse(uint16(groundspeed), trueCourse)
			mySituation.TrueCourse = uint16(trueCourse)
		} else {
			// Negligible movement. Don't update course, but do use the slow speed.
			// TO-DO: pass average course over last n seconds to setTrueCourse ?
		}
		mySituation.LastGroundTrackTime = stratuxClock.Time

	} else if (x[0] == "GNGGA") || (x[0] == "GPGGA") { // GPS fix.
		if len(x) < 15 {
			return false
		}
		//mySituation.mu_GPS.Lock()
		//defer mySituation.mu_GPS.Unlock()

		// Quality indicator.
		q, err1 := strconv.Atoi(x[6])
		if err1 != nil {
			return false
		}
		mySituation.quality = uint8(q) // 1 = 3D GPS; 2 = DGPS (SBAS /WAAS)

		// Timestamp.
		if len(x[1]) < 7 {
			return false
		}
		hr, err1 := strconv.Atoi(x[1][0:2])
		min, err2 := strconv.Atoi(x[1][2:4])
		sec, err3 := strconv.ParseFloat(x[1][4:], 32)
		if err1 != nil || err2 != nil || err3 != nil {
			return false
		}

		mySituation.lastFixSinceMidnightUTC = float32(3600*hr+60*min) + float32(sec)
		if globalStatus.GPS_detected_type != GPS_TYPE_UBX {
			thisGpsPerf.nmeaTime = mySituation.lastFixSinceMidnightUTC
		}

		// Latitude.
		if len(x[2]) < 4 {
			return false
		}

		hr, err1 = strconv.Atoi(x[2][0:2])
		minf, err2 := strconv.ParseFloat(x[2][2:], 32)
		if err1 != nil || err2 != nil {
			return false
		}

		mySituation.Lat = float32(hr) + float32(minf/60.0)
		if x[3] == "S" { // South = negative.
			mySituation.Lat = -mySituation.Lat
		}

		// Longitude.
		if len(x[4]) < 5 {
			return false
		}
		hr, err1 = strconv.Atoi(x[4][0:3])
		minf, err2 = strconv.ParseFloat(x[4][3:], 32)
		if err1 != nil || err2 != nil {
			return false
		}

		mySituation.Lng = float32(hr) + float32(minf/60.0)
		if x[5] == "W" { // West = negative.
			mySituation.Lng = -mySituation.Lng
		}

		/* Satellite count and horizontal accuracy deprecated. Using PUBX,00 with fallback to GSA.
		// Satellites.
		sat, err1 := strconv.Atoi(x[7])
		if err1 != nil {
			return false
		}
		mySituation.Satellites = uint16(sat)

		// Accuracy.
		hdop, err1 := strconv.ParseFloat(x[8], 32)
		if err1 != nil {
			return false
		}
		if mySituation.quality == 2 {
			mySituation.Accuracy = float32(hdop * 4.0) //Estimate for WAAS / DGPS solution
		} else {
			mySituation.Accuracy = float32(hdop * 8.0) //Estimate for 3D non-WAAS solution
		}

		// NACp estimate.
		mySituation.NACp = calculateNACp(mySituation.Accuracy)
		*/

		// Altitude.
		alt, err1 := strconv.ParseFloat(x[9], 32)
		if err1 != nil {
			return false
		}
		mySituation.Alt = float32(alt * 3.28084) // Convert to feet.

		if globalStatus.GPS_detected_type != GPS_TYPE_UBX {
			thisGpsPerf.alt = float32(mySituation.Alt)
		}

		// Geoid separation (Sep = HAE - MSL)
		// (needed for proper MSL offset on PUBX,00 altitudes)

		geoidSep, err1 := strconv.ParseFloat(x[11], 32)
		if err1 != nil {
			return false
		}
		mySituation.GeoidSep = float32(geoidSep * 3.28084) // Convert to feet.
		mySituation.HeightAboveEllipsoid = mySituation.GeoidSep + mySituation.Alt

		// Timestamp.
		mySituation.LastFixLocalTime = stratuxClock.Time
		if globalStatus.GPS_detected_type != GPS_TYPE_UBX {
			updateGPSPerf = true
			thisGpsPerf.msgType = x[0]
		}

	} else if (x[0] == "GNRMC") || (x[0] == "GPRMC") {
		//$GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W*6A
		/*						check RY835 man for NMEA version, if >2.2, add mode field
				Where:
		     RMC          Recommended Minimum sentence C
		     123519       Fix taken at 12:35:19 UTC
		     A            Status A=active or V=Void.
		     4807.038,N   Latitude 48 deg 07.038' N
		     01131.000,E  Longitude 11 deg 31.000' E
		     022.4        Speed over the ground in knots
		     084.4        Track angle in degrees True
		     230394       Date - 23rd of March 1994
		     003.1,W      Magnetic Variation
		     D				mode field (nmea 2.3 and higher)
		     *6A          The checksum data, always begins with *
		*/
		if len(x) < 11 {
			return false
		}
		//mySituation.mu_GPS.Lock()
		//defer mySituation.mu_GPS.Unlock()

		if x[2] != "A" { // invalid fix
			mySituation.quality = 0
			return false
		} else if mySituation.quality == 0 {
			mySituation.quality = 1 // fallback option; indicate if the position fix is valid even if GGA or PUBX,00 aren't received
		}

		// Timestamp.
		if len(x[1]) < 7 {
			return false
		}
		hr, err1 := strconv.Atoi(x[1][0:2])
		min, err2 := strconv.Atoi(x[1][2:4])
		sec, err3 := strconv.ParseFloat(x[1][4:], 32)
		if err1 != nil || err2 != nil || err3 != nil {
			return false
		}

		mySituation.lastFixSinceMidnightUTC = float32(3600*hr+60*min) + float32(sec)
		if globalStatus.GPS_detected_type != GPS_TYPE_UBX {
			thisGpsPerf.nmeaTime = mySituation.lastFixSinceMidnightUTC
		}

		if len(x[9]) == 6 {
			// Date of Fix, i.e 191115 =  19 November 2015 UTC  field 9
			gpsTimeStr := fmt.Sprintf("%s %02d:%02d:%06.3f", x[9], hr, min, sec)
			gpsTime, err := time.Parse("020106 15:04:05.000", gpsTimeStr)
			if err == nil {
				mySituation.LastGPSTimeTime = stratuxClock.Time
				if time.Since(gpsTime) > 3*time.Second || time.Since(gpsTime) < -3*time.Second {
					setStr := gpsTime.Format("20060102 15:04:05.000") + " UTC"
					log.Printf("setting system time to: '%s'\n", setStr)
					if err := exec.Command("date", "-s", setStr).Run(); err != nil {
						log.Printf("Set Date failure: %s error\n", err)
					} else {
						log.Printf("Time set from GPS. Current time is %v\n", time.Now())
					}
				}
			}
		}

		// Latitude.
		if len(x[3]) < 4 {
			return false
		}
		hr, err1 = strconv.Atoi(x[3][0:2])
		minf, err2 := strconv.ParseFloat(x[3][2:], 32)
		if err1 != nil || err2 != nil {
			return false
		}
		mySituation.Lat = float32(hr) + float32(minf/60.0)
		if x[4] == "S" { // South = negative.
			mySituation.Lat = -mySituation.Lat
		}
		// Longitude.
		if len(x[5]) < 5 {
			return false
		}
		hr, err1 = strconv.Atoi(x[5][0:3])
		minf, err2 = strconv.ParseFloat(x[5][3:], 32)
		if err1 != nil || err2 != nil {
			return false
		}
		mySituation.Lng = float32(hr) + float32(minf/60.0)
		if x[6] == "W" { // West = negative.
			mySituation.Lng = -mySituation.Lng
		}

		mySituation.LastFixLocalTime = stratuxClock.Time

		// ground speed in kts (field 7)
		groundspeed, err := strconv.ParseFloat(x[7], 32)
		if err != nil {
			return false
		}
		mySituation.GroundSpeed = uint16(groundspeed)
		if globalStatus.GPS_detected_type != GPS_TYPE_UBX {
			thisGpsPerf.gsf = float32(groundspeed)
		}

		// ground track "True" (field 8)
		trueCourse := uint16(0)
		tc, err := strconv.ParseFloat(x[8], 32)
		if err != nil {
			return false
		}

		if groundspeed > 3 { // TO-DO: use average groundspeed over last n seconds to avoid random "jumps"
			trueCourse = uint16(tc)
			setTrueCourse(uint16(groundspeed), trueCourse)
			mySituation.TrueCourse = uint16(trueCourse)
			if globalStatus.GPS_detected_type != GPS_TYPE_UBX {
				thisGpsPerf.coursef = float32(tc)
			}
		} else {
			if globalStatus.GPS_detected_type != GPS_TYPE_UBX {
				thisGpsPerf.coursef = -999.9
			}

			// Negligible movement. Don't update course, but do use the slow speed.
			// TO-DO: use average course over last n seconds?
		}

		if globalStatus.GPS_detected_type != GPS_TYPE_UBX {
			updateGPSPerf = true
			thisGpsPerf.msgType = x[0]
		}
		mySituation.LastGroundTrackTime = stratuxClock.Time

	} else if (x[0] == "GNGSA") || (x[0] == "GPGSA") {
		if len(x) < 18 {
			return false
		}

		// field 1: operation mode
		// M: manual forced to 2D or 3D mode
		// A: automatic switching between 2D and 3D modes
		if (x[1] != "A") && (x[1] != "M") { // invalid fix
			mySituation.quality = 0
			return false
		}

		// field 2: solution type
		// 1 = no solution; 2 = 2D fix, 3 = 3D fix. WAAS status is parsed from GGA message, so no need to get here

		// fields 3-14: satellites in solution
		sat := 0
		for _, svtxt := range x[3:15] {
			_, err := strconv.Atoi(svtxt)
			if err == nil {
				sat++
			}
		}
		mySituation.Satellites = uint16(sat)

		// Satellites tracked / seen should be parsed from GSV message (TO-DO) ... since we don't have it, just use satellites from solution
		if mySituation.SatellitesTracked == 0 {
			mySituation.SatellitesTracked = uint16(sat)
		}

		if mySituation.SatellitesSeen == 0 {
			mySituation.SatellitesSeen = uint16(sat)
		}

		// field 16: HDOP
		// Accuracy estimate
		hdop, err1 := strconv.ParseFloat(x[16], 32)
		if err1 != nil {
			return false
		}
		if mySituation.quality == 2 {
			mySituation.Accuracy = float32(hdop * 4.0) // Rough 95% confidence estimate for WAAS / DGPS solution
		} else {
			mySituation.Accuracy = float32(hdop * 8.0) // Rough 95% confidence estimate for 3D non-WAAS solution
		}

		// NACp estimate.
		mySituation.NACp = calculateNACp(mySituation.Accuracy)

		// field 17: VDOP
		// accuracy estimate
		vdop, err1 := strconv.ParseFloat(x[17], 32)
		if err1 != nil {
			return false
		}
		mySituation.AccuracyVert = float32(vdop * 5) // rough estimate for 95% confidence
	}

	if updateGPSPerf {
		mySituation.mu_GPSPerf.Lock()
		myGPSPerfStats = append(myGPSPerfStats, thisGpsPerf)
		lenGPSPerfStats := len(myGPSPerfStats)
		//log.Printf("GPSPerf array has %n elements. Contents are: %v\n",lenGPSPerfStats,myGPSPerfStats)
		if lenGPSPerfStats > 299 { //30 seconds @ 10 Hz for UBX, 30 seconds @ 5 Hz for MTK or SIRF with 2x messages per 200 ms)
			myGPSPerfStats = myGPSPerfStats[(lenGPSPerfStats - 299):] // remove the first n entries if more than 300 in the slice
		}
		mySituation.mu_GPSPerf.Unlock()
	}

	return true
}

func sendGPRMCString() {
	msg := makeGPRMCString()
	log.Printf("FLARM GPRMC String: %s\n", msg) // TO-DO: Send this to /dev/ttyAMA0
}

func makeGPRMCString() string {
	/*
				 RMC          Recommended Minimum sentence C
			     123519       Fix taken at 12:35:19 UTC
			     A            Status A=active or V=Void.
			     4807.038,N   Latitude 48 deg 07.038' N
			     01131.000,E  Longitude 11 deg 31.000' E
			     022.4        Speed over the ground in knots
			     084.4        Track angle in degrees True
			     230394       Date - 23rd of March 1994
			     003.1,W      Magnetic Variation
			     D				mode field (nmea 2.3 and higher)
			     *6A          The checksum data, always begins with *

		lastFixSinceMidnightUTC uint32
		Lat                     float32
		Lng                     float32
		quality                 uint8
		GeoidSep                float32 // geoid separation, ft, MSL minus HAE (used in altitude calculation)
		Satellites              uint16  // satellites used in solution
		SatellitesTracked       uint16  // satellites tracked (almanac data received)
		SatellitesSeen          uint16  // satellites seen (signal received)
		Accuracy                float32 // 95% confidence for horizontal position, meters.
		NACp                    uint8   // NACp categories are defined in AC 20-165A
		Alt                     float32 // Feet MSL
		AccuracyVert            float32 // 95% confidence for vertical position, meters
		GPSVertVel              float32 // GPS vertical velocity, feet per second
		LastFixLocalTime        time.Time
		TrueCourse              uint16
		GroundSpeed             uint16
		LastGroundTrackTime     time.Time


	*/

	lastFix := float64(mySituation.lastFixSinceMidnightUTC)
	hr := math.Floor(lastFix / 3600)
	lastFix -= 3600 * hr
	mins := math.Floor(lastFix / 60)
	sec := lastFix - mins*60

	status := "V"
	if isGPSValid() && mySituation.quality > 0 {
		status = "A"
	}

	lat := float64(mySituation.Lat)
	deg := math.Floor(lat)
	min := (lat - deg) * 60
	lat = deg*100 + min

	ns := "N"
	if lat < 0 {
		lat = -lat
		ns = "S"
	}

	lng := float64(mySituation.Lng)
	deg = math.Floor(lng)
	min = (lng - deg) * 60
	lng = deg*100 + min

	ew := "E"
	if lng < 0 {
		lng = -lng
		ew = "W"
	}

	gs := float32(mySituation.GroundSpeed)
	trueCourse := float32(mySituation.TrueCourse)
	yy, mm, dd := time.Now().UTC().Date()
	yy = yy % 100
	var magVar, mvEW string
	mode := "N"
	if mySituation.quality == 1 {
		mode = "A"
	} else if mySituation.quality == 2 {
		mode = "D"
	}

	msg := fmt.Sprintf("GPRMC,%02.f%02.f%05.2f,%s,%010.5f,%s,%011.5f,%s,%.1f,%.1f,%02d%02d%02d,%s,%s,%s", hr, mins, sec, status, lat, ns, lng, ew, gs, trueCourse, dd, mm, yy, magVar, mvEW, mode)

	var checksum byte
	for i := range msg {
		checksum = checksum ^ byte(msg[i])
	}
	msg = fmt.Sprintf("$%s*%X", msg, checksum)
	return msg
}

func gpsSerialReader() {
	defer serialPort.Close()
	readyToInitGPS = false // TO-DO: replace with channel control to terminate goroutine when complete

	i := 0 //debug monitor
	scanner := bufio.NewScanner(serialPort)
	for scanner.Scan() && globalStatus.GPS_connected && globalSettings.GPS_Enabled {
		i++
		if i%100 == 0 {
			log.Printf("gpsSerialReader() scanner loop iteration i=%d\n", i) // debug monitor
		}

		s := scanner.Text()
		//fmt.Printf("Output: %s\n", s)
		if !(processNMEALine(s)) {
			//	fmt.Printf("processNMEALine() exited early -- %s\n",s) //debug code.
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("reading standard input: %s\n", err.Error())
	}

	log.Printf("Exiting gpsSerialReader() after i=%d loops\n", i) // debug monitor
	globalStatus.GPS_connected = false
	readyToInitGPS = true // TO-DO: replace with channel control to terminate goroutine when complete
	return
}

var i2cbus embd.I2CBus
var myBMP180 *bmp180.BMP180
var myMPU6050 *mpu6050.MPU6050

func readBMP180() (float64, float64, error) { // ºCelsius, Meters
	temp, err := myBMP180.Temperature()
	if err != nil {
		return temp, 0.0, err
	}
	altitude, err := myBMP180.Altitude()
	altitude = float64(1/0.3048) * altitude // Convert meters to feet.
	if err != nil {
		return temp, altitude, err
	}
	return temp, altitude, nil
}

func readMPU6050() (float64, float64, error) { //TODO: error checking.
	pitch, roll := myMPU6050.PitchAndRoll()
	return pitch, roll, nil
}

func initBMP180() error {
	myBMP180 = bmp180.New(i2cbus) //TODO: error checking.
	return nil
}

func initMPU6050() error {
	myMPU6050 = mpu6050.New() //TODO: error checking.
	return nil
}

func initI2C() error {
	i2cbus = embd.NewI2CBus(1) //TODO: error checking.
	return nil
}

// Unused at the moment. 5 second update, since read functions in bmp180 are slow.
func tempAndPressureReader() {
	timer := time.NewTicker(5 * time.Second)
	for globalStatus.RY835AI_connected && globalSettings.AHRS_Enabled {
		<-timer.C
		// Read temperature and pressure altitude.
		temp, alt, err_bmp180 := readBMP180()
		// Process.
		if err_bmp180 != nil {
			log.Printf("readBMP180(): %s\n", err_bmp180.Error())
			globalStatus.RY835AI_connected = false
		} else {
			mySituation.Temp = temp
			mySituation.Pressure_alt = alt
			mySituation.lastTempPressTime = stratuxClock.Time
		}
	}
	globalStatus.RY835AI_connected = false
}

func makeFFAHRSSimReport() {
	s := fmt.Sprintf("XATTStratux,%f,%f,%f", mySituation.Gyro_heading, mySituation.Pitch, mySituation.Roll)

	sendMsg([]byte(s), NETWORK_AHRS_FFSIM, false)
}

func makeAHRSGDL90Report() {
	msg := make([]byte, 16)
	msg[0] = 0x4c
	msg[1] = 0x45
	msg[2] = 0x01
	msg[3] = 0x00

	pitch := int16(float64(mySituation.Pitch) * float64(10.0))
	roll := int16(float64(mySituation.Roll) * float64(10.0))
	hdg := uint16(float64(mySituation.Gyro_heading) * float64(10.0))
	slip_skid := int16(float64(0) * float64(10.0))
	yaw_rate := int16(float64(0) * float64(10.0))
	g := int16(float64(1.0) * float64(10.0))

	// Roll.
	msg[4] = byte((roll >> 8) & 0xFF)
	msg[5] = byte(roll & 0xFF)

	// Pitch.
	msg[6] = byte((pitch >> 8) & 0xFF)
	msg[7] = byte(pitch & 0xFF)

	// Heading.
	msg[8] = byte((hdg >> 8) & 0xFF)
	msg[9] = byte(hdg & 0xFF)

	// Slip/skid.
	msg[10] = byte((slip_skid >> 8) & 0xFF)
	msg[11] = byte(slip_skid & 0xFF)

	// Yaw rate.
	msg[12] = byte((yaw_rate >> 8) & 0xFF)
	msg[13] = byte(yaw_rate & 0xFF)

	// "G".
	msg[14] = byte((g >> 8) & 0xFF)
	msg[15] = byte(g & 0xFF)

	sendMsg(prepareMessage(msg), NETWORK_AHRS_GDL90, false)
}

func gpsAttitudeSender() {
	timer := time.NewTicker(100 * time.Millisecond) // ~10Hz update.
	for {
		<-timer.C
		myGPSPerfStats = make([]gpsPerfStats, 0) // reinitialize statistics on disconnect / reconnect
		for globalSettings.GPS_Enabled && globalStatus.GPS_connected && globalSettings.GPSAttitude_Enabled && !(globalSettings.AHRS_Enabled) {
			<-timer.C

			if !calcGPSAttitude() {
				if globalSettings.VerboseLogs {
					log.Printf("Error calculating GPS-based attitude statistics\n")
				}
			} else {
				mySituation.mu_GPSPerf.Lock()
				index := len(myGPSPerfStats) - 1
				if index > 1 {
					mySituation.Pitch = myGPSPerfStats[index].gpsPitch
					mySituation.Roll = myGPSPerfStats[index].gpsRoll
					mySituation.Gyro_heading = float64(mySituation.TrueCourse)
					mySituation.LastAttitudeTime = stratuxClock.Time
					if globalSettings.ForeFlightSimMode == true {
						makeFFAHRSSimReport()
					} else {
						makeAHRSGDL90Report()
					}
				}
				mySituation.mu_GPSPerf.Unlock()
			}
		}
	}
}
func attitudeReaderSender() {
	timer := time.NewTicker(100 * time.Millisecond) // ~10Hz update.
	for globalStatus.RY835AI_connected && globalSettings.AHRS_Enabled {
		<-timer.C
		// Read pitch and roll.
		pitch, roll, err_mpu6050 := readMPU6050()

		if err_mpu6050 != nil {
			log.Printf("readMPU6050(): %s\n", err_mpu6050.Error())
			globalStatus.RY835AI_connected = false
			break
		}

		mySituation.mu_Attitude.Lock()

		mySituation.Pitch = pitch
		mySituation.Roll = roll
		mySituation.Gyro_heading = myMPU6050.Heading() //FIXME. Experimental.
		mySituation.LastAttitudeTime = stratuxClock.Time

		// Send, if valid.
		//		if isGPSGroundTrackValid(), etc.

		if globalSettings.ForeFlightSimMode == true {
			makeFFAHRSSimReport()
		} else {
			makeAHRSGDL90Report()
		}

		mySituation.mu_Attitude.Unlock()
	}
	globalStatus.RY835AI_connected = false
}

func isOwnshipPressureAltValid() bool {
	return stratuxClock.Since(mySituation.OwnshipLastSeen) < 10*time.Second
}

func isGPSConnected() bool {
	return stratuxClock.Since(mySituation.LastNMEAMessage) < 5*time.Second
}

func isGPSValid() bool {
	return (stratuxClock.Since(mySituation.LastFixLocalTime) < 15*time.Second) && isGPSConnected()
}

func isGPSGroundTrackValid() bool {
	return stratuxClock.Since(mySituation.LastGroundTrackTime) < 15*time.Second
}

func isGPSClockValid() bool {
	return stratuxClock.Since(mySituation.LastGPSTimeTime) < 15*time.Second
}

func isAHRSValid() bool {
	return stratuxClock.Since(mySituation.LastAttitudeTime) < 1*time.Second // If attitude information gets to be over 1 second old, declare invalid.
}

func isTempPressValid() bool {
	return stratuxClock.Since(mySituation.lastTempPressTime) < 15*time.Second
}

func initAHRS() error {
	if err := initI2C(); err != nil { // I2C bus.
		return err
	}
	if err := initBMP180(); err != nil { // I2C temperature and pressure altitude.
		i2cbus.Close()
		return err
	}
	if err := initMPU6050(); err != nil { // I2C accel/gyro.
		i2cbus.Close()
		myBMP180.Close()
		return err
	}
	globalStatus.RY835AI_connected = true
	go attitudeReaderSender()
	go tempAndPressureReader()

	return nil
}

func pollRY835AI() {
	readyToInitGPS = true //TO-DO: Implement more robust method (channel control) to kill zombie serial readers
	timer := time.NewTicker(4 * time.Second)
	go gpsAttitudeSender()
	for {
		<-timer.C
		// GPS enabled, was not connected previously?
		if globalSettings.GPS_Enabled && !globalStatus.GPS_connected && readyToInitGPS { //TO-DO: Implement more robust method (channel control) to kill zombie serial readers
			globalStatus.GPS_connected = initGPSSerial()
			if globalStatus.GPS_connected {
				go gpsSerialReader()
			}
		}
		// RY835AI I2C enabled, was not connected previously?
		if globalSettings.AHRS_Enabled && !globalStatus.RY835AI_connected {
			err := initAHRS()
			if err != nil {
				log.Printf("initAHRS(): %s\ndisabling AHRS sensors.\n", err.Error())
				globalStatus.RY835AI_connected = false
			}
		}

		// temporary home for gps message statistics
		lenGPSMsgStats := len(myGPSMsgStats)
		countMsg := 0
		countMsgPosn := 0
		countMsgInvalid := 0

		if lenGPSMsgStats > 1 {
			for _, stat := range myGPSMsgStats {
				if stratuxClock.Milliseconds-stat.stratuxTime < 60000 {
					if !(stat.msgValid) {
						countMsgInvalid++
					} else {
						countMsg++
						if stat.msgType == "PUBX00" || stat.msgType == "GPGGA" || stat.msgType == "GNGGA" {
							countMsgPosn++
						}
					}
				}
			}

			//timeInit := myGPSMsgStats[0].stratuxTime
			//timeFinal := myGPSMsgStats[lenGPSMsgStats-1].stratuxTime
			//deltaTime := timeFinal - timeInit
			//log.Printf("%d messages in slice. Time range of GPS mgs stat slice is %d to %d; difference is %d\n", lenGPSMsgStats, timeInit, timeFinal, deltaTime)
			log.Printf("NMEA messages last minute. Valid: %d. Position: %d. Invalid: %d.\n", countMsg, countMsgPosn, countMsgInvalid)
			//log.Printf("Position data slice: %v\n", myGPSPerfStats)
			globalStatus.GPS_invalid_msgs_last_minute = uint(countMsgInvalid)
			globalStatus.GPS_msgs_last_minute = uint(countMsg)
			globalStatus.GPS_pos_msgs_last_minute = uint(countMsgPosn)
		}
	}
}

func initRY835AI() {
	mySituation.mu_GPS = &sync.Mutex{}
	mySituation.mu_Attitude = &sync.Mutex{}
	mySituation.mu_GPSPerf = &sync.Mutex{}

	go pollRY835AI()

}
