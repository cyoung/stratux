/*
	Copyright (c) 2016 AvSquirrel (https://github.com/AvSquirrel)
	Distributable under the terms of the "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	equations.go: Math and statistics library used to support AHRS
         and other fuctions of Stratux package
*/

package main

import (
	"fmt"
	"math"
)

// linReg calculates slope and intercept for a least squares linear regression of y[] vs x[]
// Returns error if fewer than two data points in each series, or if series lengths are different

func linReg(x, y []float64) (slope, intercept float64, valid bool) {

	n := len(x)
	nf := float64(n)

	if n != len(y) {
		fmt.Printf("linReg: Lengths not equal\n")
		return math.NaN(), math.NaN(), false
	}

	if n < 2 {
		fmt.Printf("linReg: Lengths too short\n")
		return math.NaN(), math.NaN(), false
	}

	var Sx, Sy, Sxx, Sxy, Syy float64

	for i := range x {
		Sx += x[i]
		Sy += y[i]
		Sxx += x[i] * x[i]
		Sxy += x[i] * y[i]
		Syy += y[i] * y[i]
	}

	if nf*Sxx == Sx*Sx {
		fmt.Printf("linReg: Infinite slope\n")
		return math.NaN(), math.NaN(), false
	}

	// Calculate slope and intercept
	slope = (nf*Sxy - Sx*Sy) / (nf*Sxx - Sx*Sx)
	intercept = Sy/nf - slope*Sx/nf
	valid = true
	return
}

// linRegWeighted calculates slope and intercept for a weighted least squares
// linear regression of y[] vs x[], given weights w[] for each point.
// Returns error if fewer than two data points in each series, if series lengths are different,
// if weights sum to zero, or if slope is infinite

func linRegWeighted(x, y, w []float64) (slope, intercept float64, valid bool) {

	n := len(x)

	if n != len(y) || n != len(w) {
		fmt.Printf("linRegWeighted: Lengths not equal\n")
		return math.NaN(), math.NaN(), false
	}

	if n < 2 {
		fmt.Printf("linRegWeighted: Lengths too short\n")
		return math.NaN(), math.NaN(), false
	}

	//var Sx, Sy, Sxx, Sxy, Syy float64
	var Sw, Swx, Swy, Swxx, Swxy, Swyy float64

	for i := range x {
		Sw += w[i]
		Swxy += w[i] * x[i] * y[i]
		Swx += w[i] * x[i]
		Swy += w[i] * y[i]
		Swxx += w[i] * x[i] * x[i]
		Swyy += w[i] * y[i] * y[i]
		/*
			Sx += x[i]
			Sy += y[i]
			Sxx += x[i]*x[i]
			Sxy += x[i]*y[i]
			Syy += y[i]*y[i]
		*/
	}

	if Sw == 0 {
		fmt.Printf("linRegWeighted: Sum of weights is zero\n")
		return math.NaN(), math.NaN(), false
	}

	if Sw*Swxx == Swx*Swx {
		fmt.Printf("linRegWeighted: Infinite slope\n")
		return math.NaN(), math.NaN(), false
	}

	// Calculate slope and intercept
	slope = (Sw*Swxy - Swx*Swy) / (Sw*Swxx - Swx*Swx)
	intercept = Swy/Sw - slope*Swx/Sw
	valid = true
	return
}

// triCubeWeight returns the value of the tricube weight function
// at point x, for the given center and halfwidth.
func triCubeWeight(center, halfwidth, x float64) float64 {
	var weight, x_t float64
	x_t = math.Abs((x - center) / halfwidth)
	if x_t < 1 {
		weight = math.Pow((1 - math.Pow(x_t, 3)), 3)
	} else {
		weight = 0
	}
	return weight
}

// arrayMin calculates the minimum value in array x
func arrayMin(x []float64) (float64, bool) {
	if len(x) < 1 {
		fmt.Printf("arrayMin: Length too short\n")
		return math.NaN(), false
	}

	min := x[0]
	for i := range x {
		if x[i] < min {
			min = x[i]
		}
	}
	return min, true
}

// arrayMax calculates the maximum value in array x
func arrayMax(x []float64) (float64, bool) {
	if len(x) < 1 {
		fmt.Printf("arrayMax: Length too short\n")
		return math.NaN(), false
	}

	max := x[0]
	for i := range x {
		if x[i] > max {
			max = x[i]
		}
	}
	return max, true
}

// arrayRange calculates the range of values in array x
func arrayRange(x []float64) (float64, bool) {
	max, err1 := arrayMax(x)
	min, err2 := arrayMin(x)

	if !err1 || !err2 {
		fmt.Printf("Error calculating range\n")
		return math.NaN(), false
	}

	return (max - min), true
}

// mean returns the arithmetic mean of array x
func mean(x []float64) (float64, bool) {
	if len(x) < 1 {
		fmt.Printf("mean: Length too short\n")
		return math.NaN(), false
	}

	sum := 0.0
	nf := float64(len(x))

	for i := range x {
		sum += x[i]
	}

	return sum / nf, true
}

