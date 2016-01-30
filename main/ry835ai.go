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
	mu_GPS *sync.Mutex

	// From GPS.
	lastFixSinceMidnightUTC float32 // Time of most recent GPS fix time, UTC seconds
	Lat                     float32 // decimal latitude DD.dddddd (North is positive)
	Lng                     float32 // decimal longitude DDD.dddddd (East is positive)
	quality                 uint8   // 0 = no fix; 1 = GPS/GLONASS; 2 = differential GPS (e.g. WAAS); 6 = dead reckoning
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

	mu_Attitude *sync.Mutex

	// From BMP180 pressure sensor.
	Temp              float64
	Pressure_alt      float64
	lastTempPressTime time.Time

	// From MPU6050 accel/gyro.
	Pitch            float64
	Roll             float64
	Gyro_heading     float64
	LastAttitudeTime time.Time
}

var serialConfig *serial.Config
var serialPort *serial.Port

/*
file:///Users/c/Downloads/u-blox5_Referenzmanual.pdf
Platform settings
Airborne <2g Recommended for typical airborne environment. No 2D position fixes supported.
p.91 - CFG-MSG
Navigation/Measurement Rate Settings
Header 0xB5 0x62
ID 0x06 0x08
0x0064 (100 ms)
0x0001
0x0001 (GPS time)
{0xB5, 0x62, 0x06, 0x08, 0x00, 0x64, 0x00, 0x01, 0x00, 0x01}
p.109 CFG-NAV5 (0x06 0x24)
Poll Navigation Engine Settings
*/

func chksumUBX(msg []byte) []byte {
	ret := make([]byte, 2)
	for i := 0; i < len(msg); i++ {
		ret[0] = ret[0] + msg[i]
		ret[1] = ret[1] + ret[0]
	}
	return ret
}

// p.62
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

