package bmp388

/*
taken from: https://github.com/tinygo-org/drivers/blob/release/bmp388/bmp388.go
and converted to use embed
*/
import (
	"errors"
	"github.com/kidoman/embd"
)

var (
	errConfigWrite  = errors.New("bmp388: failed to configure sensor, check connection")
	errConfig       = errors.New("bmp388: there is a problem with the configuration, try reducing ODR")
	errCaliRead     = errors.New("bmp388: failed to read calibration coefficient register")
	errSoftReset    = errors.New("bmp388: failed to perform a soft reset")
	ErrNotConnected = errors.New("bmp388: not connected")
)

type Oversampling byte
type Mode byte
type OutputDataRate byte
type FilterCoefficient byte
type Config struct {
	Pressure    Oversampling
	Temperature Oversampling
	Mode        Mode
	ODR         OutputDataRate
	IIR         FilterCoefficient
}

// BMP388 wraps the I2C connection and configuration values for the BMP388
type BMP388 struct {
	Bus     *embd.I2CBus
	Address uint8
	cali    calibrationCoefficients
	Config  Config
}

type calibrationCoefficients struct {
	// Temperature compensation
	t1 uint16
	t2 uint16
	t3 int8

	// Pressure compensation
	p1  int16
	p2  int16
	p3  int8
	p4  int8
	p5  uint16
	p6  uint16
	p7  int8
	p8  int8
	p9  int16
	p10 int8
	p11 int8
}

