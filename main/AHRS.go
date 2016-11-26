/*
	Copyright (c) 2016 Tino B.
	Distributable under the terms of The "BSD New"" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	BNO055.go: AHRS
*/

package main

import (
	//"flag"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	//"bufio"
	"bytes"
	"io/ioutil"

	//"github.com/kidoman/embd"
	//_ "github.com/kidoman/embd/host/all"
	//"github.com/kidoman/embd/sensor/bmp180"
	"github.com/tarm/serial"
	//"serial"
	//"github.com/mikepb/go-serial"

	"os"
	//"BNO055"
	//"os/exec"
)

type AHRS_ReadError byte

const (
	AHRS_ReadErrorNone        = 0x00
	AHRS_ReadErrorSensor      = 0x01
	AHRS_ReadErrorTimeout     = 0x02
	AHRS_ReadErrorWrongAnswer = 0x03
	AHRS_ReadErrorWrongLength = 0x04
	AHRS_ReadErrorWrongData   = 0x05
	AHRS_ReadErrorSerialport  = 0x06
)

type AHRS_WriteError byte

const (
	AHRS_WriteErrorNone        = 0x00
	AHRS_WriteErrorSensor      = 0x01
	AHRS_WriteErrorTimeout     = 0x02
	AHRS_WriteErrorWrongAnswer = 0x03
	AHRS_WriteErrorSerialport  = 0x06
)

const (
	calibrationLocation_rpi     = "/etc/stratux_AHRS_calibration.conf"
	calibrationLocation_windows = "stratux_AHRS_calibration.conf"
)

//var serialConfig *serial.Config
var AHRSserialPort *serial.Port

var bf bytes.Buffer    // serial receive buffer
var bfMutex sync.Mutex // sync access to the buffer

var WaitRead bool  // waiting for a Read
var WaitWrite bool // waiting for a Write

var WaitForData int // Amount of bytes expected to be read. 2 for a write, > 2 for a read
var mbtLastAHRS_WriteError byte
var mbtLastAHRS_ReadError byte
var mbtLastBNO055_WriteError byte
var mbtLastBNO055_ReadError byte

var ReadRetryDelay time.Duration
var WriteRetryDelay time.Duration

var run bool             // set to false to stop
var connected bool       // if the serial port is open
var BNO055NeedsInit bool // if the BNO055 is initialized

var DataReadBuffer []byte // where received data will be stored

var isWindows bool // if we are in windows or linux

func startAHRS() {
	globalSettings.DEBUG = true
	globalSettings.DEBUGLevel = DebugLevelInfo

	DebugPrintf(DebugLevelInfo, "AHRS start")
	/*DebugPrintf(DebugLevel, "GOHOSTOS: %s", os.Getenv("HOSTOS"))
	DebugPrintf(DebugLevel, "GOOS: %s", os.Getenv("OS"))
	envs := os.Environ()

	for i:=0;i< len(envs);i++ {
		DebugPrintf(DebugLevel, "en $iv: %s", i, envs[i])
	}*/

	/*ports, err := getPorts()
	if err != nil {
		DebugPrintf(DebugLevel, "getPorts. Error:", err)
	} else {
		for i := 0; i < len(ports); i++ {
			DebugPrintf(ports[i])
		}
	}*/

	ReadRetryDelay = 100
	WriteRetryDelay = 100

	//mySituation.mu_AHRS = &sync.Mutex{}
	isWindows = strings.Contains(strings.ToLower(os.Getenv("OS")), "windows")

	BNO055_GetCalibrationData()

	// create the serial receive buffer
	ReadBuffer := make([]byte, 500)
	bf := bytes.NewBuffer(ReadBuffer)
	bf.Reset()

	BNO055NeedsInit = true
	run = true
	go BNO055_Receiver()
	go BNO055_QueueWorker()

	timer := time.NewTicker(100 * time.Millisecond) // ~10Hz update.
	for {
		if connected { // serial port is open
			if BNO055NeedsInit {
				if BNO055_InitLoop() { // TODO infinite loop if never initializes
					BNO055_ReadCalibration()
				}
			} else {
				AHRS_LoopSimple()
			}
		}
		//time.Sleep(200 * time.Millisecond)
		<-timer.C
	}
}

/*func PadLeft(str, pad string, length int) string {
	for {
		str = pad + str
		if len(str) > length {
			return str[0:length]
		}
	}
}*/

