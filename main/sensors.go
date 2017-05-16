package main

import (
	"fmt"
	"log"
	"math"
	"path/filepath"
	"time"

	"../sensors"

	"../goflying/ahrs"
	"../goflying/ahrsweb"
	"github.com/kidoman/embd"
	_ "github.com/kidoman/embd/host/all"
)

const (
	numRetries uint8   = 5
	invalid    float32 = float32(ahrs.Invalid)
)

var (
	i2cbus           embd.I2CBus
	myPressureReader sensors.PressureReader
	myIMUReader      sensors.IMUReader
	cage             chan (bool)
	analysisLogger   *ahrs.AHRSLogger
	ahrsCalibrating  bool
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
			log.Println("AHRS Info: attempting pressure sensor connection.")
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
		failnum  uint8
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
			failnum++
			if failnum > numRetries {
				log.Printf("AHRS Error: Couldn't read pressure from sensor %d times, closing BMP: %s", failnum, err)
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
	log.Println("AHRS Info: attempting to connect to MPU9250")
	imu, err := sensors.NewMPU9250()
	if err == nil {
		myIMUReader = imu
		time.Sleep(200 * time.Millisecond)
		log.Println("AHRS Info: Successfully connected MPU9250")
		return true
	}

	// TODO westphae: try to connect to MPU9150 or other IMUs.

	log.Println("AHRS Error: couldn't initialize MPU9250")
	return false
}

func sensorAttitudeSender() {
	var (
		roll, pitch, heading float64
		t                    time.Time
		s                    ahrs.AHRSProvider
		m                    *ahrs.Measurement
		a, b, c, d, mm       [3]float64    // IMU measurements: accel, gyro, accel bias, gyro bias, magnetometer
		ff                   [3][3]float64 // Sensor orientation matrix
		cc                   float64
		mpuError, magError   error
		failnum              uint8
	)
	log.Println("AHRS Info: initializing new Simple AHRS")
	s = ahrs.InitializeSimple()
	SetAHRSConfig(globalSettings.AHRSSmoothingConstant, globalSettings.AHRSGPSWeight)
	m = ahrs.NewMeasurement()
	cage = make(chan (bool), 1)

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
		select { // Don't block if cage isn't receiving: only need one cage in the queue at a time.
		case cage <- true:
		default:
		}

		failnum = 0
		<-timer.C
		for globalSettings.IMU_Sensor_Enabled && globalStatus.IMUConnected {
			<-timer.C
			select {
			case <-cage:
				log.Println("AHRS Info: Calibrating IMU")
				ahrsCalibrating = true
				//TODO westphae: check for errors when reading IMU
				myIMUReader.Read() // Clear out the averages
				time.Sleep(1 * time.Second)
				_, d[0], d[1], d[2], c[0], c[1], c[2], _, _, _, _, _ = myIMUReader.Read()
				ff = *makeSensorRotationMatrix([3]float64{c[0], c[1], c[2]})
				log.Printf("AHRS Info: IMU Calibrated: accel %6f %6f %6f; gyro %6f %6f %6f\n",
					c[0], c[1], c[2], d[0], d[1], d[2])
				ahrsCalibrating = false
				cc = math.Sqrt(c[0]*c[0] + c[1]*c[1] + c[2]*c[2])
				s.Reset()
			default:
			}

			t = stratuxClock.Time
			m.T = float64(t.UnixNano()/1000) / 1e6

			_, b[0], b[1], b[2], a[0], a[1], a[2], mm[0], mm[1], mm[2], mpuError, magError = myIMUReader.Read()
			a[0] /= cc
			a[1] /= cc
			a[2] /= cc
			b[0] -= d[0]
			b[1] -= d[1]
			b[2] -= d[2]
			m.A1 = -(ff[0][0]*a[0] + ff[0][1]*a[1] + ff[0][2]*a[2])
			m.A2 = -(ff[1][0]*a[0] + ff[1][1]*a[1] + ff[1][2]*a[2])
			m.A3 = -(ff[2][0]*a[0] + ff[2][1]*a[1] + ff[2][2]*a[2])
			m.B1 = ff[0][0]*b[0] + ff[0][1]*b[1] + ff[0][2]*b[2]
			m.B2 = ff[1][0]*b[0] + ff[1][1]*b[1] + ff[1][2]*b[2]
			m.B3 = ff[2][0]*b[0] + ff[2][1]*b[1] + ff[2][2]*b[2]
			m.M1 = ff[0][0]*mm[0] + ff[0][1]*mm[1] + ff[0][2]*mm[2]
			m.M2 = ff[1][0]*mm[0] + ff[1][1]*mm[1] + ff[1][2]*mm[2]
			m.M3 = ff[2][0]*mm[0] + ff[2][1]*mm[1] + ff[2][2]*mm[2]
			m.SValid = mpuError == nil
			m.MValid = magError == nil
			if mpuError != nil {
				log.Printf("AHRS Gyro/Accel Error: %s\n", mpuError)
				failnum++
				if failnum > numRetries {
					log.Printf("AHRS Gyro/Accel Error: failed to read %d times, restarting: %s\n",
						failnum-1, mpuError)
					myIMUReader.Close()
					globalStatus.IMUConnected = false
				}
				continue
			}
			failnum = 0
			if magError != nil {
				log.Printf("AHRS Magnetometer Error, not using for this run: %s\n", magError)
				m.MValid = false
				// Don't necessarily disconnect here, unless AHRSProvider deeply depends on magnetometer
			}

			m.TW = float64(mySituation.GPSLastGroundTrackTime.UnixNano()/1000) / 1e6
			m.WValid = t.Sub(mySituation.GPSLastGroundTrackTime) < 3000*time.Millisecond
			if m.WValid {
				m.W1 = mySituation.GPSGroundSpeed * math.Sin(float64(mySituation.GPSTrueCourse)*ahrs.Deg)
				m.W2 = mySituation.GPSGroundSpeed * math.Cos(float64(mySituation.GPSTrueCourse)*ahrs.Deg)
				if globalSettings.BMP_Sensor_Enabled && globalStatus.BMPConnected {
					m.W3 = float64(mySituation.BaroVerticalSpeed * 60 / 6076.12)
				} else {
					m.W3 = float64(mySituation.GPSVerticalSpeed) * 3600 / 6076.12
				}
			}

			// Run the AHRS calcs
			s.Compute(m)

			// If we have valid AHRS info, then update mySituation
			mySituation.muAttitude.Lock()
			if s.Valid() {
				roll, pitch, heading = s.RollPitchHeading()
				mySituation.AHRSRoll = float32(roll / ahrs.Deg)
				mySituation.AHRSPitch = float32(pitch / ahrs.Deg)
				mySituation.AHRSGyroHeading = float32(heading / ahrs.Deg)

				// TODO westphae: until magnetometer calibration is performed, no mag heading
				mySituation.AHRSMagHeading = invalid
				mySituation.AHRSSlipSkid = float32(s.SlipSkid())
				mySituation.AHRSTurnRate = float32(s.RateOfTurn())
				mySituation.AHRSGLoad = float32(s.GLoad())

				mySituation.AHRSLastAttitudeTime = t
			} else {
				s.Reset()
				mySituation.AHRSRoll = invalid
				mySituation.AHRSPitch = invalid
				mySituation.AHRSGyroHeading = invalid
				mySituation.AHRSMagHeading = invalid
				mySituation.AHRSSlipSkid = invalid
				mySituation.AHRSTurnRate = invalid
				mySituation.AHRSGLoad = invalid
				mySituation.AHRSLastAttitudeTime = time.Time{}
			}
			mySituation.muAttitude.Unlock()

			makeAHRSGDL90Report() // Send whether or not valid - the function will invalidate the values as appropriate
			// makeFFAHRSSimReport() // Simultaneous use of GDL90 and FFSIM not supported in FF 7.5.1 or later. Function definition will be kept for AHRS debugging and future workarounds.

			// Send to AHRS debugging server:
			if ahrswebListener != nil {
				if err = ahrswebListener.Send(s.GetState(), m); err != nil {
					log.Printf("Error writing to ahrsweb: %s\n", err)
					ahrswebListener = nil
				}
			}

			// Log it to csv for analysis
			if globalSettings.AHRSLog && usage.Usage() < 0.95 {
				if analysisLogger == nil {
					analysisFilename := filepath.Join(logDirf, fmt.Sprintf("sensors_%s.csv",
						time.Now().Format("20060102_150405")))
					analysisLogger = ahrs.NewAHRSLogger(analysisFilename, s.GetLogMap())
				}

				if analysisLogger != nil {
					analysisLogger.Log()
				}
			} else {
				analysisLogger = nil
			}
		}
	}
}