func (d *BMP388) Configure(config Config) (err error) {
	d.Config = config

	if d.Config == (Config{}) {
		d.Config.Mode = Normal
	}

	// Turning on the pressure and temperature sensors and setting the measurement mode
	err = d.writeRegister(RegPwrCtrl, PwrPress|PwrTemp|byte(d.Config.Mode))

	// Configure the oversampling, output data rate, and iir filter coefficient settings
	err = d.writeRegister(RegOSR, byte(d.Config.Pressure|d.Config.Temperature<<3))
	err = d.writeRegister(RegODR, byte(d.Config.ODR))
	err = d.writeRegister(RegIIR, byte(d.Config.IIR<<1))

	if err != nil {
		return errConfigWrite
	}

	// Check if there is a problem with the given configuration
	if d.configurationError() {
		return errConfig
	}

	// Reading the builtin calibration coefficients and parsing them per the datasheet. The compensation formula given
	// in the datasheet is implemented in floating point
	buffer, err := d.readRegister(RegCali, 21)
	if err != nil {
		return errCaliRead
	}

	d.cali.t1 = uint16(buffer[1])<<8 | uint16(buffer[0])
	d.cali.t2 = uint16(buffer[3])<<8 | uint16(buffer[2])
	d.cali.t3 = int8(buffer[4])

	d.cali.p1 = int16(buffer[6])<<8 | int16(buffer[5])
	d.cali.p2 = int16(buffer[8])<<8 | int16(buffer[7])
	d.cali.p3 = int8(buffer[9])
	d.cali.p4 = int8(buffer[10])
	d.cali.p5 = uint16(buffer[12])<<8 | uint16(buffer[11])
	d.cali.p6 = uint16(buffer[14])<<8 | uint16(buffer[13])
	d.cali.p7 = int8(buffer[15])
	d.cali.p8 = int8(buffer[16])
	d.cali.p9 = int16(buffer[18])<<8 | int16(buffer[17])
	d.cali.p10 = int8(buffer[19])
	d.cali.p11 = int8(buffer[20])

	return nil
}
func (d *BMP388) tlinCompensate() (int64, error) {
	rawTemp, err := d.readSensorData(RegTemp)
	if err != nil {
		return 0, err
	}

	// pulled from C driver: https://github.com/BoschSensortec/BMP3-Sensor-API/blob/master/bmp3.c
	partialData1 := rawTemp - (256 * int64(d.cali.t1))
	partialData2 := int64(d.cali.t2) * partialData1
	partialData3 := (partialData1 * partialData1)
	partialData4 := partialData3 * int64(d.cali.t3)
	partialData5 := (partialData2 * 262144) + partialData4
	return partialData5 / 4294967296, nil

}
func (d *BMP388) ReadTemperature() (float64, error) {

	tlin, err := d.tlinCompensate()
	if err != nil {
		return 0, err
	}

	temp := (tlin * 25) / 16384
	return float64(temp), nil
}
func (d *BMP388) ReadPressure() (float64, error) {

	tlin, err := d.tlinCompensate()
	if err != nil {
		return 0, err
	}
	rawPress, err := d.readSensorData(RegPress)
	if err != nil {
		return 0, err
	}

	// code pulled from bmp388 C driver: https://github.com/BoschSensortec/BMP3-Sensor-API/blob/master/bmp3.c
	partialData1 := tlin * tlin
	partialData2 := partialData1 / 64
	partialData3 := (partialData2 * tlin) / 256
	partialData4 := (int64(d.cali.p8) * partialData3) / 32
	partialData5 := (int64(d.cali.p7) * partialData1) * 16
	partialData6 := (int64(d.cali.p6) * tlin) * 4194304
	offset := (int64(d.cali.p5) * 140737488355328) + partialData4 + partialData5 + partialData6
	partialData2 = (int64(d.cali.p4) * partialData3) / 32
	partialData4 = (int64(d.cali.p3) * partialData1) * 4
	partialData5 = (int64(d.cali.p2) - 16384) * tlin * 2097152
	sensitivity := ((int64(d.cali.p1) - 16384) * 70368744177664) + partialData2 + partialData4 + partialData5
	partialData1 = (sensitivity / 16777216) * rawPress
	partialData2 = int64(d.cali.p10) * tlin
	partialData3 = partialData2 + (65536 * int64(d.cali.p9))
	partialData4 = (partialData3 * rawPress) / 8192

	// dividing by 10 followed by multiplying by 10
	// To avoid overflow caused by (pressure * partial_data4)
	partialData5 = (rawPress * (partialData4 / 10)) / 512
	partialData5 = partialData5 * 10
	partialData6 = (int64)(uint64(rawPress) * uint64(rawPress))
	partialData2 = (int64(d.cali.p11) * partialData6) / 65536
	partialData3 = (partialData2 * rawPress) / 128
	partialData4 = (offset / 4) + partialData1 + partialData5 + partialData3
	compPress := ((uint64(partialData4) * 25) / uint64(1099511627776))
	return float64(compPress), nil
}
func (d *BMP388) Connected() bool {
	data, err := d.readRegister(RegChipId, 1)
	return err == nil && data[0] == ChipId // returns true if i2c comm was good and response equals 0x50
}
func (d *BMP388) SetMode(mode Mode) error {
	d.Config.Mode = mode
	return d.writeRegister(RegPwrCtrl, PwrPress|PwrTemp|byte(d.Config.Mode))
}
func (d *BMP388) readSensorData(register byte) (data int64, err error) {

	if !d.Connected() {
		return 0, ErrNotConnected
	}

	// put the sensor back into forced mode to get a reading, the sensor goes back to sleep after taking one read in
	// forced mode
	if d.Config.Mode != Normal {
		err = d.SetMode(Forced)
		if err != nil {
			return
		}
	}

	bytes, err := d.readRegister(register, 3)
	if err != nil {
		return
	}
	data = int64(bytes[2])<<16 | int64(bytes[1])<<8 | int64(bytes[0])
	return
}
func (d *BMP388) configurationError() bool {
	data, err := d.readRegister(RegErr, 1)
	return err == nil && (data[0]&0x04) != 0
}

func (d *BMP388) readRegister(register byte, len int) (data []byte, err error) {
	data = make([]byte, len)
	err = (*d.Bus).ReadFromReg(d.Address, register, data)
	return
}

func (d *BMP388) writeRegister(register byte, data byte) error {
	return (*d.Bus).WriteToReg(d.Address, register, []byte{data})
}
