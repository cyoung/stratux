// Package mpu6050 allows interfacing with InvenSense mpu6050 barometric pressure sensor. This sensor
// has the ability to provided compensated temperature and pressure readings.
package mpu6050

import (
	"../linux-mpu9150/mpu"
	"log"
	"math"
	"time"
)

//https://www.olimex.com/Products/Modules/Sensors/MOD-MPU6050/resources/RM-MPU-60xxA_rev_4.pdf
const (
	pollDelay = 98 * time.Millisecond // ~10Hz
)

// MPU6050 represents a InvenSense MPU6050 sensor.
type MPU6050 struct {
	Poll time.Duration

	started bool

	pitch float64
	roll  float64

	// Calibration variables.
	calibrated    bool
	pitch_history []float64
	roll_history  []float64
	pitch_resting float64
	roll_resting  float64

	// For tracking heading (mixing GPS track and the gyro output).
	heading              float64 // Current heading.
	gps_track            float64 // Last reading directly from the gyro for comparison with current heading.
	gps_track_valid      bool
	heading_when_gps_set float64

	quit chan struct{}
}

// New returns a handle to a MPU6050 sensor.
func New() *MPU6050 {
	n := &MPU6050{Poll: pollDelay}
	n.StartUp()
	return n
}

func (d *MPU6050) StartUp() error {
	mpu.InitMPU()

	d.pitch_history = make([]float64, 0)
	d.roll_history = make([]float64, 0)

	d.started = true
	d.Run()

	return nil
}

/*

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
	d.calibrated = true
}

*/

func normalizeHeading(h float64) float64 {
	for h < float64(0.0) {
		h = h + float64(360.0)
	}
	for h >= float64(360.0) {
		h = h - float64(360.0)
	}
	return h
}

func (d *MPU6050) getMPUData() {
	pr, rr, hr, err := mpu.ReadMPU()

	// Convert from radians to degrees.
	pitch := float64(pr) * (float64(180.0) / math.Pi)
	roll := float64(rr) * (float64(180.0) / math.Pi)
	heading := float64(hr) * (float64(180.0) / math.Pi)
	if heading < float64(0.0) {
		heading = float64(360.0) + heading
	}

	if err == nil {
		d.pitch = pitch
		d.roll = roll

		// Calculate the change in direction from current and previous IMU reading.
		if d.gps_track_valid {
			d.heading = normalizeHeading((heading - d.heading_when_gps_set) + d.gps_track)
		} else {
			d.heading = heading
		}
	} else {
		//		log.Printf("mpu6050.calculatePitchAndRoll(): mpu.ReadMPU() err: %s\n", err.Error())
	}
}

// Temperature returns the current temperature reading.
func (d *MPU6050) PitchAndRoll() (float64, float64) {
	return (d.pitch - d.pitch_resting), (d.roll - d.roll_resting)
}

func (d *MPU6050) Heading() float64 {
	return d.heading
}

func (d *MPU6050) Run() {
	time.Sleep(d.Poll)
	go func() {
		d.quit = make(chan struct{})
		timer := time.NewTicker(d.Poll)
		//		calibrateTimer := time.NewTicker(1 * time.Minute)
		for {
			select {
			case <-timer.C:
				d.getMPUData()
				//			case <-calibrateTimer.C:
				//				d.calibrate()
				//				calibrateTimer.Stop()
			case <-d.quit:
				mpu.CloseMPU()
				return
			}
		}
	}()
	return
}

// Set heading from a known value (usually GPS true heading).
func (d *MPU6050) ResetHeading(heading float64) {
	log.Printf("reset true heading: %f\n", heading)
	d.gps_track = heading
	d.gps_track_valid = true
	d.heading_when_gps_set = d.heading
}

// Close.
func (d *MPU6050) Close() {
	if d.quit != nil {
		d.quit <- struct{}{}
	}
}