func DebugPrintf(pintDebugLevel int, format string, v ...interface{}) {
	if globalSettings.DEBUG && globalSettings.DEBUGLevel >= pintDebugLevel {
		log.Printf(format, v...)
	}
}

func AHRS_LoopSimple() {
	// Read calibration status
	//ReadCalibration()

	result, temp := BNO055_ReadQuaternions() // Read the x/y/z adc values
	if result {
		var quat [4]float64 // vector to hold quaternion
		// Calculate the quaternion values
		quat[0] = (float64)(temp[0]) / 16384.0
		quat[1] = (float64)(temp[1]) / 16384.0
		quat[2] = (float64)(temp[2]) / 16384.0
		quat[3] = (float64)(temp[3]) / 16384.0

		q_pitch, q_roll, q_yaw := toEulerianAngle(quat[0], quat[1], quat[2], quat[3])
		q_pitch = -(q_pitch * 180 / math.Pi) // invert for stratux
		q_roll = -(q_roll * 180 / math.Pi)   // invert for stratux
		if q_yaw > 0 {
			q_yaw = q_yaw * 360 / (2 * math.Pi)
		} else {
			q_yaw = (2*math.Pi + q_yaw) * 360 / (2 * math.Pi)
		}
		DebugPrintf(DebugLevelInfo, "HW Quat Y %f, P %f, R %f\r\n", q_yaw, q_pitch, q_roll)

		mySituation.Gyro_heading = q_yaw
		mySituation.Pitch = q_pitch
		mySituation.Roll = q_roll
		mySituation.LastAttitudeTime = time.Now()

		makeFFAHRSSimReport()
		makeAHRSGDL90Report()
	} else {
		DebugPrintf(DebugLevelError, "AHRS Error - ReadQuaternions")
	}
}

// Reads the System & Calibration Status
func BNO055_ReadStatus() (result bool, MAG byte, ACC byte, GYR byte, SYS byte, System byte, ERR byte) {
	calibrationStatus, result := BNO055_ReadByteBool(BNO055_CALIB_STAT, 3)
	if result {
		MAG = calibrationStatus & 0x03
		ACC = (calibrationStatus >> 2) & 0x03
		GYR = (calibrationStatus >> 4) & 0x03
		SYS = (calibrationStatus >> 6) & 0x03
		System, result = BNO055_ReadByteBool(BNO055_SYS_STATUS, 3)
		if result {
			ERR, result = BNO055_ReadByteBool(BNO055_SYS_ERR, 3)
			if !result {
				DebugPrintf(DebugLevelError, "BNO055 - Error BNO055_ReadStatus ERR")
			}
		} else {
			DebugPrintf(DebugLevelError, "BNO055 - Error BNO055_ReadStatus SYS")
		}
	} else {
		DebugPrintf(DebugLevelError, "BNO055 - Error BNO055_ReadStatus CALIB")
	}
	return result, MAG, ACC, GYR, SYS, System, ERR
}

func BNO055_ReadStatusString() string {
	var text string
	result, MAG, ACC, GYR, SYS, System, ERR := BNO055_ReadStatus()
	if result {
		text = fmt.Sprintf("BNO055 Calibration status: MAG: %d/3, ACC: %d/3, GYR: %d/3, SYS: %d/3\r\nBNO055 System status: ", MAG, ACC, GYR, SYS)
		switch System {
		case 0:
			text += "System idle"
		case 1:
			text += "System Error"
		case 2:
			text += "Initializing peripherals"
		case 3:
			text += "System Initialization"
		case 4:
			text += "Executing selftest"
		case 5:
			text += "Sensor fusion algorithm running"
		case 6:
			text += "System running without fusion algorithm"
		default:
			text += fmt.Sprintf("UNKNOWN (%d)", System)
		}
		text += "\r\nSystem Error: "
		switch ERR {
		case 0:
			text += "No error"
		case 1:
			text += "Peripheral initialization error"
		case 2:
			text += "System initialization error"
		case 3:
			text += "Self test result failed"
		case 4:
			text += "Register map value out of range"
		case 5:
			text += "Register map address out of range"
		case 6:
			text += "Register map write error"
		case 7:
			text += "BNO low power mode not available for selected operation mode"
		case 8:
			text += "Accelerometer power mode not available"
		case 9:
			text += "Fusion algorithm configuration error"
		case 0xA:
			text += "Sensor configuration error"
		default:
			text += fmt.Sprintf("UNKNOWN (%d)", System)
		}
		text += "\r\n"
	} else {
		DebugPrintf(DebugLevelError, "BNO055 - Error BNO055_ReadStatusString")
	}
	return text
}

