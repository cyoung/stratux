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
	decodingCommand := exec.Command("flarm_decode", fmt.Sprintf("%.3f", lonDeg), fmt.Sprintf("%.3f", latDeg))
	// decodingCommand := exec.Command("/root/stratux/sandbox/file_reader.py", "/root/stratux/sandbox/flarm_decode_1.txt")

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
			log.Println("Received FLARM data:", inputScanner.Text())

			var flarmMessage FLARMMessage
			if err := json.Unmarshal(inputScanner.Bytes(), &flarmMessage); err != nil {
				log.Println("FLARM JSON decoding error:", err)
			} else {
				log.Println("Decoded FLARM message:", flarmMessage)

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
	var err error

	// create rtl_sdr receive process
	receiveProcess := exec.Command("rtl_sdr", "-d", "1", "-f", "868.05m", "-s", "1.6m", "-g", "49.6", "-p", "49", "-")

	// create demodulation process
	demodProcess := exec.Command("nrf905_demod", "29")

	// connect process pipes
	if demodProcess.Stdin, err = receiveProcess.StdoutPipe(); err != nil {
		log.Printf("Error while connecting receive to demodulation process: %s ", err)
	}

	// // debug
	// flarmOutput := bytes.NewBufferString("{\"addr\":\"3D1F63\",\"time\":1499611315.018839,\"rssi\":-40.0,\"channel\":118,\"lat\":48.0918336,\"lon\":11.6618432,\"dist\":1975.58,\"alt\":1182,\"vs\":4,\"type\":2,\"stealth\":0,\"no_track\":0,\"ns\":[-61,-61,-60,-60],\"ew\":[-22,-23,-23,-24]}")

	demodStdout, err := demodProcess.StdoutPipe()
	if err != nil {
		log.Printf("Error while getting Stdout pipe of demodulation process: %s ", err)
	}

	// start demodulation process
	if err = demodProcess.Start(); err != nil {
		log.Printf("Error while starting demodulation process: %s ", err)
	}
	log.Printf("Started FLARM demodulation process (pid=%d)", demodProcess.Process.Pid)
	go watchCommand(demodProcess)

	// wait for demodulation process to get ready
	time.Sleep(5 * time.Second)

	// start receive process
	if err = receiveProcess.Start(); err != nil {
		log.Printf("Error while starting receive process: %s ", err)
	}
	log.Printf("Started FLARM receive process (pid=%d)", receiveProcess.Process.Pid)
	go watchCommand(receiveProcess)

	// wait for receive process to get ready
	time.Sleep(5 * time.Second)

	// initialize decoding infrastructure
	var decodingProcess *os.Process
	var flarmOutput io.Reader

	// set timer for (re-)starting decoding process (to use latest position)
	flarmDecoderRestartTimer := time.NewTicker(60 * time.Second)

	// initialize last position
	var lastLon, lastLat float32 = 0.0, 0.0

	// re-start loop to adapt decoding to latest position
	for {
		select {
		case <-flarmDecoderRestartTimer.C:
			// restart timer has triggered

			// check if position has changes significantly
			if math.Abs(float64(mySituation.GPSLongitude-lastLon)) > 0.001 || math.Abs(float64(mySituation.GPSLatitude-lastLat)) > 0.001 {
				log.Println("Position has changed. Restarting FLARM decoder.")

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
