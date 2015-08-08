package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ipadAddr        = "192.168.10.255:49002"
	dump1090Addr    = "127.0.0.1:30003"
	maxDatagramSize = 8192
)

type PositionInfo struct {
	lat       string
	lng       string
	alt       string
	hdg       string
	vel       string
	vr        string
	tail      string
	last_seen time.Time
}

func cleanupOldEntries() {
	for icaoDec, pi := range blips {
		s := time.Since(pi.last_seen)
		if s.Seconds() >= float64(15) { // Timeout time.
			//fmt.Printf("REMOVED %d\n", icaoDec)
			delete(blips, icaoDec)
		}
	}
}

func ipadUpdate(mutex *sync.Mutex) {
	addr, err := net.ResolveUDPAddr("udp", ipadAddr)
	if err != nil {
		panic(err)
	}
	outConn, err := net.DialUDP("udp", nil, addr)

	for {
		mutex.Lock()
		cleanupOldEntries()
		for icaoDec, pi := range blips {
			msg := fmt.Sprintf("XTRAFFICMy Sim,%d,%s,%s,%s,%s,1,%s,%s,%s", icaoDec, pi.lat, pi.lng, pi.alt, pi.vr, pi.hdg, pi.vel, pi.tail)
			//fmt.Println(msg)
			outConn.Write([]byte(msg))
		}
		mutex.Unlock()
		time.Sleep(1 * time.Second)
	}
	//			c.Write([]byte("XTRAFFICMy Sim,168,42.503464,-83.622551,3749.9,-213.0,1,68.2,126.0,KS6"))
}

var blips map[int64]PositionInfo

func main() {
	mutex := &sync.Mutex{}
	blips = make(map[int64]PositionInfo)
	go ipadUpdate(mutex)

	context := new(daemon.Context)
	child, _ := context.Reborn()
	if child != nil {
		fmt.Printf("going into background.\n")
	} else {
		defer context.Release()
		for {
			inConn, err := net.Dial("tcp", dump1090Addr)
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
			rdr := bufio.NewReader(inConn)
			for {
				buf, err := rdr.ReadString('\n')
				if err != nil { // Must have disconnected?
					break
				}
				buf = strings.Trim(buf, "\r\n")
				//fmt.Printf("%s\n", buf)
				x := strings.Split(buf, ",")
				//TODO: Add more sophisticated stuff that combines heading/speed updates with the location.
				if len(x) < 22 {
					continue
				}
				icao := x[4]
				icaoDec, err := strconv.ParseInt(icao, 16, 32)
				if err != nil {
					continue
				}
				mutex.Lock()
				// Retrieve previous information on this ICAO code.
				var pi PositionInfo
				if _, ok := blips[icaoDec]; ok {
					pi = blips[icaoDec]
				}

				if x[1] == "3" {
					//MSG,3,111,11111,AC2BB7,111111,2015/07/28,03:59:12.363,2015/07/28,03:59:12.353,,5550,,,42.35847,-83.42212,,,,,,0
					alt := x[11]
					lat := x[14]
					lng := x[15]

					//fmt.Printf("icao=%s, icaoDec=%d, alt=%s, lat=%s, lng=%s\n", icao, icaoDec, alt, lat, lng)
					pi.alt = alt
					pi.lat = lat
					pi.lng = lng
				}
				if x[1] == "4" {
					// MSG,4,111,11111,A3B557,111111,2015/07/28,06:13:36.417,2015/07/28,06:13:36.398,,,414,278,,,-64,,,,,0
					vel := x[12]
					hdg := x[13]
					vr := x[16]

					//fmt.Printf("icao=%s, icaoDec=%d, vel=%s, hdg=%s, vr=%s\n", icao, icaoDec, vel, hdg, vr)
					pi.vel = vel
					pi.hdg = hdg
					pi.vr = vr
				}
				if x[1] == "1" {
					// MSG,1,,,%02X%02X%02X,,,,,,%s,,,,,,,,0,0,0,0
					tail := x[10]
					pi.tail = tail
				}

				// Update "last seen" (any type of message).
				pi.last_seen = time.Now()

				blips[icaoDec] = pi // Update information on this ICAO code.
				mutex.Unlock()
			}
		}
	}
}