func initGPSSerial() bool {
	var device string
	if _, err := os.Stat("/dev/ttyACM0"); err == nil {
		device = "/dev/ttyACM0"
	} else {
		device = "/dev/ttyAMA0"
	}
	log.Printf("Using %s for GPS\n", device)

	/* Developer option -- uncomment to allow "hot" configuration of GPS (assuming 38.4 kpbs on warm start)
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

		-- End developer option */

	// Open port at 9600 baud for config.
	serialConfig = &serial.Config{Name: device, Baud: 9600}
	p, err := serial.OpenPort(serialConfig)
	if err != nil {
		log.Printf("serial port err: %s\n", err.Error())
		return false
	}

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
	//	time.Sleep(100* time.Millisecond) // pause and wait for the GPS to finish configuring itself before closing / reopening the port

	p.Close()

	time.Sleep(250 * time.Millisecond)
	// Re-open port at 38400 baud so we can read messages. TO-DO: Fault detection / fallback to 9600 baud after failed config
	serialConfig = &serial.Config{Name: device, Baud: 38400}
	p, err = serial.OpenPort(serialConfig)
	if err != nil {
		log.Printf("serial port err: %s\n", err.Error())
		return false
	}

	serialPort = p
	log.Printf("GPS configuration complete\n")
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

func processNMEALine(l string) bool {
	replayLog(l, MSGCLASS_GPS)
	l_valid, validNMEAcs := validateNMEAChecksum(l)
	if !validNMEAcs {
		log.Printf("GPS error. Invalid NMEA string: %s\n", l_valid) // remove log message once validation complete
		return false
	}
	x := strings.Split(l_valid, ",")

	if x[0] == "PUBX" { // UBX proprietary message
		if x[1] == "00" { // position message
			if len(x) < 20 {
				return false
			}

			mySituation.mu_GPS.Lock()
			defer mySituation.mu_GPS.Unlock()

			// field 2 = time
			if len(x[2]) < 9 {
				return false
			}
			hr, err1 := strconv.Atoi(x[2][0:2])
			min, err2 := strconv.Atoi(x[2][2:4])
			sec, err3 := strconv.ParseFloat(x[2][4:], 32)
			if err1 != nil || err2 != nil || err3 != nil {
				return false
			}

			mySituation.lastFixSinceMidnightUTC = float32((hr*60*60)+(min*60)) + float32(sec)

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
			alt := float32(hae*3.28084) - mySituation.GeoidSep // convert to feet and offset by geoid separation
			mySituation.Alt = alt

			// field 8 = nav status
			// DR = dead reckoning, G2= 2D GPS, G3 = 3D GPS, D2= 2D diff, D3 = 3D diff, RK = GPS+DR, TT = time only

			if x[8] == "D2" || x[8] == "D3" {
				mySituation.quality = 2
			} else if x[8] == "G2" || x[8] == "G3" {
				mySituation.quality = 1
			} else if x[8] == "DR" || x[8] == "RK" {
				mySituation.quality = 6
			} else if x[8] == "NF" {
				mySituation.quality = 0
				return false // return false if no valid fix.
			} else {
				mySituation.quality = 0
			}

			// field 9 = horizontal accuracy, m
			hAcc, err := strconv.ParseFloat(x[9], 32)
			if err != nil {
				return false
			}
			mySituation.Accuracy = float32(hAcc * 2) // UBX reports 1-sigma variation; NACp is 95% confidence (2-sigma)

			// NACp estimate.
			mySituation.NACp = calculateNACp(mySituation.Accuracy)

			// field 10 = vertical accuracy, m
			vAcc, err := strconv.ParseFloat(x[10], 32)
			if err != nil {
				return false
			}
			mySituation.AccuracyVert = float32(vAcc * 2) // UBX reports 1-sigma variation; we want 95% confidence

			// field 11 = groundspeed, km/h
			groundspeed, err := strconv.ParseFloat(x[11], 32)
			if err != nil {
				return false
			}
			groundspeed = groundspeed * 0.540003 // convert to knots

			// field 12 = track, deg
			trueCourse := uint16(0)
			if len(x[12]) > 0 && groundspeed > 3 {
				tc, err := strconv.ParseFloat(x[12], 32)
				if err != nil {
					return false
				}
				trueCourse = uint16(tc)
			} else {
				// No movement.
				mySituation.TrueCourse = 0
				mySituation.GroundSpeed = 0
				mySituation.LastGroundTrackTime = time.Time{}
			}

			setTrueCourse(uint16(groundspeed), trueCourse)

			mySituation.TrueCourse = uint16(trueCourse)
			mySituation.GroundSpeed = uint16(groundspeed)
			mySituation.LastGroundTrackTime = stratuxClock.Time

			// field 13 = vertical velocity, m/s
			vv, err := strconv.ParseFloat(x[13], 32)
			if err != nil {
				return false
			}
			mySituation.GPSVertVel = float32(vv * -3.28084) // convert to ft/sec and positive = up

			// field 14 = age of diff corrections

			// field 18 = number of satellites
			sat, err1 := strconv.Atoi(x[18])
			if err1 != nil {
				return false
			}
			mySituation.Satellites = uint16(sat)

			mySituation.LastFixLocalTime = stratuxClock.Time

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
				if x[7+6*i] != "" {
					satSeen++
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
			if len(x[2]) < 9 {
				return false
			}
			hr, err1 := strconv.Atoi(x[2][0:2])
			min, err2 := strconv.Atoi(x[2][2:4])
			sec, err3 := strconv.ParseFloat(x[2][4:], 32)
			if err1 != nil || err2 != nil || err3 != nil {
				return false
			}
			mySituation.lastFixSinceMidnightUTC = float32((3600*hr)+(60*min)) + float32(sec)

			// field 3 is date

			if len(x[3]) == 6 {
				// Date of Fix, i.e 191115 =  19 November 2015 UTC  field 9
				gpsTimeStr := fmt.Sprintf("%s %02d:%02d:%06.3f", x[3], hr, min, sec)
				gpsTime, err := time.Parse("020106 15:04:05.000", gpsTimeStr)
				if err == nil {
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

		// otherwise parse the NMEA standard messages as a fall-back / compatibility option
	} else if (x[0] == "GNVTG") || (x[0] == "GPVTG") { // Ground track information.
		mySituation.mu_GPS.Lock()
		defer mySituation.mu_GPS.Unlock()
		if len(x) < 10 {
			return false
		}
		trueCourse := uint16(0)
		if len(x[1]) > 0 {
			tc, err := strconv.ParseFloat(x[1], 32)
			if err != nil {
				return false
			}
			trueCourse = uint16(tc)
		} else {
			// No movement.
			mySituation.TrueCourse = 0
			mySituation.GroundSpeed = 0
			mySituation.LastGroundTrackTime = time.Time{}
			return true
		}
		groundSpeed, err := strconv.ParseFloat(x[5], 32) // Knots.
		if err != nil {
			return false
		}

		setTrueCourse(uint16(groundSpeed), trueCourse)

		mySituation.TrueCourse = uint16(trueCourse)
		mySituation.GroundSpeed = uint16(groundSpeed)
		mySituation.LastGroundTrackTime = stratuxClock.Time

	} else if (x[0] == "GNGGA") || (x[0] == "GPGGA") { // GPS fix.
		if len(x) < 15 {
			return false
		}
		mySituation.mu_GPS.Lock()
		defer mySituation.mu_GPS.Unlock()
		// Timestamp.
		if len(x[1]) < 9 {
			return false
		}
		hr, err1 := strconv.Atoi(x[1][0:2])
		min, err2 := strconv.Atoi(x[1][2:4])
		sec, err3 := strconv.ParseFloat(x[1][4:], 32)
		if err1 != nil || err2 != nil || err3 != nil {
			return false
		}

		mySituation.lastFixSinceMidnightUTC = float32((3600*hr)+(60*min)) + float32(sec)

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

		// Quality indicator.
		q, err1 := strconv.Atoi(x[6])
		if err1 != nil {
			return false
		}
		mySituation.quality = uint8(q) // 1 = 3D GPS; 2 = DGPS (SBAS /WAAS)

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

		// Geoid separation (Sep = HAE - MSL)
		// (needed for proper MSL offset on PUBX,00 altitudes)

		geoidSep, err1 := strconv.ParseFloat(x[11], 32)
		if err1 != nil {
			return false
		}
		mySituation.GeoidSep = float32(geoidSep * 3.28084) // Convert to feet.

		// Timestamp.
		mySituation.LastFixLocalTime = stratuxClock.Time

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
		if len(x) < 12 {
			return false
		}
		mySituation.mu_GPS.Lock()
		defer mySituation.mu_GPS.Unlock()

		// Timestamp.
		if len(x[1]) < 9 {
			return false
		}
		hr, err1 := strconv.Atoi(x[1][0:2])
		min, err2 := strconv.Atoi(x[1][2:4])
		sec, err3 := strconv.ParseFloat(x[1][4:], 32)
		if err1 != nil || err2 != nil || err3 != nil {
			return false
		}
		mySituation.lastFixSinceMidnightUTC = float32((3600*hr)+(60*min)) + float32(sec)

		if len(x[9]) == 6 {
			// Date of Fix, i.e 191115 =  19 November 2015 UTC  field 9
			gpsTimeStr := fmt.Sprintf("%s %02d:%02d:%06.3f", x[9], hr, min, sec)
			gpsTime, err := time.Parse("020106 15:04:05.000", gpsTimeStr)
			if err == nil {
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

		if x[2] != "A" { // invalid fix
			return false
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
		// ground speed in kts (field 7)
		groundspeed, err := strconv.ParseFloat(x[7], 32)
		if err != nil {
			return false
		}
		mySituation.GroundSpeed = uint16(groundspeed)
		// ground track "True" field 8
		tc, err := strconv.ParseFloat(x[8], 32)
		if err != nil {
			return false
		}
		mySituation.TrueCourse = uint16(tc)

	} else if (x[0] == "GNGSA") || (x[0] == "GPGSA") {
		if len(x) < 18 {
			return false
		}

		// field 1: operation mode
		if x[1] != "A" { // invalid fix
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
	for globalSettings.GPS_Enabled && globalStatus.GPS_connected {

		scanner := bufio.NewScanner(serialPort)
		for scanner.Scan() {
			s := scanner.Text()
			// log.Printf("Output: %s\n", s)
			processNMEALine(s)
		}
		if err := scanner.Err(); err != nil {
			log.Printf("reading standard input: %s\n", err.Error())
		}
	}
	globalStatus.GPS_connected = false
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
	hdg := uint16(float64(mySituation.Gyro_heading) * float64(10.0)) //TODO.
	slip_skid := int16(float64(0) * float64(10.0))                   //TODO.
	yaw_rate := int16(float64(0) * float64(10.0))                    //TODO.
	g := int16(float64(1.0) * float64(10.0))                         //TODO.

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

func isGPSValid() bool {
	return stratuxClock.Since(mySituation.LastFixLocalTime) < 15*time.Second
}

func isGPSGroundTrackValid() bool {
	return stratuxClock.Since(mySituation.LastGroundTrackTime) < 15*time.Second
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
	timer := time.NewTicker(5 * time.Second)
	for {
		<-timer.C
		// GPS enabled, was not connected previously?
		if globalSettings.GPS_Enabled && !globalStatus.GPS_connected {
			globalStatus.GPS_connected = initGPSSerial() // via USB for now.
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
	}
}

func initRY835AI() {
	mySituation.mu_GPS = &sync.Mutex{}
	mySituation.mu_Attitude = &sync.Mutex{}

	go pollRY835AI()
}
