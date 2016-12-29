// Package sensors provides a stratux interface to sensors used for AHRS calculations.
package sensors

// IMUReader provides an interface to various Inertial Measurement Unit sensors,
// such as the InvenSense MPU9150 or MPU9250.
type IMUReader interface {
	// ReadRaw returns the time, Gyro X-Y-Z, Accel X-Y-Z, Mag X-Y-Z, error reading Gyro/Accel, and error reading Mag.
	ReadRaw() (T int64, G1, G2, G3, A1, A2, A3, M1, M2, M3 float64, GAError, MagError error)
	Calibrate(duration, retries int) error // Calibrate kicks off a calibration for specified duration (s) and retries.
	Close() // Close stops reading the MPU.
	MagHeading() (hdg float64, MagError error) // MagHeading returns the magnetic heading in degrees.
	SlipSkid() (slipSkid float64, err error) // SlipSkid returns the slip/skid angle in degrees.
	RateOfTurn() (turnRate float64, err error) // RateOfTurn returns the turn rate in degrees per second.
	GLoad() (gLoad float64, err error) // GLoad returns the current G load, in G's.
}
