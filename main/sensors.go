package main

import (
	"fmt"
	"log"
	"math"
	"path/filepath"
	"time"

	"../goflying/ahrs"
	"../goflying/ahrsweb"
	"../sensors"
	"github.com/kidoman/embd"
	_ "github.com/kidoman/embd/host/all"
)

const (
	numRetries uint8 = 5
)

var (
	i2cbus                     embd.I2CBus
	myPressureReader           sensors.PressureReader
	myIMUReader                sensors.IMUReader
	cal                        chan (bool)
	analysisLogger             *ahrs.AHRSLogger
	ahrsCalibrating, needsCage bool // Does the sensor orientation matrix need to be recalculated?
	logMap                     map[string]interface{}
)

func initI2CSensors() {
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
			log.Println("AHRS Info: attempting IMU connection.")
			globalStatus.IMUConnected = initIMU() // I2C accel/gyro/mag.
		}
	}
}

func initPressureSensor() (ok bool) {
	bmp, err := sensors.NewBMP280(&i2cbus, 100*time.Millisecond)
	if err == nil {
		myPressureReader = bmp
		log.Println("AHRS Info: Successfully initialized BMP280")
		return true
	}

	//TODO westphae: make bmp180.go to fit bmp interface

	log.Println("AHRS Info: couldn't initialize BMP280 or BMP180")
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
			log.Printf("AHRS Error: Couldn't read temperature from sensor: %s", err)
		}
		press, err = myPressureReader.Pressure()
		if err != nil {
			log.Printf("AHRS Error: Couldn't read pressure from sensor: %s", err)
			failNum++
			if failNum > numRetries {
				log.Printf("AHRS Error: Couldn't read pressure from sensor %d times, closing BMP: %s", failNum, err)
				myPressureReader.Close()
				globalStatus.BMPConnected = false // Try reconnecting a little later
				break
			}
		}

		// Update the Situation data.
		mySituation.muBaro.Lock()
		mySituation.BaroLastMeasurementTime = stratuxClock.Time
		mySituation.BaroTemperature = float32(temp)
		altitude = CalcAltitude(press)
		mySituation.BaroPressureAltitude = float32(altitude)
		if altLast < -2000 {
			altLast = altitude // Initialize
		}
		// Assuming timer is reasonably accurate, use a regular ewma
		mySituation.BaroVerticalSpeed = u*mySituation.BaroVerticalSpeed + (1-u)*float32(altitude-altLast)/(float32(dt)/60)
		mySituation.muBaro.Unlock()
		altLast = altitude
	}
	mySituation.BaroPressureAltitude = 99999
	mySituation.BaroVerticalSpeed = 99999
}

func initIMU() (ok bool) {
	imu, err := sensors.NewMPU9250()
	if err == nil {
		myIMUReader = imu
		log.Println("AHRS Info: Successfully connected MPU9250")
		return true
	}

	// TODO westphae: try to connect to MPU9150 or other IMUs.

	log.Println("AHRS Error: couldn't initialize an IMU")
	return false
}

func sensorAttitudeSender() {
	var (
		t                    time.Time
		roll, pitch, heading float64
		mpuError, magError   error
		failNum              uint8
	)

	log.Println("AHRS Info: initializing new Simple AHRS")
	s := ahrs.NewSimpleAHRS()
	m := ahrs.NewMeasurement()
	cal = make(chan (bool), 1)

	// Set up loggers for analysis
	ahrswebListener, err := ahrsweb.NewKalmanListener()
	if err != nil {
		log.Printf("AHRS Info: couldn't start ahrswebListener: %s\n", err.Error())
	} else {
		log.Println("AHRS Info: ahrswebListener started on port 8000")
		defer ahrswebListener.Close()
	}

	// Need a sampling freq faster than 10Hz
	timer := time.NewTicker(50 * time.Millisecond) // ~20Hz update.
	for {
		// Do an initial calibration
		select { // Don't block if cal isn't receiving: only need one calibration in the queue at a time.
		case cal <- true:
		default:
		}

		// Set sensor rotation matrix / quaternion
		if f := &globalSettings.SensorQuaternion; f[0]*f[0] + f[1]*f[1] + f[2]*f[2] + f[3]*f[3] > 0 {
			s.SetSensorQuaternion(f)
			needsCage = false
		} else {
			needsCage = true
		}

		failNum = 0
		<-timer.C
		for globalSettings.IMU_Sensor_Enabled && globalStatus.IMUConnected {
			<-timer.C

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

			// Process caging if necessary.
			if needsCage {
				globalSettings.SensorQuaternion = *makeOrientationQuaternion([3]float64{m.A1, m.A2, m.A3})
				saveSettings()
				s.SetSensorQuaternion(&globalSettings.SensorQuaternion)
				s.Reset()
				needsCage = false
				log.Println("AHRS Info: Caged")
			}

			// Run the AHRS calculations.
			s.Compute(m)

			// Calibrate the sensors if requested.
			// Must come after Compute so that the calibrations aren't zeroed out.
			select {
			case <-cal:
				log.Println("AHRS Info: Calibrating IMU")
				ahrsCalibrating = true
				myIMUReader.Read() // Clear out the averages
				time.Sleep(1 * time.Second)
				_, d1, d2, d3, c1, c2, c3, _, _, _, mpuError, _ := myIMUReader.Read()
				if mpuError != nil {
					log.Printf("AHRS Info: Error reading IMU while calibrating: %s\n", mpuError)
				} else {
					s.SetCalibrations(&[3]float64{c1, c2, c3}, &[3]float64{d1, d2, d3})
					log.Printf("AHRS Info: IMU Calibrated: accel %6f %6f %6f; gyro %6f %6f %6f\n",
						c1, c2, c3, d1, d2, d3)
				}
				ahrsCalibrating = false
			default:
			}

			// If we have valid AHRS info, then update mySituation.
			mySituation.muAttitude.Lock()
			if s.Valid() {
				roll, pitch, heading = s.RollPitchHeading()
				mySituation.AHRSRoll = roll / ahrs.Deg
				mySituation.AHRSPitch = pitch / ahrs.Deg
				mySituation.AHRSGyroHeading = heading / ahrs.Deg

				// TODO westphae: until magnetometer calibration is performed, no mag heading
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

			// Send to AHRS debugging server.
			if ahrswebListener != nil {
				if err = ahrswebListener.Send(s.GetState(), m); err != nil {
					log.Printf("Error writing to ahrsweb: %s\n", err)
					ahrswebListener = nil
				}
			}

			// Log it to csv for later analysis.
			if globalSettings.AHRSLog && usage.Usage() < 0.95 {
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
	needsCage = true
	cal <- true
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
		if globalSettings.BMP_Sensor_Enabled && globalStatus.BMPConnected {
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
