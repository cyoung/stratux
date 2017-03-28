// Package sensors provides a stratux interface to sensors used for AHRS calculations.
package sensors

// PressureReader provides an interface to a sensor reading pressure and maybe
// temperature or humidity, like the BMP180 or BMP280.
type PressureReader interface {
	Temperature() (temp float64, tempError error) // Temperature returns the temperature in degrees C.
	Pressure() (press float64, pressError error) // Pressure returns the atmospheric pressure in mBar.
	Close() // Close stops reading from the sensor.
}
