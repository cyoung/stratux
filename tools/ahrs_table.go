package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
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

var currentAHRSString string

func listener() {
	t := time.Now()
	addr := net.UDPAddr{Port: 41504, IP: net.ParseIP("0.0.0.0")}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Printf("error listening: %s\n", err.Error())
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

		fmt.Sprintf("%d,%s\n", time_diff/time.Millisecond, buf_encoded)
		currentAHRSString = string(buf_encoded)
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

func main() {
	situationMutex = &sync.Mutex{}

	go listener()
	go situationUpdater()

	tm := time.NewTicker(125 * time.Millisecond)
	for {
		<-tm.C
		fmt.Printf("%f,%f,%s\n", Location.AHRSRoll, Location.AHRSPitch, currentAHRSString)
	}

}
