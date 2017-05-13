package main

import (
	"flag"
	"fmt"
	"github.com/takama/daemon"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// #include <wiringPi.h>
// #cgo LDFLAGS: -lwiringPi
import "C"

const (
	// CPU temperature target, degrees C
	defaultTempTarget = 65.
	hysteresis        = float32(1.)

	/* This puts our PWM frequency at 19.2 MHz / 2048 =
	/* 9.4kHz. Higher frequencies will reduce audible switching
	/* noise but will be less efficient */
	pwmClockDivisor = 2048

	/* Minimum duty cycle is the point below which the fan does
	/* not spin. This depends on both your fan and the switching
	/* transistor used. */
	defaultPwmDutyMin = 80
	pwmDutyMax        = 256

	// how often to update
	delaySeconds = 2

	// GPIO-12 on a Raspberry PI 3
	defaultPin = 26

	// name of the service
	name        = "fancontrol"
	description = "cooling fan speed control based on CPU temperature"

	// port which daemon should be listen
	port = ":9977"
)

var stdlog, errlog *log.Logger

func fanControl(pwmDutyMin int, pin int, tempTarget float32) {
	cPin := C.int(pin)
	C.wiringPiSetup()
	C.pwmSetMode(C.PWM_MODE_BAL)
	C.pinMode(cPin, C.PWM_OUTPUT)
	C.pwmSetRange(pwmDutyMax)
	C.pwmSetClock(pwmClockDivisor)
	C.pwmWrite(cPin, C.int(pwmDutyMin))
	temp := float32(0.)
	go cpuTempMonitor(func(cpuTemp float32) {
		if isCPUTempValid(cpuTemp) {
			temp = cpuTemp
		}
	})
	pwmDuty := 0
	for {
		if temp > (tempTarget + hysteresis) {
			pwmDuty = iMax(iMin(pwmDutyMax, pwmDuty+1), pwmDutyMin)
		} else if temp < (tempTarget - hysteresis) {
			pwmDuty = iMax(pwmDuty-1, 0)
			if pwmDuty < pwmDutyMin {
				pwmDuty = 0
			}
		}
		//log.Println(temp, " ", pwmDuty)
		C.pwmWrite(cPin, C.int(pwmDuty))
		time.Sleep(delaySeconds * time.Second)
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

	go fanControl(*pwmDutyMin, *pin, float32(*tempTarget))

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)

	// Set up listener for defined host and port
	listener, err := net.Listen("tcp", port)
	if err != nil {
		return "Possibly was a problem with the port binding", err
	}

	// set up channel on which to send accepted connections
	listen := make(chan net.Conn, 100)
	go acceptConnection(listener, listen)

	// loop work cycle with accept connections or interrupt
	// by system signal
	for {
		select {
		case conn := <-listen:
			go handleClient(conn)
		case killSignal := <-interrupt:
			stdlog.Println("Got signal:", killSignal)
			stdlog.Println("Stoping listening on ", listener.Addr())
			listener.Close()
			if killSignal == os.Interrupt {
				return "Daemon was interrupted by system signal", nil
			}
			return "Daemon was killed", nil
		}
	}
}

// Accept a client connection and collect it in a channel
func acceptConnection(listener net.Listener, listen chan<- net.Conn) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		listen <- conn
	}
}

func handleClient(client net.Conn) {
	for {
		buf := make([]byte, 4096)
		numbytes, err := client.Read(buf)
		if numbytes == 0 || err != nil {
			return
		}
		client.Write(buf[:numbytes])
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
