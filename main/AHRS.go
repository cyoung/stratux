/*
	Copyright (c) 2016 Tino B.
	Distributable under the terms of The "BSD New"" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	BNO055.go: AHRS
*/

package main

import (
	"bytes"
	"fmt"
	"github.com/tarm/serial"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
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

var msAHRSserialPort *serial.Port

var mReceiveBuffer bytes.Buffer    // serial receive buffer
var mReceiveBufferMutex sync.Mutex // sync access to the buffer

var mBNO055RWMutex sync.Mutex     // syncs access to the BNO055
var mBNO055ConfigMutex sync.Mutex // syncs the config mode of the BNO055. TODO should eventually sync all access to the BNO055

var mfWaitRead bool  // waiting for a Read
var mfWaitWrite bool // waiting for a Write

var mintWaitForData int // Amount of bytes expected to be read. 2 for a write, > 2 for a read
var mbtLastAHRS_WriteError byte
var mbtLastAHRS_ReadError byte
var mbtLastBNO055_WriteError byte
var mbtLastBNO055_ReadError byte

var mtReadRetryDelay time.Duration
var mtWriteRetryDelay time.Duration

var mfRun bool             // set to false to stop
var mfConnected bool       // if the serial port is open
var mfBNO055NeedsInit bool // if the BNO055 is initialized

var mbtDataReadBuffer []byte // where received data will be stored

var mfIsWindows bool // if we are in windows or linux
var mfInfoRead bool

func DebugPrintf(pintDebugLevel int, format string, v ...interface{}) {
	if globalSettings.DEBUG && globalSettings.DEBUGLevel >= pintDebugLevel {
		log.Printf(format, v...)
	}
}

func startAHRS() {
	mfInfoRead = false
	globalSettings.DEBUG = true
	globalSettings.DEBUGLevel = DebugLevelAll

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

	mtReadRetryDelay = 100
	mtWriteRetryDelay = 100

	//mySituation.mu_AHRS = &sync.Mutex{}
	mfIsWindows = strings.Contains(strings.ToLower(os.Getenv("OS")), "windows")

	//BNO055_LoadCalibrationData()

	// create the serial receive buffer
	ReadBuffer := make([]byte, 500)
	mReceiveBuffer := bytes.NewBuffer(ReadBuffer)
	mReceiveBuffer.Reset()

	mfBNO055NeedsInit = true
	mfRun = true
	go BNO055_Receiver()
	go BNO055_QueueWorker()

	//_, _, calibrationData := BNO055_ReadCalibrationData()
	//BNO055_SaveCalibrationData(BNO055_CalibrationDataFilename(), calibrationData)

	timer := time.NewTicker(100 * time.Millisecond) // ~10Hz update.
	for {
		if mfConnected { // serial port is open
			if mfBNO055NeedsInit {
				if BNO055_InitLoop() { // TODO infinite loop if never initializes
					result, calibrationStatus, calibrationData := BNO055_ReadCalibrationData()
					if result {
						if calibrationStatus == 0x0F { // only save if it's fully calibrated
							// write calibration data to file
							BNO055_SaveCalibrationData(BNO055_CalibrationDataFilename(), calibrationData)
						} else {
							DebugPrintf(DebugLevelWarning, "BNO055 - Not fully calibrated, don't write calibration data.")
						}
					} else {
						DebugPrintf(DebugLevelWarning, "BNO055 - Couldn't read calibration data.")
					}
				}
			} else {
				if !mfInfoRead {
					mfInfoRead = true
					BNO055GetInfo()
				}
				mBNO055ConfigMutex.Lock() // so no configuration takes places while reading
				BNO055_LoopSimple()
				mBNO055ConfigMutex.Unlock()
			}
		}
		//time.Sleep(200 * time.Millisecond)
		<-timer.C
	}
}

func BNO055_Reset() bool {
	//result := true
	BNO055_Write(BNO055_SYS_TRIGGER, []byte{0x20}, 1) // answer is 0xEE, but this will be ignored. Only write once
	time.Sleep(300 * time.Millisecond)
	// try to read system status
	return BNO055_ReadBool(BNO055_SYS_STATUS, 5)
}

func BNO055_Init(pfWriteCalibration bool) bool {
	// Reset
	mfBNO055NeedsInit = true
	result := BNO055_Reset()
	if result {
		if pfWriteCalibration {
			// Write Calibration data
			result, data := BNO055_LoadCalibrationData()
			if result {
				result = BNO055_WriteCalibrationData(data)
				if !result {
					DebugPrintf(DebugLevelError, "BNO055_Init - Couldn't write data!")
				}
			} else {
				result = false
				DebugPrintf(DebugLevelError, "BNO055_Init - Couldn't get data!")
			}
		}
	} else {
		DebugPrintf(DebugLevelError, "BNO055_Init - Couldn't reset!")
	}
	return result
}

func BNO055_InitLoop() bool {
	result := false
	for mfRun && mfConnected {
		temp := BNO055_Init(true)
		if temp {
			DebugPrintf(DebugLevelInfo, "Init BNO055 successful!")
			mfBNO055NeedsInit = false
			result = true
			break
		} else {
			DebugPrintf(DebugLevelError, "BNO055_InitLoop - Init FAILED!")
			time.Sleep(time.Millisecond * 500)
		}
	}
	return result
}

func BNO055_SetOPRMode(pBNO055_OPRMode byte) bool {
	return BNO055_Write(BNO055_OPR_MODE, []byte{pBNO055_OPRMode}, 3)
}

func BNO055GetInfo() {
	globalSettings.BNO055Status = BNO055_ReadStatusString()
	globalSettings.BNO055IDs = BNO055_ReadIDsRevsString()

	result, _, calibrationData := BNO055_ReadCalibrationData()
	if result {
		var data string
		for i := 0; i < 18; i++ {
			data += fmt.Sprintf("%02X", calibrationData[i])
		}
		globalSettings.BNO055Calibration = data
	} else {
		DebugPrintf(DebugLevelWarning, "BNO055GetInfo - Couldn't read calibration data.")
	}
}

func BNO055_LoopSimple() {
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
				DebugPrintf(DebugLevelError, "BNO055_ReadStatus - Error ERR")
			}
		} else {
			DebugPrintf(DebugLevelError, "BNO055_ReadStatus - Error SYS")
		}
	} else {
		DebugPrintf(DebugLevelError, "BNO055_ReadStatus - Error CALIB")
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
		DebugPrintf(DebugLevelError, "BNO055_ReadStatusString - Error")
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
						result, temp := BNO055_ReadBytesBool(BNO055_SW_REV_ID_LSB, 2, 3)
						if result {
							SW_REV_ID = uint16(temp[0]) | uint16(temp[1])<<8
						} else {
							DebugPrintf(DebugLevelError, "BNO055_ReadIDsRevs - Error SW_REV_ID")
						}
					} else {
						DebugPrintf(DebugLevelError, "BNO055_ReadIDsRevs - Error BL_REV_ID")
					}
				} else {
					DebugPrintf(DebugLevelError, "BNO055_ReadIDsRevs - Error GYR_ID")
				}
			} else {
				DebugPrintf(DebugLevelError, "BNO055_ReadIDsRevs - Error MAG_ID")
			}
		} else {
			DebugPrintf(DebugLevelError, "BNO055_ReadIDsRevs - Error ACC_ID")
		}
	} else {
		DebugPrintf(DebugLevelError, "BNO055_ReadIDsRevs - Error CHIP_ID")
	}
	return result, BNO_CHIP_ID, ACC_ID, MAG_ID, GYR_ID, SW_REV_ID, BL_REV_ID
}

