package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"bufio"
	"github.com/kidoman/embd"
	_ "github.com/kidoman/embd/host/all"
	"github.com/kidoman/embd/sensor/bmp180"
	"github.com/tarm/serial"

	"../mpu6050"
	"os"
)

type SituationData struct {
	mu_GPS *sync.Mutex

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

	mu_Attitude *sync.Mutex

	// From BMP180 pressure sensor.
	temp              float64
	pressure_alt      float64
	lastTempPressTime time.Time

	// From MPU6050 accel/gyro.
	pitch            float64
	roll             float64
	gyro_heading     float64
	lastAttitudeTime time.Time
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
	ret[5] = byte((msglen & 0xFF00) << 8)
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
	serialConfig = &serial.Config{Name: device, Baud: 115200}
	p, err := serial.OpenPort(serialConfig)
	if err != nil {
		log.Printf("serial port err: %s\n", err.Error())
		return false
	}

	serialPort = p
	// Open port at 9600 baud for config.
	serialConfig = &serial.Config{Name: device, Baud: 9600}
	p, err = serial.OpenPort(serialConfig)
	if err != nil {
		log.Printf("serial port err: %s\n", err.Error())
		return false
	}

	// Set 10Hz update.
	p.Write(makeUBXCFG(0x06, 0x08, 6, []byte{0x64, 0x00, 0x00, 0x01, 0x00, 0x01}))

	// Set navigation settings.
	nav := make([]byte, 36)
	nav[0] = 0x05 // Set dyn and fixMode only.
	nav[1] = 0x00
	// dyn.
	nav[2] = 0x07 // "Airborne with >2g Acceleration".
	nav[3] = 0x02 // 3D only.

	p.Write(makeUBXCFG(0x06, 0x24, 36, nav))

	// Reconfigure serial port.
	cfg := make([]byte, 20)
	cfg[0] = 0x01 // portID.
	cfg[1] = 0x00 // res0.
	cfg[2] = 0x00 // res1.
	cfg[3] = 0x00 // res1.

	//	0000 0000 0000 0010 0011 0000 0000 0000 
	// UART mode. 0 stop bits, no parity, 8 data bits.
	cfg[4] = 0x00
	cfg[5] = 0x20
	cfg[6] = 0x30
	cfg[7] = 0x00

	// Baud rate.
	bdrt := uint32(115200)
	cfg[8] = byte((bdrt & 0xFF000000) << 24)
	cfg[9] = byte((bdrt & 0xFF0000) << 16)
	cfg[10] = byte((bdrt & 0xFF00) << 8)
	cfg[11] = byte(bdrt & 0xFF)

	// inProtoMask. NMEA and UBX.
	cfg[12] = 0x00
	cfg[13] = 0x03

	// outProtoMask. NMEA.
	cfg[14] = 0x00
	cfg[15] = 0x02

	cfg[16] = 0x00 // flags.
	cfg[17] = 0x00 // flags.

	cfg[18] = 0x00 //pad.
	cfg[19] = 0x00 //pad.


	p.Write(makeUBXCFG(0x06, 0x00, 20, cfg))

	p.Close()

	return true
}

func processNMEALine(l string) bool {
	replayLog(l, MSGCLASS_GPS)
	x := strings.Split(l, ",")
	if (x[0] == "$GNVTG") || (x[0] == "$GPVTG") { // Ground track information.
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
			//FIXME: Experimental. Set heading to true heading on the MPU6050 reader.
			if myMPU6050 != nil && globalStatus.RY835AI_connected && globalSettings.AHRS_Enabled {
				myMPU6050.ResetHeading(float64(tc))
			}
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

	} else if (x[0] == "$GNGGA") || (x[0] == "$GPGGA") { // GPS fix.
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
		mySituation.alt = float32(alt * 3.28084) // Convert to feet.

		//TODO: Altitude accuracy.
		mySituation.alt_accuracy = 0

		// Timestamp.
		mySituation.lastFixLocalTime = time.Now()

	} else if (x[0] == "$GNRMC") || (x[0] == "$GPRMC") {
		//$GNRMC,141228.00,A,x,N,y,W,0.289,,160915,,,A*7C
		//TODO.
	}
	return true
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

func makeFFAHRSSimReport() {
	s := fmt.Sprintf("XATTStratux,%f,%f,%f", mySituation.gyro_heading, mySituation.pitch, mySituation.roll)

	sendMsg([]byte(s), NETWORK_AHRS_FFSIM, false)
}

func makeAHRSGDL90Report() {
	msg := make([]byte, 16)
	msg[0] = 0x4c
	msg[1] = 0x45
	msg[2] = 0x01
	msg[3] = 0x00

	pitch := int16(float64(mySituation.pitch) * float64(10.0))
	roll := int16(float64(mySituation.roll) * float64(10.0))
	hdg := uint16(float64(mySituation.gyro_heading) * float64(10.0)) //TODO.
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

		mySituation.mu_Attitude.Lock()

		if err_mpu6050 != nil {
			log.Printf("readMPU6050(): %s\n", err_mpu6050.Error())
			globalStatus.RY835AI_connected = false
			break
		} else {
			mySituation.pitch = pitch
			mySituation.roll = roll
			mySituation.gyro_heading = myMPU6050.Heading() //FIXME. Experimental.
			mySituation.lastAttitudeTime = time.Now()
		}

		// Send, if valid.
		//		if isGPSGroundTrackValid(), etc.

		makeFFAHRSSimReport()
		makeAHRSGDL90Report()

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

func isTempPressValid() bool {
	return time.Since(mySituation.lastTempPressTime).Seconds() < 15
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
