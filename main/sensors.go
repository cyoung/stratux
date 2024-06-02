package main

import (
	"fmt"
	"log"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/b3nn0/stratux/sensors/bmp388"

	"github.com/b3nn0/goflying/ahrs"
	"github.com/b3nn0/goflying/ahrsweb"
	"github.com/b3nn0/stratux/common"
	"github.com/b3nn0/stratux/sensors"
	"github.com/kidoman/embd"
	_ "github.com/kidoman/embd/host/all"
	"github.com/ricochet2200/go-disk-usage/du"
)

const (
	numRetries uint8 = 50
	calCLimit        = 0.15
	calDLimit        = 10.0

	// WHO_AM_I values to differentiate between the different IMUs.
	MPUREG_WHO_AM_I             = 0x75
	MPUREG_WHO_AM_I_VAL         = 0x71 // Expected value.
	MPUREG_WHO_AM_I_VAL_9255    = 0x73 // Expected value for MPU9255, seems to be compatible to 9250
	MPUREG_WHO_AM_I_VAL_6500    = 0x70 // Expected value for MPU6500, seems to be same as 9250 but without magnetometer
	MPUREG_WHO_AM_I_VAL_60X0    = 0x68 // Expected value for MPU6000 and MPU6050 (and MPU9150)
	MPUREG_WHO_AM_I_VAL_UNKNOWN = 0x75 // Unknown MPU found on recent batch of gy91 boards see discussion 182
	ICMREG_WHO_AM_I             = 0x00
	ICMREG_WHO_AM_I_VAL         = 0xEA             // Expected value.
	PRESSURE_WHO_AM_I           = bmp388.RegChipId // Expected address for bosch pressure sensors bmpXXX.
)

var (
	i2cbus           embd.I2CBus
	myPressureReader sensors.PressureReader
	myIMUReader      sensors.IMUReader
	cal              chan (string)
	analysisLogger   *ahrs.AHRSLogger
	ahrsCalibrating  bool
	logMap           map[string]interface{}
)

func initI2CSensors() {
	defer func() {
		if err := recover(); err != nil {
			// still want to update status in case external GPS delivers pressure data (OGN Tracker, SoftRF with BMP)
			// This usually happens on X86, where there is no embd supported I2C
			fmt.Println("Panic during i2c initialization!")
			go updateAHRSStatus()
		}
	}()
	embd.SetHost(embd.HostRPi, 3)
	i2cbus = embd.NewI2CBus(1)
	go pollSensors()
	go sensorAttitudeSender()
	go updateAHRSStatus()
}

func pollSensors() {
	timer := time.NewTicker(4 * time.Second)
	for {
		<-timer.C

		// If it's not currently connected, try connecting to pressure sensor
		if globalSettings.BMP_Sensor_Enabled && !globalStatus.BMPConnected {
			globalStatus.BMPConnected = initPressureSensor() // I2C temperature and pressure altitude.
			go tempAndPressureSender()
		}

		// If it's not currently connected, try connecting to IMU
		if globalSettings.IMU_Sensor_Enabled && !globalStatus.IMUConnected {
			globalStatus.IMUConnected = initIMU() // I2C accel/gyro/mag.
		}
	}
}

func initPressureSensor() (ok bool) {

	v, err := i2cbus.ReadByteFromReg(0x76, PRESSURE_WHO_AM_I)

	if err != nil {
		log.Printf("Error identifying BMP: %s\n", err.Error())
		return false
	}
	if v == bmp388.ChipId || v == bmp388.ChipId390 {
		log.Printf("BMP-388 detected")
		bmp, err := sensors.NewBMP388(&i2cbus)
		if err == nil {
			myPressureReader = bmp
			return true
		}
	} else {
		log.Printf("using BMP-280")
		bmp, err := sensors.NewBMP280(&i2cbus, 100*time.Millisecond)
		if err == nil {
			myPressureReader = bmp
			return true
		}
	}

	return false
}