// Reads the System IDs & Revisions
func BNO055_ReadIDsRevs() (result bool, BNO_CHIP_ID byte, ACC_ID byte, MAG_ID byte, GYR_ID byte, SW_REV_ID uint16, BL_REV_ID byte) {
	BNO_CHIP_ID, result = BNO055_ReadByteBool(BNO055_CHIP_ID, 3)
	if result {
		ACC_ID, result = BNO055_ReadByteBool(BNO055_ACC_ID, 3)
		if result {
			MAG_ID, result = BNO055_ReadByteBool(BNO055_MAG_ID, 3)
			if result {
				GYR_ID, result = BNO055_ReadByteBool(BNO055_GYR_ID, 3)
				if result {
					BL_REV_ID, result = BNO055_ReadByteBool(BNO055_BL_REV_ID, 3)
					if result {
						var temp []byte
						result := BNO055_ReadBytesBool(BNO055_SW_REV_ID_LSB, 2, temp, 3)
						if result {
							SW_REV_ID = uint16(temp[0]) | uint16(temp[1])<<8
						} else {
							DebugPrintf(DebugLevelError, "BNO055 - Error BNO055_ReadIDsRevs SW_REV_ID")
						}
					} else {
						DebugPrintf(DebugLevelError, "BNO055 - Error BNO055_ReadIDsRevs BL_REV_ID")
					}
				} else {
					DebugPrintf(DebugLevelError, "BNO055 - Error BNO055_ReadIDsRevs GYR_ID")
				}
			} else {
				DebugPrintf(DebugLevelError, "BNO055 - Error BNO055_ReadIDsRevs MAG_ID")
			}
		} else {
			DebugPrintf(DebugLevelError, "BNO055 - Error BNO055_ReadIDsRevs ACC_ID")
		}
	} else {
		DebugPrintf(DebugLevelError, "BNO055 - Error BNO055_ReadIDsRevs CHIP_ID")
	}
	return result, BNO_CHIP_ID, ACC_ID, MAG_ID, GYR_ID, SW_REV_ID, BL_REV_ID
}

func BNO055_ReadIDsRevsString() string {
	var text string
	result, BNO_CHIP_ID, ACC_ID, MAG_ID, GYR_ID, SW_REV_ID, BL_REV_ID := BNO055_ReadIDsRevs()
	if result {
		text = fmt.Sprintf("BNO055 ID: %d, ACC_ID: %d,  MAG_ID: %d, GYR_ID: %d, SW_REV_ID: %d, BL_REV_ID: %d", BNO_CHIP_ID, ACC_ID, MAG_ID, GYR_ID, SW_REV_ID, BL_REV_ID)
	} else {
		DebugPrintf(DebugLevelError, "BNO055 - Error BNO055_ReadIDsRevsString")
	}
	return text
}

func BNO055_ReadAxis() (byte, bool) {
	var data []byte
	var config byte
	result := BNO055_ReadBytesBool(BNO055_AXIS_MAP_CONFIG, 2, data, 3)
	if result {
		if data[0]&0x3F == 0x21 {
			switch data[1] & 0x03 {
			case 0x04:
				config = 0
			case 0x02:
				config = 3
			case 0x01:
				config = 5
			case 0x07:
				config = 6
			default:
				config = 8 // error
			}
		} else if data[0]&0x3F == 0x24 {
			switch data[2] & 0x03 {
			case 0x00:
				config = 1
			case 0x06:
				config = 2
			case 0x03:
				config = 4
			case 0x05:
				config = 7
			default:
				config = 8 // error
			}
		}
	} else {
		DebugPrintf(DebugLevelError, "BNO055 - Error BNO055_ReadAxis")
	}
	return config, result
}

func BNO055_WriteAxis(pbtAxis byte) bool {
	result := false
	var data = make([]byte, 2)
	if pbtAxis <= 7 {
		switch pbtAxis {
		case 0:
			data[0] = 0x21
			data[1] = 0x04
		case 1:
			data[0] = 0x24
			data[1] = 0x00
		case 2:
			data[0] = 0x24
			data[1] = 0x06
		case 3:
			data[0] = 0x21
			data[1] = 0x02
		case 4:
			data[0] = 0x24
			data[1] = 0x03
		case 5:
			data[0] = 0x21
			data[1] = 0x01
		case 6:
			data[0] = 0x21
			data[1] = 0x07
		case 7:
			data[0] = 0x24
			data[1] = 0x05
		default:
			data[0] = 0x24
			data[1] = 0x00
		}
		result = BNO055_Write(BNO055_AXIS_MAP_CONFIG, data, 3)
	}
	return result
}

