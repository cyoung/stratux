/*
	Copyright (c) 2020 Adrian Batzill
	Distributable under the terms of The "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	ais.go: Routines for reading traffic from ais-rx-eu
*/

package main

import (
	"bufio"
	"encoding/json"
//	"io/ioutil"
	"log"
	"net"
	"strings"
	"time"
	"fmt"

	"github.com/b3nn0/stratux/common"

	"github.com/BertoldVdb/go-ais"
	"github.com/BertoldVdb/go-ais/aisnmea"
)


func aisPublishNmea(nmea string) {
	if globalStatus.AIS_connected {
		if !strings.HasSuffix(nmea, "\r\n") {
			nmea += "\r\n"
		}
		aisOutgoingMsgChan <- nmea
	}
}

var aisOutgoingMsgChan chan string = make(chan string, 100)
var aisIncomingMsgChan chan string = make(chan string, 100)
var aisExitChan chan bool = make(chan bool, 1)

func aisListen() {
	//go predTest()
	nm := aisnmea.NMEACodecNew(ais.CodecNew(false, false))
	for {
		if !globalSettings.AIS_Enabled || AISDev == nil {
			// wait until AIS is enabled
			time.Sleep(1 * time.Second)
			continue
		}
		// log.Printf("ais connecting...")
		aisAddr := "127.0.0.1:10111"
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
			<- aisExitChan
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

		loop: for globalSettings.AIS_Enabled {
			select {
			case data := <- aisOutgoingMsgChan:
				aisReadWriter.Write([]byte(data))
				aisReadWriter.Flush()
			case data := <- aisIncomingMsgChan:
				msg, err := nm.ParseSentence(data)

//				if msg.Sys == "status" {
//					importAISStatusMessage(msg)
//				} else {
					// TODO: RVT Renable
					//msgLogAppend(thisMsg)
					// TODO: RVT Renable
					//logMsg(thisMsg) // writes to replay logs
//				}
				if (msg!=nil) {
					importAISTrafficMessage(msg)
				} else if err!=nil {
					log.Printf("Invalid Data from AIS: " + err.Error())
				} else {
					// Multiline sentences will have msg as nill without err 
				}
			case <- aisExitChan:
				break loop

			}
		}
		globalStatus.AIS_connected = false
		conn.Close()
		time.Sleep(3*time.Second)
	}
}

// Update something....
// func importAISStatusMessage(msg AISMessage) {
// 	if msg.Tx_enabled {
// 		aisPublishNmea(getOgnTrackerConfigString())
// 	}
// }