func tempAndPressureSender() {
	var (
		temp     float64
		press    float64
		altLast  = -9999.9
		altitude float64
		err      error
		dt       = 0.1
		failNum  uint8
	)

	// Initialize variables for rate of climb calc
	u := 5 / (5 + float32(dt)) // Use 5 sec decay time for rate of climb, slightly faster than typical VSI

	timer := time.NewTicker(time.Duration(1000*dt) * time.Millisecond)
	for globalSettings.BMP_Sensor_Enabled && globalStatus.BMPConnected {
		<-timer.C

		// Read temperature and pressure altitude.
		temp, err = myPressureReader.Temperature()
		if err != nil {
			addSingleSystemErrorf("pressure-sensor-temp-read", "AHRS Error: Couldn't read temperature from sensor: %s", err)
		}
		press, err = myPressureReader.Pressure()
		if press == 0 || err != nil {
			if err != nil {
				addSingleSystemErrorf("pressure-sensor-pressure-read", "AHRS Error: Couldn't read pressure from sensor: %s", err)
			}
			failNum++
			if failNum > numRetries {
				//				log.Printf("AHRS Error: Couldn't read pressure from sensor %d times, closing BMP: %s", failNum, err)
				myPressureReader.Close()
				globalStatus.BMPConnected = false // Try reconnecting a little later
				errStr := "Pressure is 0"
				if err != nil {
					errStr = err.Error()
				}
				addSingleSystemErrorf("pressure-sensor-pressure-read", "AHRS Error: Couldn't read pressure from sensor: %s", errStr)
				break
			}
			continue
		}

		altitude = common.CalcAltitude(press, globalSettings.AltitudeOffset)
		if altitude > 70000 || (isGPSValid() && mySituation.GPSAltitudeMSL != 0 && math.Abs(float64(mySituation.GPSAltitudeMSL)-altitude) > 5000) {
			addSingleSystemErrorf("BaroBroken", "Barometric altitude %d' out of expected range. Ignoring. Pressure sensor potentially broken.", int32(altitude))
			continue
		}

		// Update the Situation data.
		mySituation.muBaro.Lock()
		mySituation.BaroLastMeasurementTime = stratuxClock.Time
		mySituation.BaroTemperature = float32(temp)
		mySituation.BaroPressureAltitude = float32(altitude)
		if altLast < -2000 {
			altLast = altitude // Initialize
		}
		// Assuming timer is reasonably accurate, use a regular ewma
		mySituation.BaroVerticalSpeed = u*mySituation.BaroVerticalSpeed + (1-u)*float32(altitude-altLast)/(float32(dt)/60)
		mySituation.BaroSourceType = BARO_TYPE_BMP280
		mySituation.muBaro.Unlock()
		altLast = altitude
	}
	//mySituation.BaroPressureAltitude = 99999
	//mySituation.BaroVerticalSpeed = 99999
}

func initIMU() (ok bool) {
	// Check if the chip is the ICM-20948 or MPU-9250.
	v, err := i2cbus.ReadByteFromReg(0x68, ICMREG_WHO_AM_I)
	if err != nil {
		log.Printf("Error identifying IMU: %s\n", err.Error())
		return false
	}
	v2, err := i2cbus.ReadByteFromReg(0x68, MPUREG_WHO_AM_I)
	if err != nil {
		log.Printf("Error identifying IMU: %s\n", err.Error())
		return false
	}

	if v == ICMREG_WHO_AM_I_VAL {
		log.Println("ICM-20948 detected.")
		imu, err := sensors.NewICM20948(&i2cbus)
		if err == nil {
			myIMUReader = imu
			return true
		}
	} else if v2 == MPUREG_WHO_AM_I_VAL || v2 == MPUREG_WHO_AM_I_VAL_9255 || v2 == MPUREG_WHO_AM_I_VAL_6500 ||
		v2 == MPUREG_WHO_AM_I_VAL_60X0 || v2 == MPUREG_WHO_AM_I_VAL_UNKNOWN {

		log.Printf("MPU detected (%02x).\n", v2)
		imu, err := sensors.NewMPU9250(&i2cbus)
		if err == nil {
			myIMUReader = imu
			return true
		}
	} else {
		log.Printf("Could not identify MPU. v=%02x, v2=%02x.\n", v, v2)
		return false
	}

	return false
}

//FIXME: Shoud be moved to managementinterface.go and standardized on management interface port.

