package main

import (
	"errors"
	"strings"
	"time"
	"uatparse"
)

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
