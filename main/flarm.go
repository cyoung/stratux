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
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

// OGNConfigData stores the data required for generating the OGN configuration file
type OGNConfigData struct {
	DeviceIndex, Ppm, Longitude, Latitude, Altitude string
}

// OGNConfigDataCache is the global data structure for storing the latest OGN configuration
var OGNConfigDataCache OGNConfigData

// flag that indicates if OG decoding process is running
var ognDecoderIsRunning bool

// AprsFlarmData stores content of FLARM APRS aircraft beacon
type AprsFlarmData struct {
	Identifier     string
	ReceiverName   string
	Timestamp      string
	Latitude       float64
	Longitude      float64
	SymbolTable    string
	SymbolCode     string
	Track          int
	HSpeed         int
	Altitude       int
	StealthMode    bool
	AircraftType   byte
	AddressType    byte
	Address        uint32
	VSpeed         int
	SignalStrength float64
	BitErrorCount  int
	Valid          bool
}

func decodeFLARMAircraftType(aircraftType byte) string {
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

func decodeFLARMAddressType(addressType byte) string {
	switch addressType {
	case 0:
		return "RANDOM"
	case 1:
		return "ICAO"
	case 2:
		return "FLARM"
	case 3:
		return "OGN"
	default:
		return "UNKNOWN"
	}
}

var ognTailNumberCache = make(map[string]string)
func lookupOgnTailNumber(flarmid string) string {
	if len(ognTailNumberCache) == 0 {
		log.Printf("Parsing OGN device db")
		ddb, err := ioutil.ReadFile("/etc/ddb.json")
		if err != nil {
			log.Printf("Failed to read OGN device db")
			return flarmid
		}
		var data map[string]interface{}
		err = json.Unmarshal(ddb, &data)
		if err != nil {
			log.Printf("Failed to parse OGN device db")
			return flarmid
		}
		devlist := data["devices"].([]interface{})
		for i := 0; i < len(devlist); i++ {
			dev := devlist[i].(map[string]interface{})
			flarmid := dev["device_id"].(string)
			tail := dev["registration"].(string)
			ognTailNumberCache[flarmid] = tail
		}
		log.Printf("Successfully parsed OGN device db")
	}
	if tail, ok := ognTailNumberCache[flarmid]; ok {
		return tail
	}
	return flarmid
}

func getTailNumber(flarmid string) string {
	tail := lookupOgnTailNumber(flarmid)
	if globalSettings.DisplayTrafficSource {
		tail = "fl" + tail
	}
	return tail
}

func watchCommand(command *exec.Cmd) {
	// wait for command to terminate
	err := command.Wait()

	// set status flag
	ognDecoderIsRunning = false

	if globalSettings.DEBUG {
		log.Printf("FLARM: Process %s terminated: %v", command.Path, err)
	}
}

func replaceFlarmDecodingProcess(lonDeg float32, latDeg float32, oldDecodingProcess *os.Process, configFileName string) *os.Process {
	var err error

	// kill old decoding process
	if oldDecodingProcess != nil {
		if globalSettings.DEBUG {
			log.Printf("FLARM: Stopping old decoding process (pid=%d)", oldDecodingProcess.Pid)
		}

		if err = oldDecodingProcess.Kill(); err != nil {
			log.Printf("FLARM: Error while killing old decoding process: %s ", err)

			return nil
		}
	}

	// create new decoding process
	decodingCommand := exec.Command("ogn-decode", configFileName)

	// get new decoding process' input stream
	decoderInput, err := decodingCommand.StdinPipe()
	if err != nil {
		log.Printf("FLARM: Error while getting Stdin pipe of decoding process: %s ", err)

		return nil
	}

	// get new decoding process' output stream
	decoderOutput, err := decodingCommand.StdoutPipe()
	if err != nil {
		log.Printf("FLARM: Error while getting Stdout pipe of decoding process: %s ", err)

		return nil
	}

	// get new decoding process' error stream
	decoderError, err := decodingCommand.StderrPipe()
	if err != nil {
		log.Printf("FLARM: Error while getting Stderr pipe of decoding process: %s ", err)

		return nil
	}

	// start new process
	if err = decodingCommand.Start(); err != nil {
		log.Printf("FLARM: Error while starting decoding process: %s ", err)

		return nil
	}
	if globalSettings.DEBUG {
		log.Printf("FLARM: Started new decoding process (pid=%d)", decodingCommand.Process.Pid)
	}

	// set status flag
	ognDecoderIsRunning = true

	go watchCommand(decodingCommand)

	// show stdout
	go func() {
		for {
			line, err := bufio.NewReader(decoderOutput).ReadString('\n')
			parseOgnStdoutMessage(line)
			if err == nil {
				if globalSettings.DEBUG {
					log.Println("FLARM: ogn-decode stdout:", strings.TrimSpace(line))
				}
			} else {
				return
			}
		}
	}()

	// show stderr
	go func() {
		for {
			line, err := bufio.NewReader(decoderError).ReadString('\n')
			if err == nil {
				log.Println("FLARM: ogn-decode stderr:", strings.TrimSpace(line))
			} else {
				return
			}
		}
	}()

	io.WriteString(decoderInput, "\n")

	return decodingCommand.Process
}

func createOGNConfigFile(templateFileName string, outputFileName string) {
	configTemplate, err := template.ParseFiles(templateFileName)
	if err != nil {
		log.Printf("FLARM: Unable to open OGN config template file: %s", err)
		return
	}

	outputFile, err := os.Create(outputFileName)
	defer outputFile.Close()
	if err != nil {
		log.Printf("FLARM: Unable to open OGN config file: %s", err)
		return
	}

	err = configTemplate.Execute(outputFile, OGNConfigDataCache)
	if err != nil {
		log.Printf("FLARM: Problem while executing OGN config file template: %s", err)
		return
	}
}

func ognCoordToDegrees(coordinate float64) float64 {
	// extract degree part
	degrees := float64(int(coordinate / 100.0))

	// extract minutes part
	minutes := coordinate - (degrees * 100.0)

	// add minutes to degrees
	degrees += minutes / 60.0

	return degrees
}

func atof32(val string) float32 {
	res, _ := strconv.ParseFloat(val, 32)
	return float32(res)
}

// Traffic messages look like this
// 0.458sec:868.188MHz:   8:2:AABBCC 085804: [ +45.5312, +8.1234]deg  123m  +0.0m/s   0.0m/s 000.0deg  +0.0deg/sec 0 03x05m 00f_-12.36kHz 45.2/61.0dB/0  0e    0.0km 000.0deg +69.4deg   ?
func parseOgnStdoutMessage(message string) {
	// See https://github.com/glidernet/python-ogn-client/blob/master/ogn/parser/pattern.py
	// PATTERN_TELNET_50001
	rx := `(?P<pps_offset>\d\.\d+)sec:(?P<frequency>\d+\.\d+)MHz:\s+
(?P<aircraft_type>.):(?P<address_type>\d):(?P<address>[A-F0-9]{6})\s
(?P<timestamp>\d{6}):\s
\[\s*(?P<latitude>[+-]\d+\.\d+),\s*(?P<longitude>[+-]\d+\.\d+)\]deg\s*
(?P<altitude>\d+)m\s*
(?P<climb_rate>[+-]\d+\.\d+)m/s\s*
(?P<ground_speed>\d+\.\d+)m/s\s*
(?P<track>\d+\.\d+)deg\s*
(?P<turn_rate>[+-]\d+\.\d+)deg/sec\s*
(?P<magic_number>\d+)\s*
(?P<gps_status>[0-9x]+)m\s*
(?P<channel>\d+)(?P<flarm_timeslot>[f_])(?P<ogn_timeslot>[o_])\s*
(?P<frequency_offset>[+-]\d+\.\d+)kHz\s*
(?P<decode_quality>\d+\.\d+)/(?P<signal_quality>\d+\.\d+)dB/(?P<demodulator_type>\d+)\s+
(?P<error_count>\d+)e\s*
(?P<distance>\d+\.\d+)km\s*
(?P<bearing>\d+\.\d+)deg\s*
(?P<phi>[+-]\d+\.\d+)deg\s*
(?P<multichannel>\+)?\s*
\?\s*
R?\s*
(B(?P<baro_altitude>\d+))?`
	rx = strings.Replace(rx, "\n", "", -1)
	var reAttrs = regexp.MustCompile(rx)
	match := reAttrs.FindStringSubmatch(message)
	if match == nil {
		//log.Printf("Match error for %s", message)
		return
	}
	// Append flarm message to message log
	var thisMsg msg
	thisMsg.MessageClass = MSGCLASS_FLARM
	thisMsg.TimeReceived = stratuxClock.Time
	thisMsg.Data = message
	MsgLog = append(MsgLog, thisMsg)

	attrMap := make(map[string]string)
	for i, name := range reAttrs.SubexpNames() {
        if i != 0 && name != "" {
			if len(match) > i {
				attrMap[name] = match[i]
			} else {
				log.Printf("??: " + message)
				attrMap[name] = ""
			}
			//log.Printf("%s: %s", name, match[i])
        }
	}

	// store aircraft information
	var ti TrafficInfo
	addressBytes, _ := hex.DecodeString(attrMap["address"])
	addressBytes = append([]byte{0}, addressBytes...)
	address := binary.BigEndian.Uint32(addressBytes)


	trafficMutex.Lock()
	defer trafficMutex.Unlock()
	
	// check if traffic is already known
	if existingTi, ok := traffic[address]; ok {
		ti = existingTi
	}
	ti.Icao_addr = address
	if len(ti.Tail) == 0 {
		ti.Tail = getTailNumber(attrMap["address"]) // Might have better tail from ADS-B. Don't overwrite.
	}
	ti.Last_source = TRAFFIC_SOURCE_FLARM

	ti.Timestamp = time.Now().UTC()
	var age time.Duration
	ts := attrMap["timestamp"]
	if len(ts) == 6 {
		// 112233 (hour/min/sec)
		currTs := time.Now().UTC()
		hour, _ := strconv.Atoi(ts[0:2])
		min, _ := strconv.Atoi(ts[2:4])
		sec, _ := strconv.Atoi(ts[4:6])
		signalTs := time.Date(currTs.Year(), currTs.Month(), currTs.Day(), hour, min, sec, 0, time.UTC)
		age = ti.Timestamp.Sub(signalTs)
		if age.Seconds() > 30 || age.Seconds() < -1 {
			// Sometimes we get some invalid traffic that is wrongly detected by OGN. Make sure that
			// at least the timestamp makes some sense, so our new traffic info does not time out immediately (or never if far in the future)
			log.Printf("Discarding likely invalid OGN target: %s", message)
			return 
		}
		ti.Timestamp = signalTs
	}


	// set altitude
	// To keep the rest of the system as simple as possible, we want to work with barometric altitude everywhere.
	// To do so, we use our own known geoid separation and pressure difference to compute the expected barometric altitude of the traffic.
	alt := atof32(attrMap["altitude"]) * 3.28084 // m in ft
	if isGPSValid() && isTempPressValid() {
		ti.Alt = int32(float32(alt) - mySituation.GPSHeightAboveEllipsoid + mySituation.BaroPressureAltitude)
	} else {
		ti.Alt = int32(alt)
		ti.AltIsGNSS = true
	}

	// set vertical speed
	vVel, _ := strconv.ParseFloat(attrMap["climb_rate"], 32)
	ti.Vvel = int16(vVel * 196.85) // m/s in ft/min

	// set latitude
	ti.Lat = atof32(attrMap["latitude"])

	// set longitude
	ti.Lng = atof32(attrMap["longitude"])

	// set track
	ti.Track = uint16(atof32(attrMap["track"]))

	// set speed
	ti.Speed = uint16(atof32(attrMap["ground_speed"]) * 1.94384)  // m/s in kt
	ti.Speed_valid = true

	ti.SignalLevel = float64(atof32(attrMap["decode_quality"]))

	// TODO: The timestamp of the position might actually be older (see signalTs computed above)
	// However, that would result in a jumping "age" column (older from flarm, newer from ads-b overwriting each other)
	// We can also not skip the flarm signal if it's older than the old ti's Timestamp, because that way,
	// we would throw away flarm data, even if the older ti is only from Mode-S and has no position info.
	// For now, we just live with it and set the timestamp to current.
	ti.Timestamp = time.Now().UTC()

	if isGPSValid() {
		ti.Distance, ti.Bearing = distance(float64(mySituation.GPSLatitude), float64(mySituation.GPSLongitude), float64(ti.Lat), float64(ti.Lng))
		ti.BearingDist_valid = true
	}

	ti.Position_valid = true
	ti.ExtrapolatedPosition = false
	ti.Last_seen = stratuxClock.Time.Add(-age)
	ti.Last_alt = stratuxClock.Time.Add(-age)

	if acType, ok := attrMap["aircraft_type"]; ok {
		switch(acType) {
		case "1": ti.Emitter_category = 9 // glider = glider
		case "2", "5", "8": ti.Emitter_category = 1 // tow, drop, piston = light
		case "3": ti.Emitter_category = 7 // helicopter = helicopter
		case "4": ti.Emitter_category = 11 // skydiver
		case "6", "7": ti.Emitter_category = 12 // hang glider / paraglider
		case "9": ti.Emitter_category = 3 // jet = large
		case "B", "C": ti.Emitter_category = 10 // Balloon, airship = lighter than air
		}
	}

	// update traffic database
	traffic[ti.Icao_addr] = ti

	// notify
	registerTrafficUpdate(ti)

	// mark traffic as seen
	seenTraffic[ti.Icao_addr] = true
}

// TODO: processing of APRS data is deprecated. Instead, we parse stdout of ogn-decode, which
// seems to be more up to date and has a higher update rate
// DEPRECATED
func processAprsData(aprsData string) {
	// prepare all regular expressions
	var reBeaconData = regexp.MustCompile(`^(.+?)>APRS,(.+?):/(\d{6})+h(\d{4}\.\d{2})(N|S)(.)(\d{5}\.\d{2})(E|W)(.)((\d{3})/(\d{3}))?/A=(\d{6})`)
	var reIdentifier = regexp.MustCompile(`id(\S{2})(\S{6})`)
	var reVSpeed = regexp.MustCompile(`([\+\-]\d+)fpm`)
	// var reTurnRate = regexp.MustCompile(`([\+\-]\d+\.\d+)rot`)
	var reSignalStrength = regexp.MustCompile(`(\d+\.\d+)dB`)
	var reBitErrorCount = regexp.MustCompile(`(\d+)e`)
	var reCoordinatesExtension = regexp.MustCompile(`\!W(.)(.)!`)
	// var reHearId = regexp.MustCompile(`hear(\w{4})`)
	// var reFrequencyOffset = regexp.MustCompile(`([\+\-]\d+\.\d+)kHz`)
	// var reGpsStatus = regexp.MustCompile(`gps(\d+x\d+)`)
	// var reSoftwareVersion = regexp.MustCompile(`s(\d+\.\d+)`)
	// var reHardwareVersion = regexp.MustCompile(`h(\d+)`)
	// var reRealId = regexp.MustCompile(`r(\w{6})`)
	// var reFlightLevel = regexp.MustCompile(`FL(\d{3}\.\d{2})`)

	aprsDataFields := strings.Split(aprsData, " ")

	var data AprsFlarmData
	data.Valid = false

	for _, aprsDataField := range aprsDataFields {
		if match := reBeaconData.FindStringSubmatch(aprsDataField); match != nil {
			data.Identifier = match[1]

			data.ReceiverName = match[2]

			data.Timestamp = match[3]

			latitudeRaw, _ := strconv.ParseFloat(match[4], 64)
			data.Latitude = ognCoordToDegrees(latitudeRaw)
			if match[5] == "S" {
				data.Latitude = -1.0 * data.Latitude
			}

			data.SymbolTable = match[6]

			longitudeRaw, _ := strconv.ParseFloat(match[7], 64)
			data.Longitude = ognCoordToDegrees(longitudeRaw)
			if match[8] == "W" {
				data.Longitude = -1.0 * data.Longitude
			}

			data.SymbolCode = match[9]

			data.Track, _ = strconv.Atoi(match[11])

			data.HSpeed, _ = strconv.Atoi(match[12])

			data.Altitude, _ = strconv.Atoi(match[13])

			// discard all receiver beacons
			if data.Identifier != "Stratux" {
				data.Valid = true
			}
		}

		if match := reCoordinatesExtension.FindStringSubmatch(aprsDataField); match != nil {
			// position precision enhancement is third decimal digit of minute
			latitudeDelta, _ := strconv.Atoi(match[1])
			latitudeDeltaDegrees := float64(latitudeDelta) / 1000.0 / 60.0

			longitudeDelta, _ := strconv.Atoi(match[2])
			longitudeDeltaDegrees := float64(longitudeDelta) / 1000.0 / 60.0

			data.Latitude += latitudeDeltaDegrees
			data.Longitude += longitudeDeltaDegrees
		}

		if match := reIdentifier.FindStringSubmatch(aprsDataField); match != nil {
			// Flarm ID type byte in APRS msg: SAAA AAII
			// S => stealth mode
			// AAAAA => aircraftType
			// II => addressType
			// (see https://groups.google.com/forum/#!msg/openglidernetwork/lMzl5ZsaCVs/YirmlnkaJOYJ).

			flagsBytes, err := hex.DecodeString(match[1])
			flagsDecoded := flagsBytes[0]
			if err != nil {
				log.Println("FLARM: Error while decoding identifier flags")
			} else {
				data.StealthMode = ((flagsDecoded&0x80)>>7 == 1)

				data.AircraftType = (flagsDecoded & 0x7C) >> 2

				data.AddressType = flagsDecoded & 0x03
			}

			addressBytes, err := hex.DecodeString(match[2])
			addressBytes = append([]byte{0}, addressBytes...)
			data.Address = binary.BigEndian.Uint32(addressBytes)
		}

		if match := reVSpeed.FindStringSubmatch(aprsDataField); match != nil {
			data.VSpeed, _ = strconv.Atoi(match[1])
		}

		if match := reSignalStrength.FindStringSubmatch(aprsDataField); match != nil {
			data.SignalStrength, _ = strconv.ParseFloat(match[1], 64)
		}

		if match := reBitErrorCount.FindStringSubmatch(aprsDataField); match != nil {
			data.BitErrorCount, _ = strconv.Atoi(match[1])
		}
	}

	if data.Valid == true {
		// store aircraft information

		var ti TrafficInfo

		trafficMutex.Lock()

		// check if traffic is already known
		if existingTi, ok := traffic[data.Address]; ok {
			ti = existingTi
		}

		ti.Icao_addr = data.Address
		ti.Tail = strings.ToUpper(fmt.Sprintf("F%s%s", decodeFLARMAircraftType(data.AircraftType), strconv.FormatInt(int64(data.Address), 16)))
		ti.Last_source = TRAFFIC_SOURCE_FLARM

		// set altitude
		// To keep the rest of the system as simple as possible, we want to work with barometric altitude everywhere.
		// To do so, we use our own known geoid separation and pressure difference to compute the expected barometric altitude of the traffic.
		if isGPSValid() && isTempPressValid() {
			ti.Alt = int32(float32(data.Altitude) - mySituation.GPSHeightAboveEllipsoid + mySituation.BaroPressureAltitude)
		} else {
			ti.Alt = int32(data.Altitude)
			ti.AltIsGNSS = true
		}


		ti.Last_alt = stratuxClock.Time

		// set vertical speed
		ti.Vvel = int16(data.VSpeed)

		// set latitude
		ti.Lat = float32(data.Latitude)

		// set longitude
		ti.Lng = float32(data.Longitude)

		// set track
		ti.Track = uint16(data.Track)

		// set speed
		ti.Speed = uint16(data.HSpeed)
		ti.Speed_valid = true

		// set RSSI
		ti.SignalLevel = data.SignalStrength

		// add timestamp
		ti.Timestamp = time.Now().UTC()
		var age time.Duration
		if len(data.Timestamp) == 6 {
			// 112233 (hour/min/sec)
			currTs := time.Now().UTC()
			hour, _ := strconv.Atoi(data.Timestamp[0:2])
			min, _ := strconv.Atoi(data.Timestamp[2:4])
			sec, _ := strconv.Atoi(data.Timestamp[4:6])
			signalTs := time.Date(currTs.Year(), currTs.Month(), currTs.Day(), hour, min, sec, 0, time.UTC)
			age = ti.Timestamp.Sub(signalTs)
			ti.Timestamp = signalTs
		}

		if isGPSValid() {
			ti.Distance, ti.Bearing = distance(float64(mySituation.GPSLatitude), float64(mySituation.GPSLongitude), float64(ti.Lat), float64(ti.Lng))
			ti.BearingDist_valid = true
		}

		ti.Position_valid = true
		ti.ExtrapolatedPosition = false
		ti.Last_seen = stratuxClock.Time.Add(-age)
		ti.Last_alt = stratuxClock.Time.Add(-age)

		// update traffic database
		traffic[ti.Icao_addr] = ti

		// notify
		registerTrafficUpdate(ti)

		// mark traffic as seen
		seenTraffic[ti.Icao_addr] = true

		trafficMutex.Unlock()

		if globalSettings.DEBUG {
			log.Printf("FLARM APRS: Decoded data: %+v\n", data)
		}
	}
}

func sendAprsConnectionHeartBeat(conn net.Conn) {
	for {
		time.Sleep(20 * time.Second)

		_, err := conn.Write([]byte(fmt.Sprintf("# %s %s %s %s %s\r\n", "stratux", globalStatus.Version, time.Now().UTC().Format("2 Jan 2006 15:04:05 GMT"), "STRATUX", "127.0.0.1:14580")))

		if err != nil {
			return
		}
	}
}

func handleAprsConnection(conn net.Conn) {
	if globalSettings.DEBUG {
		log.Println("FLARM APRS: Incoming connection:", conn.RemoteAddr())
	}
	globalStatus.FLARM_connected = true

	// send initial message
	conn.Write([]byte(fmt.Sprintf("# %s %s\r\n", "stratux", globalStatus.Version)))

	var reLoginRequest = regexp.MustCompile(`user (\w+) pass (\w+) vers (.+)`)

	go sendAprsConnectionHeartBeat(conn)

	for {
		// listen for message to process ending in newline (\n)
		message, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			log.Println("FLARM APRS: Error while reading from connection:", err)

			break
		}
		var thisMsg msg
		thisMsg.MessageClass = MSGCLASS_FLARM
		thisMsg.TimeReceived = stratuxClock.Time
		thisMsg.Data = message
		MsgLog = append(MsgLog, thisMsg)

		// check if message is not a receiver beacon
		if !strings.HasPrefix(string(message), "Stratux") {
			if globalSettings.DEBUG {
				log.Println("FLARM APRS: Message received:", string(message))
			}
		}

		if match := reLoginRequest.FindStringSubmatch(message); match != nil {
			username := match[1]

			// return authentication successful (credentials are not verified in current implementation)
			conn.Write([]byte(fmt.Sprintf("# logresp %s verified, server %s\r\n", username, "STRATUX")))
		} else {
			// processAprsData(message)
		}
	}

	globalStatus.FLARM_connected = false
	// Close the connection when you're done with it.
	conn.Close()
}

