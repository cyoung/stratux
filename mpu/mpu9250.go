// Package MPU9250 provides a stratux interface to the MPU9250 IMU
package mpu

import (
	"errors"
	"github.com/westphae/goflying/mpu9250"
	"log"
	"math"
	"time"
)

const (
	DECAY            = 0.8
	GYRORANGE        = 250
	ACCELRANGE       = 4
	UPDATEFREQ       = 1000
)

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

func NewMPU9250() (*MPU9250, error) {
	var (
		m   MPU9250
		mpu *mpu9250.MPU9250
		err error
	)

	mpu, err = mpu9250.NewMPU9250(GYRORANGE, ACCELRANGE, UPDATEFREQ, true, false)
	if err != nil {
		log.Println("AHRS Error: couldn't initialize MPU9250")
		return nil, err
	}

	// Set Gyro (Accel) LPFs to 20 (21) Hz to filter out prop/glareshield vibrations above 1200 (1260) RPM
	mpu.SetGyroLPF(21)
	mpu.SetAccelLPF(21)

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
				data := <-m.mpu.CAvg

				if data.GAError == nil && data.N > 0 {
					m.T = data.T.UnixNano()
					smooth(&m.turnRate, -data.G3) // TODO westphae: gross approx, depends on attitude!
					smooth(&m.gLoad, data.A3)
					smooth(&m.slipSkid, math.Atan2(-data.A1, data.A3)*180/math.Pi)
				}

				if data.MagError == nil && data.NM > 0 {
					hM := math.Atan2(-data.M2, data.M1)*180/math.Pi
					if hM - m.headingMag < -180 {
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
	*val = DECAY**val + (1-DECAY)*new
}

func (m *MPU9250) ResetHeading(newHeading float64, gain float64) {
	m.heading = newHeading
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

func (m *MPU9250) Calibrate(dur, retries int) (err error) {
	for i:=0; i<retries; i++ {
		m.mpu.CCal<- dur
		err = <-m.mpu.CCalResult
		if err == nil {
			break
		}
		time.Sleep(time.Duration(50) * time.Millisecond)
	}
	return
}

func (m *MPU9250) Close() {
	if m.quit != nil {
		m.quit <- struct{}{}
	}
}
