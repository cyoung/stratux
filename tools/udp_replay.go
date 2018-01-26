package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("%s: <file>\n", os.Args[0])
		return
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	// Open a socket.
	BROADCAST_IPv4 := net.IPv4(255, 255, 255, 255)

	//	addr, _ := net.ResolveUDPAddr("udp", "192.168.10.10:41501")
	//	laddr, _ := net.ResolveUDPAddr("udp", "192.168.10.1:58814")
	//	conn, err := net.DialUDP("udp", laddr, addr)
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   BROADCAST_IPv4,
		Port: 41504,
	})

	if err != nil {
		fmt.Printf("DialUDP(): %s\n", err.Error())
		return
	}

	for scanner.Scan() {
		s := scanner.Text()
		x := strings.Split(s, ",")
		if len(x) < 2 {
			continue
		}
		i, err := strconv.ParseInt(x[0], 10, 32)
		if err != nil {
			fmt.Printf("error parsing '%s': %s.\n", x[0], err.Error())
			continue
		}
		buf := make([]byte, hex.DecodedLen(len(x[1])))
		n, err := hex.Decode(buf, []byte(x[1]))
		if err != nil {
			fmt.Printf("error parsing '%s': %s.\n", x[1], err.Error())
			continue
		}

		fmt.Printf("sleeping %dms, sending %s\n", i, x[1])
		time.Sleep(time.Duration(i) * time.Millisecond)
		conn.Write(buf[:n])
	}
}
