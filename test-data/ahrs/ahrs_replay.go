//https://www.youtube.com/watch?v=sQxJkSFmy_M&t=12m14s
//https://www.youtube.com/watch?v=sQxJkSFmy_M&t=14m12s
//815

package main

import (
	"fmt"
	"log"
	"os"
	"bufio"
	"strconv"
	"encoding/json"
	"errors"
	"strings"
	"net"
	"math"
	"time"
)

var Crc16Table [256]uint16


const (
	IPAD_ADDR = "192.168.1.133"
	LON_LAT_RESOLUTION = float64(180.0 / 8388608.0)
	TRACK_RESOLUTION   = float32(360.0 / 256.0)
)

type GPSData struct {
	Timestamp int64
	Lat       float64
	Lng       float64
	Alt       float64
	Speed     float64
	Course   float64
}

type AHRSData struct {
	Timestamp		int64
	RawGyro         []int64
	RawAccel        []int64
	RawQuat         []int64
	DmpTimestamp    int64
	RawMag          []int64
	MagTimestamp    int64
	CalibratedAccel []int64
	CalibratedMag   []int64
	FusedQuat       []float64
	FusedEuler      []float64
	LastDMPYaw      float64
	LastYaw         float64
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


func cleanStr(str string) string {
	str = strings.Trim(str, "\r\n ")
	str = strings.Replace(str, "\x00", "", -1)
	return str
}

func seekTimestampInAHRS(fn string, startnsec, tol int64) (*os.File, int64, error) {
	fp, err := os.OpenFile(fn, os.O_RDONLY, 0)
	if err != nil {
		return nil, 0, err
	}
	rdr := bufio.NewReader(fp)
	for {
		buf, err := rdr.ReadString('\n')
		if err != nil {
			break
		}
		buf = cleanStr(buf)
		if len(buf) == 0 {
			continue
		}

		var ahrs AHRSData
		err = json.Unmarshal([]byte(buf), &ahrs)
		if err != nil {
			continue
		}

		i := ahrs.Timestamp
		if ((i > startnsec) && (i - startnsec) <= tol) || ((i < startnsec) && (startnsec - i) <= tol) { // Found it.
			return fp, i, nil
		}
	}
	return nil, 0, errors.New("can't find start.")
}

func getLine(rdr *bufio.Reader) ([]string, int64) {
	ret := make([]string, 0)
	retIdx := int64(0)
	for len(ret) == 0 {
		buf, err := rdr.ReadString('\n')
		if err != nil {
			fmt.Printf("quitting. err: %s\n", err.Error())
			os.Exit(0)
		}
		buf = cleanStr(buf)
		ln := strings.Split(buf, ",")
		if len(ln) < 2 {
			continue
		}
		idx, err := strconv.ParseInt(ln[0], 10, 64)
		if err != nil {
			continue
		}
		ret = ln
		retIdx = idx
	}
	return ret, retIdx
}

func getAHRS (rdr *bufio.Reader) (AHRSData, int64) {
	var ahrs AHRSData
	for ahrs.Timestamp == 0 {
		buf, err := rdr.ReadString('\n')
		if err != nil {
			fmt.Printf("quitting. err: %s\n", err.Error())
			os.Exit(0)
		}
		buf = cleanStr(buf)
		err = json.Unmarshal([]byte(buf), &ahrs)
		if err != nil {
			continue
		}
	}
	return ahrs, ahrs.Timestamp
}

func getGPS (rdr *bufio.Reader) (GPSData, int64) {
	var gps GPSData
	for gps.Timestamp == 0 {
		buf, err := rdr.ReadString('\n')
		if err != nil {
			fmt.Printf("quitting. err: %s\n", err.Error())
			os.Exit(0)
		}
		buf = cleanStr(buf)
		err = json.Unmarshal([]byte(buf), &gps)
		if err != nil {
			continue
		}
	}
	return gps, gps.Timestamp
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


func makeHeartbeat() []byte {
	msg := make([]byte, 7)
	// See p.10.
	msg[0] = 0x00 // Message type "Heartbeat".
	msg[1] = 0x01 // "UAT Initialized".
	msg[1] = msg[1] | 0x80
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

func makeLatLng(v float64) []byte {
	ret := make([]byte, 3)

	v = v / LON_LAT_RESOLUTION
	wk := int32(v)

	ret[0] = byte((wk & 0xFF0000) >> 16)
	ret[1] = byte((wk & 0x00FF00) >> 8)
	ret[2] = byte((wk & 0x0000FF))

	return ret
}


func makeOwnshipReport(gps GPSData) []byte {
	msg := make([]byte, 28)
	// See p.16.
	msg[0] = 0x0A // Message type "Ownship".

	msg[1] = 0x01 // Alert status, address type.

	msg[2] = 1 // Address.
	msg[3] = 1 // Address.
	msg[4] = 1 // Address.

	tmp := makeLatLng(gps.Lat)
	msg[5] = tmp[0] // Latitude.
	msg[6] = tmp[1] // Latitude.
	msg[7] = tmp[2] // Latitude.

	tmp = makeLatLng(gps.Lng)
	msg[8] = tmp[0]  // Longitude.
	msg[9] = tmp[1]  // Longitude.
	msg[10] = tmp[2] // Longitude.

	// This is **PRESSURE ALTITUDE**
	//FIXME: Temporarily removing "invalid altitude" when pressure altitude not available - using GPS altitude instead.
	//	alt := uint16(0xFFF) // 0xFFF "invalid altitude."

	alt := uint16(gps.Alt) //FIXME: This should not be here.
	alt = (alt + 1000) / 25

	alt = alt & 0xFFF // Should fit in 12 bits.

	msg[11] = byte((alt & 0xFF0) >> 4) // Altitude.
	msg[12] = byte((alt & 0x00F) << 4)

	msg[12] = byte(((alt & 0x00F) << 4) | 0xB) // "Airborne" + "True Heading"

	msg[13] = 0xBB // NIC and NACp.


	gdSpeed := uint16(gps.Speed)
	gdSpeed = gdSpeed & 0x0FFF // Should fit in 12 bits.

	msg[14] = byte((gdSpeed & 0xFF0) >> 4)
	msg[15] = byte((gdSpeed & 0x00F) << 4)

	verticalVelocity := int16(1000 / 64) // ft/min. 64 ft/min resolution.
	//TODO: 0x800 = no information available.
	verticalVelocity = verticalVelocity & 0x0FFF // Should fit in 12 bits.
	msg[15] = msg[15] | byte((verticalVelocity&0x0F00)>>8)
	msg[16] = byte(verticalVelocity & 0xFF)

	// Showing magnetic (corrected) on ForeFlight. Needs to be True Heading.
	groundTrack := uint16(gps.Course)
	trk := uint8(float32(groundTrack) / TRACK_RESOLUTION) // Resolution is ~1.4 degrees.

	msg[17] = byte(trk)

	msg[18] = 0x01 // "Light (ICAO) < 15,500 lbs"

	return prepareMessage(msg)
}

//TODO
func makeOwnshipGeometricAltitudeReport(gps GPSData) []byte {
	msg := make([]byte, 5)
	// See p.28.
	msg[0] = 0x0B                 // Message type "Ownship Geo Alt".
	alt := int16(gps.Alt) // GPS Altitude.
	alt = alt / 5
	msg[1] = byte(alt >> 8)     // Altitude.
	msg[2] = byte(alt & 0x00FF) // Altitude.

	//TODO: "Figure of Merit". 0x7FFF "Not available".
	msg[3] = 0x00
	msg[4] = 0x0A

	return prepareMessage(msg)
}

var myGPS GPSData

func heartBeatSender() {
	addr, err := net.ResolveUDPAddr("udp", IPAD_ADDR + ":4000")
	if err != nil {
		log.Printf("ResolveUDPAddr(%s): %s\n", IPAD_ADDR, err.Error())
		return
	}
	gdlConn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Printf("DialUDP(%s): %s\n", IPAD_ADDR, err.Error())
		return
	}
	
	timer := time.NewTicker(1 * time.Second)
	for {
		<-timer.C
		gdlConn.Write(makeHeartbeat())
		gdlConn.Write(makeOwnshipReport(myGPS))
		gdlConn.Write(makeOwnshipGeometricAltitudeReport(myGPS))
	}
}

var cal_pitch float64
var cal_roll float64

var cal_num int

func main() {
	crcInit()
	if len(os.Args) < 5 {
		fmt.Printf("%s <start second> <ahrs file> <gps file> <replay speed>\n", os.Args[0])
		return
	}
	startsec, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Printf("invalid: %s\n", os.Args[1])
		return
	}

	replayspeed, err := strconv.Atoi(os.Args[4])
	if err != nil {
		fmt.Printf("invalid: %s\n", os.Args[4])
		return
	}

	startnsec := int64(startsec) * 1000000000

	ahrsfp, ahrsIdx, err := seekTimestampInAHRS(os.Args[2], startnsec, 1000000000) // Find the index with 1.00s tolerance.
	if err != nil {
		panic(err)
	}
	defer ahrsfp.Close()
	gpsfp, err := os.OpenFile(os.Args[3], os.O_RDONLY, 0)
	if err != nil {
		panic(err)
	}
	defer gpsfp.Close()

	addr, err := net.ResolveUDPAddr("udp", IPAD_ADDR + ":49002")
	if err != nil {
		log.Printf("ResolveUDPAddr(%s): %s\n", IPAD_ADDR, err.Error())
		return
	}
	outConn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Printf("DialUDP(%s): %s\n", IPAD_ADDR, err.Error())
		return
	}
	go heartBeatSender()

	ahrsReader := bufio.NewReader(ahrsfp)
	gpsReader := bufio.NewReader(gpsfp)

	lastTs := ahrsIdx
	for {
		gps, gpsIdx := getGPS(gpsReader)
		ahrs, ahrsIdx := getAHRS(ahrsReader)

		// Correct for drift between the samples.
		drift := int64(math.Abs(float64(ahrsIdx - gpsIdx)))
//		fmt.Printf("drift: %d\n", drift)
		if drift >= 200000000 {
			// There's a problem. One of the files is ahead of the other by more than 0.10s.
			if gpsIdx > ahrsIdx { // GPS got ahead of AHRS? When does this happen?
//				fmt.Printf("GPS sample ahead of AHRS - correcting\n")
				for gpsIdx - ahrsIdx >= 200000000 {
					ln, nidx := getAHRS(ahrsReader)
					ahrsIdx = nidx
					ahrs = ln
				}
			} else { // AHRS got ahead of GPS? This usually happens.
				for ahrsIdx - gpsIdx >= 200000000 {
//					fmt.Printf("AHRS sample ahead of GPS - correcting\n")
					ln, nidx := getGPS(gpsReader)
					gpsIdx = nidx
					gps = ln
				}
			}
		}

		myGPS = gps
		fmt.Printf("matchy: %d, %d\n", ahrs.Timestamp, gps.Timestamp)
		pitch := ahrs.FusedEuler[0] * (180.0 / math.Pi)
		roll := -ahrs.FusedEuler[1] * (180.0 / math.Pi)

		if cal_num < 20 { // Average the first 5 measurements and call this "level".
			cal_pitch = ((cal_pitch * float64(cal_num)) + pitch) / float64(cal_num + 1)
			cal_roll = ((cal_roll * float64(cal_num)) + roll) / float64(cal_num + 1)
			cal_num++
		}

		// Apply the calibration values.
		pitch -= cal_pitch
		roll -= cal_roll

		fmt.Printf("%f %f\n", pitch, roll)
		s := fmt.Sprintf("XATTStratux,%f,%f,%f", gps.Course, pitch, roll)
		outConn.Write([]byte(s))
		time.Sleep(time.Duration((ahrs.Timestamp - lastTs)/int64(replayspeed)))
		lastTs = ahrs.Timestamp

		// Now we're working with synced samples.
	}
}
