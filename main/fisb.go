/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New"" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	fisb.go: FISB service. Inputs raw UAT packets, parses, and creates a standard buffer to be saved for general purpose use.
*/

package main

import (
	"errors"
	"strings"
	"time"
	"uatparse"
)

/*
	Examples:

METAR KIND 261654Z 10005KT 10SM SCT250 11/02 A3015 RMK AO2 SLP212
    T01060017=
SPECI KTDZ 261738Z AUTO 04006KT 8SM SCT020 07/00 A3025 RMK AO2
    T00670000=
TAF KCMI 261131Z 2612/2712 12007KT P6SM SCT250
     FM261600 13013KT P6SM SCT100
     FM270000 15006KT P6SM BKN200=
TAF.AMD KMKG 261725Z 2617/2718 VRB05KT P6SM SCT250
     FM270100 11006KT P6SM BKN150
     FM271200 13008KT P6SM VCSH BKN150=
WINDS COU 271200Z  FT 3000 6000      9000   12000       18000   24000   30000    34000  39000
   2109 2019+05 2134+01 2347-04 2353-21 2454-33 257949 259857 257955
PIREP HNN 261618Z CRW UA /OV HNN/TM 1618/FL360/TP B737/TB OCNL LGT CHOP 360/RM OVER HNN AWC-WEB:SWA

*/

const (
	FISB_METAR = iota
	FISB_SPECI
	FISB_TAF
	FISB_TAFAMD
	FISB_WINDS
	FISB_PIREP
	FISB_NEXRAD //TODO. Not implemented.
)

type FISB struct {
	MessageType      int
	MessageData      string
	MessageTimestamp string
}

var uatIn chan string

var fisbBuffer map[int]map[string]FISB // fisbBuffer[messageType][identifier] = FISB

// Return (err, message type constant, identifier, timestamp-if-any).
func identifyMessage(s string) (error, int, string, string) {
	x := strings.Split(s, " ")
	if len(x) < 3 {
		return errors.New("unsupported text update format."), 0, nil, nil
	}
	switch x[0] {
	case "METAR":
		return nil, FISB_METAR, x[1], x[2]
	case "SPECI":
		return nil, FISB_SPECI, x[1], x[2]
	case "TAF":
		return nil, FISB_TAF, x[1], x[2]
	case "TAFAMD":
		return nil, FISB_TAFAMD, x[1], x[2]
	case "WINDS":
		return nil, FISB_WINDS, x[1], x[2]
	case "PIREP":
		return nil, FISB_PIREP, x[1] + " " + x[2], nil
	default:
		return errors.New("unknown type: " + s), 0, nil, nil
	}
	return errors.New("unknown type: " + s), 0, nil, nil
}

/*
	textReportTimestampParse().
		Parse timestamps of type '261654Z'.
*/

func textReportTimestampParse(t string) time.Time {
	if len(t) != 7 || t[6] != 'Z' {
		return time.Time{} //FIXME: Fails silently.
	}

	// Mon Jan 2 15:04:05 -0700 MST 2006.
	tT, err := time.Parse("021504", t[:6])
	if err != nil {
		return time.Time{} //FIXME: Fails silently.
	}
	return tT
}

/*
	textReportTimestampIsNewer().
		Compare timestamps of type '261654Z'.
*/
func textReportTimestampIsNewer(past, current string) bool {
	pastT := textReportTimestampParse(past)
	currentT := textReportTimestampParse(current)
	return currentT.After(pastT)
}

func fisbService() {
	uatIn = make(chan string, 10240)
	fisbBuffer = make(map[int]map[string]FISB)
	for {
		m := <-uatIn
		// Parse the message.
		uatMsg, err := uatparse.New(m)
		if err != nil {
			continue // Ignore if invalid.
		}
		// Decode uplink message. At this point, an error is thrown by uatparse if it's not an uplink message - no further checking necessary.
		uatMsg.DecodeUplink()

		for _, textReport := range uatMsg.GetTextReports() {
			err, msgType, ident, ts := identifyMessage(textReport)
			if err != nil {
				continue // If there was a parsing/identification error, just skip this message.
			}

			var thisFISB FISB
			thisFISB.MessageType = msgType
			thisFISB.MessageData = textReport
			thisFISB.MessageTimestamp = ts
			if curFISB, ok := fisbBuffer[msgType][ident]; !ok {
				// We haven't seen anything for this identifier yet. Store it and continue.
				fisbBuffer[msgType][ident] = thisFISB
			} else {
				// We already have received a report for this identifier. Check if this report is newer.
				if textReportTimestampIsNewer(curFISB.MessageTimestamp, ts) {
					// We've received a newer text report. Replace the current text report with this one.
					fisbBuffer[msgType][ident] = thisFISB
				}
			}
		}

	}
}
