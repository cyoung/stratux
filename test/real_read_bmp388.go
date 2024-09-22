package main

import (
	"fmt"
	"time"

	"github.com/b3nn0/stratux/sensors/bmp388"
	"github.com/kidoman/embd"
	_ "github.com/kidoman/embd/host/all"
)

func main() {

	i2cbus := embd.NewI2CBus(1)
	bmp := bmp388.BMP388{Bus: &i2cbus, Address: bmp388.Address}
	bmp.Configure(bmp.Config)

	fmt.Println("t,temp,press,alt")

	clock := time.NewTicker(time.Millisecond)
	for {
		for _ = range clock.C {

			p, _ := bmp.ReadPressure()
			t, _ := bmp.ReadTemperature()
			fmt.Printf("%3.2f,%4.2f\n", p, t)
		}

	}
}
