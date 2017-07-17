// Package sensors provides a stratux interface to sensors used for AHRS calculations.
package sensors

import (
	"errors"
	"time"

	"../goflying/bmp280"
	"github.com/kidoman/embd"
)

const (
	bmp280PowerMode   = bmp280.NormalMode
	bmp280Standby     = bmp280.StandbyTime63ms
	bmp280FilterCoeff = bmp280.FilterCoeff16
	bmp280TempRes     = bmp280.Oversamp16x
	bmp280PressRes    = bmp280.Oversamp16x
)

// BMP280 represents a BMP280 sensor and implements the PressureSensor interface.
type BMP280 struct {
	sensor  *bmp280.BMP280
	data    *bmp280.BMPData
	running bool
}

var errBMP = errors.New("BMP280 Error: BMP280 is not running")

// NewBMP280 looks for a BMP280 connected on the I2C bus having one of the valid addresses and begins reading it.
func NewBMP280(i2cbus *embd.I2CBus, freq time.Duration) (*BMP280, error) {
	var (
		bmp    *bmp280.BMP280
		errbmp error
	)

	bmp, errbmp = bmp280.NewBMP280(i2cbus, bmp280.Address1,
		bmp280PowerMode, bmp280Standby, bmp280FilterCoeff, bmp280TempRes, bmp280PressRes)
	if errbmp != nil { // Maybe the BMP280 isn't at Address1, try Address2
		bmp, errbmp = bmp280.NewBMP280(i2cbus, bmp280.Address2,
			bmp280PowerMode, bmp280Standby, bmp280FilterCoeff, bmp280TempRes, bmp280PressRes)
	}
	if errbmp != nil {
		return nil, errbmp
	}

	newbmp := BMP280{sensor: bmp, data: new(bmp280.BMPData)}
	go newbmp.run()

	return &newbmp, nil
}

func (bmp *BMP280) run() {
	bmp.running = true
	clock := time.NewTicker(100 * time.Millisecond)
	for bmp.running {
		<-clock.C
		bmp.data = <-bmp.sensor.C
	}
}

// Temperature returns the current temperature in degrees C measured by the BMP280
func (bmp *BMP280) Temperature() (float64, error) {
	if !bmp.running {
		return 0, errBMP
	}
	return bmp.data.Temperature, nil
}

// Pressure returns the current pressure in mbar measured by the BMP280
func (bmp *BMP280) Pressure() (float64, error) {
	if !bmp.running {
		return 0, errBMP
	}
	return bmp.data.Pressure, nil
}

// Close stops the measurements of the BMP280
func (bmp *BMP280) Close() {
	bmp.running = false
	bmp.sensor.Close()
}
