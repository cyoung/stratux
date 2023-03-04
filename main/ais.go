/*
	Copyright (c) 2021 R. van Twisk
	Distributable under the terms of The "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	ais.go: Routines for reading AIS traffic
*/

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/b3nn0/stratux/common"

	"github.com/BertoldVdb/go-ais"
	"github.com/BertoldVdb/go-ais/aisnmea"
)

var aisIncomingMsgChan chan string = make(chan string, 100)
var aisExitChan chan bool = make(chan bool, 1)
var aisNmeaParser = aisnmea.NMEACodecNew(ais.CodecNew(false, false))

func aisListen() {
	//go predTest()
	for {
		if !globalSettings.AIS_Enabled || AISDev == nil {
			// wait until AIS is enabled
			time.Sleep(1 * time.Second)
			continue
		}
		// log.Printf("ais connecting...")
		aisAddr := "127.0.0.1:10110"
		conn, err := net.Dial("tcp", aisAddr)
		if err != nil { // Local connection failed.
			time.Sleep(3 * time.Second)
			continue
		}
		log.Printf("ais successfully connected")
		aisReadWriter := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
		globalStatus.AIS_connected = true

		// Make sure the exit channel is empty, so we don't exit immediately
		for len(aisExitChan) > 0 {
			<-aisExitChan
		}

		go func() {
			scanner := bufio.NewScanner(aisReadWriter.Reader)
			for scanner.Scan() {
				aisIncomingMsgChan <- scanner.Text()
			}
			if scanner.Err() != nil {
				log.Printf("ais-rx-eu connection lost: " + scanner.Err().Error())
			}
			aisExitChan <- true
		}()

	loop:
		for globalSettings.AIS_Enabled {
			select {
			case data := <-aisIncomingMsgChan:
				TraceLog.Record(CONTEXT_AIS, []byte(data))
				parseAisMessage(data)
			case <-aisExitChan:
				break loop

			}
		}
		globalStatus.AIS_connected = false
		conn.Close()
		time.Sleep(3 * time.Second)
	}
}

func parseAisMessage(data string) {
	var thisMsg msg
	thisMsg.MessageClass = MSGCLASS_AIS
	thisMsg.TimeReceived = stratuxClock.Time
	thisMsg.Data = data
	msgLogAppend(thisMsg)
	logMsg(thisMsg) // writes to replay logs

	msg, err := aisNmeaParser.ParseSentence(data)
	if err == nil && msg != nil && msg.Packet != nil {
		importAISTrafficMessage(msg)
	} else if err != nil {
		log.Printf("Invalid Data from AIS: " + err.Error())
	} else {
		// Multiline sentences will have msg as nill without err
	}
}

// Datastructure for AIS parsing 
// explanation can be found at https://www.navcen.uscg.gov/?pageName=AISMessages
func importAISTrafficMessage(msg *aisnmea.VdmPacket) {
	var ti TrafficInfo

	var header *ais.Header = msg.Packet.GetHeader()
	var key = header.UserID

	trafficMutex.Lock()
	defer trafficMutex.Unlock()

	if existingTi, ok := traffic[key]; ok {
		ti = existingTi
	} else {
		ti.Reg = fmt.Sprintf("%d", header.UserID)
		ti.Emitter_category = 18 // Ground Vehicle, see also gdl90EmitterCatToNMEA
		ti.TargetType = TARGET_TYPE_AIS
		ti.Last_source = TRAFFIC_SOURCE_AIS
		ti.Alt = 0
		ti.Addr_type = uint8(1) // Non-ICAO Address
		ti.SignalLevel = 0.0
		ti.Squawk = 0
		ti.AltIsGNSS = false
		ti.GnssDiffFromBaroAlt = 0
		ti.NIC = 0
		ti.NACp = 0
		ti.Vvel = 0
		ti.PriorityStatus = 0

		ti.Age = 0
		ti.AgeLastAlt = 0
	}

	ti.Icao_addr = header.UserID
	ti.Timestamp = time.Now().UTC()
	ti.Last_seen = stratuxClock.Time
	ti.Last_alt = stratuxClock.Time

	// Handle ShipStaticData
	if header.MessageID == 5 {
		var shipStaticData ais.ShipStaticData = msg.Packet.(ais.ShipStaticData)

		ti.Tail = strings.TrimSpace(shipStaticData.Name)
		ti.Reg = strings.TrimSpace(shipStaticData.CallSign)
		ti.SurfaceVehicleType = uint16(shipStaticData.Type)
		// Store in case this was the first message and we disgard die to GPS not available 
		traffic[key] = ti
	}

	// Handle LongRangeAisBroadcastMessage
	if header.MessageID == 27 {
		var positionReport ais.LongRangeAisBroadcastMessage = msg.Packet.(ais.LongRangeAisBroadcastMessage)

		ti.Lat = float32(positionReport.Latitude)
		ti.Lng = float32(positionReport.Longitude)

		if positionReport.Cog != 511 {
			cog := float32(positionReport.Cog)
			ti.Track = cog
		}
		if positionReport.Sog < 63 {
			ti.Speed = uint16(positionReport.Sog)
			ti.Speed_valid = true
		}
	}

	// Handle MessageID 1,2 & 3 Position reports
	if header.MessageID == 1 || header.MessageID == 2 || header.MessageID == 3 {
		var positionReport ais.PositionReport = msg.Packet.(ais.PositionReport)

		ti.OnGround = true
		ti.Position_valid = true
		ti.Lat = float32(positionReport.Latitude)
		ti.Lng = float32(positionReport.Longitude)

		if positionReport.Sog < 102.3 {
			ti.Speed = uint16(positionReport.Sog) // I think Sog is in knt
			ti.Speed_valid = true
			ti.Last_speed = ti.Last_seen
		}

		// We assume that when we have speed, 
		// we also have a proper course over ground so we take that over heading.
		// Otherwise Track will be heading so boats will orient correctly
		if positionReport.Sog > 0.0 && positionReport.Sog < 102.3 {
			var cog float32 = 0.0
			if positionReport.Cog != 360 {
				cog = float32(positionReport.Cog)
			}
			ti.Track = cog
		} else {
			var heading float32 = 0.0
			if positionReport.TrueHeading != 511 {
				heading = float32(positionReport.TrueHeading)
			}
			ti.Track = heading
		}

		var rot float32 = 0.0
		if positionReport.RateOfTurn != -128 {
			rot = float32(positionReport.RateOfTurn)
		}
		ti.TurnRate = (rot / 4.733) * (rot / 4.733)

		ti.ExtrapolatedPosition = false
	}

	// Prevent wild lat/long coordinates
	if ti.Lat > 360 || ti.Lat < -360 || ti.Lng > 360 || ti.Lng < -360 {
		return
	}

	// Validate the position report
	if isGPSValid() && (ti.Lat != 0 && ti.Lng != 0) {
		ti.Distance, ti.Bearing = common.Distance(float64(mySituation.GPSLatitude), float64(mySituation.GPSLongitude), float64(ti.Lat), float64(ti.Lng))
		ti.BearingDist_valid = true
	}

	// Do not display targets more than 150km
	if ti.BearingDist_valid == false || ti.Distance >= 150000 {
		return
	}

	traffic[key] = ti
	postProcessTraffic(&ti)   // This will not estimate distance for non ES sources, pffff
	registerTrafficUpdate(ti) // Sends this one to the web interface
	seenTraffic[key] = true

	if globalSettings.DEBUG {
		txt, _ := json.Marshal(ti)
		log.Printf("AIS traffic imported: " + string(txt))
	}
}
