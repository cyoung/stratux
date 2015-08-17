package main

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/kidoman/embd"
	_ "github.com/kidoman/embd/host/all"
	"github.com/kidoman/embd/sensor/bmp180"
	"github.com/tarm/serial"
)

type GPSData struct {
	lastFixSinceMidnightUTC uint32
	lat                     float32
	lng                     float32
	quality                 uint8
	satellites              uint16
	accuracy                float32 // Meters.
	alt                     float32 // Feet.
	alt_accuracy            float32
	lastFixLocalTime        time.Time
	trueCourse              uint16
	groundSpeed             uint16
	lastGroundTrackTime     time.Time
}

var serialConfig *serial.Config
var serialPort *serial.Port

func initGPSSerialReader() bool {
	serialConfig = &serial.Config{Name: "/dev/ttyACM0", Baud: 9600}
	p, err := serial.OpenPort(serialConfig)
	if err != nil {
		log.Printf("serial port err: %s\n", err.Error())
		return false
	}
	serialPort = p
	return true
}

func processNMEALine(l string) bool {
	x := strings.Split(l, ",")
	if x[0] == "$GNVTG" { // Ground track information.
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
			myGPS.trueCourse = 0
			myGPS.groundSpeed = 0
			myGPS.lastGroundTrackTime = time.Time{}
			return true
		}
		groundSpeed, err := strconv.ParseFloat(x[5], 32) // Knots.
		if err != nil {
			return false
		}

		myGPS.trueCourse = uint16(trueCourse)
		myGPS.groundSpeed = uint16(groundSpeed)
		myGPS.lastGroundTrackTime = time.Now()
	} else if x[0] == "$GNGGA" { // GPS fix.
		if len(x) < 15 {
			return false
		}
		var fix GPSData

		fix = myGPS

		// Timestamp.
		if len(x[1]) < 9 {
			return false
		}
		hr, err1 := strconv.Atoi(x[1][0:2])
		min, err2 := strconv.Atoi(x[1][2:4])
		sec, err3 := strconv.Atoi(x[1][4:6])
		if err1 != nil || err2 != nil || err3 != nil {
			return false
		}

		fix.lastFixSinceMidnightUTC = uint32((hr * 60 * 60) + (min * 60) + sec)

		// Latitude.
		if len(x[2]) < 10 {
			return false
		}
		hr, err1 = strconv.Atoi(x[2][0:2])
		minf, err2 := strconv.ParseFloat(x[2][2:10], 32)
		if err1 != nil || err2 != nil {
			return false
		}

		fix.lat = float32(hr) + float32(minf/60.0)
		if x[3] == "S" { // South = negative.
			fix.lat = -fix.lat
		}

		// Longitude.
		if len(x[4]) < 11 {
			return false
		}
		hr, err1 = strconv.Atoi(x[4][0:3])
		minf, err2 = strconv.ParseFloat(x[4][3:11], 32)
		if err1 != nil || err2 != nil {
			return false
		}

		fix.lng = float32(hr) + float32(minf/60.0)
		if x[5] == "W" { // West = negative.
			fix.lng = -fix.lng
		}

		// Quality indicator.
		q, err1 := strconv.Atoi(x[6])
		if err1 != nil {
			return false
		}
		fix.quality = uint8(q)

		// Satellites.
		sat, err1 := strconv.Atoi(x[7])
		if err1 != nil {
			return false
		}
		fix.satellites = uint16(sat)

		// Accuracy.
		hdop, err1 := strconv.ParseFloat(x[8], 32)
		if err1 != nil {
			return false
		}
		fix.accuracy = float32(hdop * 5.0) //FIXME: 5 meters ~ 1.0 HDOP?

		// Altitude.
		alt, err1 := strconv.ParseFloat(x[9], 32)
		if err1 != nil {
			return false
		}
		fix.alt = float32(alt * 3.28084) // Covnert to feet.

		//TODO: Altitude accuracy.
		fix.alt_accuracy = 0

		// Timestamp.
		fix.lastFixLocalTime = time.Now()

		myGPS = fix

	}
	return true
}

func gpsSerialReader() {
	defer serialPort.Close()
	buf := make([]byte, 1024)
	for {
		if !globalSettings.GPS_Enabled { // GPS was turned off. Shut down.
			break
		}
		n, err := serialPort.Read(buf)
		if err != nil {
			log.Printf("gps unit read error: %s\n", err.Error())
			return
		}
		s := string(buf[:n])
		x := strings.Split(s, "\n")
		for _, l := range x {
			processNMEALine(l)
		}
	}
}

func gpsReader() {
	if initGPSSerialReader() {
		gpsSerialReader()
	} else {
		globalSettings.GPS_Enabled = false
	}
}

var bus embd.I2CBus
var i2csensor *bmp180.BMP180

func readBMP180() (float64, float64, error) { // ºCelsius, Meters
	temp, err := i2csensor.Temperature()
	if err != nil {
		return temp, 0.0, err
	}
	altitude, err := i2csensor.Altitude()
	if err != nil {
		return temp, altitude, err
	}
	return temp, altitude, nil
}

func initBMP180() {
	bus = embd.NewI2CBus(1)
	i2csensor = bmp180.New(bus)
}
