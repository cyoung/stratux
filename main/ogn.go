
/*
	Copyright (c) 2020 Adrian Batzill
	Distributable under the terms of The "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	ogn.go: Routines for reading traffic from ogn-rx-eu
*/

package main

import (
	"encoding/json"
	"net"
	"bufio"
	"time"
	"log"
	"io/ioutil"
)


var ognReadWriter *bufio.ReadWriter

func ognPublishNmea(nmea string) {
	if ognReadWriter != nil {
		// TODO: we could filter a bit more to only send RMC/GGA, but for now it's just everything
		ognReadWriter.Write([]byte(nmea + "\r\n"))
		ognReadWriter.Flush()
	}
}

func ognListen() {
	for {
		if !globalSettings.OGN_Enabled || OGNDev == nil {
			// wait until OGN is enabled
			time.Sleep(1 * time.Second)
			continue
		}
		ognAddr := "127.0.0.1:8888"
		conn, err := net.Dial("tcp", ognAddr)
		if err != nil { // Local connection failed.
			time.Sleep(1 * time.Second)
			continue
		}
		ognReadWriter = bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
		for globalSettings.OGN_Enabled {
			//log.Printf("ES enabled. Ready to read next message from dump1090\n")
			buf, err := ognReadWriter.ReadString('\n')
			if err != nil {
				break
			}
			log.Printf(string(buf))
			// TODO: parse buf
		}
		ognReadWriter = nil
		conn.Close()
		
	}
}

var ognTailNumberCache = make(map[string]string)
func lookupOgnTailNumber(flarmid string) string {
	if len(ognTailNumberCache) == 0 {
		log.Printf("Parsing OGN device db")
		ddb, err := ioutil.ReadFile("/etc/ddb.json")
		if err != nil {
			log.Printf("Failed to read OGN device db")
			return flarmid
		}
		var data map[string]interface{}
		err = json.Unmarshal(ddb, &data)
		if err != nil {
			log.Printf("Failed to parse OGN device db")
			return flarmid
		}
		devlist := data["devices"].([]interface{})
		for i := 0; i < len(devlist); i++ {
			dev := devlist[i].(map[string]interface{})
			flarmid := dev["device_id"].(string)
			tail := dev["registration"].(string)
			ognTailNumberCache[flarmid] = tail
		}
		log.Printf("Successfully parsed OGN device db")
	}
	if tail, ok := ognTailNumberCache[flarmid]; ok {
		return tail
	}
	return flarmid
}

func getTailNumber(flarmid string) string {
	tail := lookupOgnTailNumber(flarmid)
	if globalSettings.DisplayTrafficSource {
		tail = "fl" + tail
	}
	return tail
}