func BNO055_ReadQuaternions() (bool, [4]int16) {
	result := false
	var data [4]int16
	temp := BNO055_Read(BNO055_QUA_DATA_W_LSB, 8, 3) // read in all data
	if temp != nil && len(temp) == 8 {
		data[0] = int16(temp[1])<<8 | int16(temp[0]) // convert to 16bit
		data[1] = int16(temp[3])<<8 | int16(temp[2])
		data[2] = int16(temp[5])<<8 | int16(temp[4])
		data[3] = int16(temp[7])<<8 | int16(temp[6])
		result = true
	}
	return result, data
}

func BNO055_ReadCalibration() (bool, []byte) {
	calibrationStatus, result := BNO055_ReadByteBool(BNO055_CALIB_STAT, 3)
	calibrationData := make([]byte, 18)

	if result {
		DebugPrintf(DebugLevelInfo, "BNO055 Calibration status: MAG: %d, ACC: %d, GYR: %d, SYS: %d\r\n", calibrationStatus&0x03, (calibrationStatus>>2)&0x03, (calibrationStatus>>4)&0x03, (calibrationStatus>>6)&0x03)

		// Select BNO055 config mode
		if BNO055_WriteByte(BNO055_OPR_MODE, BNO055_OPRModeCONFIGMODE, 3) {
			temp := BNO055_Read(BNO055_ACC_OFFSET_X_LSB, 6, 3)
			if temp != nil && len(temp) == 6 {
				DebugPrintf(DebugLevelInfo, "ACC X: %d, Y: %d, Z: %d\r\n", int16(temp[0])|int16(temp[1])<<8, int16(temp[2])|int16(temp[3])<<8, int16(temp[4])|int16(temp[5])<<8)
				//DebugPrintf(DebugLevel, "%d %d %d ", (temp[0] | temp[1]<<8), (temp[2] | temp[3]<<8), (temp[4] | temp[5]<<8))
				copy(calibrationData[0:6], temp)
			} else {
				DebugPrintf(DebugLevelInfo, "0 0 0 ")
			}
			temp = BNO055_Read(BNO055_MAG_OFFSET_X_LSB, 6, 3)
			if temp != nil && len(temp) == 6 {
				DebugPrintf(DebugLevelInfo, "MAG X: %d, Y: %d, Z: %d\r\n", int16(temp[0])|int16(temp[1])<<8, int16(temp[2])|int16(temp[3])<<8, int16(temp[4])|int16(temp[5])<<8)
				//DebugPrintf(DebugLevel, "%d %d %d ", (temp[0] | temp[1]<<8), (temp[2] | temp[3]<<8), (temp[4] | temp[5]<<8))
				copy(calibrationData[6:12], temp)
			} else {
				DebugPrintf(DebugLevelInfo, "0 0 0 ")
			}
			temp = BNO055_Read(BNO055_GYR_OFFSET_X_LSB, 6, 3)
			if temp != nil && len(temp) == 6 {
				DebugPrintf(DebugLevelInfo, "GYR X: %d, Y: %d, Z: %d\r\n", int16(temp[0])|int16(temp[1])<<8, int16(temp[2])|int16(temp[3])<<8, int16(temp[4])|int16(temp[5])<<8)
				//DebugPrintf(DebugLevel, "%d %d %d ", (temp[0] | temp[1]<<8), (temp[2] | temp[3]<<8), (temp[4] | temp[5]<<8))
				copy(calibrationData[12:18], temp)
			} else {
				DebugPrintf(DebugLevelInfo, "0 0 0 ")
			}

			DebugPrintf(DebugLevelInfo, "CalibrationData: %x", calibrationData)

			// Select BNO055 system operation mode
			BNO055_WriteByte(BNO055_OPR_MODE, BNO055_OPRModeNDOF, 3)

			if calibrationStatus == 0x0F { // only save if it's fully calibrated
				// write calibration data to file
				var data string
				for i := 0; i < 18; i++ {
					//data += PadLeft(strconv.FormatUint(uint64(calibrationData[i]), 16),"0",2)
					data += fmt.Sprintf("%02X", calibrationData[i])
				}
				var file string
				if isWindows {
					file = calibrationLocation_windows
				} else {
					file = calibrationLocation_rpi
				}
				// file exists
				text := make([]byte, 36)
				copy(text[:], data)
				err := ioutil.WriteFile(file, text, os.ModePerm)
				if err != nil {
					DebugPrintf(DebugLevelError, "BNO055 - Cannot open %s to write calibration data!", file)
				} else {
					DebugPrintf(DebugLevelInfo, "Calibration data written.")
				}
			} else {
				DebugPrintf(DebugLevelWarning, "BNO055 - Not fully calibrated, don't write calibration data.")
			}
		} else {
			DebugPrintf(DebugLevelError, "BNO055 - Couldn't get BNO055 config mode")
		}
	} else {
		DebugPrintf(DebugLevelError, "BNO055 - BNO055 - Couldn't get calibration status")
	}

	return result, calibrationData
}

