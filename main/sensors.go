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
)

var (
	i2cbus           embd.I2CBus
	myPressureReader sensors.PressureReader
	myIMUReader      sensors.IMUReader
)

func initI2CSensors() {
	i2cbus = embd.NewI2CBus(1)

	globalStatus.PressureSensorConnected = initPressureSensor() // I2C temperature and pressure altitude.
	log.Printf("AHRS Info: pressure sensor connected: %t\n", globalStatus.PressureSensorConnected)
	globalStatus.IMUConnected = initIMU()                       // I2C accel/gyro/mag.
	log.Printf("AHRS Info: IMU connected: %t\n", globalStatus.IMUConnected)

	if !(globalStatus.PressureSensorConnected || globalStatus.IMUConnected) {
		i2cbus.Close()
		log.Println("AHRS Info: I2C bus closed")
	}

	if globalStatus.PressureSensorConnected {
		go tempAndPressureSender()
		log.Println("AHRS Info: monitoring pressure sensor")
	}
}

func initPressureSensor() (ok bool) {
	bmp, err := sensors.NewBMP280(&i2cbus, 100*time.Millisecond)
	if err == nil {
		myPressureReader = bmp
		ok = true
		log.Println("AHRS Info: Successfully initialized BMP280")
		return
	}

	// TODO westphae: make bmp180.go to fit bmp interface
	//for i := 0; i < 5; i++ {
	//	myBMPX80 = bmp180.New(i2cbus)
	//	_, err := myBMPX80.Temperature() // Test to see if it works, since bmp180.New doesn't return err
	//	if err != nil {
	//		time.Sleep(250 * time.Millisecond)
	//	} else {
	//		globalStatus.PressureSensorConnected = true
	//		log.Println("AHRS Info: Successfully initialized BMP180")
	//		return nil
	//	}
	//}

	log.Println("AHRS Info: couldn't initialize BMP280 or BMP180")
	return
}

func initIMU() (ok bool) {
	var err error

	for i := 0; i < 5; i++ {
		log.Printf("AHRS Info: attempt %d to connect to MPU9250\n", i)
		myIMUReader, err = sensors.NewMPU9250()
		if err != nil {
			log.Printf("AHRS Info: attempt %d failed to connect to MPU9250\n", i)
			time.Sleep(100 * time.Millisecond)
		} else {
			time.Sleep(time.Second)
			log.Println("AHRS Info: Successfully connected MPU9250, running calibration")
			myIMUReader.Calibrate(1, 5)
			log.Println("AHRS Info: Successfully initialized MPU9250")
			return true
		}
	}

	//for i := 0; i < 5; i++ {
	//	myIMUReader, err = sensors.NewMPU6050()
	//	if err != nil {
	//		log.Printf("AHRS Info: attempt %d failed to connect to MPU6050\n", i)
	//		time.Sleep(100 * time.Millisecond)
	//	} else {
	//		ok = true
	//		log.Println("AHRS Info: Successfully initialized MPU6050")
	//		return
	//	}
	//}

	log.Println("AHRS Error: couldn't initialize MPU9250 or MPU6050")
	return
}

func tempAndPressureSender() {
	var (
		temp     float64
		press    float64
		altLast  float64
		altitude float64
		err      error
		dt       float64 = 0.1
	)

	// Initialize variables for rate of climb calc
	u := 5 / (5 + float64(dt)) // Use 5 sec decay time for rate of climb, slightly faster than typical VSI
	if press, err = myPressureReader.Pressure(); err != nil {
		log.Printf("AHRS Error: Couldn't read temp from sensor: %s", err)
	}
	altLast = CalcAltitude(press)

	timer := time.NewTicker(time.Duration(1000*dt) * time.Millisecond)
	for globalStatus.PressureSensorConnected {
		<-timer.C

		// Read temperature and pressure altitude.
		temp, err = myPressureReader.Temperature()
		if err != nil {
			log.Printf("AHRS Error: Couldn't read temperature from sensor: %s", err)
		}
		press, err = myPressureReader.Pressure()
		if err != nil {
			log.Printf("AHRS Error: Couldn't read pressure from sensor: %s", err)
			continue
		}

		// Update the Situation data.
		mySituation.mu_Pressure.Lock()
		mySituation.LastTempPressTime = stratuxClock.Time
		mySituation.Temp = temp
		altitude = CalcAltitude(press)
		mySituation.Pressure_alt = altitude
		// Assuming timer is reasonably accurate, use a regular ewma
		mySituation.RateOfClimb = u*mySituation.RateOfClimb + (1-u)*(altitude-altLast)/(float64(dt)/60)
		mySituation.mu_Pressure.Unlock()
		altLast = altitude
	}
}

