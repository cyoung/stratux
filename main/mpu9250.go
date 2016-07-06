package main

import (
	"fmt"
	"log"
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

func initMPU9250() {
	globalSettings.AHRS_Enabled = true

	//TODO: Calibration.

	setSetting(0x6B, 0x80) // Reset.
	setSetting(0x6B, 0x01) // Clock source.
	setSetting(0x6C, 0x00) // Enable accelerometer and gyro.

	setSetting(0x6A, 0x20) // I2C Master mode.
	setSetting(0x24, 0x0D) // I2C configuration multi-master, IIC 400KHz.

	setSetting(0x25, 0x0C) // Set the I2C slave addres of AK8963 and set for write.
	setSetting(0x26, 0x0B) // I2C slave 0 register address from where to begin data transfer.

	setSetting(0x63, 0x01) // Reset AK8963.
	setSetting(0x27, 0x81) // Enable I2C and set 1 byte.

	setSetting(0x26, 0x0A) // I2C slave 0 register address from where to begin data transfer.
	setSetting(0x63, 0x12) // Register value to 8Hz continuous measurement in 16bit.

	go readRawData()
}

func readRawData() {
	for {
		// Get accelerometer data.
		x_acc, err := i2cbus.ReadWordFromReg(0x68, 0x3b)
		chkErr(err)
		y_acc, err := i2cbus.ReadWordFromReg(0x68, 0x3d)
		chkErr(err)
		z_acc, err := i2cbus.ReadWordFromReg(0x68, 0x3f)

		log.Printf("x_acc=%d, y_acc=%d, z_acc=%d\n", x_acc, y_acc, z_acc)

		// Get gyro data.
		x_gyro, err := i2cbus.ReadWordFromReg(0x68, 0x43)
		chkErr(err)
		y_gyro, err := i2cbus.ReadWordFromReg(0x68, 0x45)
		chkErr(err)
		z_gyro, err := i2cbus.ReadWordFromReg(0x68, 0x47)

		log.Printf("x_gyro=%d, y_gyro=%d, z_gyro=%d\n", x_gyro, y_gyro, z_gyro)

		// Get magnetometer data.
		setSetting(0x25, 0x0c|0x80) // Set the I2C slave addres of AK8963 and set for read.
		setSetting(0x26, 0x03)      // I2C slave 0 register address from where to begin data transfer.
		setSetting(0x27, 0x87)      // Read 6 bytes from the magnetometer.
		i2cbus.ReadByteFromReg(0x68, 0x49)
		x_mag, err := i2cbus.ReadWordFromReg(0x68, 0x50)
		chkErr(err)
		y_mag, err := i2cbus.ReadWordFromReg(0x68, 0x52)
		chkErr(err)
		z_mag, err := i2cbus.ReadWordFromReg(0x68, 0x54)

		log.Printf("x_mag=%d, y_mag=%d, z_mag=%d\n", x_mag, y_mag, z_mag)

		AHRSupdate(convertToRadians(x_gyro), convertToRadians(y_gyro), convertToRadians(z_gyro), convertToRadians(x_acc), convertToRadians(y_acc), convertToRadians(z_acc), convertToRadians(x_mag), convertToRadians(y_mag), convertToRadians(z_mag))

		time.Sleep(100 * time.Millisecond)
	}
}

func convertToRadians(value uint16) float64 {
	//return float64((value/65535)*360.0) * math.Pi / 180.0
	return float64(value)
}