func toEulerianAngle(quatw float64, quatx float64, quaty float64, quatz float64) (pitch float64, roll float64, yaw float64) {
	ysqr := quaty * quaty
	t0 := -2.0*(ysqr+quatz*quatz) + 1.0
	t1 := +2.0 * (quatx*quaty - quatw*quatz)
	t2 := -2.0 * (quatx*quatz + quatw*quaty)
	t3 := +2.0 * (quaty*quatz - quatw*quatx)
	t4 := -2.0*(quatx*quatx+ysqr) + 1.0

	if t2 > 1.0 {
		t2 = 1.0
	}
	if t2 < -1.0 {
		t2 = -1.0
	}

	// pitch and roll are switched
	//pitch = Math.Asin(t2)
	//roll = Math.Atan2(t3, t4)
	roll = math.Asin(t2)
	pitch = math.Atan2(t3, t4)
	yaw = math.Atan2(t1, t0)

	return pitch, roll, yaw
}

func BNO055_GetCalibrationData() (bool, []byte) {
	// get from settings
	result := true
	var file string
	var calibration []byte
	if isWindows {
		file = calibrationLocation_windows
	} else {
		file = calibrationLocation_rpi
	}
	if _, err := os.Stat(file); err == nil {
		// file exists
		content, err := ioutil.ReadFile(file)
		if err != nil {
			DebugPrintf(DebugLevelError, "BNO055 - Cannot open %s to read calibration data!", file)
		} else {
			lines := strings.Split(string(content), "\n")
			if len(lines) >= 1 && len(lines[0]) == 36 {
				calibration = make([]byte, 18)
				for i := 0; i < 18; i++ {
					// try to parse
					temp, err := strconv.ParseUint(lines[0][i*2:i*2+2], 16, 8)
					if err != nil {
						DebugPrintf(DebugLevelError, "BNO055 - Calibration data error: %s", file, err)
					} else {
						calibration[i] = byte(temp)
					}
				}
				DebugPrintf(DebugLevelInfo, "Calibration data: %x", calibration)
			}
		}
	} else {
		DebugPrintf(DebugLevelWarning, "BNO055 - Cannot open %s to read calibration data!\r\n%s", file, err)
	}
	//return []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	//return []byte{0xFF, 0xF8, 0xFF, 0xFF, 0x00, 0x05, 0xFF, 0x57, 0x00, 0x59, 0xFF, 0x5A, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0x01}
	return result, calibration
}

func BNO055_InitLoop() bool {
	result := false
	for run && connected {
		temp := BNO055_Init(true)
		if temp {
			DebugPrintf(DebugLevelInfo, "Init BNO055 successful!")
			BNO055NeedsInit = false
			result = true
			break
		} else {
			DebugPrintf(DebugLevelError, "BNO055 - Init FAILED!")
			time.Sleep(time.Millisecond * 500)
		}
	}
	return result
}