func sensorAttitudeSender() {
	log.Println("AHRS Info: Setting up sensorAttitudeSender")
	var (
		roll, pitch, heading                              float64
		t                                                 time.Time
		s                                                 ahrs.AHRSProvider
		m                                                 *ahrs.Measurement
		bx, by, bz, ax, ay, az, mx, my, mz                float64
		mpuError, magError                                error
		headingMag, slipSkid, turnRate, gLoad             float64
		errHeadingMag, errSlipSkid, errTurnRate, errGLoad error
	)
	m = ahrs.NewMeasurement()

	//TODO westphae: remove this logging when finished testing, or make it optional in settings
	logger := ahrs.NewSensorLogger(fmt.Sprintf("/var/log/sensors_%s.csv", time.Now().Format("20060102_150405")),
		"T", "TS", "A1", "A2", "A3", "H1", "H2", "H3", "M1", "M2", "M3", "TW", "W1", "W2", "W3", "TA", "Alt",
		"pitch", "roll", "heading", "mag_heading", "slip_skid", "turn_rate", "g_load", "T_Attitude")
	defer logger.Close()

	ahrswebListener, err := ahrsweb.NewKalmanListener()
	if err != nil {
		log.Printf("AHRS Error: couldn't start ahrswebListener: %s\n", err.Error())
	}

	// Need a 10Hz sampling freq
	timer := time.NewTicker(100 * time.Millisecond) // ~10Hz update.
	for globalStatus.IMUConnected {
		<-timer.C
		t = stratuxClock.Time
		m.T = float64(t.UnixNano()/1000) / 1e6

		_, bx, by, bz, ax, ay, az, mx, my, mz, mpuError, magError = myIMUReader.ReadRaw()
		//TODO westphae: allow user configuration of this mapping from a file, plus UI modification
		//m.B1, m.B2, m.B3 = +by, -bx, +bz // This is how the RY83XAI is wired up
		//m.A1, m.A2, m.A3 = -ay, +ax, -az // This is how the RY83XAI is wired up
		m.B1, m.B2, m.B3 = -bx, +by, -bz // This is how the OpenFlightBox board is wired up
		m.A1, m.A2, m.A3 = -ay, +ax, +az // This is how the OpenFlightBox board is wired up
		m.M1, m.M2, m.M3 = +mx, +my, +mz
		m.SValid = mpuError == nil
		m.MValid = magError == nil
		if mpuError != nil {
			log.Printf("AHRS Gyro/Accel Error, not using for this run: %s\n", mpuError.Error())
			//TODO westphae: disconnect?
		}
		if magError != nil {
			log.Printf("AHRS Magnetometer Error, not using for this run: %s\n", magError.Error())
			m.MValid = false
		}

		m.WValid = t.Sub(mySituation.LastGroundTrackTime) < 500*time.Millisecond
		if m.WValid {
			m.W1 = mySituation.GroundSpeed * math.Sin(float64(mySituation.TrueCourse)*ahrs.Deg)
			m.W2 = mySituation.GroundSpeed * math.Cos(float64(mySituation.TrueCourse)*ahrs.Deg)
			if globalStatus.PressureSensorConnected {
				m.W3 = mySituation.RateOfClimb * 3600 / 6076.12
			} else {
				m.W3 = float64(mySituation.GPSVertVel) * 3600 / 6076.12
			}
		}

		// Run the AHRS calcs
		if s == nil { // s is nil if we should (re-)initialize the Kalman state
			log.Println("AHRS Info: initializing new simple AHRS")
			s = ahrs.InitializeSimple(m, "")
		}
		s.Compute(m)

		// Debugging server:
		if ahrswebListener != nil {
			ahrswebListener.Send(s.GetState(), m)
		}

		// If we have valid AHRS info, then send
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
			s = nil
			mySituation.LastAttitudeTime = time.Time{}
		}

		makeAHRSGDL90Report() // Send whether or not valid - the function will invalidate the values as appropriate

		logger.Log(
			float64(t.UnixNano()/1000)/1e6,
			m.T, m.A1, m.A2, m.A3, m.B1, m.B2, m.B3, m.M1, m.M2, m.M3,
			float64(mySituation.LastGroundTrackTime.UnixNano()/1000)/1e6, m.W1, m.W2, m.W3,
			float64(mySituation.LastTempPressTime.UnixNano()/1000)/1e6, mySituation.Pressure_alt,
			pitch/ahrs.Deg, roll/ahrs.Deg, heading/ahrs.Deg, headingMag, slipSkid, turnRate, gLoad,
			float64(mySituation.LastAttitudeTime.UnixNano()/1000)/1e6)
	}
	log.Println("AHRS Info: Exited sensorAttitudeSender loop")
	globalStatus.IMUConnected = false
	ahrswebListener.Close()
	myPressureReader.Close()
	myIMUReader.Close()
}
