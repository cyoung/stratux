package main

import (
	"flag"
	"fmt"
	"github.com/takama/daemon"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

// #include <wiringPi.h>
// #cgo LDFLAGS: -lwiringPi
import "C"

const (
	// CPU temperature target, degrees C
	defaultTempTarget = 50.
	hysteresis        = float32(1.)

	// In Mark-Sweep mode:
	// PWM clock = 19.2 MHz / divisor / range
	defaultPwmClockDiv = 7

	// Minimum duty cycle is the point below which the fan does
	// not spin. This depends on both your fan and the switching
	// transistor used.
	defaultPwmDutyMin = 30
	pwmDutyMax        = 100

	// how often to update
	delaySeconds = 2

	// GPIO-1/BCM "18"/Pin 12 on a Raspberry Pi 3
	defaultFanPin = 1
	// GPIO-29/BCM "21"/Pin 40 on a Raspberry Pi 3
	defaultShutdownPin = 29

	// name of the service
	name        = "hwcontrol"
	description = "control for cooling fan and shutdown"
)

var stdlog, errlog *log.Logger

type hwControlArgs struct {
	pwmDutyMin  int
	pwmClockDiv int
	fanPin      int
	shutdownPin int
	tempTarget  float32
}

func hwControl(args hwControlArgs) {
	stdlog.Printf("Starting up with %+v", args)

	wiringPiVersionMajor := C.int(0)
	wiringPiVersionMinor := C.int(0)
	C.wiringPiVersion(&wiringPiVersionMajor, &wiringPiVersionMinor)
	stdlog.Printf("Using wiringPi %d.%d", wiringPiVersionMajor, wiringPiVersionMinor)
	C.wiringPiSetup()

	// Fan PWM setup. The order of these commands appears to
	// matter so be sure to test after a reboot when making
	// changes.
	cFanPin := C.int(args.fanPin)
	C.pinMode(cFanPin, C.PWM_OUTPUT)
	C.pwmSetClock(C.int(args.pwmClockDiv))
	C.pwmSetRange(pwmDutyMax)
	// MS means "mark-sweep" a.k.a. normal PWM. The other mode is
	// "balanced" which tries to reinterpret clock frequency and
	// should not be used for motor control.
	C.pwmSetMode(C.PWM_MODE_MS)
	C.pwmWrite(cFanPin, C.int(args.pwmDutyMin))

	// low power shutdown setup
	cShutdownPin := C.int(args.shutdownPin)
	C.pinMode(cShutdownPin, C.INPUT)
	C.pullUpDnControl(cShutdownPin, C.PUD_UP)

	temp := float32(0.)
	go cpuTempMonitor(func(cpuTemp float32) {
		if isCPUTempValid(cpuTemp) {
			temp = cpuTemp
		}
	})
	pwmDuty := 0
	shutdownLowCount := 0
	for {
		if temp > (args.tempTarget + hysteresis) {
			if pwmDuty == 0 {
				stdlog.Println("Starting fan at temperature", temp)
			} else if pwmDuty+1 == pwmDutyMax {
				stdlog.Println("Fan is at maximum with temperature", temp)
			}
			pwmDuty = iMax(iMin(pwmDutyMax, pwmDuty+1), args.pwmDutyMin)

		} else if temp < (args.tempTarget - hysteresis) {
			pwmDuty = iMax(pwmDuty-1, 0)
			if pwmDuty == args.pwmDutyMin-1 {
				pwmDuty = 0
				stdlog.Println("Stopping fan at temperature", temp)
			}
		}
		C.pwmWrite(cFanPin, C.int(pwmDuty))
		time.Sleep(delaySeconds * time.Second)

		if C.digitalRead(cShutdownPin) == 0 {
			shutdownLowCount += 1
		} else {
			shutdownLowCount = 0
		}
		// We debounce the shutdown input because, when used
		// with the "low battery" output of a boost converter,
		// we may see flickering when the battery is nearly
		// dead but still has a bit left
		if shutdownLowCount == 10 {
			stdlog.Println("Shutting down the system")
			cmd := exec.Command("systemctl", "poweroff")
			err := cmd.Run()
			if err != nil {
				errlog.Fatal(err)
			}
		}
	}
}

// Service has embedded daemon
type Service struct {
	daemon.Daemon
}

// Manage by daemon commands or run the daemon
func (service *Service) Manage() (string, error) {

	tempTarget := flag.Float64("temp", defaultTempTarget, "Target CPU Temperature, degrees C")
	pwmDutyMin := flag.Int("minduty", defaultPwmDutyMin, "Minimum PWM duty cycle")
	fanPin := flag.Int("fanpin", defaultFanPin, "Fan PWM pin (wiringPi numbering)")
	shutdownPin := flag.Int("shutdownpin", defaultShutdownPin, "Shutdown pin (wiringPi numbering)")
	pwmClockDiv := flag.Int("pwmclockdiv", defaultPwmClockDiv, "PWM Clock Divisor")
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

	go hwControl(hwControlArgs{
		pwmDutyMin:  *pwmDutyMin,
		fanPin:      *fanPin,
		shutdownPin: *shutdownPin,
		tempTarget:  float32(*tempTarget),
		pwmClockDiv: *pwmClockDiv})

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)

	// loop work cycle with accept connections or interrupt
	// by system signal
	for {
		select {
		case killSignal := <-interrupt:
			stdlog.Println("Got signal:", killSignal)
			if killSignal == os.Interrupt {
				return "Daemon was interrupted by system signal", nil
			}
			return "Daemon was killed", nil
		}
	}
}

func init() {
	stdlog = log.New(os.Stdout, "", 0)
	errlog = log.New(os.Stderr, "", 0)
}

func main() {
	srv, err := daemon.New(name, description, []string{}...)
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
