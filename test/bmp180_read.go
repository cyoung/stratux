package main

import (
	"fmt"
	"github.com/kidoman/embd"
	_ "github.com/kidoman/embd/host/all"
	"github.com/kidoman/embd/sensor/bmp180"
)

var i2cbus embd.I2CBus
var myBMP180 *bmp180.BMP180

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

func initBMP180() error {
	myBMP180 = bmp180.New(i2cbus) //TODO: error checking.
	return nil
}

func initI2C() error {
	i2cbus = embd.NewI2CBus(1) //TODO: error checking.
	return nil
}

// Unused at the moment. 5 second update, since read functions in bmp180 are slow.
func tempAndPressureReader() {
	// Read temperature and pressure altitude.
	temp, alt, err_bmp180 := readBMP180()
	// Process.
	if err_bmp180 != nil {
		fmt.Printf("readBMP180(): %s\n", err_bmp180.Error())
	} else {
		fmt.Printf("Temp %f Alt %f\n", temp, alt)
	}
}

func initAHRS() error {
	if err := initI2C(); err != nil { // I2C bus.
		return err
	}
	if err := initBMP180(); err != nil { // I2C temperature and pressure altitude.
		i2cbus.Close()
		return err
	}
	go tempAndPressureReader()

	return nil
}

func main() {
	fmt.Printf("Hello world!\n")
	initAHRS()
	for {
		tempAndPressureReader()
	}
}
