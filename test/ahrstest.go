package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

type SituationUpdate struct {
	pitch   float64
	roll    float64
	heading float64
}

var updateChan chan SituationUpdate

func updateSender(addr string) {
	updateChan = make(chan SituationUpdate)

	ipAndPort := addr + ":49002"
	udpaddr, err := net.ResolveUDPAddr("udp", ipAndPort)
	if err != nil {
		fmt.Printf("ResolveUDPAddr(%s): %s\n", ipAndPort, err.Error())
		return
	}

	conn, err := net.DialUDP("udp", nil, udpaddr)
	if err != nil {
		fmt.Printf("DialUDP(%s): %s\n", ipAndPort, err.Error())
		return
	}

	// Get updates from the channel, send.
	for {
		u := <-updateChan
		s := fmt.Sprintf("XATTtesting,%f,%f,%f", u.heading, u.pitch, u.roll)
		fmt.Printf("%s\n", s)
		conn.Write([]byte(s))
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("%s <ip>\n", os.Args[0])
		return
	}
	go updateSender(os.Args[1])

	initMPU9250()
	time.Sleep(100 * time.Millisecond)

	for i := 0; i < 10; i++ {
		pitch, roll := GetCurrentAttitudeXY()
		fmt.Printf("%f,%f", pitch, roll)
	}
}