func BNO055_ReadIDsRevsString() string {
	var text string
	result, BNO_CHIP_ID, ACC_ID, MAG_ID, GYR_ID, SW_REV_ID, BL_REV_ID := BNO055_ReadIDsRevs()
	if result {
		text = fmt.Sprintf("BNO055 ID: %02X, ACC_ID: %02X,  MAG_ID: %02X, GYR_ID: %02X, SW_REV_ID: %04X, BL_REV_ID: %d", BNO_CHIP_ID, ACC_ID, MAG_ID, GYR_ID, SW_REV_ID, BL_REV_ID)
	} else {
		DebugPrintf(DebugLevelError, "BNO055_ReadIDsRevsString - Error")
	}
	return text
}

func BNO055_ReadAxis() (byte, bool) {
	var config byte
	result, data := BNO055_ReadBytesBool(BNO055_AXIS_MAP_CONFIG, 2, 3)
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
		DebugPrintf(DebugLevelError, "BNO055_ReadAxis - Error BNO055_ReadAxis")
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

		mBNO055ConfigMutex.Lock() // TODO put this in BNO055_SetOPRMode
		if BNO055_SetOPRMode(BNO055_OPRModeCONFIGMODE) {
			result = BNO055_Write(BNO055_AXIS_MAP_CONFIG, data, 3)

			// Select BNO055 system operation mode
			if !BNO055_SetOPRMode(BNO055_OPRModeNDOF) {
				DebugPrintf(DebugLevelError, "BNO055_WriteAxis - Couldn't get BNO055 NDOF mode")
			}
		} else {
			DebugPrintf(DebugLevelError, "BNO055_WriteAxis - Couldn't get BNO055 config mode")
		}
		mBNO055ConfigMutex.Unlock()
	}
	return result
}

