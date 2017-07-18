/*
	Copyright (c) 2017 Thorsten Biermann
	Distributable under the terms of The "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	xplane.go: Routines for generating X-Plane data feed
*/

package main

import "fmt"

func createXPlaneGpsMsg(latDeg float32, lonDeg float32, altMslFt float32) []byte {
	// example: XGPS1,-122.298432,47.450756,420.9961,349.7547,57.9145
	// TODO find out what the remaining parameters are for
	// could be: track (in degrees), speed (in ?)
	return []byte(fmt.Sprintf("XGPS1,%.6f,%.6f,%.4f,%.4f,%.4f", lonDeg, latDeg, convertFeetToMeters(altMslFt), 0.0, 0.0))
}

func createXPlaneAttitudeMsg(headingDeg float32, pitchDeg float32, rollDeg float32) []byte {
	// example: XATT1,345.1,-1.1,-12.5,0.1374,0.0954,-0.0444,-17.0,-1.2,-65.0,-0.01,1.63,0.02
	// TODO find out what the remaining parameters are for
	return []byte(fmt.Sprintf("XATT1,%.1f,%.1f,%.1f,%.4f,%.4f,%.4f,%.1f,%.1f,%.1f,%.2f,%.2f,%.2f", headingDeg, pitchDeg, rollDeg, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0))
}

func createXPlaneTrafficMsg(targetId uint32, latDeg float32, lonDeg float32, altFt float32, callSign string) []byte {
	// example: XTRA1,1,47.435484,-122.304048,351,1,0,62,0,N172SP
	// TODO find out what the remaining parameters are for
	// could be: vertical speed (in ?), unknown, unknown, horizontal speed (in ?)
	return []byte(fmt.Sprintf("XTRA1,%d,%.6f,%.6f,%d,%d,%d,%d,%d,%s", targetId, latDeg, lonDeg, int64(convertFeetToMeters(altFt)), 0, 0, 0, 0, callSign))
}
