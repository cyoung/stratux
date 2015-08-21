// Package mpu6050 allows interfacing with InvenSense mpu6050 barometric pressure sensor. This sensor
// has the ability to provided compensated temperature and pressure readings.
package mpu6050

import (
	"math"
	"time"
	//	"github.com/golang/glog"
	"github.com/kidoman/embd"
	"log"
)

//https://www.olimex.com/Products/Modules/Sensors/MOD-MPU6050/resources/RM-MPU-60xxA_rev_4.pdf
const (
	address = 0x68

	GYRO_XOUT_H = 0x43
	GYRO_YOUT_H = 0x45
	GYRO_ZOUT_H = 0x47

	ACCEL_XOUT_H = 0x3B
	ACCEL_YOUT_H = 0x3D
	ACCEL_ZOUT_H = 0x3F

	PWR_MGMT_1 = 0x6B

	GYRO_CONFIG = 0x1B
	ACCEL_CONFIG = 0x1C

	ACCEL_SCALE = 16384.0 // Assume AFS_SEL = 0.
	GYRO_SCALE  = 131.0   // Assume FS_SEL = 0.

	pollDelay = 500 * time.Microsecond // 2000Hz
)

type XYZ struct {
	x float32
	y float32
	z float32
}

// MPU6050 represents a InvenSense MPU6050 sensor.
type MPU6050 struct {
	Bus  embd.I2CBus
	Poll time.Duration

	started bool

	//TODO
	gyro_reading  XYZ // "Integrated".
	accel_reading XYZ // Directly from sensor.

	pitch_history []float64
	roll_history  []float64

	pitch_resting float64
	roll_resting  float64

	pitch   float64
	roll    float64
	heading float64
	//	gyro		chan XYZ
	//	accel		chan XYZ

	quit chan struct{}
}

// New returns a handle to a MPU6050 sensor.
func New(bus embd.I2CBus) *MPU6050 {
	n := &MPU6050{Bus: bus, Poll: pollDelay}
	n.StartUp()
	return n
}

//TODO
func (d *MPU6050) StartUp() error {
	d.Bus.WriteByteToReg(address, PWR_MGMT_1, 0) // Wake device up.

	d.Bus.WriteByteToReg(address, GYRO_CONFIG, 0) // FS_SEL = 0
	d.Bus.WriteByteToReg(address, ACCEL_CONFIG, 0) // AFS_SEL = 0

	d.pitch_history = make([]float64, 0)
	d.roll_history = make([]float64, 0)

	d.started = true
	d.Run()

	return nil
}

func (d *MPU6050) calibrate() {
	//TODO: Error checking to make sure that the histories are extensive enough to be significant.
	//TODO: Error checking to do continuous calibrations.
	pitch_adjust := float64(0)
	for _, v := range d.pitch_history {
		pitch_adjust = pitch_adjust + v
	}
	pitch_adjust = pitch_adjust / float64(len(d.pitch_history))
	d.pitch_resting = pitch_adjust

	roll_adjust := float64(0)
	for _, v := range d.roll_history {
		roll_adjust = roll_adjust + v
	}
	roll_adjust = roll_adjust / float64(len(d.roll_history))
	d.roll_resting = roll_adjust
	log.Printf("calibrate: pitch %f, roll %f\n", pitch_adjust, roll_adjust)
}

func (d *MPU6050) readGyro() (XYZ, error) {
	var ret XYZ

	x, err := d.Bus.ReadWordFromReg(address, GYRO_XOUT_H)
	if err != nil {
		return ret, err
	}
	y, err := d.Bus.ReadWordFromReg(address, GYRO_YOUT_H)
	if err != nil {
		return ret, err
	}
	z, err := d.Bus.ReadWordFromReg(address, GYRO_ZOUT_H)
	if err != nil {
		return ret, err
	}

	ret.x = float32(int16(x)) / GYRO_SCALE // º/sec
	ret.y = float32(int16(y)) / GYRO_SCALE // º/sec
	ret.z = float32(int16(z)) / GYRO_SCALE // º/sec

	return ret, nil
}

