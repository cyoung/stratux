package mpu

type MPU interface {
	Close()
	ResetHeading(float64, float64)
	Pitch() (float64, error)
	Roll() (float64, error)
	Heading() (float64, error)
	MagHeading() (float64, error)
	SlipSkid() (float64, error)
	RateOfTurn() (float64, error)
	GLoad() (float64, error)
	ReadRaw() (int64, float64, float64, float64, float64, float64, float64, float64, float64, float64, error, error)
	Calibrate(int) error
}