func sensorAttitudeSender() {
	var (
		t                    time.Time
		roll, pitch, heading float64
		mpuError, magError   error
		failNum              uint8
	)

	s := ahrs.NewSimpleAHRS()
	m := ahrs.NewMeasurement()
	cal = make(chan (string), 1)

	// Set up loggers for analysis
	ahrswebListener, err := ahrsweb.NewKalmanListener()
	if err != nil {
		// addSingleSystemErrorf("ahrs-web-start", "AHRS Info: couldn't start ahrswebListener: %s\n", err.Error())
	} else {
		defer ahrswebListener.Close()
	}

	// Need a sampling freq faster than 10Hz
	timer := time.NewTicker(50 * time.Millisecond) // ~20Hz update.
	for {
		// Set sensor gyro calibrations
		if c, d := &globalSettings.C, &globalSettings.D; d[0]*d[0]+d[1]*d[1]+d[2]*d[2] > 0 {
			s.SetCalibrations(c, d)
			// log.Printf("AHRS Info: IMU Calibrations read from settings: accel %6f %6f %6f; gyro %6f %6f %6f\n",
			//	c[0], c[1], c[2], d[0], d[1], d[2])
		} else {
			// Do an initial calibration
			select { // Don't block if cal isn't receiving: only need one calibration in the queue at a time.
			case cal <- "cal":
			default:
			}
		}

		// Set sensor quaternion
		if f := &globalSettings.SensorQuaternion; f[0]*f[0]+f[1]*f[1]+f[2]*f[2]+f[3]*f[3] > 0 {
			s.SetSensorQuaternion(f)
		} else {
			select { // Don't block if cal isn't receiving: only need one calibration in the queue at a time.
			case cal <- "level":
			default:
			}
		}

		failNum = 0
		<-timer.C
		time.Sleep(950 * time.Millisecond)
		for globalSettings.IMU_Sensor_Enabled && globalStatus.IMUConnected {
			<-timer.C

			// Process calibration and level requests
			select {
			case action := <-cal:
				log.Printf("AHRS Info: cal received action %s\n", action)
				ahrsCalibrating = true
				myIMUReader.Read() // Clear out the averages
				var (
					nTries uint8
					cc, dd float64
				)
				for (math.Abs(cc-1) > calCLimit || dd > calDLimit) && nTries < numRetries {
					time.Sleep(1 * time.Second)
					_, d1, d2, d3, c1, c2, c3, _, _, _, mpuError, _ := myIMUReader.Read()
					cc = math.Sqrt(c1*c1 + c2*c2 + c3*c3)
					dd = math.Sqrt(d1*d1 + d2*d2 + d3*d3)
					nTries++
					log.Printf("AHRS Info: IMU calibration attempt #%d\n", nTries)
					if mpuError != nil {
						log.Printf("AHRS Info: Error reading IMU while calibrating: %s\n", mpuError)
					} else {
						if strings.Contains(action, "cal") { // Calibrate gyros
							globalSettings.D = [3]float64{d1, d2, d3}
							s.SetCalibrations(nil, &globalSettings.D)
							log.Printf("AHRS Info: IMU gyro calibration: %3f %3f %3f\n", d1, d2, d3)
						}
						if strings.Contains(action, "level") { // Calibrate accel / level
							globalSettings.C = [3]float64{c1, c2, c3}
							s.SetCalibrations(&globalSettings.C, nil)
							globalSettings.SensorQuaternion = *makeOrientationQuaternion(globalSettings.C)
							s.SetSensorQuaternion(&globalSettings.SensorQuaternion)
							s.Reset()
							log.Printf("AHRS Info: IMU accel calibration: %3f %3f %3f\n", c1, c2, c3)
							log.Printf("AHRS Info: Caged to quaternion %v\n", globalSettings.SensorQuaternion)
						}
						saveSettings()
					}
				}
				ahrsCalibrating = false
				<-timer.C // Make sure we get data for the actual algorithm
			default:
			}

			// Make the IMU sensor measurements.
			t = stratuxClock.Time
			m.T = float64(t.UnixNano()/1000) / 1e6
			_, m.B1, m.B2, m.B3, m.A1, m.A2, m.A3, m.M1, m.M2, m.M3, mpuError, magError = myIMUReader.Read()
			m.SValid = mpuError == nil
			m.MValid = magError == nil
			if mpuError != nil {
				log.Printf("AHRS Gyro/Accel Error: %s\n", mpuError)
				failNum++
				if failNum > numRetries {
					log.Printf("AHRS Gyro/Accel Error: failed to read %d times, restarting: %s\n",
						failNum-1, mpuError)
					myIMUReader.Close()
					globalStatus.IMUConnected = false
				}
				continue
			}
			failNum = 0
			if magError != nil {
				if globalSettings.DEBUG {
					log.Printf("AHRS Magnetometer Error, not using for this run: %s\n", magError)
				}
				m.MValid = false
			}

			// Make the GPS measurements.
			m.TW = float64(mySituation.GPSLastGroundTrackTime.UnixNano()/1000) / 1e6
			m.WValid = isGPSGroundTrackValid()
			if m.WValid {
				m.W1 = mySituation.GPSGroundSpeed * math.Sin(float64(mySituation.GPSTrueCourse)*ahrs.Deg)
				m.W2 = mySituation.GPSGroundSpeed * math.Cos(float64(mySituation.GPSTrueCourse)*ahrs.Deg)
				if globalSettings.BMP_Sensor_Enabled && globalStatus.BMPConnected {
					m.W3 = float64(mySituation.BaroVerticalSpeed * 60 / 6076.12)
				} else {
					m.W3 = float64(mySituation.GPSVerticalSpeed) * 3600 / 6076.12
				}
			}

			// Run the AHRS calculations.
			s.Compute(m)

			// If we have valid AHRS info, then update mySituation.
			mySituation.muAttitude.Lock()
			if s.Valid() {
				roll, pitch, heading = s.RollPitchHeading()
				mySituation.AHRSRoll = roll / ahrs.Deg
				mySituation.AHRSPitch = pitch / ahrs.Deg
				mySituation.AHRSGyroHeading = heading
				if !isAHRSInvalidValue(heading) {
					mySituation.AHRSGyroHeading /= ahrs.Deg
				}

				//TODO westphae: until magnetometer calibration is performed, no mag heading
				mySituation.AHRSMagHeading = ahrs.Invalid
				mySituation.AHRSSlipSkid = s.SlipSkid()
				mySituation.AHRSTurnRate = s.RateOfTurn()
				mySituation.AHRSGLoad = s.GLoad()
				if mySituation.AHRSGLoad < mySituation.AHRSGLoadMin || mySituation.AHRSGLoadMin == 0 {
					mySituation.AHRSGLoadMin = mySituation.AHRSGLoad
				}
				if mySituation.AHRSGLoad > mySituation.AHRSGLoadMax {
					mySituation.AHRSGLoadMax = mySituation.AHRSGLoad
				}

				mySituation.AHRSLastAttitudeTime = t
			} else {
				mySituation.AHRSRoll = ahrs.Invalid
				mySituation.AHRSPitch = ahrs.Invalid
				mySituation.AHRSGyroHeading = ahrs.Invalid
				mySituation.AHRSMagHeading = ahrs.Invalid
				mySituation.AHRSSlipSkid = ahrs.Invalid
				mySituation.AHRSTurnRate = ahrs.Invalid
				mySituation.AHRSGLoad = ahrs.Invalid
				mySituation.AHRSGLoadMin = ahrs.Invalid
				mySituation.AHRSGLoadMax = 0
				mySituation.AHRSLastAttitudeTime = time.Time{}
				s.Reset()
			}
			mySituation.muAttitude.Unlock()

			makeAHRSGDL90Report() // Send whether or not valid - the function will invalidate the values as appropriate
			makeAHRSSimReport()
			makeAHRSLevilReport()

			// Send to AHRS debugging server.
			if ahrswebListener != nil {
				if err = ahrswebListener.Send(s.GetState(), m); err != nil {
					log.Printf("AHRS Error: couldn't write to ahrsweb: %s\n", err)
					ahrswebListener = nil
				}
			}

			// Log it to csv for later analysis.
			if globalSettings.AHRSLog && du.NewDiskUsage("/").Usage() < 0.95 {
				if analysisLogger == nil {
					analysisFilename := fmt.Sprintf("sensors_%s.csv", time.Now().Format("20060102_150405"))
					logMap = s.GetLogMap()
					updateExtraLogging()
					analysisLogger = ahrs.NewAHRSLogger(filepath.Join(logDirf, analysisFilename), logMap)
				}

				if analysisLogger != nil {
					updateExtraLogging()
					analysisLogger.Log()
				}
			} else {
				analysisLogger = nil
			}
		}
	}
}