func (d *MPU6050) readAccel() (XYZ, error) {
	var ret XYZ

	x, err := d.Bus.ReadWordFromReg(address, ACCEL_XOUT_H)
	if err != nil {
		return ret, err
	}
	y, err := d.Bus.ReadWordFromReg(address, ACCEL_YOUT_H)
	if err != nil {
		return ret, err
	}
	z, err := d.Bus.ReadWordFromReg(address, ACCEL_ZOUT_H)
	if err != nil {
		return ret, err
	}

	ret.x = float32(int16(x)) / ACCEL_SCALE
	ret.y = float32(int16(y)) / ACCEL_SCALE
	ret.z = float32(int16(z)) / ACCEL_SCALE

	return ret, nil
}

func (d *MPU6050) calculatePitchAndRoll() {
	accel := d.accel_reading
	//	log.Printf("accel: %f, %f, %f\n", accel.x, accel.y, accel.z)

	// Accel.

	p1 := math.Atan2(float64(accel.y), dist(accel.x, accel.z))
	p1_deg := p1 * (180.0 / math.Pi)

	r1 := math.Atan2(float64(accel.x), dist(accel.y, accel.z))
	r1_deg := -r1 * (180.0 / math.Pi)

	// Gyro.

	p2 := float64(d.gyro_reading.x)
	r2 := float64(d.gyro_reading.y) // Backwards?

	// "Noise filter".
	ft := float64(0.98)
	sample_period := float64(1 / 2000.0)
	d.pitch = float64(ft*(sample_period*p2+d.pitch) + (1-ft)*p1_deg)
	d.roll = float64((ft*(sample_period*r2+d.roll) + (1-ft)*r1_deg))

	d.pitch_history = append(d.pitch_history, d.pitch)
	d.roll_history = append(d.roll_history, d.roll)

	//FIXME: Experimental (heading).



	f := math.Atan2(float64(d.gyro_reading.z), dist(float32(d.pitch), float32(d.roll)))
	h1_deg := -float64(3.42857142857)*f * (180.0 / math.Pi)
	//	d.heading = float64((float64(3.42857142857)*sample_period * float64(-d.gyro_reading.z)) + d.heading)
	d.heading = float64((sample_period * h1_deg) + d.heading)
	if d.heading > 360.0 {
		d.heading = d.heading - float64(360.0)
	} else if d.heading < 0.0 {
		d.heading = d.heading + float64(360.0)
	}
}

func (d *MPU6050) measureGyro() error {
	XYZ_gyro, err := d.readGyro()
	if err != nil {
		return err
	}
	d.gyro_reading = XYZ_gyro
	return nil
}

func (d *MPU6050) measureAccel() error {
	XYZ_accel, err := d.readAccel()
	if err != nil {
		return err
	}
	d.accel_reading = XYZ_accel
	return nil
}

func (d *MPU6050) measure() error {
	if err := d.measureGyro(); err != nil {
		return err
	}
	if err := d.measureAccel(); err != nil {
		return err
	}
	return nil
}

func dist(a, b float32) float64 {
	a64 := float64(a)
	b64 := float64(b)
	return math.Sqrt((a64 * a64) + (b64 * b64))
}

// Temperature returns the current temperature reading.
func (d *MPU6050) PitchAndRoll() (float64, float64) {
	return (d.pitch - d.pitch_resting), (d.roll - d.roll_resting)
}

func (d *MPU6050) Heading() float64 {
	return d.heading
}

func (d *MPU6050) Run() {
	go func() {
		d.quit = make(chan struct{})
		timer := time.NewTicker(d.Poll)
		calibrateTimer := time.NewTicker(1 * time.Minute)
		for {
			select {
			case <-timer.C:
				d.measureGyro()
				d.measureAccel()
				d.calculatePitchAndRoll()
			case <-calibrateTimer.C:
				d.calibrate()
				calibrateTimer.Stop()
			case <-d.quit:
				return
			}
		}
	}()
	return
}

// Set heading from a known value (usually GPS true heading).
func (d *MPU6050) ResetHeading(heading float64) {
	log.Printf("reset true heading: %f\n", heading)
	d.heading = heading
}

// Close.
func (d *MPU6050) Close() {
	if d.quit != nil {
		d.quit <- struct{}{}
	}
}
