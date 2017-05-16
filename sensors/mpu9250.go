// Package sensors provides a stratux interface to sensors used for AHRS calculations.
package sensors

import (
	"log"
	"time"

	"../goflying/mpu9250"
)

const (
	gyroRange  = 250 // gyroRange is the default range to use for the Gyro.
	accelRange = 4   // accelRange is the default range to use for the Accel.
	updateFreq = 50  // updateFreq is the rate at which to update the sensor values.
)

// MPU9250 represents an InvenSense MPU9250 attached to the I2C bus and satisfies
// the IMUReader interface.
type MPU9250 struct {
	mpu *mpu9250.MPU9250
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
	time.Sleep(100 * time.Millisecond)

	m.mpu = mpu

	log.Println("AHRS Info: monitoring IMU")
	return &m, nil
}

// Read returns the average (since last reading) time, Gyro X-Y-Z, Accel X-Y-Z, Mag X-Y-Z,
// error reading Gyro/Accel, and error reading Mag.
func (m *MPU9250) Read() (T int64, G1, G2, G3, A1, A2, A3, M1, M2, M3 float64, GAError, MAGError error) {
	var (
		data *mpu9250.MPUData
		i    int8
	)
	data = new(mpu9250.MPUData)

	for data.N == 0 && i < 5 {
		data = <-m.mpu.CAvg
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
		i++
	}
	return
}

// Close stops reading the MPU.
func (m *MPU9250) Close() {
	m.mpu.CloseMPU()
}
