package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"../sensors"

	"github.com/kidoman/embd"
	_ "github.com/kidoman/embd/host/all"
	// "github.com/kidoman/embd/sensor/bmp180"
	"github.com/westphae/goflying/ahrs"
	"github.com/westphae/goflying/ahrsweb"
	"path/filepath"
)

const numRetries uint8 = 5

var (
	i2cbus           embd.I2CBus
	myPressureReader sensors.PressureReader
	myIMUReader      sensors.IMUReader
	cage             chan(bool)
)

func initI2CSensors() {
	i2cbus = embd.NewI2CBus(1)

	go pollSensors()
	go sensorAttitudeSender()
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

	// TODO westphae: make bmp180.go to fit bmp interface
	//for i := 0; i < 5; i++ {
	//	myBMPX80 = bmp180.New(i2cbus)
	//	_, err := myBMPX80.Temperature() // Test to see if it works, since bmp180.New doesn't return err
	//	if err != nil {
	//		time.Sleep(250 * time.Millisecond)
	//	} else {
	//		globalStatus.BMPConnected = true
	//		log.Println("AHRS Info: Successfully initialized BMP180")
	//		return nil
	//	}
	//}

	log.Println("AHRS Info: couldn't initialize BMP280 or BMP180")
	return false
}

func tempAndPressureSender() {
	var (
		temp     float64
		press    float64
		altLast  float64 = -9999
		altitude float64
		err      error
		dt       float64 = 0.1
		failnum  uint8
	)

	// Initialize variables for rate of climb calc
	u := 5 / (5 + float64(dt)) // Use 5 sec decay time for rate of climb, slightly faster than typical VSI

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
			failnum += 1
			if failnum > numRetries {
				log.Printf("AHRS Error: Couldn't read pressure from sensor %d times, closing BMP: %s", failnum, err)
				myPressureReader.Close()
				globalStatus.BMPConnected = false // Try reconnecting a little later
				break
			}
		}

		// Update the Situation data.
		mySituation.mu_Pressure.Lock()
		mySituation.LastTempPressTime = stratuxClock.Time
		mySituation.Temp = temp
		altitude = CalcAltitude(press)
		mySituation.Pressure_alt = altitude
		if altLast < -2000 {
			altLast = altitude // Initialize
		}
		// Assuming timer is reasonably accurate, use a regular ewma
		mySituation.RateOfClimb = u*mySituation.RateOfClimb + (1-u)*(altitude-altLast)/(float64(dt)/60)
		mySituation.mu_Pressure.Unlock()
		altLast = altitude
	}
}

func initIMU() (ok bool) {
	log.Println("AHRS Info: attempting to connect to MPU9250")
	imu, err := sensors.NewMPU9250()
	if err == nil {
		myIMUReader = imu
		time.Sleep(200 * time.Millisecond)
		log.Println("AHRS Info: Successfully connected MPU9250, running calibration")
		if err := myIMUReader.Calibrate(1, 1); err == nil {
			log.Println("AHRS Info: Successfully calibrated MPU9250")
			return true
		} else {
			log.Println("AHRS Info: couldn't calibrate MPU9250")
			return false
		}
	}

	// TODO westphae: try to connect to MPU9150

	log.Println("AHRS Error: couldn't initialize MPU9250 or MPU9150")
	return false
}