func BNO055_CalibrationDataFilename() string {
	var file string
	if mfIsWindows {
		file = calibrationLocation_windows
	} else {
		file = calibrationLocation_rpi
	}
	return file
}

func BNO055_LoadCalibrationDataString() string {
	var text string
	result, data := BNO055_LoadCalibrationData()
	if result {
		for i := 0; i < len(data); i++ {
			text += fmt.Sprintf("%02X", data[i])
		}
	}
	return text
}

func BNO055_ParseCalibrationData(pstrCalibrationData string) (bool, []byte) {
	result := false
	calibration := make([]byte, 18)
	if len(pstrCalibrationData) == 36 {
		for i := 0; i < 18; i++ {
			// try to parse
			temp, err := strconv.ParseUint(pstrCalibrationData[i*2:i*2+2], 16, 8)
			if err != nil {
				DebugPrintf(DebugLevelError, "BNO055_ParseCalibrationData - Calibration data (%s) error: %s", pstrCalibrationData, err)
			} else {
				calibration[i] = byte(temp)
				result = true
				DebugPrintf(DebugLevelInfo, "BNO055_ParseCalibrationData - Calibration data: %x", calibration)
			}
		}
	} else {
		DebugPrintf(DebugLevelError, "BNO055_ParseCalibrationData - Calibration data (%s) has the wrong length. 36 expected", pstrCalibrationData)
	}
	return result, calibration
}

func BNO055_LoadCalibrationData() (bool, []byte) {
	// get from settings
	result := false
	var calibration []byte
	file := BNO055_CalibrationDataFilename()
	if _, err := os.Stat(file); err == nil {
		// file exists
		content, err := ioutil.ReadFile(file)
		if err != nil {
			DebugPrintf(DebugLevelError, "BNO055_LoadCalibrationData - Cannot open %s to read calibration data!", file)
		} else {
			lines := strings.Split(string(content), "\n")
			if len(lines) >= 1 {
				result, calibration = BNO055_ParseCalibrationData(file)
			}
		}
	} else {
		DebugPrintf(DebugLevelWarning, "BNO055_LoadCalibrationData - Cannot open %s to read calibration data!\r\n%s", file, err)
	}
	//return []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	//return []byte{0xFF, 0xF8, 0xFF, 0xFF, 0x00, 0x05, 0xFF, 0x57, 0x00, 0x59, 0xFF, 0x5A, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0x01}
	return result, calibration
}

func BNO055_SaveCalibrationData(pstrFile string, calibrationData []byte) bool {
	// write calibration data to file
	var data string
	canwrite := false
	result := false
	for i := 0; i < 18; i++ {
		//data += PadLeft(strconv.FormatUint(uint64(calibrationData[i]), 16),"0",2)
		data += fmt.Sprintf("%02X", calibrationData[i])
	}
	text := make([]byte, 36)
	copy(text[:], data)
	if _, err := os.Stat(pstrFile); err != nil {
		// file doesn't exist, create it
		file, err := os.Create(pstrFile)
		if err != nil {
			DebugPrintf(DebugLevelError, "BNO055_SaveCalibrationData - Cannot create %s to write calibration data! Error: %s", pstrFile, err)
		} else {
			DebugPrintf(DebugLevelInfo, "BNO055_SaveCalibrationData - Calibration file %s created.", file.Name())
			canwrite = true
		}
	}
	if canwrite {
		err := ioutil.WriteFile(pstrFile, text, os.ModePerm)
		if err != nil {
			DebugPrintf(DebugLevelError, "BNO055_SaveCalibrationData - Cannot open %s to write calibration data! Error: %s", pstrFile, err)
		} else {
			DebugPrintf(DebugLevelInfo, "BNO055_SaveCalibrationData - Calibration data written.")
			result = true
		}
	}
	return result
}

