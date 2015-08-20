package main

import (
	"log"
	"strconv"
	"strings"
	"time"
	"sync"
	"fmt"

	"github.com/kidoman/embd"
	_ "github.com/kidoman/embd/host/all"
	"github.com/kidoman/embd/sensor/bmp180"
	"github.com/tarm/serial"

	"./mpu6050"
)

type SituationData struct {
	mu_GPS					*sync.Mutex

	// From GPS.
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

	mu_Attitude				*sync.Mutex

	// From BMP180 pressure sensor.
	temp					float64
	pressure_alt			float64
	lastTempPressTime		time.Time

	// From MPU6050 accel/gyro.
	pitch					float64
	roll					float64
	lastAttitudeTime		time.Time
}

var serialConfig *serial.Config
var serialPort *serial.Port

func initGPSSerial() bool {
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
			mySituation.trueCourse = 0
			mySituation.groundSpeed = 0
			mySituation.lastGroundTrackTime = time.Time{}
			return true
		}
		groundSpeed, err := strconv.ParseFloat(x[5], 32) // Knots.
		if err != nil {
			return false
		}

		mySituation.trueCourse = uint16(trueCourse)
		mySituation.groundSpeed = uint16(groundSpeed)
		mySituation.lastGroundTrackTime = time.Now()

	} else if x[0] == "$GNGGA" { // GPS fix.
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
		sec, err3 := strconv.Atoi(x[1][4:6])
		if err1 != nil || err2 != nil || err3 != nil {
			return false
		}

		mySituation.lastFixSinceMidnightUTC = uint32((hr * 60 * 60) + (min * 60) + sec)

		// Latitude.
		if len(x[2]) < 10 {
			return false
		}
		hr, err1 = strconv.Atoi(x[2][0:2])
		minf, err2 := strconv.ParseFloat(x[2][2:10], 32)
		if err1 != nil || err2 != nil {
			return false
		}

		mySituation.lat = float32(hr) + float32(minf/60.0)
		if x[3] == "S" { // South = negative.
			mySituation.lat = -mySituation.lat
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

		mySituation.lng = float32(hr) + float32(minf/60.0)
		if x[5] == "W" { // West = negative.
			mySituation.lng = -mySituation.lng
		}

		// Quality indicator.
		q, err1 := strconv.Atoi(x[6])
		if err1 != nil {
			return false
		}
		mySituation.quality = uint8(q)

		// Satellites.
		sat, err1 := strconv.Atoi(x[7])
		if err1 != nil {
			return false
		}
		mySituation.satellites = uint16(sat)

		// Accuracy.
		hdop, err1 := strconv.ParseFloat(x[8], 32)
		if err1 != nil {
			return false
		}
		mySituation.accuracy = float32(hdop * 5.0) //FIXME: 5 meters ~ 1.0 HDOP?

		// Altitude.
		alt, err1 := strconv.ParseFloat(x[9], 32)
		if err1 != nil {
			return false
		}
		mySituation.alt = float32(alt * 3.28084) // Covnert to feet.

		//TODO: Altitude accuracy.
		mySituation.alt_accuracy = 0

		// Timestamp.
		mySituation.lastFixLocalTime = time.Now()

	}
	return true
}

func gpsSerialReader() {
	defer serialPort.Close()
	for globalSettings.GPS_Enabled && globalStatus.GPS_connected {
		buf := make([]byte, 1024)
		n, err := serialPort.Read(buf)
		if err != nil {
			log.Printf("gps serial read error: %s\n", err.Error())
			globalStatus.GPS_connected = false
			break
		}
		s := string(buf[:n])
		x := strings.Split(s, "\n")
		for _, l := range x {
			processNMEALine(l)
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
	if err != nil {
		return temp, altitude, err
	}
	return temp, altitude, nil
}

func readMPU6050() (float64, float64, error) {//TODO: error checking.
	pitch, roll := myMPU6050.PitchAndRoll()
	return pitch, roll, nil
}

func initBMP180() error {
	myBMP180 = bmp180.New(i2cbus) //TODO: error checking.
	return nil
}

func initMPU6050() error {
	myMPU6050 = mpu6050.New(i2cbus) //TODO: error checking.
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
			mySituation.temp = temp
			mySituation.pressure_alt = alt
			mySituation.lastTempPressTime = time.Now()
		}		
	}
	globalStatus.RY835AI_connected = false
}

func attitudeReaderSender() {
	timer := time.NewTicker(100 * time.Millisecond) // ~10Hz update.
	for globalStatus.RY835AI_connected && globalSettings.AHRS_Enabled {
		<-timer.C
		// Read pitch and roll.
		pitch, roll, err_mpu6050 := readMPU6050()

		mySituation.mu_Attitude.Lock()

		if err_mpu6050 != nil {
			log.Printf("readMPU6050(): %s\n", err_mpu6050.Error())
			globalStatus.RY835AI_connected = false
			break
		} else {
			mySituation.pitch = pitch
			mySituation.roll = roll
			mySituation.lastAttitudeTime = time.Now()
		}

		// Send, if valid.
//		if isGPSGroundTrackValid()
		s := fmt.Sprintf("XATTStratux,%d,%f,%f", mySituation.trueCourse, mySituation.pitch, mySituation.roll)

		sendMsg([]byte(s), NETWORK_AHRS)

		mySituation.mu_Attitude.Unlock()
	}
	globalStatus.RY835AI_connected = false
}

func isGPSValid() bool {
	return time.Since(mySituation.lastFixLocalTime).Seconds() < 15
}

func isGPSGroundTrackValid() bool {
	return time.Since(mySituation.lastGroundTrackTime).Seconds() < 15
}

func isAHRSValid() bool {
	return time.Since(mySituation.lastAttitudeTime).Seconds() < 1 // If attitude information gets to be over 1 second old, declare invalid.
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
	timer := time.NewTicker(10 * time.Second)
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
