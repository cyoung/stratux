package mpu

type BMP interface {
	Temperature() (float64, error)
	Altitude() (float64, error)
	Close()
}
