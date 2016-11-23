package mpu

type MPU interface {
	Close()
	ResetHeading(float64, float64)
	MagHeading() (float64, error)
	SlipSkid() (float64, error)
	RateOfTurn() (float64, error)
	GLoad() (float64, error)
	ReadRaw() (int64, float64, float64, float64, float64, float64, float64, float64, float64, float64, error, error)
	Calibrate(int, int) error
}
