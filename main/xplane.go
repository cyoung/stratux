/*
	Copyright (c) 2017 Thorsten Biermann
	Distributable under the terms of The "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	xplane.go: Routines for generating X-Plane data feed
*/

package main

import "fmt"

func convertFeetToMeters(feet float32) float32 {
	return feet * 0.3048
}

func createXPlaneGpsMsg(latDeg float32, lonDeg float32, altMslFt float32) []byte {
	// TODO find out what the remaining parameters are for
	return []byte(fmt.Sprintf("XGPS1,%.6f,%.6f,%.4f,%.4f,%.4f", lonDeg, latDeg, convertFeetToMeters(altMslFt), 0.0, 0.0))
}

func createXPlaneAttitudeMsg(headingDeg float32, pitchDeg float32, rollDeg float32) []byte {
	// TODO find out what the remaining parameters are for
	return []byte(fmt.Sprintf("XATT1,%.1f,%.1f,%.1f,%.4f,%.4f,%.4f,%.1f,%.1f,%.1f,%.2f,%.2f,%.2f", headingDeg, pitchDeg, rollDeg, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0))
}
