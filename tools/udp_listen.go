package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"time"
)

func main() {
	t := time.Now()
	addr := net.UDPAddr{Port: 41504, IP: net.ParseIP("0.0.0.0")}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Printf("ffMonitor(): error listening: %s\n", err.Error())
		return
	}
	defer conn.Close()
	for {
		buf := make([]byte, 1024)
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			fmt.Printf("Err receive: %s\n", err.Error())
			continue
		}
		buf_encoded := make([]byte, hex.EncodedLen(n))
		hex.Encode(buf_encoded, buf[:n])
		t2 := time.Now()
		time_diff := t2.Sub(t)
		t = t2

		fmt.Printf("%d,%s\n", time_diff/time.Millisecond, buf_encoded)
	}
}