func updateExtraLogging() {
	logMap["GPSNACp"] = float64(mySituation.GPSNACp)
	logMap["GPSTrueCourse"] = mySituation.GPSTrueCourse
	logMap["GPSVerticalAccuracy"] = mySituation.GPSVerticalAccuracy
	logMap["GPSHorizontalAccuracy"] = mySituation.GPSHorizontalAccuracy
	logMap["GPSAltitudeMSL"] = mySituation.GPSAltitudeMSL
	logMap["GPSFixQuality"] = float64(mySituation.GPSFixQuality)
	logMap["BaroPressureAltitude"] = float64(mySituation.BaroPressureAltitude)
	logMap["BaroVerticalSpeed"] = float64(mySituation.BaroVerticalSpeed)
}

func makeOrientationQuaternion(g [3]float64) (f *[4]float64) {
	if globalSettings.IMUMapping[0] == 0 { // if unset, default to some standard orientation
		globalSettings.IMUMapping[0] = -1 // +2 for RY836AI
	}

	// This is the "forward direction" chosen during the orientation process.
	var x *[3]float64 = new([3]float64)
	if globalSettings.IMUMapping[0] < 0 {
		x[-globalSettings.IMUMapping[0]-1] = -1
	} else {
		x[+globalSettings.IMUMapping[0]-1] = +1
	}

	// Normalize the gravity vector to be 1 G.
	z, _ := ahrs.MakeUnitVector(g)

	rotmat, _ := ahrs.MakeHardSoftRotationMatrix(*z, *x, [3]float64{0, 0, 1}, [3]float64{1, 0, 0})
	f = new([4]float64)
	f[0], f[1], f[2], f[3] = ahrs.RotationMatrixToQuaternion(*rotmat)
	return
}

