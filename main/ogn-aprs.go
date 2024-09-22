/*
	Copyright (c) 2022 Quentin Bossard
	Distributable under the terms of The "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	aprs.go: Routines for reading traffic from aprs
*/

package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var aprsOutgoingMsgChan chan string = make(chan string, 100)
var aprsIncomingMsgChan chan string = make(chan string, 100)
var aprsExitChan chan bool = make(chan bool, 1)


var aprsRegex = regexp.MustCompile(
	`(?P<protocol>ICA|FLR|SKY|PAW|OGN|RND|FMT|MTK|XCG|FAN|FNT)(?P<id>[\dA-Z]{6})>` +        // protocol, id
		`[A-Z]+,qAS,([\d\w]+):[\/]` +                                                       //
		`(?P<time>\d{6})h(?P<longitude>\d*\.?\d*[NS])[\/\\](?P<lattitude>\d*\.?\d*[EW])` +  // time, lon, lat
		`\D` +                                                                              // sep
		`((?P<track>\d{3})\/(?P<speed>\d{3})\/A=(?P<altitude>\d*))*` +                      // optional track, speed, alt
		`(\s!W(?P<lonlatprecision>\d+)!\s)*` +                                              // optional lon lat precision
		`(id(?P<id>[\dA-F]{8}))*`)                                                          // optional id


func authenticate(c net.Conn) {
	filter := ""
	if mySituation.GPSFixQuality > 0 {
		filter = fmt.Sprintf(
			"filter r/%.7f/%.7f/%d\r\n", 
			mySituation.GPSLatitude, mySituation.GPSLongitude, 
			globalSettings.RadarRange*2)   // RadarRange is an int in NM, APRS wants an int in km and 2~=1.852
	}
	auth := fmt.Sprintf("user OGNNOCALL pass -1 vers stratux %s %s\r\n",  globalStatus.Version, filter)
	log.Printf(auth)
	fmt.Fprintf(c, auth)
}

func keepalive(c net.Conn) {
	go func() {
		ticker := time.NewTicker(240 * time.Second)
		for t := range ticker.C {
			if globalStatus.APRS_connected {
				fmt.Fprintf(c, "# stratux keepalive %s\n", t)
			} else {
				break
			}
		}
	}()
}

func updateFilter(c net.Conn) {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			if globalStatus.APRS_connected {
				if mySituation.GPSFixQuality > 0 {
					filter := fmt.Sprintf(
						"#filter r/%.7f/%.7f/%d\r\n", 
						mySituation.GPSLatitude, mySituation.GPSLongitude, 
						globalSettings.RadarRange*2)  // RadarRange is an int in NM, APRS wants an int in km and 2~=1.852
					fmt.Fprintf(c, filter)
				}
			} else {
				break
			}
		}
	}()
}

func aprsListen() {
	for {
		if !globalSettings.APRS_Enabled || !isGPSValid() {
			// wait until APRS is enabled
			time.Sleep(1 * time.Second)
			continue
		}
		if globalSettings.DEBUG {
			log.Printf("aprs connecting...")
		}
		conn, err := net.Dial("tcp", "aprs.glidernet.org:14580")
		if err != nil { // Local connection failed.
			time.Sleep(3 * time.Second)
			continue
		}
		authenticate(conn)
		keepalive(conn)
		updateFilter(conn)

		aprsReader := bufio.NewReader(conn)

		log.Printf("APRS successfully connected")
		globalStatus.APRS_connected = true

		// Make sure the exit channel is empty, so we don't exit immediately
		for len(aprsExitChan) > 0 {
			<-aprsExitChan
		}

		go func() {
			scanner := bufio.NewScanner(aprsReader)
			for scanner.Scan() {
				var temp string = scanner.Text()
				select {
				case aprsIncomingMsgChan <- temp: // Put in the channel unless it is full
				default:
					if globalSettings.DEBUG {
						log.Println("aprsIncomingMsgChan full. Discarding " + temp)
					}
				}
			}
			if scanner.Err() != nil {
				log.Printf("APRS issue: " + scanner.Err().Error())
			}
			aprsExitChan <- true
		}()

	loop:
		for globalSettings.APRS_Enabled {
			select {
			case data := <-aprsIncomingMsgChan:
				TraceLog.Record(CONTEXT_APRS, []byte(data))
				parseAprsMessage(data, false)

			case <-aprsExitChan:
				break loop
			}
		}
		globalStatus.APRS_connected = false
		log.Printf("closing connection")
		conn.Close()
		time.Sleep(3 * time.Second)
	}
}