func makeSensorRotationMatrix(g [3]float64) (rotmat *[3][3]float64) {
	f := globalSettings.IMUMapping
	if globalSettings.IMUMapping[0] == 0 { // if unset, default to some standard orientation
		globalSettings.IMUMapping[0] = -1 // +2 for RY836AI
		globalSettings.IMUMapping[1] = -3 // +3 for RY836AI
		saveSettings()
	}

	// This is the "forward direction" chosen during the orientation process.
	var x *[3]float64 = new([3]float64)
	if f[0] < 0 {
		x[-f[0]-1] = -1
	} else {
		x[+f[0]-1] = +1
	}

	// Normalize the gravity vector to be 1 G.
	z, _ := ahrs.MakeUnitVector(g)

	rotmat, _ = ahrs.MakeHardSoftRotationMatrix(*z, *x, [3]float64{0, 0, 1}, [3]float64{1, 0, 0})
	return rotmat
}

// This is used in the orientation process where the user specifies the forward and up directions.
func getMinAccelDirection() (i int, err error) {
	_, _, _, _, a1, a2, a3, _, _, _, err, _ := myIMUReader.Read()
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

// CageAHRS sends a signal to the AHRSProvider that it should be reset.
func CageAHRS() {
	cage <- true
}

// SetAHRSConfig TODO westphae remove after debugging
func SetAHRSConfig(smoothConst, weight float64) {
	ahrs.SetConfig(smoothConst, weight)
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

		// GPS valid
		if stratuxClock.Time.Sub(mySituation.GPSLastGroundTrackTime) < 3000*time.Millisecond {
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
