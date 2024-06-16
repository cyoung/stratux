/*
	Copyright (c) 2024 Adrian Batzill
	Distributable under the terms of The "BSD New"" License
	that can be found in the LICENSE file, herein included
	as part of this header.
	flarm-nmea.go: Functions for generating FLARM-related NMEA sentences
		to communicate traffic bearing / distance to glider computers
		and UK / EU oriented EFBs.
	Accept cursor on target events on UDP port 8087
*/

package main

import (
	"encoding/json"
	"encoding/xml"
	"hash/fnv"
	"log"
	"net"
	"strings"
	"time"
)

type CotEvent struct {
	Version string  `xml:"version,attr"`
	Uid     string  `xml:"uid,attr"`
	Type_   string  `xml:"type,attr"`
	Time    string  `xml:"time,attr"`
	Start   string  `xml:"start,attr"`
	Stale   string  `xml:"stale,attr"`
	Point   CotPoint `xml:"point"`
	Detail  CotDetail `xml:"detail"`
}

type CotPoint struct {
	Lat float32 `xml:"lat,attr"`
	Lon float32 `xml:"lon,attr"`
	Hae float32 `xml:"hae,attr"`
	//ce float32  `xml:"ce,attr"`
	//ce float32  `xml:"le,attr"`
}

type CotDetail struct {
	Track CotTrack `xml:"track"`
}

type CotTrack struct {
	Speed float32 `xml:"speed,attr"`
	Course float32 `xml:"course,attr"`
}


func cotListen() {
	server, err := net.ListenPacket("udp", ":8087")
	if err != nil {
		log.Printf("cotListen() failed to listen on udp:8087. cot support disabled")
		return
	}
	messageBuf := ""
	buf := make([]byte, 2048)
	for {
		n, _, err := server.ReadFrom(buf)
		if err != nil || n == 0{
			continue
		}
		messageBuf += string(buf[0:n])
		startIndex := strings.Index(messageBuf, "<event")
		endIndex := strings.Index(messageBuf, "</event>")
		if startIndex >= 0 && endIndex > 0 {
			msg := messageBuf[startIndex:endIndex+8]
			processCotMessage(msg)
			messageBuf = messageBuf[endIndex+8:]
		}

	}
}

func processCotMessage(msg string) {
	var event CotEvent
	err := xml.Unmarshal([]byte(msg), &event)
	if err != nil {
		log.Printf("Failed to parse COT event: " + msg)
		return
	}
	if event.Point.Lat == 0 && event.Point.Lon == 0 {
		return
	}

	// Ugly.. we just use the lower 24 bit of the hash of the uid to generate an address
	hasher := fnv.New32a()
	hasher.Write([]byte(event.Uid))
	addr := hasher.Sum32()
	addr = addr & 0x00FFFFFF
	key := 1 << 24 | addr // mark as non-icao


	var ti TrafficInfo

	trafficMutex.Lock()
	defer trafficMutex.Unlock()

	
	if existingTi, ok := traffic[key]; ok {
		ti = existingTi
	}
	ti.Addr_type = 1
	ti.Icao_addr = addr
	ti.Timestamp = time.Now()
	ti.Lat = event.Point.Lat
	ti.Lng = event.Point.Lon
	ti.Position_valid = true
	ti.Reg = event.Uid
	ti.Tail = event.Uid
	ti.Last_source = TRAFFIC_SOURCE_OGN // TODO: properly implement TRAFFIC_SOURCE_COT
	ti.Age = 0
	ti.AgeLastAlt = 0
	ti.Last_seen = stratuxClock.Time
	ti.Speed = uint16(event.Detail.Track.Speed * 1.94384449) // m/s to kts
	ti.Speed_valid = ti.Speed != 0
	ti.Track = event.Detail.Track.Course

	// convert altitudes..
	alt := event.Point.Hae * 3.28084 // to feet
	if isGPSValid() && isTempPressValid() {
		ti.Alt = int32(alt - mySituation.GPSAltitudeMSL + mySituation.BaroPressureAltitude)
		ti.AltIsGNSS = false
	} else {
		// Fall back to GNSS alt
		ti.Alt = int32(alt)
		ti.AltIsGNSS = true
	}



	traffic[key] = ti

	postProcessTraffic(&ti)
	registerTrafficUpdate(ti)
	seenTraffic[key] = true

	if globalSettings.DEBUG {
		txt, _ := json.Marshal(ti)
		log.Printf("COT traffic imported: " + string(txt))
	}
}