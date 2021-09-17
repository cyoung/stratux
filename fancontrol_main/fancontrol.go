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
	"math"

	"github.com/b3nn0/stratux/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/takama/daemon"
	"github.com/felixge/pidctrl"
	"github.com/stianeikeland/go-rpio/v4"
)

import "C"

// Initialize Prometheus metrics.
var (
	currentTemp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "current_temp",
		Help: "Current CPU temp.",
	})

	currentPWM = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "current_pwm",
		Help: "Current PWM Value",
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

	/* Minimum duty cycle in % is the point below which the fan  */
	/* stops running nicely from a running situation */
	defaultPwmDutyMin = 50

	/* Maximum duty for PWM controller */
	pwmDutyMax        = 100
	pwmFanFrequency   = 3000

	// Temperature at which we give up attempting active fan speed control and set it to full speed.
	failsafeTemp = 65

	// how often to update
	updateDelayMS = 5000

	// start delay of the fan to start the fan to 80% to give the fan a kick to start spinning
	PWMDuty80PStartDelay = 500

	// GPIO-1/BCM "18"/Pin 12 on a Rev 2 and 3,4 Raspberry Pi   
	defaultPin = 18

	// name of the service
	name        = "fancontrol"
	description = "cooling fan speed control based on CPU temperature"

	// Address on which daemon should be listen.
	addr = ":9977"
)

type FanControl struct {
	TempTarget           float64
	TempCurrent          float64
	PWMDutyMin           uint32
	PWMDuty80PStartDelay uint32
	PWMDutyCurrent       uint32
	PWMPin               int
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
		currentPWM.Set(float64(myFanControl.PWMDutyCurrent))
		if myFanControl.PWMDutyCurrent > 0 {
			totalFanOnTime.With(prometheus.Labels{"all": "all"}).Inc()
		}
	}
}

func fmap( x, in_min, in_max, out_min, out_max float64) float64 {
	return (x - in_min) * (out_max - out_min) / (in_max - in_min) + out_min;
}



func fanControl() {
	myFanControl.PWMDuty80PStartDelay = PWMDuty80PStartDelay
	myFanControl.TempCurrent = 0
	myFanControl.PWMDutyCurrent = 0
	updateControlDelay := time.NewTicker(updateDelayMS * time.Millisecond)

	// Monitor Temperature
	go common.CpuTempMonitor(func(cpuTemp float32) {
		if common.IsCPUTempValid(cpuTemp) {
			myFanControl.TempCurrent = float64(cpuTemp)
		}
	})

	// Open Raspberry GPIO pins		
	err := rpio.Open()
	if err != nil {
			os.Exit(1)
	}
	defer rpio.Close()

	// Set PWM Mode
	pin := rpio.Pin(myFanControl.PWMPin)
	pin.Mode(rpio.Pwm)
	pin.Freq(pwmFanFrequency)
 
	// Calculate the dutyCycle to the hardware value
	dutyCycleToHW := func(dutyCycle float64) uint32 {
		mappedMinimum := fmap(float64(myFanControl.PWMDutyMin), 0.0, 100.0, 0, float64(pwmDutyMax))
		return uint32(math.Ceil(fmap(dutyCycle, 0.0, 100.0, mappedMinimum, float64(pwmDutyMax))))
	}
	
	// Fan test function
	turnOnFanTest := func () {
		myFanControl.PWMDutyCurrent = 100
		pin.DutyCycle(dutyCycleToHW(100.0), pwmDutyMax)
		time.Sleep(time.Duration(myFanControl.PWMDuty80PStartDelay) * time.Millisecond)	
		myFanControl.PWMDutyCurrent = myFanControl.PWMDutyMin
		pin.DutyCycle(dutyCycleToHW(float64(myFanControl.PWMDutyMin)), pwmDutyMax)
		time.Sleep(10 * time.Second)
	}

	// Power on "test". Allows the user to verify that their fan is working at the selected minimum duty cycle
	// Turns on the fan at minimum duty for 10 seconds. User should see that the fan keeps running all the time
	turnOnFanTest()

	// Start Prometheus		
	prometheus.MustRegister(currentTemp)
	prometheus.MustRegister(currentPWM)
	prometheus.MustRegister(totalFanOnTime)
	prometheus.MustRegister(totalUptime)
	go updateStats()

	// Create a PID controller
	pidControl := pidctrl.NewPIDController(0.2, 0.2, 0.1)
	pidControl.SetOutputLimits(-100, 0.0)
	pidControl.Set(myFanControl.TempTarget)

	var lastPWMControlValue float64 = 0.0
	for {

		// Update the PID controller.
		pidValueOut := -pidControl.UpdateDuration(myFanControl.TempCurrent, updateDelayMS * time.Millisecond)

		// If fan is starting up eg from 0 to some value, start it up for myFanControl.PWMDuty80PStartDelay at 80%
		if (lastPWMControlValue <=5.0 && pidValueOut>5.0) {
			log.Println("Starting up fan for" ,myFanControl.PWMDuty80PStartDelay, "ms")
			myFanControl.PWMDutyCurrent = 100
			pin.DutyCycle(pwmDutyMax, pwmDutyMax)
			time.Sleep(time.Duration(myFanControl.PWMDuty80PStartDelay) * time.Millisecond)
		}

		var pwmDutyMapped uint32 = 0
		if (pidValueOut > 5.0) {
			pwmDutyMapped = dutyCycleToHW(pidValueOut)
			myFanControl.PWMDutyCurrent = uint32(pidValueOut)
		} else {
			myFanControl.PWMDutyCurrent = 0
		}
		log.Println(myFanControl.TempCurrent, " ", pwmDutyMapped, " ",lastPWMControlValue, " ", pidValueOut)
		pin.DutyCycle(pwmDutyMapped, pwmDutyMax)

		lastPWMControlValue = pidValueOut

		select {
			case <-updateControlDelay.C:
				break;
			case <-configChan:
				pidControl.Set(myFanControl.TempTarget)
				// set lastPWMControlValue so we go through a cycle of starting the fan
				lastPWMControlValue = 0
				turnOnFanTest()
		}
	}

	// Default to "ON" when we bail out
	pin.DutyCycle(pwmDutyMax, pwmDutyMax)
}

// Service has embedded daemon
type Service struct {
	daemon.Daemon
}

// Manage by daemon commands or run the daemon
func (service *Service) Manage() (string, error) {

	tempTarget := flag.Float64("temp", defaultTempTarget, "Target CPU Temperature, degrees C")
	pwmDutyMin := flag.Int("minduty", defaultPwmDutyMin, "Minimum PWM duty cycle")
	pin := flag.Int("pin", defaultPin, "PWM pin (BCM numbering)")
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

	myFanControl.TempTarget = float64(*tempTarget)
	myFanControl.PWMDutyMin = uint32(*pwmDutyMin)
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