func parseAprsMessage(data string, fakeCurrentTime bool) {

	if globalSettings.DEBUG {
		log.Printf("%+v\n", data)
	}

	// APRS,qAS: aircraft beacon
	// APRS,TCPIP*,qAC: ground station beacon
	res := aprsRegex.FindStringSubmatch(data)
	if res == nil { // no match
		if strings.Contains(data, "TCPIP*") {
			// log.Printf("GW data: " + data)
		} else {
			if globalSettings.DEBUG {
				log.Printf("No match for: " + data)
			}
		}
		return
	} else if len(res) < 15 { // too few captures
		log.Printf("Invalid APRS data format: " + data)
	} else if len(res[14]) > 0 {
		ts := time.Now().UTC()
		hh, _ := strconv.ParseInt(res[4][:2], 10, 8)
		mm, _ := strconv.ParseInt(res[4][2:4], 10, 8)
		ss, err := strconv.ParseInt(res[4][4:], 10, 8)
		if err != nil {
			return
		}
		ts = time.Date(ts.Year(), ts.Month(), ts.Day(), int(hh), int(mm), int(ss), 0, time.UTC)


		lat, err := strconv.ParseFloat(res[5][:2], 64)
		if err != nil {
			return
		}
		lat_m, err := strconv.ParseFloat(res[5][2:len(res[5])-1], 64)
		if err != nil {
			return
		}
		lat_m3d, err := strconv.ParseFloat(res[12][:1], 64)
		if err != nil {
			return
		}
		if strings.Contains(res[5], "S") {
			lat = -lat
		}
		lon, err := strconv.ParseFloat(res[6][:3], 64)
		if err != nil {
			return
		}
		lon_m, err := strconv.ParseFloat(res[6][3:len(res[6])-1], 64)
		if err != nil {
			return
		}
		lon_m3d, err := strconv.ParseFloat(res[12][1:], 64)
		if err != nil {
			return
		}
		if strings.Contains(res[6], "W") {
			lon = -lon
		}

		track, err := strconv.ParseFloat(res[8], 64)
		if err != nil {
			return
		}
		speed, err := strconv.ParseFloat(res[9], 64)
		if err != nil {
			return
		}
		alt, err := strconv.ParseFloat(res[10], 64)
		if err != nil {
			return
		}

		details, err := hex.DecodeString(res[14][:2])
		if err != nil {
			log.Fatal(err)
		}
		detail_byte := details[0]
		addr_type := detail_byte & 0b00000011
		acft_type := (detail_byte & 0b00111100) >> 2

		msg := OgnMessage{
			Sys:       res[1],
			Time:      float64(ts.Unix()),
			Addr:      res[2],
			Addr_type: int32(addr_type),
			Acft_type: fmt.Sprintf("%d", acft_type),
			Lat_deg:   float32(lat + lat_m/60 + lat_m3d/60000),
			Lon_deg:   float32(lon + lon_m/60 + lon_m3d/60000),
			Alt_msl_m: float32(alt * 0.3048),
			Track_deg: track,
			Speed_mps: speed * 0.514444}

		if globalSettings.DEBUG {
			// log.Printf("%+v\n", res)
			log.Printf("%+v\n", msg)
		}

		importOgnTrafficMessage(msg, data, fakeCurrentTime)
	}
}
