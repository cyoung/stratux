// Package sensors provides a stratux interface to sensors used for AHRS calculations.
package sensors

import (
	"errors"
	"math"
	"time"

	"github.com/westphae/goflying/mpu9250"
	"log"
)

const (
	decay      = 0.8 // decay is the decay constant used for exponential smoothing of sensor values.
	gyroRange  = 250 // gyroRange is the default range to use for the Gyro.
	accelRange = 4 // accelRange is the default range to use for the Accel.
	updateFreq = 1000 // updateFreq is the rate at which to update the sensor values.
)

// MPU9250 represents an InvenSense MPU9250 attached to the I2C bus and satisfies
// the IMUReader interface.
type MPU9250 struct {
	mpu                  *mpu9250.MPU9250
	pitch, roll, heading float64
	headingMag           float64
	slipSkid             float64
	turnRate             float64
	gLoad                float64
	T                    int64
	valid                bool
	nextCalibrateT       int64
	quit                 chan struct{}
}

// NewMPU9250 returns an instance of the MPU9250 IMUReader, connected to an
// MPU9250 attached on the I2C bus with either valid address.
func NewMPU9250() (*MPU9250, error) {
	var (
		m   MPU9250
		mpu *mpu9250.MPU9250
		err error
	)

	log.Println("AHRS Info: Making new MPU9250")
	mpu, err = mpu9250.NewMPU9250(gyroRange, accelRange, updateFreq, true, false)
	if err != nil {
		return nil, err
	}

	// Set Gyro (Accel) LPFs to 20 (21) Hz to filter out prop/glareshield vibrations above 1200 (1260) RPM
	log.Println("AHRS Info: Setting MPU9250 LPF")
	mpu.SetGyroLPF(21)
	mpu.SetAccelLPF(21)

	m.mpu = mpu
	m.valid = true

	time.Sleep(100 * time.Millisecond)
	log.Println("AHRS Info: monitoring IMU")
	m.run()

	return &m, nil
}

func (m *MPU9250) run() {
	time.Sleep(100 * time.Millisecond)
	go func() {
		m.quit = make(chan struct{})
		clock := time.NewTicker(100 * time.Millisecond)

		for {
			select {
			case <-clock.C:
				data := <-m.mpu.CAvg

				if data.GAError == nil && data.N > 0 {
					m.T = data.T.UnixNano()
					smooth(&m.turnRate, -data.G3) // TODO westphae: gross approx, depends on attitude!
					smooth(&m.gLoad, data.A3)
					smooth(&m.slipSkid, math.Atan2(-data.A1, data.A3)*180/math.Pi)
				}

				if data.MagError == nil && data.NM > 0 {
					hM := math.Atan2(-data.M2, data.M1) * 180 / math.Pi
					if hM-m.headingMag < -180 {
						hM += 360
					}
					smooth(&m.headingMag, hM)
					for m.headingMag < 0 {
						m.headingMag += 360
					}
					for m.headingMag >= 360 {
						m.headingMag -= 360
					}
				}
			case <-m.quit:
				m.mpu.CloseMPU()
				return
			}
		}
	}()
}

func smooth(val *float64, new float64) {
	*val = decay**val + (1-decay)*new
}

// MagHeading returns the magnetic heading in degrees.
func (m *MPU9250) MagHeading() (float64, error) {
	if m.valid {
		return m.headingMag, nil
	}
	return 0, errors.New("MPU error: data not available")
}

// SlipSkid returns the slip/skid angle in degrees.
func (m *MPU9250) SlipSkid() (float64, error) {
	if m.valid {
		return m.slipSkid, nil
	}
	return 0, errors.New("MPU error: data not available")
}

// RateOfTurn returns the turn rate in degrees per second.
func (m *MPU9250) RateOfTurn() (float64, error) {
	if m.valid {
		return m.turnRate, nil
	}
	return 0, errors.New("MPU error: data not available")
}

// GLoad returns the current G load, in G's.
func (m *MPU9250) GLoad() (float64, error) {
	if m.valid {
		return m.gLoad, nil
	}
	return 0, errors.New("MPU error: data not available")
}

// ReadRaw returns the time, Gyro X-Y-Z, Accel X-Y-Z, Mag X-Y-Z, error reading Gyro/Accel, and error reading Mag.
func (m *MPU9250) ReadRaw() (T int64, G1, G2, G3, A1, A2, A3, M1, M2, M3 float64, GAError, MAGError error) {
	data := <-m.mpu.C
	T = data.T.UnixNano()
	G1 = data.G1
	G2 = data.G2
	G3 = data.G3
	A1 = data.A1
	A2 = data.A2
	A3 = data.A3
	M1 = data.M1
	M2 = data.M2
	M3 = data.M3
	GAError = data.GAError
	MAGError = data.MagError
	return
}

// Calibrate kicks off a calibration for specified duration (s) and retries.
func (m *MPU9250) Calibrate(dur, retries int) (err error) {
	if dur > 0 {
		for i := 0; i < retries; i++ {
			m.mpu.CCal <- dur
			log.Printf("AHRS Info: Waiting for calibration result try %d of %d\n", i, retries)
			err = <-m.mpu.CCalResult
			if err == nil {
				log.Println("AHRS Info: MPU9250 calibrated")
				break
			}
			time.Sleep(time.Duration(50) * time.Millisecond)
			log.Println("AHRS Info: MPU9250 wasn't inertial, retrying calibration")
		}
	}
	return
}

// Close stops reading the MPU.
func (m *MPU9250) Close() {
	if m.quit != nil {
		m.quit <- struct{}{}
	}
}
