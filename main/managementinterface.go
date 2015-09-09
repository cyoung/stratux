package main

import (
	"encoding/json"
	"golang.org/x/net/websocket"
	"io"
	"log"
	"net/http"
	"time"
)

type SettingMessage struct {
	Setting string `json:"setting"`
	Value   bool   `json:"state"`
}

func statusSender(conn *websocket.Conn) {
	timer := time.NewTicker(1 * time.Second)
	for {
		<-timer.C

		statResp, _ := json.Marshal(&globalStatus)
		conn.Write(statResp)

		settingResp, _ := json.Marshal(&globalSettings)
		_, err := conn.Write(settingResp)

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

func managementInterface() {
	http.Handle("/", http.FileServer(http.Dir("/var/www")))
	http.Handle("/logs/", http.StripPrefix("/logs/", http.FileServer(http.Dir("/var/log"))))
	http.HandleFunc("/control",
		func(w http.ResponseWriter, req *http.Request) {
			s := websocket.Server{
				Handler: websocket.Handler(handleManagementConnection)}
			s.ServeHTTP(w, req)
		})

	err := http.ListenAndServe(managementAddr, nil)

	if err != nil {
		log.Printf("managementInterface ListenAndServe: %s\n", err.Error())
	}
}
