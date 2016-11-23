package mpu

import (
	"github.com/westphae/goflying/bmp280"
	"github.com/kidoman/embd"
	"log"
	"time"
	"errors"
)

const (
	BMP280PowerMode = bmp280.NormalMode
	BMP280Standby   = bmp280.StandbyTime63ms
	BMP280FilterCoeff = bmp280.FilterCoeff16
	BMP280TempRes     = bmp280.Oversamp16x
	BMP280PressRes     = bmp280.Oversamp16x
)

type BMP280 struct {
	bmp     *bmp280.BMP280
	bmpdata *bmp280.BMPData
	running bool
}

var bmperr = errors.New("BMP280 Error: BMP280 is not running")

func NewBMP280(i2cbus *embd.I2CBus, freq time.Duration) (*BMP280, error) {
	var (
		bmp *bmp280.BMP280
		errbmp error
	)

	bmp, errbmp = bmp280.NewBMP280(i2cbus, bmp280.Address1,
		BMP280PowerMode, BMP280Standby, BMP280FilterCoeff, BMP280TempRes, BMP280PressRes)
	if errbmp != nil { // Maybe the BMP280 isn't at Address1, try Address2
		bmp, errbmp = bmp280.NewBMP280(i2cbus, bmp280.Address2,
			BMP280PowerMode, BMP280Standby, BMP280FilterCoeff, BMP280TempRes, BMP280PressRes)
	}
	if errbmp != nil {
		log.Println("AHRS Error: couldn't initialize BMP280")
		return nil, errbmp
	}

	newbmp := BMP280{bmp: bmp, bmpdata: new(bmp280.BMPData)}
	go newbmp.run()

	return &newbmp, nil
}

func (bmp *BMP280) run() {
	bmp.running = true
	for bmp.running {
		bmp.bmpdata = <-bmp.bmp.C
	}
}

func (bmp *BMP280) Temperature() (float64, error) {
	if !bmp.running {
		return 0, bmperr
	}
	return bmp.bmpdata.Temperature, nil
}

func (bmp *BMP280) Pressure() (float64, error) {
	if !bmp.running {
		return 0, bmperr
	}
	return bmp.bmpdata.Pressure, nil
}

func (bmp *BMP280) Close() {
	bmp.running = false
	bmp.bmp.Close()
}
