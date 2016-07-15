// Package MPU9250 provides a stratux interface to the MPU9250 IMU
package mpu

import (
	"errors"
	"github.com/westphae/goflying/mpu9250"
	"math"
	"time"
	"log"
)

const (
	DECAY = 0.98
	GYRORANGE = 250
	ACCELRANGE = 4
	UPDATEFREQ = 100
	CALIBTIME int64 = 5*60*1000000000
)

type MPU9250 struct {
	mpu                  *mpu9250.MPU9250
	pitch, roll, heading float64
	headingMag           float64
	slipSkid             float64
	turnRate             float64
	gLoad                float64
	T 		     int64
	valid                bool
	nextCalibrateT 	     int64
	quit 		     chan struct{}
}

func NewMPU9250() (*MPU9250, error) {
	var (
		m	MPU9250
		mpu	*mpu9250.MPU9250
		err 	error
	)

	mpu, err = mpu9250.NewMPU9250(GYRORANGE, ACCELRANGE, UPDATEFREQ, false)
	if err != nil {
		log.Println("AHRS Error: couldn't initialize MPU9250")
		return nil, err
	}

	err = mpu.CalibrateGyro(1)
	if err != nil {
		log.Printf("AHRS: Gyro calibration failed: %s\n", err.Error())
	} else {
		log.Println("AHRS: Gyro calibration successful")
		m.nextCalibrateT = time.Now().UnixNano() + CALIBTIME
	}

	m.mpu = mpu
	m.valid = true

	time.Sleep(100 * time.Millisecond)
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
				Ts, Gx, Gy, Gz, Ax, Ay, Az, Mx, My, _, gaError, magError := m.mpu.Read()

				if gaError == nil {
					m.T = Ts
					smooth(&m.turnRate, Gz)
					smooth(&m.gLoad, math.Sqrt(Ax * Ax + Ay * Ay + Az * Az))
					smooth(&m.slipSkid, Ay / Az)

					// Quick and dirty calcs just to test - these are no good for pitch >> 0
					m.pitch += 0.1 * Gx
					m.roll += 0.1 * Gy
					m.heading -= 0.1 * Gz

					if m.pitch > 90 {
						m.pitch = 180-m.pitch
					}
					if m.pitch < -90 {
						m.pitch = -180 - m.pitch
					}
					if (m.roll > 180) || (m.roll < -180) {
						m.roll = -m.roll
					}
					if m.heading > 360 {
						m.heading -= 360
					}
					if m.heading < 0 {
						m.heading += 360
					}
				}

				if magError == nil {
					smooth(&m.headingMag, math.Atan2(My, Mx))
				}

				// Calibrate if past-due
				if time.Now().UnixNano() > m.nextCalibrateT {
					err := m.mpu.CalibrateGyro(1)
					if err != nil {
						log.Printf("AHRS: Error calibrating gyro, %s\n", err)
					} else {
						log.Println("AHRS: Gyro calibration successful")
						m.nextCalibrateT = time.Now().UnixNano() + CALIBTIME
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
	*val = DECAY * *val + (1-DECAY)*new
}

func (m *MPU9250) ResetHeading(newHeading float64, gain float64) {
	m.heading = newHeading
}

func (m *MPU9250) Pitch() (float64, error) {
	if m.valid {
		return m.pitch, nil
	} else {
		return 0, errors.New("MPU error: data not available")
	}
}

func (m *MPU9250) Roll() (float64, error) {
	if m.valid {
		return m.roll, nil
	} else {
		return 0, errors.New("MPU error: data not available")
	}
}

func (m *MPU9250) Heading() (float64, error) {
	if m.valid {
		return m.heading, nil
	} else {
		return 0, errors.New("MPU error: data not available")
	}
}

func (m *MPU9250) MagHeading() (float64, error) {
	if m.valid {
		return m.headingMag, nil
	} else {
		return 0, errors.New("MPU error: data not available")
	}
}

func (m *MPU9250) SlipSkid() (float64, error) {
	if m.valid {
		return m.slipSkid, nil
	} else {
		return 0, errors.New("MPU error: data not available")
	}
}

func (m *MPU9250) RateOfTurn() (float64, error) {
	if m.valid {
		return m.turnRate, nil
	} else {
		return 0, errors.New("MPU error: data not available")
	}
}

func (m *MPU9250) GLoad() (float64, error) {
	if m.valid {
		return m.gLoad, nil
	} else {
		return 0, errors.New("MPU error: data not available")
	}
}

func (m *MPU9250) Close() {
	if m.quit != nil {
		m.quit <- struct{}{}
	}
}
