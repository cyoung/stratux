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

func handleManagementConnection(conn *websocket.Conn) {
	//	log.Printf("Web client connected.\n")

	timer := time.NewTicker(1 * time.Second)
	for {
		// The below is not used, but should be if something needs to be streamed from the web client ever in the future.
/*		var msg SettingMessage
		err := websocket.JSON.Receive(conn, &msg)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Printf("handleManagementConnection: %s\n", err.Error())
		} else {
			// Use 'msg'.
		}
*/

		// Send status.
		<-timer.C
		update, _ := json.Marshal(&globalStatus)
		_, err := conn.Write(update)

		if err != nil {
			//			log.Printf("Web client disconnected.\n")
			break
		}
	}
}

// AJAX call - /getTraffic. Responds with currently tracked traffic targets.
func handleTrafficRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
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
	w.Header().Set("Content-Type", "application/json")
	situationJSON, _ := json.Marshal(&mySituation)
	fmt.Fprintf(w, "%s\n", situationJSON)
}

// AJAX call - /getTowers. Responds with all ADS-B ground towers that have sent messages that we were able to parse, along with its stats.
func handleTowersRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	towersJSON, _ := json.Marshal(&ADSBTowers)
	fmt.Fprintf(w, "%s\n", towersJSON)
}

// AJAX call - /getSettings. Responds with all stratux.conf data.
func handleSettingsGetRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	settingsJSON, _ := json.Marshal(&globalSettings)
	fmt.Fprintf(w, "%s\n", settingsJSON)
}

// AJAX call - /setSettings. receives via POST command, any/all stratux.conf data.
func handleSettingsSetRequest(w http.ResponseWriter, r *http.Request) {
	// define header in support of cross-domain AJAX
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Method", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Set("Content-Type", "application/json")

	// for an OPTION method request, we return header without processing.
	// this insures we are recognized as supporting cross-domain AJAX REST calls
	if r.Method == "POST" {
		// raw, _ := httputil.DumpRequest(r, true)
		// log.Printf("handleSettingsSetRequest:raw: %s\n", raw)

		decoder := json.NewDecoder(r.Body)
		for {
			var msg map[string]interface{} // support arbitrary JSON

			err := decoder.Decode(&msg)
			if err == io.EOF {
				break
			} else if err != nil {
				log.Printf("handleSettingsSetRequest:error: %s\n", err.Error())
			} else {
				for key, val := range msg {
					// log.Printf("handleSettingsSetRequest:json: testing for key:%s of type %s\n", key, reflect.TypeOf(val))
					switch key {
					case "UAT_Enabled":
						globalSettings.UAT_Enabled = val.(bool)
					case "ES_Enabled":
						globalSettings.ES_Enabled = val.(bool)
					case "GPS_Enabled":
						globalSettings.GPS_Enabled = val.(bool)
					case "AHRS_Enabled":
						globalSettings.AHRS_Enabled = val.(bool)
					case "DEBUG":
						globalSettings.DEBUG = val.(bool)
					case "ReplayLog":
						globalSettings.ReplayLog = val.(bool)
					case "PPM":
						globalSettings.PPM = int(val.(float64))
					default:
						log.Printf("handleSettingsSetRequest:json: unrecognized key:%s\n", key)
					}
				}
				saveSettings()
			}
		}

		// while it may be redundent, we return the latest settings
		settingsJSON, _ := json.Marshal(&globalSettings)
		fmt.Fprintf(w, "%s\n", settingsJSON)
	}
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
