/*
	Copyright (c) 2016 uAvionix
	Distributable under the terms of The "BSD New"" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	ping.go: uAvionix Ping ADS-B monitoring and management.
*/

package main

import (
	"bufio"
	"fmt"
	"strings"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"
	"net"
	"os/exec"

	"github.com/tarm/serial"
)

// Ping device data
var pingSerialConfig *serial.Config
var pingSerialPort *serial.Port
var pingWG *sync.WaitGroup
var closeCh chan int

func initPingSerial() bool {
	var device string
	baudrate := int(2000000)

	log.Printf("Configuring Ping ADS-B\n")

	if _, err := os.Stat("/dev/ttyUSB0"); err == nil {
		device = "/dev/ttyUSB0"
	} else if _, err := os.Stat("/dev/ping"); err == nil {
		device = "/dev/ping"
	} else {
		log.Printf("No suitable device found.\n")
		return false
	}
	log.Printf("Using %s for Ping\n", device)

	// Open port
	//pingSerialConfig = &serial.Config{Name: device, Baud: baudrate, ReadTimeout: time.Millisecond * 2500}
	pingSerialConfig = &serial.Config{Name: device, Baud: baudrate}
	p, err := serial.OpenPort(pingSerialConfig)
	if err != nil {
		log.Printf("Error opening serial port: %s\n", err.Error())
		return false
	}
	log.Printf("Ping opened serial port")

	// No device configuration is needed, we should be ready

	pingSerialPort = p
	return true
}

func pingNetworkRepeater() {
	defer pingWG.Done()
	log.Println("Entered Ping network repeater ...")
	cmd := exec.Command("/usr/bin/dump1090", "--net-only")
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	err := cmd.Start()
	if err != nil {
		log.Printf("Error executing /usr/bin/dump1090: %s\n", err)
		// don't return immediately, use the proper shutdown procedure
		shutdownPing = true
		for {
			select {
			case <-closeCh:
				return
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}

	log.Println("Executed /usr/bin/dump1090 successfully...")

	scanStdout := bufio.NewScanner(stdout)
	scanStderr := bufio.NewScanner(stderr)

	for {
		select {
		case <-closeCh:
			log.Println("Ping network repeater: shutdown msg received, calling cmd.Process.Kill() ...")
			err := cmd.Process.Kill()
			if err != nil {
				log.Printf("\t couldn't kill dump1090: %s\n", err)
			} else {
				cmd.Wait()
				log.Println("\t kill successful...")
			}
			return
		default:
			for scanStdout.Scan() {
				replayLog(scanStdout.Text(), MSGCLASS_DUMP1090)
			}
			if err := scanStdout.Err(); err != nil {
				log.Printf("scanStdout error: %s\n", err)
			}

			for scanStderr.Scan() {
				replayLog(scanStderr.Text(), MSGCLASS_DUMP1090)
				if shutdownES != true {
					shutdownES = true
				}
			}
			if err := scanStderr.Err(); err != nil {
				log.Printf("scanStderr error: %s\n", err)
			}

			time.Sleep(1 * time.Second)
		}
	}
}

var dump1090Connection net.Conn = nil
var connectionError error

func pingNetworkConnection() {
	// Send to dump1090 on port 30001
	dump1090Addr := "127.0.0.1:30001"
	dump1090Connection, connectionError = net.Dial("tcp", dump1090Addr)
	// RCB monitoir for connection failure and redial
}

func pingSerialReader() {
	//defer pingWG.Done()
	defer pingSerialPort.Close()
	// RCB TODO channel control for terminate

	log.Printf("Starting Ping serial reader")

	scanner := bufio.NewScanner(pingSerialPort)
	for scanner.Scan() && globalStatus.Ping_connected && globalSettings.Ping_Enabled {
		s := scanner.Text()
		// Trimspace removes newlines as well as whitespace
		s = strings.TrimSpace(s)
		logString := fmt.Sprintf("Print received: %s", s);
		log.Println(logString)
		if s[0] == '*' {
			// 1090ES report
			//replayLog(s, MSGCLASS_DUMP1090);
			if dump1090Connection != nil {
				dump1090Connection.Write([]byte(s + "\r\n"))
				log.Println("Relaying 1090ES message")
			}
		} else {
			// UAT report
			o, msgtype := parseInput(s)
			if o != nil && msgtype != 0 {
				logString = fmt.Sprintf("Relaying message, type=%d", msgtype)
				log.Println(logString)
				relayMessage(msgtype, o)
			} else if (o == nil) {
				log.Println("Not relaying message, o == nil")
			} else {
				log.Println("Not relaying message, msgtype == 0")
			}
		}
	}
	globalStatus.Ping_connected = false
	log.Printf("Exiting Ping serial reader")
	return
}

func pingShutdown() {
	log.Println("Entered Ping shutdown() ...")
	//close(closeCh)
	//log.Println("Ping shutdown(): calling pingWG.Wait() ...")
	//pingWG.Wait() // Wait for the goroutine to shutdown
	//log.Println("Ping shutdown(): pingWG.Wait() returned...")
	// RCB TODO FINISH
	globalStatus.Ping_connected = false
}

func pingKill() {
	// Send signal to shutdown to pingWatcher().
	shutdownPing = true
	// Spin until device has been de-initialized.
	for globalStatus.Ping_connected != false {
		time.Sleep(1 * time.Second)
	}
}

// to keep our sync primitives synchronized, only exit a read
// method's goroutine via the close flag channel check, to
// include catastrophic dongle failures
var shutdownPing bool

// Watch for config/device changes.
func pingWatcher() {
	prevPingEnabled := false

	for {
		time.Sleep(1 * time.Second)

		// true when a serial call fails
		if shutdownPing {
			pingShutdown()
			shutdownPing = false
		}

		if prevPingEnabled == globalSettings.Ping_Enabled {
			continue
		}

		// Global settings have changed, reconfig
		if globalSettings.Ping_Enabled && !globalStatus.Ping_connected {
			globalStatus.Ping_connected = initPingSerial()
			count := 0
			if globalStatus.Ping_connected {
				//pingWG.Add(1)
				go pingNetworkRepeater()
				go pingNetworkConnection()
				go pingSerialReader()
				// Emulate SDR count
				count = 2
			}
			atomic.StoreUint32(&globalStatus.Devices, uint32(count))
		} else if !globalSettings.Ping_Enabled {
			pingShutdown()
		}

		prevPingEnabled = globalSettings.Ping_Enabled
	}
}

func pingInit() {
	go pingWatcher()
}
