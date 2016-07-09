package main

import (
	"fmt"
	"log"
	"math"
	"time"

	_ "github.com/kidoman/embd/host/rpi"
)

//https://github.com/brianc118/MPU9250/blob/master/MPU9250.cpp

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

	return r == 0x48
}

func initMPU9250() {
	globalSettings.AHRS_Enabled = true

	//TODO: Calibration.

	setSetting(0x6B, 0x80) // Reset.
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

		// log.Printf("x_acc=%f, y_acc=%f, z_acc=%f\n", x_acc_f, y_acc_f, z_acc_f)

		// Get gyro data.
		x_gyro, err := i2cbus.ReadWordFromReg(0x68, 0x43)
		chkErr(err)
		y_gyro, err := i2cbus.ReadWordFromReg(0x68, 0x45)
		chkErr(err)
		z_gyro, err := i2cbus.ReadWordFromReg(0x68, 0x47)

		x_gyro_f := float64(int16(x_gyro)) * 0.00006103515625
		y_gyro_f := float64(int16(y_gyro)) * 0.00006103515625
		z_gyro_f := float64(int16(z_gyro)) * 0.00006103515625

		// log.Printf("x_gyro=%f, y_gyro=%f, z_gyro=%f\n", x_gyro_f, y_gyro_f, z_gyro_f)

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

		x_mag_f := float64(int16(x_mag)) * (10.0 * 4219.0 / 32760.0)
		y_mag_f := float64(int16(y_mag)) * (10.0 * 4219.0 / 32760.0)
		z_mag_f := float64(int16(z_mag)) * (10.0 * 4219.0 / 32760.0)

		// log.Printf("x_mag=%f, y_mag=%f, z_mag=%f\n", x_mag_f, y_mag_f, z_mag_f)

		// // "heading" not working with MPU9250 breakout board.

		// hdg := math.Atan2(y_mag_f, x_mag_f)

		// if hdg < 0 {
		// 	hdg += 2 * math.Pi
		// }

		// hdgDeg := hdg * 180.0 / math.Pi

		// fmt.Printf("---x_mag=%d, y_mag=%d, z_mag=%d\n", x_mag, y_mag, z_mag)
		// fmt.Printf("---x_mag_f=%f, y_mag_f=%f, z_mag_f=%f\n", x_mag_f, y_mag_f, z_mag_f)
		// fmt.Printf("***hdgDeg=%f\n", hdgDeg)

		AHRSupdate(convertToRadians(x_gyro_f), convertToRadians(y_gyro_f), convertToRadians(z_gyro_f), float64(x_acc_f), float64(y_acc_f), float64(z_acc_f), float64(x_mag_f), float64(y_mag_f), float64(z_mag_f))
	}
}

func calculateAttitude() {
	timer := time.NewTicker(30 * time.Millisecond) // ~33.3 Hz

	for {
		<-timer.C
		CalculateCurrentAttitudeXYZ()
	}
}

func convertToRadians(value float64) float64 {
	//return float64(value)
	//return float64((value/65535)*360.0) * math.Pi / 180.0
	//return float64((value / 65535) * 360.0)
	return value * math.Pi / 180.0
}