func BNO055_ReadCalibrationData() (bool, byte, []byte) {
	mBNO055ConfigMutex.Lock()
	calibrationStatus, result := BNO055_ReadByteBool(BNO055_CALIB_STAT, 3)
	calibrationData := make([]byte, 18)

	if result {
		DebugPrintf(DebugLevelInfo, "BNO055 Calibration status: MAG: %d, ACC: %d, GYR: %d, SYS: %d\r\n", calibrationStatus&0x03, (calibrationStatus>>2)&0x03, (calibrationStatus>>4)&0x03, (calibrationStatus>>6)&0x03)

		// Select BNO055 config mode
		if BNO055_SetOPRMode(BNO055_OPRModeCONFIGMODE) {
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
			if !BNO055_SetOPRMode(BNO055_OPRModeNDOF) {
				DebugPrintf(DebugLevelError, "BNO055_ReadCalibrationData - Couldn't get BNO055 NDOF mode")
			}
		} else {
			DebugPrintf(DebugLevelError, "BNO055_ReadCalibrationData - Couldn't get BNO055 config mode")
		}
	} else {
		DebugPrintf(DebugLevelError, "BNO055_ReadCalibrationData - Couldn't get calibration status")
	}

	defer mBNO055ConfigMutex.Unlock()
	return result, calibrationStatus, calibrationData
}

// switches to config and writes
func BNO055_WriteCalibrationData(pbtCalibrationData []byte) bool {
	mBNO055ConfigMutex.Lock()
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
				DebugPrintf(DebugLevelError, "BNO055_WriteCalibrationData - Couldn't set NDOF mode!")
			}
			//}
		} else {
			DebugPrintf(DebugLevelError, "BNO055_WriteCalibrationData - Couldn't write data!")
		}
	} else {
		DebugPrintf(DebugLevelError, "BNO055_WriteCalibrationData - Couldn't set config mode!")
	}
	defer mBNO055ConfigMutex.Unlock()
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

