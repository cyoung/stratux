package main

import (
	"../mpu6050"
	"fmt"
	"net"
	"time"
)

var attSensor mpu.MPU

func readMPU6050() (float64, float64) {
	pitch, _ := attSensor.Pitch()
	roll, _ := attSensor.Roll()

	return pitch, roll
}

func initMPU6050() {
	attSensor = mpu.NewMPU6050()
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
		heading, _ := attSensor.Heading()
		s := fmt.Sprintf("XATTMy Sim,%f,%f,%f", heading, pitch, roll)
		fmt.Printf("%f, %f\n", pitch, roll)
		outConn.Write([]byte(s))
		time.Sleep(50 * time.Millisecond)
	}
}
