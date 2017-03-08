// Package sensors provides a stratux interface to sensors used for AHRS calculations.
package sensors

// IMUReader provides an interface to various Inertial Measurement Unit sensors,
// such as the InvenSense MPU9150 or MPU9250.  It is a light abstraction on top
// of the current goflying MPU9250 driver so that it can accommodate other types
// of drivers.
type IMUReader interface {
	// Calibrate kicks off a calibration for specified duration (s) and retries.
	Calibrate(duration, retries int) error
	// Read returns the average (since last reading) time, Gyro X-Y-Z, Accel X-Y-Z, Mag X-Y-Z, error reading Gyro/Accel, and error reading Mag.
	Read() (T int64, G1, G2, G3, A1, A2, A3, M1, M2, M3 float64, GAError, MagError error)
	// Close stops reading the MPU.
	Close()
}