func BNO055_Read(pbtRegister byte, pbtCount byte, pintAttempts int) []byte {
	mBNO055RWMutex.Lock()
	var result []byte
	if mfConnected && pbtCount > 0 && pbtCount < 128 /*&& openCom(cboComPort.Text)*/ {
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

		for i := 0; i < pintAttempts && mfConnected; i++ {
			if i > 0 {
				DebugPrintf(DebugLevelVerbose, "retry read")
			}
			mintWaitForData = int(pbtCount + 2)
			mfWaitRead = true
			mbtLastAHRS_ReadError = AHRS_ReadErrorNone
			DebugPrintf(DebugLevelVerbose, "rs")
			count, err := msAHRSserialPort.Write(Buffer)
			if err != nil {
				mfConnected = false
				msAHRSserialPort.Close()
				DebugPrintf(DebugLevelError, "BNO055_Read - Disconnected3. Error:", err)
				time.Sleep(500 * time.Millisecond)
			} else {
				if count == len(Buffer) {
					counter := 0
					for counter < int(pbtCount)+100 && mfWaitRead {
						time.Sleep(1 * time.Millisecond)
						counter++
					}
					if !mfWaitRead && mbtLastAHRS_ReadError == AHRS_ReadErrorNone { // success
						if len(mbtDataReadBuffer) == int(pbtCount) {
							result = mbtDataReadBuffer
							DebugPrintf(DebugLevelVerbose, "ro")
							break
						} else {
							DebugPrintf(DebugLevelError, "BNO055_Read - Read Wrong Data")
							mbtLastAHRS_ReadError = AHRS_ReadErrorWrongData
						}
					} else {
						if mbtLastAHRS_ReadError == AHRS_ReadErrorNone {
							mbtLastAHRS_ReadError = AHRS_ReadErrorTimeout
							DebugPrintf(DebugLevelError, "BNO055_Read - Read Timeout")
						} else {
							DebugPrintf(DebugLevelError, "BNO055_Read - Read Error")
						}
					}
				} else {
					DebugPrintf(DebugLevelError, "BNO055_Read - Read Error Serialport")
					mbtLastAHRS_ReadError = AHRS_ReadErrorSerialport
				}
			}
			time.Sleep(mtReadRetryDelay * time.Millisecond)
		}
	}
	defer mBNO055RWMutex.Unlock()
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

func BNO055_ReadBytesBool(pbtRegister byte, pbtCount byte, pintAttempts int) (bool, []byte) {
	pbtData := BNO055_Read(pbtRegister, pbtCount, pintAttempts)
	return pbtData != nil && len(pbtData) == int(pbtCount), pbtData
}

func BNO055_Write(pbtRegister byte, pbtData []byte, pintAttempts int) bool {
	mBNO055RWMutex.Lock()
	result := false
	if mfConnected && pbtData != nil {
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

			for i := 0; i < pintAttempts && mfConnected; i++ {
				if i > 0 {
					DebugPrintf(DebugLevelVerbose, "retry write")
				}
				mintWaitForData = 2
				mfWaitWrite = true
				mbtLastAHRS_WriteError = AHRS_WriteErrorNone
				DebugPrintf(DebugLevelVerbose, "ws")
				count, err := msAHRSserialPort.Write(Buffer)
				if err != nil {
					mfConnected = false
					msAHRSserialPort.Close()
					DebugPrintf(DebugLevelError, "BNO055_Write - Disconnected2. Error:", err)
					//time.Sleep(500 * time.Millisecond)
				} else {
					if count == len(Buffer) {
						counter := 0
						for counter < int(length)+100 && mfWaitWrite {
							time.Sleep(1 * time.Millisecond)
							counter++
						}
						if !mfWaitWrite && mbtLastAHRS_WriteError == AHRS_WriteErrorNone { // success
							result = true
							DebugPrintf(DebugLevelVerbose, "wo")
							break
						} else {
							//DebugPrintf(DebugLevelVerbose, "we")
							if mbtLastAHRS_WriteError == AHRS_WriteErrorNone {
								mbtLastAHRS_WriteError = AHRS_WriteErrorTimeout
								DebugPrintf(DebugLevelError, "BNO055_Write - Write Timeout")
							} else {
								DebugPrintf(DebugLevelError, "BNO055_Write - Write Error")

							}
						}
					} else {
						DebugPrintf(DebugLevelError, "BNO055_Write - Write Error Serialport")
						mbtLastAHRS_WriteError = AHRS_WriteErrorSerialport
					}
				}
				time.Sleep(mtWriteRetryDelay * time.Millisecond)
			}
		}
	}
	defer mBNO055RWMutex.Unlock()
	return result
}

func BNO055_WriteByte(pbtRegister byte, pbtData byte, pintAttempts int) bool {
	return BNO055_Write(pbtRegister, []byte{pbtData}, pintAttempts)
}

func BNO055_Receiver() {
	buf := make([]byte, 50)
	for mfRun {
		if mfConnected {
			len, err := msAHRSserialPort.Read(buf)
			if err != nil {
				mfConnected = false
				msAHRSserialPort.Close()
				DebugPrintf(DebugLevelError, "BNO055_Receiver - Disconnected1. Error:", err)
				//time.Sleep(500 * time.Millisecond)
			} else if len > 0 {
				mReceiveBufferMutex.Lock()
				for i := 0; i < len; i++ {
					mReceiveBuffer.WriteByte(buf[i])
				}
				mReceiveBufferMutex.Unlock()
			}
			//time.Sleep(1 * time.Microsecond)	not needed since we're reading blocking
		} else { // not connected
			// attempt to reconnect
			DebugPrintf(DebugLevelInfo, "Attempt to reconnect")
			if BNO055_InitSerial() {
				DebugPrintf(DebugLevelInfo, "Connected")
				mfBNO055NeedsInit = true
				mfConnected = true
			} else {
				DebugPrintf(DebugLevelError, "BNO055_Receiver - Reconnect failed - wait")
				mfConnected = false
				time.Sleep(500 * time.Millisecond)
			}
		}
	}
	msAHRSserialPort.Close()
}

