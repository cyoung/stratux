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
	//"fmt"
	"log"
	"os"
	"strings"
	"sync"
	//"sync/atomic"
	"net"
	"os/exec"
	"time"

	// Using forked version of tarm/serial to force Linux
	// instead of posix code, allowing for higher baud rates
	"github.com/uavionix/serial"
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

	if _, err := os.Stat("/dev/ping"); err == nil {
		device = "/dev/ping"
	} else {
		log.Printf("No suitable Ping device found.\n")
		return false
	}
	log.Printf("Using %s for Ping\n", device)

	// Open port
	// No timeout specified as Ping does not heartbeat
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
				m := Dump1090TermMessage{Text: scanStdout.Text(), Source: "stdout"}
				logDump1090TermMessage(m)
			}
			if err := scanStdout.Err(); err != nil {
				log.Printf("scanStdout error: %s\n", err)
			}

			for scanStderr.Scan() {
				m := Dump1090TermMessage{Text: scanStderr.Text(), Source: "stderr"}
				logDump1090TermMessage(m)
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
	// RCB monitor for connection failure and redial
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
		//logString := fmt.Sprintf("Ping received: %s", s)
		//log.Println(logString)
		if s[0] == '*' {
			// 1090ES report
			// Ping appends a signal strength at the end of the message
			// e.g. *8DC01C2860C37797E9732E555B23;ss=049D;
			// Remove this before forwarding to dump1090
			// We currently aren't doing anything with this information
			// and need to develop a scaling equation - we're using a
			// log detector for power so it should have a logarithmic
			// relationship. In one example, at -25dBm input (upper limit
			// of RX) we saw ~0x500. At -95dBm input (lower limit of RX)
			// we saw 0x370
			report := strings.Split(s, ";")
			//replayLog(s, MSGCLASS_DUMP1090);
			if dump1090Connection == nil {
				log.Println("Starting dump1090 network connection")
				pingNetworkConnection()
			}
			if len(report[0]) != 0 && dump1090Connection != nil {
				dump1090Connection.Write([]byte(report[0] + ";\r\n"))
				//log.Println("Relaying 1090ES message")
				//logString := fmt.Sprintf("Relaying 1090ES: %s;", report[0]);
				//log.Println(logString)
			}
		} else if s[0] == '+' || s[0] == '-' {
			// UAT report
			// Ping appends a signal strength and RS bit errors corrected
			// at the end of the message
			// e.g. -08A5DFDF3907E982585F029B00040080105C3AB4BC5C240700A206000000000000003A13C82F96C80A63191F05FCB231;rs=1;ss=A2;
			// We need to rescale the signal strength for interpretation by dump978,
			// which expects a 0-1000 base 10 (linear?) scale
			// RSSI is in hex and represents an int8 with -128 (0x80) representing an
			// errored measurement. There will be some offset from actual due to loss
			// in the path. In one example we measured 0x93 (-98) when injecting a
			// -102dBm signal
			o, msgtype := parseInput(s)
			if o != nil && msgtype != 0 {
				//logString = fmt.Sprintf("Relaying message, type=%d", msgtype)
				//log.Println(logString)
				relayMessage(msgtype, o)
			} else if o == nil {
				//log.Println("Not relaying message, o == nil")
			} else {
				//log.Println("Not relaying message, msgtype == 0")
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
			//count := 0
			if globalStatus.Ping_connected {
				//pingWG.Add(1)
				go pingNetworkRepeater()
				//pingNetworkConnection()
				go pingSerialReader()
				// Emulate SDR count
				//count = 2
			}
			//atomic.StoreUint32(&globalStatus.Devices, uint32(count))
		} else if !globalSettings.Ping_Enabled {
			pingShutdown()
		}

		prevPingEnabled = globalSettings.Ping_Enabled
	}
}

func pingInit() {
	go pingWatcher()
}
