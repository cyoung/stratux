// Package bmp388 provides a driver for Bosch's BMP388 digital temperature & pressure sensor.
// The datasheet can be found here: https://www.bosch-sensortec.com/media/boschsensortec/downloads/datasheets/bst-bmp388-ds001.pdf
package bmp388

const Address byte = 0x77 // default I2C address

const (
	RegChipId  byte = 0x00 // useful for checking the connection
	RegCali    byte = 0x31 // pressure & temperature compensation calibration coefficients
	RegPress   byte = 0x04 // start of pressure data registers
	RegTemp    byte = 0x07 // start of temperature data registers
	RegPwrCtrl byte = 0x1B // measurement mode & pressure/temperature sensor power register
	RegOSR     byte = 0x1C // oversampling settings register
	RegODR     byte = 0x1D //
	RegCmd     byte = 0x7E // miscellaneous command register
	RegStat    byte = 0x03 // sensor status register
	RegErr     byte = 0x02 // error status register
	RegIIR     byte = 0x1F
)

const (
	ChipId    byte = 0x50 // correct response if reading from chip id register
	PwrPress  byte = 0x01 // power on pressure sensor
	PwrTemp   byte = 0x02 // power on temperature sensor
	SoftReset byte = 0xB6 // command to reset all user configuration
	DRDYPress byte = 0x20 // for checking if pressure data is ready
	DRDYTemp  byte = 0x40 // for checking if pressure data is ready
)

// The difference between forced and normal mode is the bmp388 goes to sleep after taking a measurement in forced mode.
// Set it to forced if you intend to take measurements sporadically and want to save power. The driver will handle
// waking the sensor up when the sensor is in forced mode.
const (
	Normal Mode = 0x30
	Forced Mode = 0x16
	Sleep  Mode = 0x00
)

// Increasing sampling rate increases precision but also the wait time for measurements. The datasheet has a table of
// suggested values for oversampling, output data rates, and iir filter coefficients by use case.
const (
	Sampling1X Oversampling = iota
	Sampling2X
	Sampling4X
	Sampling8X
	Sampling16X
	Sampling32X
)

// Output data rates in Hz. If increasing the sampling rates you need to decrease the output data rates, else the bmp388
// will freeze and Configure() will return a configuration error message. In that case keep decreasing the data rate
// until the bmp is happy
const (
	Odr200 OutputDataRate = iota
	Odr100
	Odr50
	Odr25
	Odr12p5
	Odr6p25
	Odr3p1
	Odr1p5
	Odr0p78
	Odr0p39
	Odr0p2
	Odr0p1
	Odr0p05
	Odr0p02
	Odr0p01
	Odr0p006
	Odr0p003
	Odr0p0015
)

// IIR filter coefficients, higher values means steadier measurements but slower reaction times
const (
	Coeff0 FilterCoefficient = iota
	Coeff1
	Coeff3
	Coeff7
	Coeff15
	Coeff31
	Coeff63
	Coeff127
)