func BNO055_QueueWorker() {
	for mfRun {
		if mintWaitForData > 0 { // Waiting for data
			mReceiveBufferMutex.Lock()
			if mReceiveBuffer.Len() >= 2 { // an error could be reported - when resetting the BNO it sends a single '0xEE'. But recognizing that doesn't make sense
				result, _ := mReceiveBuffer.ReadByte() // Peek()
				if result == 0xEE {                    // Response
					result, _ = mReceiveBuffer.ReadByte() // Peek()
					if mintWaitForData == 2 {             // Write ACK/Error
						done := false
						if result == BNO055_WriteErrorWRITE_SUCCESS { // WRITE_SUCCESS
							mbtLastAHRS_WriteError = AHRS_WriteErrorNone
							done = true
						} else if result >= BNO055_WriteErrorMIN && result <= BNO055_WriteErrorMAX { // valid error
							mbtLastAHRS_WriteError = AHRS_WriteErrorSensor
							done = true
						} else { // invalid error, ignore. return byte to buffer
							mReceiveBuffer.UnreadByte()
						}
						if done {
							mbtLastBNO055_WriteError = result
							mintWaitForData = 0 // always before setting the handler
							mfWaitWrite = false
							DebugPrintf(DebugLevelVerbose, "wd")
						}
					} else { // Read Error
						if result >= BNO055_ReadErrorMIN && result <= BNO055_ReadErrorMAX { // valid error
							mbtLastBNO055_ReadError = result
							mbtLastAHRS_ReadError = AHRS_ReadErrorSensor
							mintWaitForData = 0 // always before setting the handler
							mfWaitWrite = false
							DebugPrintf(DebugLevelVerbose, "wd")
						} else { // invalid error, ignore. return byte to buffer
							mReceiveBuffer.UnreadByte()
						}
					}
				} else if result == 0xBB { // Read Response
					// check if all bytes are in
					if mReceiveBuffer.Len() >= mintWaitForData-1 { // -1 because we already read 1 byte
						length, _ := mReceiveBuffer.ReadByte()
						data := make([]byte, length)
						data[0] = length
						if length == (byte)(mintWaitForData-2) {
							mbtDataReadBuffer = make([]byte, length)
							for i := 0; i < int(length); i++ {
								mbtDataReadBuffer[i], _ = mReceiveBuffer.ReadByte()
							}
							mbtLastAHRS_ReadError = AHRS_ReadErrorNone
						} else {
							mbtLastAHRS_ReadError = AHRS_ReadErrorWrongLength
						}
						mintWaitForData = 0 // always before setting the handler
						mfWaitRead = false
						DebugPrintf(DebugLevelVerbose, "rd")
					} else { // return byte to buffer
						mReceiveBuffer.UnreadByte()
					}
				} /*else { // Wrong data - don't remove anything as the receiving routine will filter out invalid data
				}*/
			}
			mReceiveBufferMutex.Unlock()
		}
		/*else {    don't remove anything as the receiving routine will filter out invalid data
		}*/
		time.Sleep(1 * time.Microsecond) // to avoid high system usage
	}
}

func BNO055_InitSerial() bool {
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
	if mfIsWindows {
		device = "COM11"
	}

	if globalSettings.DEBUG {
		DebugPrintf(DebugLevelInfo, "Using %s for AHRS\n", device)
	}

	//serialConfig = &serial.Config{Name: device, Baud: baudrate, ReadTimeout: time.Microsecond * 1}
	serialConfig := &serial.Config{Name: device, Baud: baudrate, ReadTimeout: 0} // blocking read
	p, err := serial.OpenPort(serialConfig)
	if err != nil {
		DebugPrintf(DebugLevelError, "BNO055_InitSerial - serial port err: %s\n", err.Error())
		return false
	}

	msAHRSserialPort = p
	return true
}
