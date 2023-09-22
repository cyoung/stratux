package sensors

import (
	"github.com/b3nn0/stratux/sensors/bmp388"
	"github.com/kidoman/embd"
	"time"
)

type BMP388 struct {
	sensor      *bmp388.BMP388
	temperature float64
	pressure    float64
	running     bool
}

func NewBMP388(i2cbus *embd.I2CBus) (*BMP388, error) {

	bmp := bmp388.BMP388{Address: bmp388.Address, Config: bmp388.Config{}, Bus: i2cbus} //new sensor
	// retry to connect until sensor connected
	var connected bool
	for n := 0; n < 5; n++ {
		if bmp.Connected() {
			connected = true
		} else {
			time.Sleep(time.Millisecond)
		}
	}
	if !connected {
		return nil, bmp388.ErrNotConnected
	}
	err := bmp.Configure(bmp.Config)
	if err != nil {
		return nil, err
	}
	newBmp := BMP388{sensor: &bmp}

	go newBmp.run()
	return &newBmp, nil
}
func (bmp *BMP388) run() {
	bmp.running = true
	clock := time.NewTicker(100 * time.Millisecond)
	for bmp.running {
		for _ = range clock.C {
			var p, _ = bmp.sensor.ReadPressure()
			bmp.pressure = p
			var t, _ = bmp.sensor.ReadTemperature()
			bmp.temperature = t
		}

	}
}

func (bmp *BMP388) Close() {
	bmp.running = false
	bmp.sensor.Config.Mode = bmp388.Sleep
	_ = bmp.sensor.Configure(bmp.sensor.Config)
}

// Temperature returns the current temperature in degrees C measured by the BMP280
func (bmp *BMP388) Temperature() (float64, error) {
	if !bmp.running {
		return 0, bmp388.ErrNotConnected
	}

	return bmp.temperature, nil
}

func (bmp *BMP388) Pressure() (float64, error) {
	if !bmp.running {
		return 0, bmp388.ErrNotConnected
	}
	return bmp.pressure, nil
}
