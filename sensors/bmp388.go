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

func NewBMP388(i2cbus *embd.I2CBus) *BMP388 {

	bmp := bmp388.BMP388{Address: bmp388.Address, Config: bmp388.Config{}, Bus: i2cbus}
	newbmp := BMP388{
		sensor: &bmp}
	go newbmp.run()
	return &newbmp
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

func (d *BMP388) Close() {
	d.running = false
	d.sensor.Config.Mode = bmp388.Sleep
	_ = d.sensor.Configure(d.sensor.Config)
}

// Temperature returns the current temperature in degrees C measured by the BMP280
func (d *BMP388) Temperature() (float64, error) {
	if !d.running {
		return 0, bmp388.ErrNotConnected
	}

	return d.temperature, nil
}

func (d *BMP388) Pressure() (float64, error) {
	if !d.running {
		return 0, bmp388.ErrNotConnected
	}
	return d.pressure, nil
}