func aprsServer() {
	log.Println("FLARM APRS: Starting server")

	// listen for incoming connections
	l, err := net.Listen("tcp", "localhost:14580")
	if err != nil {
		log.Println("FLARM APRS: Unable to start APRS listening:", err)
	}
	defer l.Close()

	for {
		// wait for incoming connection
		conn, err := l.Accept()
		if err != nil {
			log.Println("FLARM APRS: Error accepting connection:", err)
		}

		// handle connection in a new goroutine
		go handleAprsConnection(conn)
	}
}

func flarmListen() {
	for {
		if !globalSettings.FLARM_Enabled {
			// wait until FLARM is enabled
			time.Sleep(10 * time.Second)
			continue
		}

		// start APRS server
		go aprsServer()

		// set OGN configuration file path
		configTemplateFileName := "/etc/stratux-ogn.conf.template"
		configFileName := "/tmp/stratux-ogn.conf"

		// initialize decoding infrastructure
		var decodingProcess *os.Process
		stopDecodingLoop := false
		ognDecoderIsRunning = false

		// set timer for (re-)starting decoding process (to use latest position)
		flarmDecoderRestartTimer := time.NewTicker(10 * time.Second)

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
					if globalSettings.DEBUG {
						log.Println("FLARM: Stopping decoding loop")
					}
					break decodingLoop
				}

				// check if position has changes significantly. 0.3 lat/lon diff is approximately 35km
				if !ognDecoderIsRunning || math.Abs(float64(mySituation.GPSLongitude-lastLon)) > 0.3 || math.Abs(float64(mySituation.GPSLatitude-lastLat)) > 0.3 {
					if globalSettings.DEBUG {
						if !ognDecoderIsRunning {
							log.Println("FLARM: Decoder is not running")
						} else {
							log.Println("FLARM: Own position has changed")
						}

						log.Println("FLARM: Restarting decoder")
					}

					// generate OGN configuration file
					OGNConfigDataCache.Longitude = fmt.Sprintf("%.4f", mySituation.GPSLongitude)
					OGNConfigDataCache.Latitude = fmt.Sprintf("%.4f", mySituation.GPSLatitude)
					OGNConfigDataCache.Altitude = fmt.Sprintf("%.0f", convertFeetToMeters(mySituation.GPSAltitudeMSL))
					createOGNConfigFile(configTemplateFileName, configFileName)

					// start new decoding process
					decodingProcess = replaceFlarmDecodingProcess(mySituation.GPSLongitude, mySituation.GPSLatitude, decodingProcess, configFileName)

					// start JSON decoding goroutine
					// go decodeFlarmJSONData(decoderOutput)

					// store current location
					lastLon = mySituation.GPSLongitude
					lastLat = mySituation.GPSLatitude
				}
			}
		}
	}
}
