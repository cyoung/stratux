// Package mpu6050 allows interfacing with InvenSense mpu6050 barometric pressure sensor. This sensor
// has the ability to provided compensated temperature and pressure readings.
package mpu

import (
	"../linux-mpu9150/mpu"
	"log"
	"math"
	"time"
	"errors"
)

//https://www.olimex.com/Products/Modules/Sensors/MOD-MPU6050/resources/RM-MPU-60xxA_rev_4.pdf
const (
	pollDelay = 98 * time.Millisecond // ~10Hz
)

// MPU6050 represents a InvenSense MPU6050 sensor.
type MPU6050 struct {
	poll time.Duration

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
	heading            float64 // Current heading.
	gps_track          float64 // Last reading directly from the gyro for comparison with current heading.
	gps_track_valid    bool
	heading_correction float64

	quit chan struct{}
}

// New returns a handle to a MPU6050 sensor.
func NewMPU6050() (*MPU6050, error) {
	n := &MPU6050{poll: pollDelay}
	if err := n.startUp(); err != nil {
		return nil, err
	}
	return n, nil
}

func (d *MPU6050) startUp() error {
	mpu_sample_rate := 10 // 10 Hz read rate of hardware IMU
	yaw_mix_factor := 0   // must be zero if no magnetometer
	err := mpu.InitMPU(mpu_sample_rate, yaw_mix_factor)
	if err != 0 {
		return errors.New("MPU6050 Error: couldn't start MPU")
	}

	d.pitch_history = make([]float64, 0)
	d.roll_history = make([]float64, 0)

	d.started = true
	d.run()

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
	roll := float64(-rr) * (float64(180.0) / math.Pi)
	heading := float64(hr) * (float64(180.0) / math.Pi)
	if heading < float64(0.0) {
		heading = float64(360.0) + heading
	}

	if err == nil {
		d.pitch = pitch
		d.roll = roll

		// Heading is raw value off the IMU. Without mag compass fusion, need to apply correction bias.
		// Amount of correction is set by ResetHeading() -- doesn't necessarily have to be based off GPS.
		d.heading = normalizeHeading((heading - d.heading_correction))

	} else {
		//		log.Printf("mpu6050.calculatePitchAndRoll(): mpu.ReadMPU() err: %s\n", err.Error())
	}
}

// Temperature returns the current temperature reading.
func (d *MPU6050) Pitch() (float64, error) {
	return (d.pitch - d.pitch_resting), nil
}

func (d *MPU6050) Roll() (float64, error) {
	return (d.roll - d.roll_resting), nil
}

func (d *MPU6050) Heading() (float64, error) {
	return d.heading, nil
}

func (d *MPU6050) run() {
	time.Sleep(d.poll)
	go func() {
		d.quit = make(chan struct{})
		timer := time.NewTicker(d.poll)
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
func (d *MPU6050) ResetHeading(newHeading float64, gain float64) {
	if gain < 0.001 { // sanitize our inputs!
		gain = 0.001
	} else if gain > 1 {
		gain = 1
	}

	old_hdg := d.heading // only used for debug log report
	//newHeading = float64(30*time.Now().Minute()) // demo input for testing
	newHeading = normalizeHeading(newHeading) // sanitize the inputs

	// By applying gain factor, this becomes a 1st order function that slowly converges on solution.
	// Time constant is poll rate over gain. With gain of 0.1, convergence to +/-2 deg on a 180 correction difference is about 4 sec; 0.01 converges in 45 sec.

	hdg_corr_bias := float64(d.heading - newHeading) // desired adjustment to heading_correction
	if hdg_corr_bias > 180 {
		hdg_corr_bias = hdg_corr_bias - 360
	} else if hdg_corr_bias < -180 {
		hdg_corr_bias = hdg_corr_bias + 360
	}
	hdg_corr_bias = hdg_corr_bias * gain
	d.heading_correction = normalizeHeading(d.heading_correction + hdg_corr_bias)
	log.Printf("Adjusted heading. Old: %f Desired: %f  Adjustment: %f  New: %f\n", old_hdg, newHeading, hdg_corr_bias, d.heading-hdg_corr_bias)
}

// Close.
func (d *MPU6050) Close() {
	if d.quit != nil {
		d.quit <- struct{}{}
	}
}

func (d *MPU6050) MagHeading() (float64, error) { return 0, nil }
func (d *MPU6050) SlipSkid() (float64, error) { return 0, nil }
func (d *MPU6050) RateOfTurn() (float64, error) { return 0, nil }
func (d *MPU6050) GLoad() (float64, error) { return 0, nil }

func (d *MPU6050) ReadRaw() (int64, float64, float64, float64, float64, float64, float64, float64, float64, float64, error, error) {
	return 0, // Ts, time of last sensor reading
		0.0, 0.0, 0.0, // Gyro x, y, z
		0.0, 0.0, 0.0, // Accel x, y, z
		0.0, 0.0, 0.0, // Mag x, y, z
		errors.New("Error: ReadRaw() not implemented yet for MPU6050"),
		errors.New("Error: MPU6050 magnetometer isn't working on RY835AI chip")
}
func (d *MPU6050) Calibrate() error {
    return nil //TODO westphae: for now, maybe we'll get lucky; but eventually we should calibrate
}
