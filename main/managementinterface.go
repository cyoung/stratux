package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/websocket"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

type SettingMessage struct {
	Setting string `json:"setting"`
	Value   bool   `json:"state"`
}

type InfoMessage struct {
	*status
	*settings
}

func statusSender(conn *websocket.Conn) {
	timer := time.NewTicker(1 * time.Second)
	for {
		<-timer.C

		update, _ := json.Marshal(InfoMessage{status: &globalStatus, settings: &globalSettings})
		_, err := conn.Write(update)

		if err != nil {
			//			log.Printf("Web client disconnected.\n")
			break
		}
	}
}

func handleManagementConnection(conn *websocket.Conn) {
	//	log.Printf("Web client connected.\n")
	go statusSender(conn)

	for {
		var msg SettingMessage
		err := websocket.JSON.Receive(conn, &msg)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Printf("handleManagementConnection: %s\n", err.Error())
		} else {
			if msg.Setting == "UAT_Enabled" {
				globalSettings.UAT_Enabled = msg.Value
			}
			if msg.Setting == "ES_Enabled" {
				globalSettings.ES_Enabled = msg.Value
			}
			if msg.Setting == "GPS_Enabled" {
				globalSettings.GPS_Enabled = msg.Value
			}
			if msg.Setting == "AHRS_Enabled" {
				globalSettings.AHRS_Enabled = msg.Value
			}
			if msg.Setting == "DEBUG" {
				globalSettings.DEBUG = msg.Value
			}

			saveSettings()
		}
	}
}

// AJAX call - /getTraffic. Responds with currently tracked traffic targets.

func handleTrafficRequest(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
	/* From JSON package docs:
	"JSON objects only support strings as keys; to encode a Go map type it must be of the form map[string]T (where T is any Go type supported by the json package)."
	*/
	t := make(map[string]TrafficInfo)
	trafficMutex.Lock()
	for icao, traf := range traffic {
		icaoStr := strconv.FormatInt(int64(icao), 16) // Convert to hex.
		t[icaoStr] = traf
	}
	trafficJSON, _ := json.Marshal(&t)
	trafficMutex.Unlock()
	fmt.Fprintf(w, "%s\n", trafficJSON)
}

// AJAX call - /getSituation. Responds with current situation (lat/lon/gdspeed/track/pitch/roll/heading/etc.)
func handleSituationRequest(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
	situationJSON, _ := json.Marshal(&mySituation)
	fmt.Fprintf(w, "%s\n", situationJSON)
}

// AJAX call - /getTowers. Responds with all ADS-B ground towers that have sent messages that we were able to parse, along with its stats.
func handleTowersRequest(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
	towersJSON, _ := json.Marshal(&ADSBTowers)
	fmt.Fprintf(w, "%s\n", towersJSON)
}

// AJAX call - /getSettings. Responds with all stratux.conf data.
func handleSettingsGetRequest(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
	settingsJSON, _ := json.Marshal(&globalSettings)
	fmt.Fprintf(w, "%s\n", settingsJSON)
}

// AJAX call - /setSettings. receives via POST command, any/all stratux.conf data.
func handleSettingsSetRequest(w http.ResponseWriter, r *http.Request) {
	//TODO need this setter function implemented
}

func managementInterface() {
	http.Handle("/", http.FileServer(http.Dir("/var/www")))
	http.Handle("/logs/", http.StripPrefix("/logs/", http.FileServer(http.Dir("/var/log"))))
	http.HandleFunc("/control",
		func(w http.ResponseWriter, req *http.Request) {
			s := websocket.Server{
				Handler: websocket.Handler(handleManagementConnection)}
			s.ServeHTTP(w, req)
		})

	http.HandleFunc("/getTraffic", handleTrafficRequest)
	http.HandleFunc("/getSituation", handleSituationRequest)
	http.HandleFunc("/getTowers", handleTowersRequest)
	http.HandleFunc("/getSettings", handleSettingsGetRequest)
	http.HandleFunc("/setSettings", handleSettingsSetRequest)

	err := http.ListenAndServe(managementAddr, nil)

	if err != nil {
		log.Printf("managementInterface ListenAndServe: %s\n", err.Error())
	}
}
