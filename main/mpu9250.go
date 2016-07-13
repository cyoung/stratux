package main

import (
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/kidoman/embd"
	_ "github.com/kidoman/embd/host/all"
	_ "github.com/kidoman/embd/host/rpi"
)

//https://github.com/brianc118/MPU9250/blob/master/MPU9250.cpp

var magXcal, magYcal, magZcal float64

var i2cbus embd.I2CBus

func initI2C() error {
	i2cbus = embd.NewI2CBus(1) //TODO: error checking.
	return nil
}

func chkErr(err error) {
	if err != nil {
		fmt.Printf("err: %s\n", err.Error())
	}
}

func setSetting(addr, val byte) {
	err := i2cbus.WriteByteToReg(0x68, addr, val)
	if err != nil {
		fmt.Printf("WriteByteToReg(0x68, 0x%02X, 0x%02X): %s\n", addr, val, err.Error())
	}
	time.Sleep(1000 * time.Microsecond)
}

// 0x00 register on AK8963 is "0x48" - DeviceID.
func checkMagConnection() bool {
	setSetting(0x25, 0x0C|0x80)
	setSetting(0x26, 0x00)
	setSetting(0x27, 0x81) // Read one byte.

	time.Sleep(100 * time.Microsecond)

	r, err := i2cbus.ReadByteFromReg(0x68, 0x49)
	chkErr(err)

	ret := r == 0x48

	// Read calibration data.
	setSetting(0x25, 0x0C|0x80)
	setSetting(0x26, 0x10)
	setSetting(0x27, 0x83) // Read three bytes, (CalX, CalY, CalZ).

	mxcal, err := i2cbus.ReadByteFromReg(0x68, 0x49)
	chkErr(err)
	mycal, err := i2cbus.ReadByteFromReg(0x68, 0x4A)
	chkErr(err)
	mzcal, err := i2cbus.ReadByteFromReg(0x68, 0x4B)
	chkErr(err)

	magXcal = (float64(mxcal)-128)/256.0 + 1.0
	magYcal = (float64(mycal)-128)/256.0 + 1.0
	magZcal = (float64(mzcal)-128)/256.0 + 1.0

	return ret
}

func initMPU9250() {
	initI2C()
	globalSettings.AHRS_Enabled = true
	mySituation.mu_Attitude = &sync.Mutex{}

	//TODO: Calibration.

	setSetting(0x6B, 0x80) // Reset.
	time.Sleep(100 * time.Millisecond)
	setSetting(0x6B, 0x01) // Clock source.
	setSetting(0x6C, 0x00) // Enable accelerometer and gyro.

	setSetting(0x6A, 0x20) // I2C Master mode.
	setSetting(0x24, 0x0D) // I2C configuration multi-master, 400KHz.

	// AK8963 init.

	setSetting(0x25, 0x0C) // Set the I2C slave addres of AK8963 and set for write.
	setSetting(0x26, 0x0B) // I2C slave 0 register address from where to begin data transfer.
	setSetting(0x63, 0x01) // Reset AK8963.
	setSetting(0x27, 0x81) // Enable I2C and set 1 byte.

	setSetting(0x25, 0x0C) // Set the I2C slave addres of AK8963 and set for write.
	setSetting(0x26, 0x0A) // I2C slave 0 register address from where to begin data transfer.
	setSetting(0x63, 0x16) // Register value to 100Hz continuous measurement in 16bit.
	setSetting(0x27, 0x81) // Enable I2C and set 1 byte.

	// Accelerometer and gyro init.

	setSetting(0x19, 0x00) // Set Gyro 1000 Hz sample rate. rate = gyroscope output rate/(1 + value)
	setSetting(0x1A, 0x03) // Set low pass filter to 92 Hz.
	setSetting(0x1B, 0x00) // Set gyro sensitivity to 250dps.
	setSetting(0x1C, 0x00) // Set accelerometer scale to +/- 2G.
	setSetting(0x1D, 0x02) // Set Accel 1000 Hz sample rate.

	if !checkMagConnection() {
		log.Printf("magnetometer is offline.\n")
		return
	}

	go readRawData()
	go calculateAttitude()
	go calculateHeading()
}

func readRawData() {
	timer := time.NewTicker(2 * time.Millisecond)

	for {
		<-timer.C
		// Get accelerometer data.
		x_acc, err := i2cbus.ReadWordFromReg(0x68, 0x3B)
		chkErr(err)
		y_acc, err := i2cbus.ReadWordFromReg(0x68, 0x3D)
		chkErr(err)
		z_acc, err := i2cbus.ReadWordFromReg(0x68, 0x3F)

		// currently manually setting resolution
		x_acc_f := float64(int16(x_acc)) * 0.00006103515625
		y_acc_f := float64(int16(y_acc)) * 0.00006103515625
		z_acc_f := float64(int16(z_acc)) * 0.00006103515625

		// Get gyro data.
		x_gyro, err := i2cbus.ReadWordFromReg(0x68, 0x43)
		chkErr(err)
		y_gyro, err := i2cbus.ReadWordFromReg(0x68, 0x45)
		chkErr(err)
		z_gyro, err := i2cbus.ReadWordFromReg(0x68, 0x47)

		x_gyro_f := float64(int16(x_gyro)) * math.Pi / 131.0
		y_gyro_f := float64(int16(y_gyro)) * math.Pi / 131.0
		z_gyro_f := float64(int16(z_gyro)) * math.Pi / 131.0

		// Get magnetometer data.
		setSetting(0x25, 0x0C|0x80) // Set the I2C slave addres of AK8963 and set for read.
		setSetting(0x26, 0x03)      // I2C slave 0 register address from where to begin data transfer.
		setSetting(0x27, 0x87)      // Read 7 bytes from the magnetometer (HX+HY+HZ+ST2).
		x_mag, err := i2cbus.ReadWordFromReg(0x68, 0x49)
		chkErr(err)
		y_mag, err := i2cbus.ReadWordFromReg(0x68, 0x4B)
		chkErr(err)
		z_mag, err := i2cbus.ReadWordFromReg(0x68, 0x4D)

		st2, err := i2cbus.ReadByteFromReg(0x68, 0x4F) // ST2 register. Unlatch measurement data for next sample.
		chkErr(err)

		if st2&0x08 != 0 { // Measurement overflow. HOFL.
			fmt.Printf("mag: measurement overflow\n")
			continue // Don't use measurement.
		}

		x_mag_f := float64(int16(y_mag)) * 1.28785103785104 * magXcal
		y_mag_f := float64(int16(x_mag)) * 1.28785103785104 * magYcal
		z_mag_f := float64(int16(-z_mag)) * 1.28785103785104 * magZcal

		AHRSupdate(convertToRadians(x_gyro_f), convertToRadians(y_gyro_f), convertToRadians(z_gyro_f), float64(x_acc_f), float64(y_acc_f), float64(z_acc_f), float64(x_mag_f), float64(y_mag_f), float64(z_mag_f))
	}
}

func calculateAttitude() {
	timer := time.NewTicker(10 * time.Millisecond) // ~33.3 Hz

	for {
		<-timer.C
		CalculateCurrentAttitudeXYZ()
		//CalculateHeading()
	}
}

func calculateHeading() {
	timer := time.NewTicker(2 * time.Millisecond)

	for {
		<-timer.C
		CalculateHeading()
	}
}

func convertToRadians(value float64) float64 {
	return value * math.Pi / 180.0
}
