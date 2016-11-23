package mpu

type BMP interface {
	Temperature() (float64, error)
	Pressure() (float64, error)
	Close()
}
