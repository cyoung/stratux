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
	"encoding/binary"
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

var ognUser = "969696"


func ognPass(user string) uint16 {
	hashb, _ := hex.DecodeString("73e2")
	userc := strings.ToUpper(user)
	userc = strings.Replace(userc, "-", "0", -1)
	for i := range userc {
		// log.Printf("%s", i)
		fmt.Printf("%c", userc[i])
		if i%2 == 0 {
			hashb[0] ^= userc[i]
		} else {
			hashb[1] ^= userc[i]
		}
	}
	hashb[0] &= 127
	hashb[1] &= 255
	return binary.BigEndian.Uint16(hashb)
}

func authenticate(c net.Conn) {
	// passwd := ognPass("OGN" + ognUser)
	passwd := -1
	filter := "r/48.8589465/2.2768241/500"
	// filter := "m/500"
	auth := fmt.Sprintf("user OGN%s pass %d vers stratux 0.28 filter %s\n", ognUser, passwd, filter)
	fmt.Printf(auth)
	fmt.Fprintf(c, auth)
}

func keepalive(c net.Conn) {
	ticker := time.NewTicker(10 * time.Second)

	go func() {
		for t := range ticker.C {
			fmt.Fprintf(c, "# stratux keepalive %s\n", t)
			fmt.Printf("# stratux keepalive %s\n", t)
		}
	}()
}

func sendPosition(c net.Conn) {
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for range ticker.C {
			fmt.Printf("%f %f %d\n", mySituation.GPSLatitude, mySituation.GPSLongitude, mySituation.GPSFixQuality)
			if true || mySituation.GPSFixQuality > 0  && globalSettings.OGNAddr != "" {
				// OGN123456>OGNAPP:/123456h5123.45N/00123.45E'180/025/A=001000 !W66! id07123456 +100fpm +1.0rot FL011.00 gps4x5
				position_report := fmt.Sprintf(
					"OGN%s>OGSTUX:/%sh4634.27N/00706.68E'%03.0f/%03.0f/A=%06.0f !W52! id07%s +%.0ffpm +1.0rot FL%03.2f gps%.0fx%.0f",
					// globalSettings.OGNAddr,
					ognUser,
					time.Now().UTC().Format("150405"),
					// mySituation.GPSLastFixLocalTime.UTC().Format("150405"),
					mySituation.GPSTrueCourse,
					mySituation.GPSGroundSpeed,
					mySituation.GPSAltitudeMSL,
					globalSettings.OGNAddr,
					mySituation.BaroVerticalSpeed,
					mySituation.BaroPressureAltitude/100,
					mySituation.GPSHorizontalAccuracy,
					mySituation.GPSVerticalAccuracy)
				fmt.Println(position_report)
				// fmt.Fprintf(c, position_report)
			}
		}
	}()
}