// stdev estimates the sample standard deviation of array x
func stdev(x []float64) (float64, bool) {
	if len(x) < 2 {
		fmt.Printf("stdev: Length too short\n")
		return math.NaN(), false
	}

	nf := float64(len(x))
	xbar, xbarValid := mean(x)

	if !xbarValid {
		fmt.Printf("stdev: Error calculating xbar\n")
		return math.NaN(), false
	}

	sumsq := 0.0

	for i := range x {
		sumsq += (x[i] - xbar) * (x[i] - xbar)
	}

	return math.Pow(sumsq/(nf-1), 0.5), true
}

// radians converts angle from degrees, and returns its value in radians
func radians(angle float64) float64 {
	return angle * math.Pi / 180.0
}

// degrees converts angle from radians, and returns its value in degrees
func degrees(angle float64) float64 {
	return angle * 180.0 / math.Pi
}

// radiansRel converts angle from degrees, and returns its value in radians in the range -Pi to + Pi
func radiansRel(angle float64) float64 {
	for angle > 180 {
		angle -= 360
	}
	for angle < -180 {
		angle += 360
	}
	return angle * math.Pi / 180.0
}

// degreesRel converts angle from radians, and returns its value in the range of -180 to +180 degrees
func degreesRel(angle float64) float64 {
	for angle > math.Pi {
		angle -= 2 * math.Pi
	}
	for angle < -math.Pi {
		angle += 2 * math.Pi
	}
	return angle * 180.0 / math.Pi
}

// degreesHdg converts angle from radians, and returns its value in the range of 0+ to 360 degrees
func degreesHdg(angle float64) float64 {
	for angle < 0 {
		angle += 2 * math.Pi
	}
	return angle * 180.0 / math.Pi
}

/*
Distance functions based on rectangular coordinate systems
Simple calculations and "good enough" on small scale (± 1° of lat / lon)
suitable for relative distance to nearby traffic
*/

// distRect returns distance and bearing to target #2 (e.g. traffic) from target #1 (e.g. ownship)
// Inputs are lat / lon of both points in decimal degrees
// Outputs are distance in meters and bearing in degrees (0° = north, 90° = east)
// Secondary outputs are north and east components of distance in meters (north, east positive)

func distRect(lat1, lon1, lat2, lon2 float64) (dist, bearing, distN, distE float64) {
	radius_earth := 6371008.8 // meters; mean radius
	dLat := radiansRel(lat2 - lat1)
	avgLat := radiansRel((lat2 + lat1) / 2)
	dLon := radiansRel(lon2 - lon1)
	distN = dLat * radius_earth
	distE = dLon * radius_earth * math.Abs(math.Cos(avgLat))
	dist = math.Pow(distN*distN+distE*distE, 0.5)
	bearing = math.Atan2(distE, distN)
	bearing = degreesHdg(bearing)
	return
}

// distRectNorth returns north-south distance from point 1 to point 2.
// Inputs are lat in decimal degrees. Output is distance in meters (east positive)
func distRectNorth(lat1, lat2 float64) float64 {
	var dist float64
	radius_earth := 6371008.8 // meters; mean radius
	dLat := radiansRel(lat2 - lat1)
	dist = dLat * radius_earth
	return dist
}

// distRectEast returns east-west distance from point 1 to point 2.
// Inputs are lat/lon in decimal degrees. Output is distance in meters (north positive)
func distRectEast(lat1, lon1, lat2, lon2 float64) float64 {
	var dist float64
	radius_earth := 6371008.8 // meters; mean radius
	//dLat := radiansRel(lat2 - lat1) // unused
	avgLat := radiansRel((lat2 + lat1) / 2)
	dLon := radiansRel(lon2 - lon1)
	dist = dLon * radius_earth * math.Abs(math.Cos(avgLat))
	return dist
}

/*
Distance functions: Polar coordinate systems
More accurate over longer distances
*/

// distance calculates distance between two points using the law of cosines.
// Inputs are lat / lon of both points in decimal degrees
// Outputs are distance in meters and bearing to the target from origin in degrees (0° = north, 90° = east)
func distance(lat1, lon1, lat2, lon2 float64) (dist, bearing float64) {
	radius_earth := 6371008.8 // meters; mean radius

	lat1 = radians(lat1)
	lon1 = radians(lon1)
	lat2 = radians(lat2)
	lon2 = radians(lon2)

	dist = math.Acos(math.Sin(lat1)*math.Sin(lat2)+math.Cos(lat1)*math.Cos(lat2)*math.Cos(lon2-lon1)) * radius_earth

	var x, y float64

	x = math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(lon2-lon1)
	y = math.Sin(lon2-lon1) * math.Cos(lat2)

	bearing = degreesHdg(math.Atan2(y, x))

	return
}
