package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/b3nn0/stratux/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/takama/daemon"
)

// #include <wiringPi.h>
// #cgo LDFLAGS: -lwiringPi
import "C"

// Initialize Prometheus metrics.
var (
	currentTemp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "current_temp",
		Help: "Current CPU temp.",
	})

	totalFanOnTime = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "total_fan_on_time",
			Help: "Total fan run time.",
		},
		[]string{"all"},
	)

	totalUptime = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "total_uptime",
			Help: "Total uptime.",
		},
		[]string{"all"},
	)
)

const (
	configLocation = "/boot/stratux.conf"

	// CPU temperature target, degrees C
	defaultTempTarget = 50.
	hysteresis        = float32(1.)

	pwmClockDivisor = 100

	/* Minimum duty cycle is the point below which the fan does
	/* not spin. This depends on both your fan and the switching
	/* transistor used. */
	defaultPwmDutyMin = 1
	pwmDutyMax        = 100

	// Temperature at which we give up attempting active fan speed control and set it to full speed.
	failsafeTemp = 65

	// how often to update
	delaySeconds = 30

	// GPIO-1/BCM "18"/Pin 12 on a Raspberry Pi 3
	defaultPin = 1

	// name of the service
	name        = "fancontrol"
	description = "cooling fan speed control based on CPU temperature"

	// Address on which daemon should be listen.
	addr = ":9977"
)

type FanControl struct {
	TempTarget     float32
	TempCurrent    float32
	PWMDutyMin     int
	PWMDutyMax     int
	PWMDutyCurrent int
	PWMPin         int
}

var myFanControl FanControl

var configChan = make(chan bool, 1)

var stdlog, errlog *log.Logger

func updateStats() {
	updateTicker := time.NewTicker(1 * time.Second)
	for {
		<-updateTicker.C
		totalUptime.With(prometheus.Labels{"all": "all"}).Inc()
		currentTemp.Set(float64(myFanControl.TempCurrent))
		if myFanControl.PWMDutyCurrent > 0 {
			totalFanOnTime.With(prometheus.Labels{"all": "all"}).Inc()
		}
	}
}

func fanControl() {
	prometheus.MustRegister(currentTemp)
	prometheus.MustRegister(totalFanOnTime)
	prometheus.MustRegister(totalUptime)

	go updateStats()

	cPin := C.int(myFanControl.PWMPin)

	C.wiringPiSetup()

	// Power on "test". Allows the user to verify that their fan is working.
	C.pinMode(cPin, C.OUTPUT)
	C.digitalWrite(cPin, C.HIGH)
	time.Sleep(5 * time.Second)
	C.digitalWrite(cPin, C.LOW)

	
	myFanControl.TempCurrent = 0
	go common.CpuTempMonitor(func(cpuTemp float32) {
		if common.IsCPUTempValid(cpuTemp) {
			myFanControl.TempCurrent = cpuTemp
		}
	})

	myFanControl.PWMDutyCurrent = 0

	delay := time.NewTicker(delaySeconds * time.Second)

	for {
		// Make sure we have 10 fanspeed steps, but if it's more than 90% duty cycle, we just increase by one every time
		pwmDutyStep := (myFanControl.PWMDutyMax - myFanControl.PWMDutyMin) / 10.0
		if pwmDutyStep < 1 {
			pwmDutyStep = 1
		}

		if myFanControl.TempCurrent > (myFanControl.TempTarget + hysteresis) {
			myFanControl.PWMDutyCurrent = common.IMax(common.IMin(myFanControl.PWMDutyMax, myFanControl.PWMDutyCurrent+pwmDutyStep), myFanControl.PWMDutyMin)
		} else if myFanControl.TempCurrent < (myFanControl.TempTarget - hysteresis) {
			myFanControl.PWMDutyCurrent = common.IMax(myFanControl.PWMDutyCurrent-pwmDutyStep, 0)
			if myFanControl.PWMDutyCurrent < myFanControl.PWMDutyMin {
				myFanControl.PWMDutyCurrent = myFanControl.PWMDutyMin
			}
		}
		//log.Println(myFanControl.TempCurrent, " ", myFanControl.PWMDutyCurrent)
		C.pwmSetMode(C.PWM_MODE_MS)
		C.pinMode(cPin, C.PWM_OUTPUT)
		C.pwmSetRange(C.uint(myFanControl.PWMDutyMax))
		C.pwmSetClock(pwmClockDivisor)
		C.pwmWrite(cPin, C.int(myFanControl.PWMDutyCurrent))
		select {
		case <-delay.C:
			if myFanControl.PWMDutyCurrent == myFanControl.PWMDutyMax && myFanControl.TempCurrent >= failsafeTemp {
				// Reached the maximum temperature. We stop using PWM control and set the fan to "on" permanently.
				break
			}
		case <-configChan:
			// Make sure we don't change speed again immediately
			myFanControl.TempCurrent = myFanControl.TempTarget
		}
	}

	// Default to "ON".
	C.pinMode(cPin, C.OUTPUT)
	C.digitalWrite(cPin, C.HIGH)
}

