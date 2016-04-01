/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New"" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	gen_gdl90.go: Input demodulated UAT and 1090ES information, output GDL90. Heartbeat,
	 ownship, status messages, stats collection.
*/

package main

import (
	"../mpu6050"
	"fmt"
	"net"
	"time"
)

var attSensor *mpu6050.MPU6050

func readMPU6050() (float64, float64) {
	pitch, roll := attSensor.PitchAndRoll()

	return pitch, roll
}

func initMPU6050() {
	attSensor = mpu6050.New()
}

func main() {
	initMPU6050()
	addr, err := net.ResolveUDPAddr("udp", "192.168.1.255:49002")
	if err != nil {
		panic(err)
	}
	outConn, err := net.DialUDP("udp", nil, addr)
	for {
		pitch, roll := readMPU6050()
		s := fmt.Sprintf("XATTMy Sim,%f,%f,%f", attSensor.Heading(), pitch, roll)
		fmt.Printf("%f, %f\n", pitch, roll)
		outConn.Write([]byte(s))
		time.Sleep(50 * time.Millisecond)
	}
}