// switches to config and writes
func BNO055_WriteCalibrationData(pbtCalibrationData []byte) bool {
	// Config mode
	result := BNO055_SetOPRMode(BNO055_OPRModeCONFIGMODE)
	if result {
		// Write Calibration data
		result = BNO055_Write(BNO055_ACC_OFFSET_X_LSB, pbtCalibrationData, 5)
		if result {
			//result = Write_BNO055Byte(BNO055_UNIT_SEL, 0x80, 3) // set Android - no influence on Quaterions
			//if result {
			// Switch to NDOF
			result = BNO055_SetOPRMode(BNO055_OPRModeNDOF)
			if !result {
				DebugPrintf(DebugLevelError, "BNO055 - BNO055_WriteCalibrationData - Couldn't set NDOF mode!")
			}
			//}
		} else {
			DebugPrintf(DebugLevelError, "BNO055 - BNO055_WriteCalibrationData - Couldn't write data!")
		}
	} else {
		DebugPrintf(DebugLevelError, "BNO055 - BNO055_WriteCalibrationData - Couldn't set config mode!")
	}
	return result
}

func BNO055_Init(pfWriteCalibration bool) bool {
	// Reset
	result := BNO055_Reset()
	if result {
		if pfWriteCalibration {
			// Write Calibration data
			result, data := BNO055_GetCalibrationData()
			if result {
				result = BNO055_WriteCalibrationData(data)
				if !result {
					DebugPrintf(DebugLevelError, "BNO055 - BNO055_Init - Couldn't write data!")
				}
			} else {
				result = false
				DebugPrintf(DebugLevelError, "BNO055 - BNO055_Init - Couldn't get data!")
			}
		}
	} else {
		DebugPrintf(DebugLevelError, "BNO055 - BNO055_Init - Couldn't reset!")
	}
	return result
}

func BNO055_SetOPRMode(pBNO055_OPRMode byte) bool {
	return BNO055_Write(BNO055_OPR_MODE, []byte{pBNO055_OPRMode}, 3)
}

func BNO055_Reset() bool {
	//result := true
	BNO055_Write(BNO055_SYS_TRIGGER, []byte{0x20}, 1) // answer is 0xEE, but this will be ignored. Only write once
	time.Sleep(300 * time.Millisecond)
	// try to read system status
	return BNO055_ReadBool(BNO055_SYS_STATUS, 5)
}

func BNO055_Receiver() {
	buf := make([]byte, 50)
	for run {
		if connected {
			len, err := AHRSserialPort.Read(buf)
			if err != nil {
				connected = false
				AHRSserialPort.Close()
				DebugPrintf(DebugLevelError, "BNO055 - Disconnected1. Error:", err)
				//time.Sleep(500 * time.Millisecond)
			} else if len > 0 {
				bfMutex.Lock()
				for i := 0; i < len; i++ {
					bf.WriteByte(buf[i])
				}
				bfMutex.Unlock()
			}
			//time.Sleep(1 * time.Microsecond)	not needed since we're reading blocking
		} else { // not connected
			// attempt to reconnect
			DebugPrintf(DebugLevelInfo, "Attempt to reconnect")
			if AHRS_InitSerial() {
				DebugPrintf(DebugLevelInfo, "Connected")
				BNO055NeedsInit = true
				connected = true
			} else {
				DebugPrintf(DebugLevelError, "BNO055 - Reconnect failed - wait")
				connected = false
				time.Sleep(500 * time.Millisecond)
			}
		}
	}
	AHRSserialPort.Close()
}