// Service has embedded daemon
type Service struct {
	daemon.Daemon
}

// Manage by daemon commands or run the daemon
func (service *Service) Manage() (string, error) {

	tempTarget := flag.Float64("temp", defaultTempTarget, "Target CPU Temperature, degrees C")
	pwmDutyMin := flag.Int("minduty", defaultPwmDutyMin, "Minimum PWM duty cycle")
	pin := flag.Int("pin", defaultPin, "PWM pin (wiringPi numbering)")
	flag.Parse()

	usage := "Usage: " + name + " install | remove | start | stop | status"
	// if received any kind of command, do it
	if flag.NArg() > 0 {
		command := os.Args[flag.NFlag()+1]
		switch command {
		case "install":
			return service.Install()
		case "remove":
			return service.Remove()
		case "start":
			return service.Start()
		case "stop":
			return service.Stop()
		case "status":
			return service.Status()
		default:
			return usage, nil
		}
	}

	myFanControl.TempTarget = float32(*tempTarget)
	myFanControl.PWMDutyMin = *pwmDutyMin
	myFanControl.PWMDutyMax = pwmDutyMax
	myFanControl.PWMPin = *pin

	readSettings()

	go fanControl()

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)

	http.HandleFunc("/", handleStatusRequest)
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(addr, nil)

	// interrupt by system signal
	for {
		killSignal := <-interrupt
		log.Println("Got signal:", killSignal)
		if killSignal == syscall.SIGINT {
			return "Daemon was interrupted by system signal", nil
		} else if (killSignal == syscall.SIGUSR1) {
			readSettings()
			myFanControl.PWMDutyCurrent = myFanControl.PWMDutyMin
			configChan<-true
		} else {
			return "Daemon was killed", nil
		}
	}

	return "", nil
}

func readSettings() {
	fd, err := os.Open(configLocation)
	if err != nil {
		log.Printf("can't read settings %s: %s\n", configLocation, err.Error())
		return
	}
	defer fd.Close()
	buf := make([]byte, 4096)
	count, err := fd.Read(buf)
	if err != nil {
		log.Printf("can't read settings %s: %s\n", configLocation, err.Error())
		return
	}
	err = json.Unmarshal(buf[0:count], &myFanControl)
	if err != nil {
		log.Printf("can't read settings %s: %s\n", configLocation, err.Error())
		return
	}
	log.Printf("read in settings.\n")
}

func handleStatusRequest(w http.ResponseWriter, r *http.Request) {
	statusJSON, _ := json.Marshal(&myFanControl)
	w.Write(statusJSON)
}

func init() {
	stdlog = log.New(os.Stdout, "", 0)
	errlog = log.New(os.Stderr, "", 0)
}

func main() {
	srv, err := daemon.New(name, description, daemon.SystemDaemon)
	if err != nil {
		errlog.Println("Error: ", err)
		os.Exit(1)
	}
	service := &Service{srv}
	status, err := service.Manage()
	if err != nil {
		errlog.Println(status, "\nError: ", err)
		os.Exit(1)
	}
	fmt.Println(status)
}
