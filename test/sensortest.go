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