func importAISTrafficMessage(msg *aisnmea.VdmPacket) {
	var ti TrafficInfo
	
	var header *ais.Header = msg.Packet.GetHeader()
	var key = header.UserID

	if existingTi, ok := traffic[key]; ok {
		ti = existingTi
	} else {		
		ti.Reg = fmt.Sprintf("%d", header.UserID)
	}
	
	trafficMutex.Lock()
	defer trafficMutex.Unlock()

	ti.Icao_addr = header.UserID

	// Binary Broadcast, we will ignore this
	// !AIVDM,3,1,8,A,8h30otA?0@<o;NPPPP<i>nskl4tSp1m>@00o;NPPPP<iCnsm<4tPG286@00o;NPP,0*3D
	// !AIVDM,3,2,8,A,PP<j>nsphTtHBR7@@00o;NPPPP<jCnssG4tCk2N0@00o;NPPPP<k>nsuPTt6m2Mt,0*44
    // !AIVDM,3,3,8,A,@00o;NPPPP<kCnsuwTt3?2lB@00,2*3D

	// Handle Ship Static Data
	if header.MessageID == 5 {
		var shipStaticData ais.ShipStaticData = msg.Packet.(ais.ShipStaticData);
		ti.Tail = strings.TrimSpace(shipStaticData.CallSign)

		// txt, _ := json.Marshal(shipStaticData)
		// log.Printf("ShipStaticData: " + string(txt))

		// https://www.navcen.uscg.gov/?pageName=AISMessagesAStatic
		ti.Emitter_category = shipStaticData.Type

		//log.Printf("ShipStatic: " + fmt.Sprintf("%d", header.UserID) + ":" + shipStaticData.Name)
	}

	// TODO: RVT further implement LongRangeAisBroadcastMessage ??
	if header.MessageID==27 {
		// var positionReport ais.LongRangeAisBroadcastMessage = msg.Packet.(ais.LongRangeAisBroadcastMessage);
	}

	// Handle MessageID 1,2 & 3 Position reports
	if header.MessageID==1 || header.MessageID==2 || header.MessageID==3 {
		// !AIVDM,1,1,,A,13aIhV?P140H?T@MVbNJVOvT00Ss,0*6D
		//log.Printf("ShipPosition: " + fmt.Sprintf("%d", header.UserID))

		var positionReport ais.PositionReport = msg.Packet.(ais.PositionReport);
	
	//	ti.Reg = ""
	//	ti.Tail // Set above
		ti.OnGround = true
		ti.Addr_type = uint8(1) // Non-ICAO Address
		ti.TargetType = TARGET_TYPE_AIS 
		ti.SignalLevel = 0.0
	//    ti.SignalLevelHist
		ti.Squawk = 0
		ti.Position_valid = true
		ti.Lat = float32(positionReport.Latitude)
		ti.Lng = float32(positionReport.Longitude)
		ti.Alt = 0 // pressure altitude
	//	ti.GnssDiffFromBaroAlt = 0
	//	ti.AltIsGNSS = 0
	//	ti.NIC = 0
	//	ti.NACp = 0

		var sog uint16 = 0
		if positionReport.Sog<102.3 {
			sog=uint16(positionReport.Sog)
		}
		ti.Speed = sog // I think Sog is in knt
		ti.Speed_valid = true

		// We assume that when we have speed, we also have a proper course.
		if (positionReport.Sog > 0.0) { // Using positionReport.Sog gives us more accuracy then using sog
			var cog float32 = 0.0
			if positionReport.Cog!=360 {
				cog=float32(positionReport.Cog)
			}
			ti.Track = cog
		} else {
			var heading float32 = 0.0
			if positionReport.TrueHeading!=511 {
				heading=float32(positionReport.TrueHeading)	
			}
			ti.Track = heading
		}

//		txt, _ := json.Marshal(positionReport)
//		log.Printf("Position report: " + string(txt))


		var rot float32 = 0.0
		if positionReport.RateOfTurn!=-128 {
			rot=float32(positionReport.RateOfTurn)
		}
		ti.TurnRate = (rot/4.733)*(rot/4.733)

		//	ti.Vvel = 0 
		ti.Timestamp = time.Now().UTC()	
	//	ti.PriorityStatus
		ti.Age = time.Now().UTC().Sub(ti.Timestamp).Seconds()
		//ti.AgeLastAlt = time.Now().UTC().Sub(ti.Timestamp).Seconds()
		ti.Last_seen = stratuxClock.Time
		ageMs := int64(ti.Age * 1000)

		ti.Last_seen = ti.Last_seen.Add(-time.Duration(ageMs) * time.Millisecond)
		ti.Last_alt = ti.Last_seen
		ti.Last_speed = ti.Last_seen
	
	//	ti.Last_GnssDiff        
	//	ti.Last_GnssDiffAlt     
		ti.Last_source = TRAFFIC_SOURCE_AIS
		ti.ExtrapolatedPosition = false
	//	ti.Last_extrapolation   time.Time
	//	ti.AgeExtrapolation     float64
	//	ti.Lat_fix              float32   // Last real, non-extrapolated latitude
	//	ti.Lng_fix              float32   // Last real, non-extrapolated longitude
	//	ti.Alt_fix              int32     // Last real, non-extrapolated altitude		return
	}
	//	DistanceEstimated    float64   // Estimated distance of the target if real distance can't be calculated, Estimated from signal strength with exponential smoothing.
	//	DistanceEstimatedLastTs time.Time // Used to compute moving average
	//	ReceivedMsgs         uint64    // Number of messages received by this aircraft

	// Sometimes there seems to be wildly invalid lat/lons, which can trip over distRect's normailization..
	if ti.Lat > 360 || ti.Lat < -360 || ti.Lng > 360 || ti.Lng < -360 {
		return
	}

	// Validate the position report 
	if isGPSValid() && (ti.Lat!=0 && ti.Lng!=0) {
		ti.Distance, ti.Bearing = common.Distance(float64(mySituation.GPSLatitude), float64(mySituation.GPSLongitude), float64(ti.Lat), float64(ti.Lng))
		ti.BearingDist_valid = true
	}
	
	if ti.TurnRate==-128 || ti.TurnRate > 1080 || ti.TurnRate < -1080 {
		ti.TurnRate = 0
	}

	// Basic plausibility check and ensure we do not overload you map
	if ti.BearingDist_valid == true && ti.Distance >= 150000 {
		// more than 150km away or invalid positions
		return
	}

	traffic[key] = ti
	postProcessTraffic(&ti) // This will not estimate distance for non ES sources, pffff
	registerTrafficUpdate(ti) // Sends this one to the web interface
	seenTraffic[key] = true	

	if globalSettings.DEBUG {
		txt, _ := json.Marshal(ti)
		log.Printf("AIS traffic imported: " + string(txt))
	}
}