func sensorAttitudeSender() {
	var (
		roll, pitch, heading                              float64
		t                                                 time.Time
		s                                                 ahrs.AHRSProvider
		m                                                 *ahrs.Measurement
		a1, a2, a3, b1, b2, b3, m1, m2, m3                float64 // IMU measurements
		ff       					  *[3][3]float64 // Sensor orientation matrix
		mpuError, magError                                error
		headingMag, slipSkid, turnRate, gLoad             float64
		errHeadingMag, errSlipSkid, errTurnRate, errGLoad error
		failnum						  uint8
		analysisLogger                                    *ahrs.AHRSLogger
	)
	log.Println("AHRS Info: initializing new simple AHRS")
	s = ahrs.InitializeSimple()
	m = ahrs.NewMeasurement()
	cage = make(chan(bool))

	// Set up loggers for analysis
	ahrswebListener, err := ahrsweb.NewKalmanListener()
	if err != nil {
		log.Printf("AHRS Error: couldn't start ahrswebListener: %s\n", err.Error())
	} else {
		defer ahrswebListener.Close()
	}

	// Need a sampling freq faster than 10Hz
	timer := time.NewTicker(50 * time.Millisecond) // ~20Hz update.
	for {
		if globalSettings.IMUMapping[0]==0 { // if unset, default to RY836AI
			globalSettings.IMUMapping[0] = -1 // +2
			globalSettings.IMUMapping[1] = -3 // +3
			saveSettings()
		}
		f := globalSettings.IMUMapping

		// Set up orientation matrix; a bit ugly for now
		ff = new([3][3]float64)
		if f[0] < 0 {
			ff[0][-f[0] - 1] = -1
		} else {
			ff[0][+f[0] - 1] = +1
		}
		if f[1] < 0 {
			ff[2][-f[1] - 1] = -1
		} else {
			ff[2][+f[1] - 1] = +1
		}
		ff[1][0] = ff[2][1] * ff[0][2] - ff[2][2] * ff[0][1]
		ff[1][1] = ff[2][2] * ff[0][0] - ff[2][0] * ff[0][2]
		ff[1][2] = ff[2][0] * ff[0][1] - ff[2][1] * ff[0][0]

		failnum = 0
		<-timer.C
		for globalSettings.IMU_Sensor_Enabled && globalStatus.IMUConnected {
			<-timer.C
			select {
			case <-cage:
				if err := myIMUReader.Calibrate(1, 1); err == nil {
					log.Println("AHRS Info: Successfully recalibrated MPU9250")
				} else {
					log.Println("AHRS Info: couldn't recalibrate MPU9250")
				}
				s.Reset()
			default:
			}

			t = stratuxClock.Time
			m.T = float64(t.UnixNano() / 1000) / 1e6

			_, b1, b2, b3, a1, a2, a3, m1, m2, m3, mpuError, magError = myIMUReader.ReadRaw()
			// This is how the RY83XAI is wired up
			//m.A1, m.A2, m.A3 = -a2, +a1, -a3
			//m.B1, m.B2, m.B3 = +b2, -b1, +b3
			//m.M1, m.M2, m.M3 = +m1, +m2, +m3
			// This is how the OpenFlightBox board is wired up
			//m.A1, m.A2, m.A3 = +a1, -a2, +a3
			//m.B1, m.B2, m.B3 = -b1, +b2, -b3
			//m.M1, m.M2, m.M3 = +m2, +m1, +m3
			m.A1 = -(ff[0][0]*a1 + ff[0][1]*a2 + ff[0][2]*a3)
			m.A2 = -(ff[1][0]*a1 + ff[1][1]*a2 + ff[1][2]*a3)
			m.A3 = -(ff[2][0]*a1 + ff[2][1]*a2 + ff[2][2]*a3)
			m.B1 =   ff[0][0]*b1 + ff[0][1]*b2 + ff[0][2]*b3
			m.B2 =   ff[1][0]*b1 + ff[1][1]*b2 + ff[1][2]*b3
			m.B3 =   ff[2][0]*b1 + ff[2][1]*b2 + ff[2][2]*b3
			m.M1 =   ff[0][0]*m1 + ff[0][1]*m2 + ff[0][2]*m3
			m.M2 =   ff[1][0]*m1 + ff[1][1]*m2 + ff[1][2]*m3
			m.M3 =   ff[2][0]*m1 + ff[2][1]*m2 + ff[2][2]*m3
			m.SValid = mpuError == nil
			m.MValid = magError == nil
			if mpuError != nil {
				log.Printf("AHRS Gyro/Accel Error: %s\n", mpuError)
				failnum += 1
				if failnum > numRetries {
					log.Printf("AHRS Gyro/Accel Error: failed to read %d times, restarting: %s\n", failnum, mpuError)
					myIMUReader.Close()
					globalStatus.IMUConnected = false
					continue
				}
			}
			if magError != nil {
				log.Printf("AHRS Magnetometer Error, not using for this run: %s\n", magError)
				m.MValid = false
				// Don't necessarily disconnect here, unless AHRSProvider deeply depends on magnetometer
			}

			m.TW = float64(mySituation.LastGroundTrackTime.UnixNano() / 1000) / 1e6
			m.WValid = t.Sub(mySituation.LastGroundTrackTime) < 3000*time.Millisecond
			if m.WValid {
				m.W1 = mySituation.GroundSpeed * math.Sin(float64(mySituation.TrueCourse) * ahrs.Deg)
				m.W2 = mySituation.GroundSpeed * math.Cos(float64(mySituation.TrueCourse) * ahrs.Deg)
				if globalSettings.BMP_Sensor_Enabled && globalStatus.BMPConnected {
					m.W3 = mySituation.RateOfClimb * 60 / 6076.12
				} else {
					m.W3 = float64(mySituation.GPSVertVel) * 3600 / 6076.12
				}
			}

			// Run the AHRS calcs
			s.Compute(m)

			// Log it to csv for analysis
			if globalSettings.AHRSLog {
				if analysisLogger == nil {
					analysisFilename := filepath.Join(logDirf, fmt.Sprintf("sensors_%s.csv",
						time.Now().Format("20060102_150405")))
					analysisLogger = ahrs.NewAHRSLogger(analysisFilename, s.GetLogMap())
				}

				if analysisLogger != nil && usage.Usage() < 0.95 {
					analysisLogger.Log()
				}
			} else {
				analysisLogger = nil
			}

			// Debugging server:
			if ahrswebListener != nil {
				if err = ahrswebListener.Send(s.GetState(), m); err != nil {
					log.Printf("Error writing to ahrsweb: %s\n", err)
					ahrswebListener = nil
				}
			}

			// If we have valid AHRS info, then update mySituation
			if s.Valid() {
				mySituation.mu_Attitude.Lock()

				roll, pitch, heading = s.CalcRollPitchHeading()
				mySituation.Roll = roll / ahrs.Deg
				mySituation.Pitch = pitch / ahrs.Deg
				mySituation.Gyro_heading = heading / ahrs.Deg

				if headingMag, errHeadingMag = myIMUReader.MagHeading(); errHeadingMag != nil {
					log.Printf("AHRS MPU Error: %s\n", errHeadingMag.Error())
				} else {
					mySituation.Mag_heading = headingMag
				}

				if slipSkid, errSlipSkid = myIMUReader.SlipSkid(); errSlipSkid != nil {
					log.Printf("AHRS MPU Error: %s\n", errSlipSkid.Error())
				} else {
					mySituation.SlipSkid = slipSkid
				}

				if turnRate, errTurnRate = myIMUReader.RateOfTurn(); errTurnRate != nil {
					log.Printf("AHRS MPU Error: %s\n", errTurnRate.Error())
				} else {
					mySituation.RateOfTurn = turnRate
				}

				if gLoad, errGLoad = myIMUReader.GLoad(); errGLoad != nil {
					log.Printf("AHRS MPU Error: %s\n", errGLoad.Error())
				} else {
					mySituation.GLoad = gLoad
				}

				mySituation.LastAttitudeTime = t
				mySituation.mu_Attitude.Unlock()

				// makeFFAHRSSimReport() // simultaneous use of GDL90 and FFSIM not supported in FF 7.5.1 or later. Function definition will be kept for AHRS debugging and future workarounds.
			} else {
				s.Reset()
			}

			makeAHRSGDL90Report() // Send whether or not valid - the function will invalidate the values as appropriate
		}
	}
}

func getMinAccelDirection() (i int, err error) {
	_, _, _, _, a1, a2, a3, _, _, _, err, _ := myIMUReader.ReadRaw()
	if err != nil {
		return
	}
	log.Printf("AHRS Info: sensor orientation accels %1.3f %1.3f %1.3f\n", a1, a2, a3)
	switch {
	case math.Abs(a1) > math.Abs(a2) && math.Abs(a1) > math.Abs(a3):
		i = int(a1 / math.Abs(a1))
	case math.Abs(a2) > math.Abs(a3) && math.Abs(a2) > math.Abs(a1):
		i = int(a2 / math.Abs(a2)) * 2
	case math.Abs(a3) > math.Abs(a1) && math.Abs(a3) > math.Abs(a2):
		i = int(a3 / math.Abs(a3)) * 3
	default:
		err = fmt.Errorf("couldn't determine biggest accel from %1.3f %1.3f %1.3f", a1, a2, a3)
	}

	return
}

func CageAHRS() {
	cage<- true
}
