// Package sensors provides a stratux interface to sensors used for AHRS calculations.
package sensors

import (
	"github.com/b3nn0/goflying/icm20948"
	"github.com/kidoman/embd"
)

const (
	gyroRange  = 250 // gyroRange is the default range to use for the Gyro.
	accelRange = 4   // accelRange is the default range to use for the Accel.
	updateFreq = 50  // updateFreq is the rate at which to update the sensor values.
)

// ICM20948 represents an InvenSense ICM-20948 attached to the I2C bus and satisfies
// the IMUReader interface.
type ICM20948 struct {
	mpu *icm20948.ICM20948
}

// NewICM20948 returns an instance of the ICM-20948 IMUReader, connected to an
// ICM-20948 attached on the I2C bus with either valid address.
func NewICM20948(i2cbus *embd.I2CBus) (*ICM20948, error) {
	var (
		m   ICM20948
		mpu *icm20948.ICM20948
		err error
	)

	mpu, err = icm20948.NewICM20948(i2cbus, gyroRange, accelRange, updateFreq, true, false)
	if err != nil {
		return nil, err
	}

	// Set Gyro (Accel) LPFs to 25 Hz to filter out prop/glareshield vibrations above 1200 (1260) RPM
	mpu.SetGyroLPF(25)
	mpu.SetAccelLPF(25)

	m.mpu = mpu
	return &m, nil
}

// Read returns the average (since last reading) time, Gyro X-Y-Z, Accel X-Y-Z, Mag X-Y-Z,
// error reading Gyro/Accel, and error reading Mag.
func (m *ICM20948) Read() (T int64, G1, G2, G3, A1, A2, A3, M1, M2, M3 float64, GAError, MAGError error) {
	var (
		data *icm20948.MPUData
		i    int8
	)
	data = new(icm20948.MPUData)

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

// ReadOne returns the most recent time, Gyro X-Y-Z, Accel X-Y-Z, Mag X-Y-Z,
// error reading Gyro/Accel, and error reading Mag.
func (m *ICM20948) ReadOne() (T int64, G1, G2, G3, A1, A2, A3, M1, M2, M3 float64, GAError, MAGError error) {
	var (
		data *icm20948.MPUData
	)
	data = new(icm20948.MPUData)

	data = <-m.mpu.C
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

// Close stops reading the MPU.
func (m *ICM20948) Close() {
	m.mpu.CloseMPU()
}