// This is used in the orientation process where the user specifies the forward and up directions.
func getMinAccelDirection() (i int, err error) {
	_, _, _, _, a1, a2, a3, _, _, _, err, _ := myIMUReader.ReadOne()
	if err != nil {
		return
	}
	log.Printf("AHRS Info: sensor orientation accels %1.3f %1.3f %1.3f\n", a1, a2, a3)
	switch {
	case math.Abs(a1) > math.Abs(a2) && math.Abs(a1) > math.Abs(a3):
		if a1 > 0 {
			i = 1
		} else {
			i = -1
		}
	case math.Abs(a2) > math.Abs(a3) && math.Abs(a2) > math.Abs(a1):
		if a2 > 0 {
			i = 2
		} else {
			i = -2
		}
	case math.Abs(a3) > math.Abs(a1) && math.Abs(a3) > math.Abs(a2):
		if a3 > 0 {
			i = 3
		} else {
			i = -3
		}
	default:
		err = fmt.Errorf("couldn't determine biggest accel from %1.3f %1.3f %1.3f", a1, a2, a3)
	}

	return
}

// CageAHRS sends a signal to the AHRSProvider that it should recalibrate and reset its level orientation.
func CageAHRS() {
	cal <- "level"
}

// CageAHRS sends a signal to the AHRSProvider that it should recalibrate and reset its level orientation.
func CalibrateAHRS() {
	cal <- "cal"
}

// ResetAHRSGLoad resets the min and max to the current G load value.
func ResetAHRSGLoad() {
	mySituation.AHRSGLoadMax = mySituation.AHRSGLoad
	mySituation.AHRSGLoadMin = mySituation.AHRSGLoad
}

func updateAHRSStatus() {
	var (
		msg    uint8
		imu    bool
		ticker *time.Ticker
	)

	ticker = time.NewTicker(250 * time.Millisecond)

	for {
		<-ticker.C
		msg = 0

		// GPS ground track valid?
		if isGPSGroundTrackValid() {
			msg++
		}
		// IMU is being used
		imu = globalSettings.IMU_Sensor_Enabled && globalStatus.IMUConnected
		if imu {
			msg += 1 << 1
		}
		// BMP is being used
		if (globalSettings.BMP_Sensor_Enabled && globalStatus.BMPConnected) || isTempPressValid() {
			msg += 1 << 2
		}
		// IMU is doing a calibration
		if ahrsCalibrating {
			msg += 1 << 3
		}
		// Logging to csv
		if imu && analysisLogger != nil {
			msg += 1 << 4
		}
		mySituation.AHRSStatus = msg
	}
}

func isAHRSInvalidValue(val float64) bool {
	return math.Abs(val-ahrs.Invalid) < 0.01
}