func BNO055_QueueWorker() {
	for run {
		if WaitForData > 0 { // Waiting for data
			bfMutex.Lock()
			if bf.Len() >= 2 { // an error could be reported - when resetting the BNO it sends a single '0xEE'. But recognizing that doesn't make sense
				result, _ := bf.ReadByte() // Peek()
				if result == 0xEE {        // Response
					result, _ = bf.ReadByte() // Peek()
					if WaitForData == 2 {     // Write ACK/Error
						done := false
						if result == BNO055_WriteErrorWRITE_SUCCESS { // WRITE_SUCCESS
							mbtLastAHRS_WriteError = AHRS_WriteErrorNone
							done = true
						} else if result >= BNO055_WriteErrorMIN && result <= BNO055_WriteErrorMAX { // valid error
							mbtLastAHRS_WriteError = AHRS_WriteErrorSensor
							done = true
						} else { // invalid error, ignore. return byte to buffer
							bf.UnreadByte()
						}
						if done {
							mbtLastBNO055_WriteError = result
							WaitForData = 0 // always before setting the handler
							WaitWrite = false
							DebugPrintf(DebugLevelVerbose, "wd")
						}
					} else { // Read Error
						if result >= BNO055_ReadErrorMIN && result <= BNO055_ReadErrorMAX { // valid error
							mbtLastBNO055_ReadError = result
							mbtLastAHRS_ReadError = AHRS_ReadErrorSensor
							WaitForData = 0 // always before setting the handler
							WaitWrite = false
							DebugPrintf(DebugLevelVerbose, "wd")
						} else { // invalid error, ignore. return byte to buffer
							bf.UnreadByte()
						}
					}
				} else if result == 0xBB { // Read Response
					// check if all bytes are in
					if bf.Len() >= WaitForData-1 { // -1 because we already read 1 byte
						length, _ := bf.ReadByte()
						data := make([]byte, length)
						data[0] = length
						if length == (byte)(WaitForData-2) {
							DataReadBuffer = make([]byte, length)
							for i := 0; i < int(length); i++ {
								DataReadBuffer[i], _ = bf.ReadByte()
							}
							mbtLastAHRS_ReadError = AHRS_ReadErrorNone
						} else {
							mbtLastAHRS_ReadError = AHRS_ReadErrorWrongLength
						}
						WaitForData = 0 // always before setting the handler
						WaitRead = false
						DebugPrintf(DebugLevelVerbose, "rd")
					} else { // return byte to buffer
						bf.UnreadByte()
					}
				} /*else { // Wrong data - don't remove anything as the receiving routine will filter out invalid data
				}*/
			}
			bfMutex.Unlock()
		}
		/*else {    don't remove anything as the receiving routine will filter out invalid data
		}*/
		time.Sleep(1 * time.Microsecond) // to avoid high system usage
	}
}

func BNO055_Write(pbtRegister byte, pbtData []byte, pintAttempts int) bool {
	result := false
	if connected && pbtData != nil {
		length := byte(len(pbtData))
		if length > 0 && length <= 128 {
			Buffer := make([]byte, 4+length)
			index := byte(0)
			Buffer[index] = 0xAA // Start
			index++
			Buffer[index] = 0x00 // Write
			index++
			Buffer[index] = pbtRegister // Register
			index++
			Buffer[index] = length // Length
			index++
			copy(Buffer[index:index+length], pbtData)

			for i := 0; i < pintAttempts && connected; i++ {
				if i > 0 {
					DebugPrintf(DebugLevelVerbose, "retry write")
				}
				WaitForData = 2
				WaitWrite = true
				mbtLastAHRS_WriteError = AHRS_WriteErrorNone
				DebugPrintf(DebugLevelVerbose, "ws")
				count, err := AHRSserialPort.Write(Buffer)
				if err != nil {
					connected = false
					AHRSserialPort.Close()
					DebugPrintf(DebugLevelError, "BNO055 - Disconnected2. Error:", err)
					//time.Sleep(500 * time.Millisecond)
				} else {
					if count == len(Buffer) {
						counter := 0
						for counter < int(length)+100 && WaitWrite {
							time.Sleep(1 * time.Millisecond)
							counter++
						}
						if !WaitWrite && mbtLastAHRS_WriteError == AHRS_WriteErrorNone { // success
							result = true
							DebugPrintf(DebugLevelVerbose, "wo")
							break
						} else {
							//DebugPrintf(DebugLevelVerbose, "we")
							if mbtLastAHRS_WriteError == AHRS_WriteErrorNone {
								mbtLastAHRS_WriteError = AHRS_WriteErrorTimeout
								DebugPrintf(DebugLevelError, "BNO055 - Write Timeout")
							} else {
								DebugPrintf(DebugLevelError, "BNO055 - Write Error")

							}
						}
					} else {
						DebugPrintf(DebugLevelError, "BNO055 - Write Error Serialport")
						mbtLastAHRS_WriteError = AHRS_WriteErrorSerialport
					}
				}
				time.Sleep(WriteRetryDelay * time.Millisecond)
			}
		}
	}
	return result
}

func BNO055_WriteByte(pbtRegister byte, pbtData byte, pintAttempts int) bool {
	return BNO055_Write(pbtRegister, []byte{pbtData}, pintAttempts)
}

