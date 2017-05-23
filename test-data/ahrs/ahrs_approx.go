package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	SITUATION_URL = "http://127.0.0.1/getSituation"
)

type MySituation struct {
	AHRSRoll  float64
	AHRSPitch float64
}

var Location MySituation

var situationMutex *sync.Mutex

func chkErr(err error) {
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		os.Exit(1)
	}
}

func situationUpdater() {
	situationUpdateTicker := time.NewTicker(100 * time.Millisecond)
	for {
		<-situationUpdateTicker.C
		situationMutex.Lock()

		resp, err := http.Get(SITUATION_URL)
		if err != nil {
			fmt.Printf("HTTP GET error: %s\n", err.Error())
			situationMutex.Unlock()
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("HTTP GET body error: %s\n", err.Error())
			resp.Body.Close()
			situationMutex.Unlock()
			continue
		}

		//		fmt.Printf("body: %s\n", string(body))
		err = json.Unmarshal(body, &Location)

		if err != nil {
			fmt.Printf("HTTP JSON unmarshal error: %s\n", err.Error())
		}
		resp.Body.Close()
		situationMutex.Unlock()

	}
}

type AHRSData struct {
	Roll    float64
	Pitch   float64
	Trigger []byte
}

func main() {
	situationMutex = &sync.Mutex{}

	BROADCAST_IPv4 := net.IPv4(255, 255, 255, 255)
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   BROADCAST_IPv4,
		Port: 41504,
	})

	if err != nil {
		fmt.Printf("err conn: %s\n", err.Error())
		return
	}

	ahrsTable := make([]AHRSData, 0)

	f, err := os.Open("ahrs_table.log")
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s := scanner.Text()
		x := strings.Split(s, ",")
		if len(x) < 3 {
			continue
		}

		buf := make([]byte, 1024)
		n, err := hex.Decode(buf, []byte(x[2]))
		if err != nil || n == 0 {
			fmt.Printf("error parsing '%s'.\n", x[2])
			continue
		}

		roll, err := strconv.ParseFloat(x[0], 64)
		if err != nil {
			fmt.Printf("error parsing '%s'.\n", x[0])
			continue
		}
		pitch, err := strconv.ParseFloat(x[1], 64)
		if err != nil {
			fmt.Printf("error parsing '%s'.\n", x[1])
			continue
		}

		newEntry := AHRSData{
			Roll:    roll,
			Pitch:   pitch,
			Trigger: buf[:n],
		}

		ahrsTable = append(ahrsTable, newEntry)

	}

	fmt.Printf("loaded %d size ahrs table.\n", len(ahrsTable))

	go situationUpdater()

	tm := time.NewTicker(125 * time.Millisecond)
	for {
		<-tm.C
		situationMutex.Lock()
		myPitch := Location.AHRSPitch
		myRoll := Location.AHRSRoll
		situationMutex.Unlock()

		mB := make([]byte, 0)
		var mV float64
		for _, v := range ahrsTable {
			roll := v.Roll
			pitch := v.Pitch
			trigger := v.Trigger
			z := ((roll - myRoll) * (roll - myRoll)) + ((pitch - myPitch) * (pitch - myPitch))
			if (z < mV) || ((mV - 0.000) < 0.00001) {
				mV = z
				mB = trigger
			}
		}
		if len(mB) > 0 {
			conn.Write(mB)
		}
	}

}
