/*
	Copyright (c) 2017 Thorsten Biermann
	Distributable under the terms of The "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	flarm.go: Routines for reading FLARM traffic information
*/

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type FLARMMessage struct {
	Addr    string  `json:"addr"`
	Time    float64 `json:"time"`
	Rssi    float32 `json:"rssi"`
	Channel uint32  `json:"channel"`
	Lat     float32 `json:"lat"`
	Lon     float32 `json:"lon"`
	Dist    float32 `json:"dist"`
	Alt     int32   `json:"alt"`
	Vs      int32   `json:"vs"`
	Type    uint32  `json:"type"`
}

func decodeFLARMAircraftType(aircraftType uint32) string {
	switch aircraftType {
	case 0:
		// unknown
		return "UKN"
	case 1:
		// glider / motor glider
		return "GLID"
	case 2:
		// tow / tug plane
		return "TOW"
	case 3:
		// helicopter / rotorcraft
		return "HEL"
	case 4:
		// skydiver
		return "SKYD"
	case 5:
		// drop plane for skydivers
		return "DROP"
	case 6:
		// hang glider (hard)
		return "HANG"
	case 7:
		// paraglider (soft)
		return "PARA"
	case 8:
		// aircraft with reciprocating engine(s)
		return "PLN"
	case 9:
		// aircraft with jet/turboprop engine(s)
		return "JET"
	case 10:
		// unknown
		return "UKN"
	case 11:
		// balloon
		return "BAL"
	case 12:
		// airship
		return "SHIP"
	case 13:
		// unmanned aerial vehicle (UAV)
		return "UAV"
	case 14:
		// unknown
		return "UKN"
	case 15:
		// static object
		return "STAT"
	default:
		return "UKN"
	}
}

func watchCommand(command *exec.Cmd) {
	// wait for command to terminate
	err := command.Wait()

	log.Printf("Process %s terminated: %v", command.Path, err)
}

func replaceFlarmDecodingProcess(lonDeg float32, latDeg float32, oldDecodingProcess *os.Process, inputStream io.Reader) (*os.Process, io.Reader) {
	var err error

	// create new decoding process
	decodingCommand := exec.Command("flarm_decode", fmt.Sprintf("%.3f", latDeg), fmt.Sprintf("%.3f", lonDeg))

	// connect input data stream
	decodingCommand.Stdin = inputStream

	// get new decoding process' output stream
	var flarmOutput io.Reader
	if flarmOutput, err = decodingCommand.StdoutPipe(); err != nil {
		log.Printf("Error while getting Stdout pipe of decoding process: %s ", err)

		return nil, nil
	}

	// start new process
	if err = decodingCommand.Start(); err != nil {
		log.Printf("Error while starting decoding process: %s ", err)

		return nil, nil
	}
	log.Printf("Started new FLARM decoding process (pid=%d)", decodingCommand.Process.Pid)

	go watchCommand(decodingCommand)

	// kill old decoding process
	if oldDecodingProcess != nil {
		log.Printf("Stopping old FLARM decoding process (pid=%d)", oldDecodingProcess.Pid)

		if err = oldDecodingProcess.Kill(); err != nil {
			log.Printf("Error while killing old decoding process: %s ", err)

			return nil, nil
		}
	}

	return decodingCommand.Process, flarmOutput
}

