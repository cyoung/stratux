package main

import (
	"fmt"
	"time"
)

func main() {
	initRY835AI()
	initMPU9250()

	for i := 0; i < 10; i++ {
		pitch, roll := GetCurrentAttitudeXY()
		fmt.Printf("%f,%f", pitch, roll)
		time.Sleep(100 * time.Millisecond)
	}
}
