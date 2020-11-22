// Package sensors provides a stratux interface to sensors used for AHRS calculations.
package sensors

import (
	"../goflying/bmx160"
	"github.com/kidoman/embd"
)

const (
	bmx160gyroRange  = 125 // gyroRange is the default range to use for the Gyro.
	bmx160accelRange = 4   // accelRange is the default range to use for the Accel.
	bmx160updateFreq = 200  // updateFreq is the rate at which to update the sensor values.
)

// BMX160 is a Bosch BMX160 attached to the I2C bus and satisfies
// the IMUReader interface.
type BMX160 struct {
	mpu *bmx160.BMX160
}

// NewBMX160 returns an instance of the BMX160 IMUReader, connected to an
// BMX160 attached on the I2C bus with either valid address.
func NewBMX160(i2cbus *embd.I2CBus) (*BMX160, error) {
	var (
		m   BMX160
		mpu *bmx160.BMX160
		err error
	)

	mpu, err = bmx160.NewBMX160(i2cbus, bmx160gyroRange, bmx160accelRange, bmx160updateFreq, false, false)
	if err != nil {
		return nil, err
	}

	// Set Gyro (Accel) LPFs to 20 (21) Hz to filter out prop/glareshield vibrations above 1200 (1260) RPM
        // This is done in the constructor of BMX160

	m.mpu = mpu
	return &m, nil
}

// Read returns the average (since last reading) time, Gyro X-Y-Z, Accel X-Y-Z, Mag X-Y-Z,
// error reading Gyro/Accel, and error reading Mag.
func (m *BMX160) Read() (T int64, G1, G2, G3, A1, A2, A3, M1, M2, M3 float64, GAError, MAGError error) {
	var (
		data *bmx160.MPUData
		i    int8
	)
	data = new(bmx160.MPUData)

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
func (m *BMX160) ReadOne() (T int64, G1, G2, G3, A1, A2, A3, M1, M2, M3 float64, GAError, MAGError error) {
	var (
		data *bmx160.MPUData
	)
	data = new(bmx160.MPUData)

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
func (m *BMX160) Close() {
	m.mpu.CloseMPU()
}