func decodeFlarmJSONData(flarmOutput io.Reader) {
	log.Println("FLARM JSON decoding started")

	if flarmOutput != nil {
		inputScanner := bufio.NewScanner(flarmOutput)
		for inputScanner.Scan() {
			// log.Println("Received FLARM data:", inputScanner.Text())

			var flarmMessage FLARMMessage
			if err := json.Unmarshal(inputScanner.Bytes(), &flarmMessage); err != nil {
				log.Println("FLARM JSON decoding error:", err)
			} else {
				// log.Println("Decoded FLARM message:", flarmMessage)

				flarmAddress, err := strconv.ParseUint(flarmMessage.Addr, 16, 24)
				if err != nil {
					log.Println("Unable to decode FLARM address:", err)
					log.Println("FLARM message:", flarmMessage)
				} else {
					flarmAddress := uint32(flarmAddress)

					var ti TrafficInfo

					trafficMutex.Lock()

					// check if traffic is already known
					if existingTi, ok := traffic[flarmAddress]; ok {
						ti = existingTi
					}

					ti.Icao_addr = flarmAddress
					ti.Tail = fmt.Sprintf("F_%s_%s", decodeFLARMAircraftType(flarmMessage.Type), flarmMessage.Addr[len(flarmMessage.Addr)-2:])
					ti.Last_source = TRAFFIC_SOURCE_FLARM

					// set altitude
					ti.Alt = int32(convertMetersToFeet(float32(flarmMessage.Alt)))
					ti.Last_alt = stratuxClock.Time

					// set vertical speed
					ti.Vvel = int16(convertMetersToFeet(float32(flarmMessage.Vs)))

					// set latitude
					ti.Lat = flarmMessage.Lat

					// set longitude
					ti.Lng = flarmMessage.Lon

					// set RSSI
					ti.SignalLevel = float64(flarmMessage.Rssi)

					// add timestamp
					ti.Timestamp = time.Unix(int64(flarmMessage.Time), 0)

					if isGPSValid() {
						ti.Distance, ti.Bearing = distance(float64(mySituation.GPSLatitude), float64(mySituation.GPSLongitude), float64(ti.Lat), float64(ti.Lng))
						ti.BearingDist_valid = true
					}

					ti.Position_valid = true
					ti.ExtrapolatedPosition = false
					ti.Last_seen = stratuxClock.Time

					// update traffic database
					traffic[ti.Icao_addr] = ti

					// notify
					registerTrafficUpdate(ti)

					// mark traffic as seen
					seenTraffic[ti.Icao_addr] = true

					trafficMutex.Unlock()
				}
			}
		}
		if inputScanner.Err() != nil {
			log.Println("Error reading from FLARM stream:", inputScanner.Err())
		}
	} else {
		log.Println("Invalid FLARM source stream")
	}

	log.Println("FLARM JSON decoding terminated")
}

func flarmListen() {
	for {
		var err error

		if !globalSettings.FLARM_Enabled {
			// wait until FLARM is enabled
			time.Sleep(10 * time.Second)
			continue
		}

		// connect to rtl_tcp
		rtlTCPAddr := "127.0.0.1:40001"
		rtlTCPConnection, err := net.Dial("tcp", rtlTCPAddr)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		// create buffered reader for data coming from receive process (via network)
		receiveReader := bufio.NewReader(rtlTCPConnection)

		// create demodulation process
		demodProcess := exec.Command("nrf905_demod", "29")

		// connect receive process (via network) to demodulation process
		demodProcess.Stdin = receiveReader

		demodStdout, err := demodProcess.StdoutPipe()
		if err != nil {
			log.Printf("Error while getting Stdout pipe of demodulation process: %s ", err)
		}

		// start demodulation process
		if err = demodProcess.Start(); err != nil {
			log.Printf("Error while starting demodulation process: %s ", err)
		}
		log.Printf("Started FLARM demodulation process (pid=%d)", demodProcess.Process.Pid)

		// initialize decoding infrastructure
		var decodingProcess *os.Process
		var flarmOutput io.Reader
		stopDecodingLoop := false

		// function that waits for the demodulation process to terminate
		go func(command *exec.Cmd, stopDecodingLoop *bool) {
			watchCommand(demodProcess)

			*stopDecodingLoop = true
		}(demodProcess, &stopDecodingLoop)

		// set timer for (re-)starting decoding process (to use latest position)
		flarmDecoderRestartTimer := time.NewTicker(60 * time.Second)

		// initialize last position
		var lastLon, lastLat float32 = 0.0, 0.0

		// re-start loop to adapt decoding to latest position
	decodingLoop:
		for {
			select {
			case <-flarmDecoderRestartTimer.C:
				// restart timer has triggered

				// stop loop if demodulation process has terminated
				if stopDecodingLoop == true {
					log.Println("FLARM demodulation stopped, stopping decoding loop")
					break decodingLoop
				}

				// check if position has changes significantly
				if math.Abs(float64(mySituation.GPSLongitude-lastLon)) > 0.001 || math.Abs(float64(mySituation.GPSLatitude-lastLat)) > 0.001 {
					log.Println("Position has changed, restarting FLARM decoder")

					flarmOutput = nil

					// start new decoding process
					decodingProcess, flarmOutput = replaceFlarmDecodingProcess(mySituation.GPSLongitude, mySituation.GPSLatitude, decodingProcess, demodStdout)

					// start JSON decoding goroutine
					go decodeFlarmJSONData(flarmOutput)

					// store current location
					lastLon = mySituation.GPSLongitude
					lastLat = mySituation.GPSLatitude
				}
			}
		}
	}
}