func aprsListen() {
	//go predTest()

	// https://regex101.com/r/Cv9mSq/1
	// rex := regexp.MustCompile(`(ICA|FLR|SKY|PAW|FNT)([\dA-Z]{6})>[A-Z]+,qAS,([\d\w]+):\/(\d{6})h(\d*\.?\d*[NS])[\/\\](\d*\.?\d*[EW])['\^nX](\d{3})\/(\d{3})\/A=\d*\s!W(\d+)!\sid([\dA-F]{8})`)
	rex := regexp.MustCompile(
		`(ICA|FLR|SKY|PAW|OGN|RND|FMT|MTK|XCG|FAN|FNT)([\dA-Z]{6})>` + // protocol, id
			// `(SKY)([\dA-Z]{6})>` + // protocol, id
			`[A-Z]+,qAS,([\d\w]+):[\/]` +
			`(\d{6})h(\d*\.?\d*[NS])[\/\\](\d*\.?\d*[EW])` + // time, lon, lat
			// `[#&0>AW\^_acnsuvz]` +
			`\D` + // sep
			`((\d{3})\/(\d{3})\/A=(\d*))*` + // optional track, speed, alt
			`(\s!W(\d+)!\s)*` + // optional lon lat precision
			`(id([\dA-F]{8}))*`) // optional id
	for {
		if !globalSettings.APRS_Enabled || OGNDev == nil {
			// wait until APRS is enabled
			time.Sleep(1 * time.Second)
			continue
		}
		log.Printf("aprs connecting...")
		conn, err := net.Dial("tcp", "aprs.glidernet.org:14580")
		if err != nil { // Local connection failed.
			time.Sleep(3 * time.Second)
			continue
		}
		log.Printf("APRS connected")
		authenticate(conn)
		log.Printf("APRS authentication sent...")
		keepalive(conn)
		sendPosition(conn)

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
				// log.Printf(temp)
				select {
				case aprsIncomingMsgChan <- temp: // Put in the channel unless it is full
				default:
					fmt.Println("aprsIncomingMsgChan full. Discarding " + temp)
				}
			}
			if scanner.Err() != nil {
				log.Printf("APRS issue: " + scanner.Err().Error())
			}
			aprsExitChan <- true
		}()

		pgrmzTimer := time.NewTicker(100 * time.Millisecond)

	loop:
		for globalSettings.APRS_Enabled {
			// log.Println(len(aprsIncomingMsgChan))
			select {
			case data := <-aprsIncomingMsgChan:
				// APRS,qAS: aircraft beacon
				// APRS,TCPIP*,qAC: ground station beacon

				res := rex.FindStringSubmatch(data)
				if res == nil { // no match
					if strings.Contains(data, "TCPIP*") {
						// log.Printf("GW data: " + data)
					} else {
						log.Printf("No match for: " + data)
					}
					continue
				} else if len(res) == 0 { // no group capture
					log.Printf("No group capture: " + data)
				} else if len(res) > 0 && len(res[14]) > 0 {
					// for i := range res {
					// 	fmt.Printf("%s(%d)|", res[i], len(res[i]))
					// }
					// fmt.Printf("\n")
					// fmt.Printf("%+v\n", res)

					lat, err := strconv.ParseFloat(res[5][:2], 64)
					if err != nil {
						log.Printf(err.Error())
					}
					lat_m, err := strconv.ParseFloat(res[5][2:len(res[5])-1], 64)
					if err != nil {
						log.Printf(err.Error())
					}
					lat_m3d, err := strconv.ParseFloat(res[12][:1], 64)
					if err != nil {
						log.Printf(err.Error())
					}
					if strings.Contains(res[5], "S") {
						lat = -lat
					}
					lon, err := strconv.ParseFloat(res[6][:3], 64)
					if err != nil {
						log.Printf(err.Error())
					}
					lon_m, err := strconv.ParseFloat(res[6][3:len(res[6])-1], 64)
					if err != nil {
						log.Printf(err.Error())
					}
					lon_m3d, err := strconv.ParseFloat(res[12][1:], 64)
					if err != nil {
						log.Printf(err.Error())
					}
					if strings.Contains(res[6], "W") {
						lon = -lon
					}
					// log.Printf(res[6][:2] + " " + res[6][2:len(res[6])-1] + " " + res[12][1:])

					track, err := strconv.ParseFloat(res[8], 64)
					if err != nil {
						log.Printf(err.Error())
					}
					speed, err := strconv.ParseFloat(res[9], 64)
					if err != nil {
						log.Printf(err.Error())
					}
					alt, err := strconv.ParseFloat(res[10], 64)
					if err != nil {
						log.Printf(err.Error())
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
						Time:      0,
						Addr:      res[2],
						Addr_type: int32(addr_type),
						Acft_type: fmt.Sprintf("%d", acft_type),
						Lat_deg:   float32(lat + lat_m/60 + lat_m3d/1000),
						Lon_deg:   float32(lon + lon_m/60 + lon_m3d/1000),
						Alt_msl_m: float32(alt * 0.3048),
						Track_deg: track,
						Speed_mps: speed * 0.514444}

					fmt.Printf("%+v\n", res)
					// fmt.Printf("%+v\n", msg)

					importOgnTrafficMessage(msg, data)
				}
			case <-pgrmzTimer.C:
				if isTempPressValid() && mySituation.BaroSourceType != BARO_TYPE_NONE && mySituation.BaroSourceType != BARO_TYPE_ADSBESTIMATE {
					select {
					case ognOutgoingMsgChan <- makePGRMZString(): // Put in the channel unless it is full
					default:
						// fmt.Println("ognOutgoingMsgChan full, discarding...")
					}
				}
			case <-aprsExitChan:
				break loop
				// default:
				// 	fmt.Println("nothing to do here")
			}
		}
		globalStatus.APRS_connected = false
		log.Printf("closing connection")
		conn.Close()
		time.Sleep(3 * time.Second)
	}
}