func BNO055_Read(pbtRegister byte, pbtCount byte, pintAttempts int) []byte {
	var result []byte
	if connected && pbtCount > 0 && pbtCount < 128 /*&& openCom(cboComPort.Text)*/ {
		Buffer := make([]byte, 4)
		index := 0
		Buffer[index] = 0xAA // Start
		index++
		Buffer[index] = 0x01 // Read
		index++
		Buffer[index] = pbtRegister // Register
		index++
		Buffer[index] = pbtCount // Count
		index++

		for i := 0; i < pintAttempts && connected; i++ {
			if i > 0 {
				DebugPrintf(DebugLevelVerbose, "retry read")
			}
			WaitForData = int(pbtCount + 2)
			WaitRead = true
			mbtLastAHRS_ReadError = AHRS_ReadErrorNone
			DebugPrintf(DebugLevelVerbose, "rs")
			count, err := AHRSserialPort.Write(Buffer)
			if err != nil {
				connected = false
				AHRSserialPort.Close()
				DebugPrintf(DebugLevelError, "BNO055 - Disconnected3. Error:", err)
				time.Sleep(500 * time.Millisecond)
			} else {
				if count == len(Buffer) {
					counter := 0
					for counter < int(pbtCount)+100 && WaitRead {
						time.Sleep(1 * time.Millisecond)
						counter++
					}
					if !WaitRead && mbtLastAHRS_ReadError == AHRS_ReadErrorNone { // success
						if len(DataReadBuffer) == int(pbtCount) {
							result = DataReadBuffer
							DebugPrintf(DebugLevelVerbose, "ro")
							break
						} else {
							DebugPrintf(DebugLevelError, "BNO055 - Read Wrong Data")
							mbtLastAHRS_ReadError = AHRS_ReadErrorWrongData
						}
					} else {
						if mbtLastAHRS_ReadError == AHRS_ReadErrorNone {
							mbtLastAHRS_ReadError = AHRS_ReadErrorTimeout
							DebugPrintf(DebugLevelError, "BNO055 - Read Timeout")
						} else {
							DebugPrintf(DebugLevelError, "BNO055 - Read Error")
						}
					}
				} else {
					DebugPrintf(DebugLevelError, "BNO055 - Read Error Serialport")
					mbtLastAHRS_ReadError = AHRS_ReadErrorSerialport
				}
			}
			time.Sleep(ReadRetryDelay * time.Millisecond)
		}
	}
	return result
}

func BNO055_ReadByteBool(pbtRegister byte, pintAttempts int) (byte, bool) {
	result := false
	data := byte(0)
	temp := BNO055_Read(pbtRegister, 1, pintAttempts)
	if temp != nil && len(temp) == 1 {
		data = temp[0]
		result = true
	} /*else {
		//pbtData = 0
	}*/
	return data, result
}

func BNO055_ReadBool(pbtRegister byte, pintAttempts int) bool {
	result := false
	temp := BNO055_Read(pbtRegister, 1, pintAttempts)
	if temp != nil && len(temp) == 1 {
		result = true
	}
	return result
}

func BNO055_ReadByte(pbtRegister byte, pintAttempts int) byte {
	data, result := BNO055_ReadByteBool(pbtRegister, pintAttempts)
	_ = result
	return data
}

func BNO055_ReadBytesBool(pbtRegister byte, pbtCount byte, pbtData []byte, pintAttempts int) bool {
	pbtData = BNO055_Read(pbtRegister, pbtCount, pintAttempts)
	return pbtData != nil && len(pbtData) == int(pbtCount)
}

func AHRS_InitSerial() bool {
	var device string
	baudrate := int(115200)

	if _, err := os.Stat("/dev/ttyUSB0"); err == nil { // USB serial adapter
		device = "/dev/ttyUSB0"
		//} else if _, err := os.Stat("/dev/ttyAMA0"); err == nil { // ttyAMA0 is PL011 UART (GPIO pins 8 and 10) on all RPi.
		//	device = "/dev/ttyAMA0"
	} else if _, err := os.Stat("COM11"); err == nil { // Windows
		device = "COM11"
	} else {
		//DebugPrintf(DebugLevel, "No suitable device found.\n")
		//return false
	}
	if isWindows {
		device = "COM11"
	}

	if globalSettings.DEBUG {
		DebugPrintf(DebugLevelInfo, "Using %s for AHRS\n", device)
	}

	//serialConfig = &serial.Config{Name: device, Baud: baudrate, ReadTimeout: time.Microsecond * 1}
	serialConfig := &serial.Config{Name: device, Baud: baudrate, ReadTimeout: 0} // blocking read
	p, err := serial.OpenPort(serialConfig)
	if err != nil {
		DebugPrintf(DebugLevelError, "BNO055 - serial port err: %s\n", err.Error())
		return false
	}

	AHRSserialPort = p
	return true
}
