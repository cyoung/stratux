/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	managementinterface.go: Web interfaces (JSON and websocket), web server for web interface HTML.
*/

package main

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"

	"github.com/b3nn0/stratux/common"
	humanize "github.com/dustin/go-humanize"
	"golang.org/x/net/websocket"
)

type SettingMessage struct {
	Setting string `json:"setting"`
	Value   bool   `json:"state"`
}

type MbTileConnectionCacheEntry struct {
	Path string
	Conn *sql.DB
	Metadata map[string]string
	fileTime time.Time
	
}

func (this *MbTileConnectionCacheEntry) IsOutdated() bool {
	file, err := os.Stat(this.Path)
	if err != nil {
		return true
	}
	modTime := file.ModTime()
	return modTime != this.fileTime
}

func NewMbTileConnectionCacheEntry(path string, conn *sql.DB) *MbTileConnectionCacheEntry {
	file, err := os.Stat(path)
	if err != nil {
		return nil
	}
	return &MbTileConnectionCacheEntry{path, conn, nil, file.ModTime()}
}

var mbtileCacheLock = sync.Mutex{}
var mbtileConnectionCache = make(map[string]MbTileConnectionCacheEntry)


// Weather updates channel.
var weatherUpdate *uibroadcaster
var trafficUpdate *uibroadcaster
var radarUpdate *uibroadcaster
var gdl90Update *uibroadcaster



func handleGDL90WS(conn *websocket.Conn) {
	// Subscribe the socket to receive updates.
	gdl90Update.AddSocket(conn)

	// Connection closes when function returns. Since uibroadcast is writing and we don't need to read anything (for now), just keep it busy.
	for {
		buf := make([]byte, 1024)
		_, err := conn.Read(buf)
		if err != nil {
			break
		}
		if buf[0] != 0 { // Dummy.
			continue
		}
		time.Sleep(1 * time.Second)
	}
}

// Situation updates channel.
var situationUpdate *uibroadcaster

// Raw weather (UATFrame packet stream) update channel.
var weatherRawUpdate *uibroadcaster

/*
	The /weather websocket starts off by sending the current buffer of weather messages, then sends updates as they are received.
*/
func handleWeatherWS(conn *websocket.Conn) {
	// Subscribe the socket to receive updates.
	weatherUpdate.AddSocket(conn)

	// Connection closes when function returns. Since uibroadcast is writing and we don't need to read anything (for now), just keep it busy.
	for {
		buf := make([]byte, 1024)
		_, err := conn.Read(buf)
		if err != nil {
			break
		}
		if buf[0] != 0 { // Dummy.
			continue
		}
		time.Sleep(1 * time.Second)
	}
}

func handleJsonIo(conn *websocket.Conn) {
	trafficMutex.Lock()
	for _, traf := range traffic {
		if !traf.Position_valid { // Don't send unless a valid position exists.
			continue
		}
		trafficJSON, _ := json.Marshal(&traf)
		conn.Write(trafficJSON)
	}
	// Subscribe the socket to receive updates.
	trafficUpdate.AddSocket(conn)
	radarUpdate.AddSocket(conn)
	weatherRawUpdate.AddSocket(conn)
	situationUpdate.AddSocket(conn)

	trafficMutex.Unlock()

	// Connection closes when function returns. Since uibroadcast is writing and we don't need to read anything (for now), just keep it busy.
	for {
		buf := make([]byte, 1024)
		_, err := conn.Read(buf)
		if err != nil {
			break
		}
		if buf[0] != 0 { // Dummy.
			continue
		}
		time.Sleep(1 * time.Second)
	}
}

// Works just as weather updates do.

func handleTrafficWS(conn *websocket.Conn) {
	trafficMutex.Lock()
	for _, traf := range traffic {
		if !traf.Position_valid { // Don't send unless a valid position exists.
			continue
		}
		trafficJSON, _ := json.Marshal(&traf)
		conn.Write(trafficJSON)
	}
	// Subscribe the socket to receive updates.
	trafficUpdate.AddSocket(conn)
	trafficMutex.Unlock()

	// Connection closes when function returns. Since uibroadcast is writing and we don't need to read anything (for now), just keep it busy.
	for {
		buf := make([]byte, 1024)
		_, err := conn.Read(buf)
		if err != nil {
			break
		}
		if buf[0] != 0 { // Dummy.
			continue
		}
		time.Sleep(1 * time.Second)
	}
}

func handleRadarWS(conn *websocket.Conn) {
	trafficMutex.Lock()
	// Subscribe the socket to receive updates. Not necessary to send old traffic 
	radarUpdate.AddSocket(conn)
	trafficMutex.Unlock()

	radarUpdate.SendJSON(globalSettings);

	// Connection closes when function returns. Since uibroadcast is writing and we don't need to read anything (for now), just keep it busy.
	for {
		buf := make([]byte, 1024)
		_, err := conn.Read(buf)
		if err != nil {
			break
		}
		if buf[0] != 0 { // Dummy.
			continue
		}
		time.Sleep(1 * time.Second)
	}
}


func handleStatusWS(conn *websocket.Conn) {
	//	log.Printf("Web client connected.\n")

	timer := time.NewTicker(1 * time.Second)
	for {
		// The below is not used, but should be if something needs to be streamed from the web client ever in the future.
		/*		var msg SettingMessage
				err := websocket.JSON.Receive(conn, &msg)
				if err == io.EOF {
					break
				} else if err != nil {
					log.Printf("handleStatusWS: %s\n", err.Error())
				} else {
					// Use 'msg'.
				}
		*/

		// Send status.
		update, _ := json.Marshal(&globalStatus)
		_, err := conn.Write(update)

		if err != nil {
			//			log.Printf("Web client disconnected.\n")
			break
		}
		<-timer.C
	}
}

func handleSituationWS(conn *websocket.Conn) {
	timer := time.NewTicker(100 * time.Millisecond)
	for {
		situationJSON, _ := json.Marshal(&mySituation)
		_, err := conn.Write(situationJSON)

		if err != nil {
			break
		}
		<-timer.C

	}

}

// AJAX call - /getStatus. Responds with current global status
// a webservice call for the same data available on the websocket but when only a single update is needed
func handleStatusRequest(w http.ResponseWriter, r *http.Request) {
	setNoCache(w)
	setJSONHeaders(w)
	statusJSON, _ := json.Marshal(&globalStatus)
	fmt.Fprintf(w, "%s\n", statusJSON)
}

// AJAX call - /getSituation. Responds with current situation (lat/lon/gdspeed/track/pitch/roll/heading/etc.)
func handleSituationRequest(w http.ResponseWriter, r *http.Request) {
	setNoCache(w)
	setJSONHeaders(w)
	situationJSON, _ := json.Marshal(&mySituation)
	fmt.Fprintf(w, "%s\n", situationJSON)
}

// AJAX call - /getTowers. Responds with all ADS-B ground towers that have sent messages that we were able to parse, along with its stats.
func handleTowersRequest(w http.ResponseWriter, r *http.Request) {
	setNoCache(w)
	setJSONHeaders(w)

	ADSBTowerMutex.Lock()
	towersJSON, err := json.Marshal(&ADSBTowers)
	if err != nil {
		log.Printf("Error sending tower JSON data: %s\n", err.Error())
	}
	// for testing purposes, we can return a fixed reply
	// towersJSON = []byte(`{"(38.490880,-76.135554)":{"Lat":38.49087953567505,"Lng":-76.13555431365967,"Signal_strength_last_minute":100,"Signal_strength_max":67,"Messages_last_minute":1,"Messages_total":1059},"(38.978698,-76.309276)":{"Lat":38.97869825363159,"Lng":-76.30927562713623,"Signal_strength_last_minute":495,"Signal_strength_max":32,"Messages_last_minute":45,"Messages_total":83},"(39.179285,-76.668413)":{"Lat":39.17928457260132,"Lng":-76.66841268539429,"Signal_strength_last_minute":50,"Signal_strength_max":24,"Messages_last_minute":1,"Messages_total":16},"(39.666309,-74.315300)":{"Lat":39.66630935668945,"Lng":-74.31529998779297,"Signal_strength_last_minute":9884,"Signal_strength_max":35,"Messages_last_minute":4,"Messages_total":134}}`)
	fmt.Fprintf(w, "%s\n", towersJSON)
	ADSBTowerMutex.Unlock()
}

// AJAX call - /getSatellites. Responds with all GNSS satellites that are being tracked, along with status information.
func handleSatellitesRequest(w http.ResponseWriter, r *http.Request) {
	setNoCache(w)
	setJSONHeaders(w)
	mySituation.muSatellite.Lock()
	satellitesJSON, err := json.Marshal(&Satellites)
	if err != nil {
		log.Printf("Error sending GNSS satellite JSON data: %s\n", err.Error())
	}
	fmt.Fprintf(w, "%s\n", satellitesJSON)
	mySituation.muSatellite.Unlock()
}

// AJAX call - /getSettings. Responds with all stratux.conf data.
func handleSettingsGetRequest(w http.ResponseWriter, r *http.Request) {
	setNoCache(w)
	setJSONHeaders(w)
	settingsJSON, err := json.Marshal(&globalSettings)
	if err != nil {
		log.Printf("%s", err)
	}
	fmt.Fprintf(w, "%s\n", settingsJSON)
}

// AJAX call - /setSettings. receives via POST command, any/all stratux.conf data.
func handleSettingsSetRequest(w http.ResponseWriter, r *http.Request) {
	// define header in support of cross-domain AJAX
	setNoCache(w)
	setJSONHeaders(w)
	w.Header().Set("Access-Control-Allow-Method", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")

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
				reconfigureOgnTracker := false
				reconfigureGXTracker := false
				reconfigureFancontrol := false
				for key, val := range msg {
					// log.Printf("handleSettingsSetRequest:json: testing for key:%s of type %s\n", key, reflect.TypeOf(val))
					switch key {
					case "DarkMode":
						globalSettings.DarkMode = val.(bool)
					case "UAT_Enabled":
						globalSettings.UAT_Enabled = val.(bool)
					case "ES_Enabled":
						globalSettings.ES_Enabled = val.(bool)
					case "OGN_Enabled":
						globalSettings.OGN_Enabled = val.(bool)
					case "AIS_Enabled":
						globalSettings.AIS_Enabled = val.(bool)
					case "APRS_Enabled":
						globalSettings.APRS_Enabled = val.(bool)
					case "Ping_Enabled":
						globalSettings.Ping_Enabled = val.(bool)
					case "OGNI2CTXEnabled":
						globalSettings.OGNI2CTXEnabled = val.(bool)
					case "GPS_Enabled":
						globalSettings.GPS_Enabled = val.(bool)
					case "IMU_Sensor_Enabled":
						globalSettings.IMU_Sensor_Enabled = val.(bool)
						if !globalSettings.IMU_Sensor_Enabled && globalStatus.IMUConnected {
							myIMUReader.Close()
							globalStatus.IMUConnected = false
						}
					case "BMP_Sensor_Enabled":
						globalSettings.BMP_Sensor_Enabled = val.(bool)
						if !globalSettings.BMP_Sensor_Enabled && globalStatus.BMPConnected {
							myPressureReader.Close()
							globalStatus.BMPConnected = false
						}
					case "DEBUG":
						globalSettings.DEBUG = val.(bool)
					case "DisplayTrafficSource":
						globalSettings.DisplayTrafficSource = val.(bool)
					case "ReplayLog":
						v := val.(bool)
						if v != globalSettings.ReplayLog { // Don't mark the files unless there is a change.
							globalSettings.ReplayLog = v
						}
					case "TraceLog":
						globalSettings.TraceLog = val.(bool)
					case "AHRSLog":
						globalSettings.AHRSLog = val.(bool)
					case "PersistentLogging":
						globalSettings.PersistentLogging = val.(bool)
						setPersistentLogging(globalSettings.PersistentLogging)
					case "IMUMapping":
						if globalSettings.IMUMapping != val.([2]int) {
							globalSettings.IMUMapping = val.([2]int)
							myIMUReader.Close()
							globalStatus.IMUConnected = false // Force a restart of the IMU reader
						}
					case "Dump1090Gain":
						globalSettings.Dump1090Gain = (val.(float64))
					case "PPM":
						globalSettings.PPM = int(val.(float64))
					case "AltitudeOffset":
						globalSettings.AltitudeOffset = int(val.(float64))
					case "RadarLimits":
						globalSettings.RadarLimits = int(val.(float64))
						radarUpdate.SendJSON(globalSettings)
					case "RadarRange":
						globalSettings.RadarRange = int(val.(float64))
						radarUpdate.SendJSON(globalSettings)
					case "Baud":
						if globalSettings.SerialOutputs != nil {
							for dev, serialOut := range globalSettings.SerialOutputs {
								newBaud := int(val.(float64))
								if newBaud == serialOut.Baud { // Same baud rate. No change.
									continue
								}
								log.Printf("changing %s baud rate from %d to %d.\n", dev, serialOut.Baud, newBaud)
								serialOut.Baud = newBaud
								globalSettings.SerialOutputs[dev] = serialOut
								closeSerial(dev)
							}
						}
					case "WatchList":
						globalSettings.WatchList = val.(string)
					case "GLimits":
						globalSettings.GLimits = val.(string)
					case "OwnshipModeS":
						codes := strings.Split(val.(string), ",")
						codesFinal :=  make([]string, 0)
						for _, code := range codes {
							code = strings.Trim(code, " ")
							// Expecting a hex string less than 6 characters (24 bits) long.
							if len(code) > 6 { // Too long.
								continue
							}
							// Pad string, must be 6 characters long.
							vals := strings.ToUpper(code)
							for len(vals) < 6 {
								vals = "0" + vals
							}
							hexn, err := hex.DecodeString(vals)
							if err != nil { // Number not valid.
								log.Printf("handleSettingsSetRequest:OwnshipModeS: %s\n", err.Error())
								continue
							}
							codesFinal = append(codesFinal, fmt.Sprintf("%02X%02X%02X", hexn[0], hexn[1], hexn[2]))
						}
						globalSettings.OwnshipModeS = strings.Join(codesFinal, ",")
					case "StaticIps":
						ipsStr := val.(string)
						ips := strings.Split(ipsStr, " ")
						if ipsStr == "" {
							ips = make([]string, 0)
						}

						re, _ := regexp.Compile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`)
						err := ""
						for _, ip := range ips {
							// Verify IP format
							if !re.MatchString(ip) {
								err = err + "Invalid IP: " + ip + ". "
							}
						}
						if err != "" {
							log.Printf("handleSettingsSetRequest:StaticIps: %s\n", err)
							continue
						}
						globalSettings.StaticIps = ips
					case "WiFiCountry":
						setWifiCountry(val.(string))
					case "WiFiSSID":
						setWifiSSID(val.(string))
					case "WiFiChannel":
						setWifiChannel(int(val.(float64)))
					case "WiFiSecurityEnabled":
						setWifiSecurityEnabled(val.(bool))
					case "WiFiPassphrase":
						setWifiPassphrase(val.(string))
					case "WiFiIPAddress":
						setWifiIPAddress(val.(string))
					case "WiFiMode":
						setWiFiMode(int(val.(float64)))
					case "WiFiDirectPin":
						setWifiDirectPin(val.(string))
					case "WiFiClientNetworks":
						var networks = make([]wifiClientNetwork, 0)
						for _, rawNetwork := range val.([]interface{}) {
							network := rawNetwork.(map[string]interface{})
							networks = append(networks, wifiClientNetwork{network["SSID"].(string), network["Password"].(string)})
						}
						setWifiClientNetworks(networks)
					case "WiFiInternetPassThroughEnabled":
						setWifiInternetPassthroughEnabled(val.(bool))
					case "EstimateBearinglessDist":
						globalSettings.EstimateBearinglessDist = val.(bool)

					case "OGNAddrType":
						globalSettings.OGNAddrType = int(val.(float64))
						reconfigureOgnTracker = true
					case "OGNAddr":
						globalSettings.OGNAddr = val.(string)
						reconfigureOgnTracker = true
					case "OGNAcftType":
						globalSettings.OGNAcftType = int(val.(float64))
						reconfigureOgnTracker = true
					case "OGNPilot":
						globalSettings.OGNPilot = val.(string)
						reconfigureOgnTracker = true
					case "OGNReg":
						globalSettings.OGNReg = val.(string)
						reconfigureOgnTracker = true
					case "OGNTxPower":
						globalSettings.OGNTxPower = int(val.(float64))
						reconfigureOgnTracker = true
					case "GXAddr":
						inter,_ := strconv.ParseInt(val.(string), 16, 0)
						globalSettings.GXAddr = int(inter) & 0xffffff
						reconfigureGXTracker = true
					case "GXAddrType":
						globalSettings.GXAddrType = int(val.(float64))
						reconfigureGXTracker = true
					case "GXAcftType":
						globalSettings.GXAcftType = int(val.(float64))
						reconfigureGXTracker = true
					case "GXPilot":
						globalSettings.GXPilot = val.(string)
						reconfigureGXTracker = true
					case "PWMDutyMin":
						globalSettings.PWMDutyMin = int(val.(float64))
						reconfigureFancontrol = true

					default:
						log.Printf("handleSettingsSetRequest:json: unrecognized key:%s\n", key)
					}
				}
				saveSettings()
				applyNetworkSettings(false, false)
				if reconfigureOgnTracker {
					configureOgnTrackerFromSettings()
				}
				if reconfigureGXTracker {
					configureGxAirComTracker()
				}
				if reconfigureFancontrol {
					exec.Command("killall", "-SIGUSR1", "fancontrol").Run();
				}
			}
		}

		// while it may be redundant, we return the latest settings
		settingsJSON, _ := json.Marshal(&globalSettings)
		fmt.Fprintf(w, "%s\n", settingsJSON)
	}
}

func setPersistentLogging(persistent bool) {
	if persistent {
		overlayctl("disable")
	} else {
		overlayctl("enable")
	}
}


func handleShutdownRequest(w http.ResponseWriter, r *http.Request) {
	syscall.Sync()
	exec.Command("systemctl", "poweroff").Run()
}

func doReboot() {
	syscall.Sync()
	exec.Command("systemctl", "reboot").Run()
}

func handleDeleteLogFile(w http.ResponseWriter, r *http.Request) {
	log.Println("handleDeleteLogFile called!!!")
	clearDebugLogFile()
}

func handleDeleteAHRSLogFiles(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir("/var/log")
	if err != nil {
		http.Error(w, fmt.Sprintf("error deleting AHRS logs: %s", err), http.StatusNotFound)
		return
	}

	var fn string
	for _, f := range files {
		fn = f.Name()
		if v, _ := filepath.Match("sensors_*.csv", fn); v {
			os.Remove("/var/log/" + fn)
			log.Printf("Deleting AHRS log file %s\n", fn)
		}
		analysisLogger = nil
	}
}

func handleDevelModeToggle(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleDevelModeToggle called!!!\n")
	globalSettings.DeveloperMode = true
	saveSettings()
}

func handleRestartRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleRestartRequest called\n")
	go doRestartApp()
}

func handleRebootRequest(w http.ResponseWriter, r *http.Request) {
	setNoCache(w)
	setJSONHeaders(w)
	w.Header().Set("Access-Control-Allow-Method", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	go delayReboot()
}

func handleOrientAHRS(w http.ResponseWriter, r *http.Request) {
	// define header in support of cross-domain AJAX
	setNoCache(w)
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Method", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")

	// For an OPTION method request, we return header without processing.
	// This ensures we are recognized as supporting cross-domain AJAX REST calls.
	if r.Method == "POST" {
		var (
			action []byte = make([]byte, 1)
			err    error
		)

		if _, err = r.Body.Read(action); err != nil {
			log.Println("AHRS Error: handleOrientAHRS received invalid request")
			http.Error(w, "orientation received invalid request", http.StatusBadRequest)
		}

		switch action[0] {
		case 'f': // Set sensor "forward" direction (toward nose of airplane).
			f, err := getMinAccelDirection()
			if err != nil {
				log.Printf("AHRS Error: sensor orientation: couldn't read accelerometer: %s\n", err)
				http.Error(w, fmt.Sprintf("couldn't read accelerometer: %s\n", err), http.StatusBadRequest)
				return
			}
			log.Printf("AHRS Info: sensor orientation success! forward axis is %d\n", f)
			globalSettings.IMUMapping = [2]int{f, 0}
		case 'd': // Set sensor "up" direction (toward top of airplane).
			globalSettings.SensorQuaternion = [4]float64{0, 0, 0, 0}
			saveSettings()
			myIMUReader.Close()
			globalStatus.IMUConnected = false // restart the processes depending on the orientation
			ResetAHRSGLoad()
			time.Sleep(2000 * time.Millisecond)
		}
	}
}

func handleCageAHRS(w http.ResponseWriter, r *http.Request) {
	// define header in support of cross-domain AJAX
	setNoCache(w)
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Method", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")

	// For an OPTION method request, we return header without processing.
	// This ensures we are recognized as supporting cross-domain AJAX REST calls.
	if r.Method == "POST" {
		CageAHRS()
	}
}

func handleCalibrateAHRS(w http.ResponseWriter, r *http.Request) {
	// define header in support of cross-domain AJAX
	setNoCache(w)
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Method", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")

	// For an OPTION method request, we return header without processing.
	// This ensures we are recognized as supporting cross-domain AJAX REST calls.
	if r.Method == "POST" {
		CalibrateAHRS()
	}
}

func handleResetGMeter(w http.ResponseWriter, r *http.Request) {
	// define header in support of cross-domain AJAX
	setNoCache(w)
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Method", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")

	// For an OPTION method request, we return header without processing.
	// This ensures we are recognized as supporting cross-domain AJAX REST calls.
	if r.Method == "POST" {
		ResetAHRSGLoad()
	}
}

func doRestartApp() {
	time.Sleep(1)
	syscall.Sync()
	out, err := exec.Command("/bin/systemctl", "restart", "stratux").Output()
	if err != nil {
		log.Printf("restart error: %s\n%s", err.Error(), out)
	} else {
		log.Printf("restart: %s\n", out)
	}
}

// AJAX call - /getClients. Responds with all connected clients.
func handleClientsGetRequest(w http.ResponseWriter, r *http.Request) {
	setNoCache(w)
	setJSONHeaders(w)
	netMutex.Lock()
	clientsJSON, _ := json.Marshal(&clientConnections)
	netMutex.Unlock()
	fmt.Fprintf(w, "%s\n", clientsJSON)
}

func delayReboot() {
	time.Sleep(1 * time.Second)
	doReboot()
}

func handleDownloadLogRequest(w http.ResponseWriter, r *http.Request) {
	//w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=stratux.log")
	http.ServeFile(w, r, "/var/log/stratux.log")
}

func handleDownloadAHRSLogsRequest(w http.ResponseWriter, r *http.Request) {
	// Common error handler
	httpErr := func(w http.ResponseWriter, e error) {
		http.Error(w, fmt.Sprintf("error zipping AHRS logs: %s", e), http.StatusNotFound)
	}

	files, err := ioutil.ReadDir("/var/log")
	if err != nil {
		httpErr(w, err)
		return
	}

	z := zip.NewWriter(w)
	defer z.Close()

	for _, f := range files {
		fn := f.Name()
		v1, _ := filepath.Match("sensors_*.csv", fn)
		v2, _ := filepath.Match("stratux.log", fn)
		if !(v1 || v2) {
			continue
		}

		unzippedFile, err := os.Open("/var/log/" + fn)
		if err != nil {
			httpErr(w, err)
			return
		}

		fh, err := zip.FileInfoHeader(f)
		if err != nil {
			httpErr(w, err)
			return
		}
		zippedFile, err := z.CreateHeader(fh)
		if err != nil {
			httpErr(w, err)
			return
		}

		_, err = io.Copy(zippedFile, unzippedFile)
		if err != nil {
			httpErr(w, err)
			return
		}
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=ahrs_logs.zip")
}

func handleDownloadDBRequest(w http.ResponseWriter, r *http.Request) {
	//w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=stratux.sqlite")
	http.ServeFile(w, r, "/var/log/stratux.sqlite")
}

// Upload an update file.
func handleUpdatePostRequest(w http.ResponseWriter, r *http.Request) {
	setNoCache(w)
	setJSONHeaders(w)
	overlayctl("unlock")
	reader, err := r.MultipartReader()
	if err != nil {
		log.Printf("Update failed from %s (%s).\n", r.RemoteAddr, err.Error())
		return
	}
	for {
		part, err := reader.NextPart();
		if err != nil {
			log.Printf("Update failed from %s (%s).\n", r.RemoteAddr, err.Error())
			return
		}
		if part == nil {
			return
		}

		if part.FormName() != "update_file" {
			continue
		}

		fi, err := os.OpenFile("/overlay/robase/root/TMP_update-stratux-v.sh", os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0666)
		if err != nil {
			log.Printf("Update failed from %s (%s).\n", r.RemoteAddr, err.Error())
			return
		}
		defer fi.Close()
		_, err = io.Copy(fi, part)
		if err != nil {
			log.Printf("Update failed from %s (%s).\n", r.RemoteAddr, err.Error())
			return
		}

		break
	}

	
	os.Rename("/overlay/robase/root/TMP_update-stratux-v.sh", "/overlay/robase/root/update-stratux-v.sh")
	log.Printf("%s uploaded %s for update.\n", r.RemoteAddr, "/overlay/robase/root/update-stratux-v.sh")
	overlayctl("disable")
	// Successful update upload. Now reboot.
	go delayReboot()
}

func setNoCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

func setJSONHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
}

func defaultServer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "max-age=360") // 5 min, so that if user installs update, he will revalidate soon enough
	//	setNoCache(w)
	http.FileServer(http.Dir(STRATUX_WWW_DIR)).ServeHTTP(w, r)
}

func handleroPartitionRebuild(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("/usr/sbin/rebuild_ro_part.sh").Output()

	if err != nil {
		addSingleSystemErrorf("partition-rebuild", "Rebuild RO Partition error: %s", err.Error())
	} else {
		addSingleSystemErrorf("partition-rebuild", "Rebuild RO Partition success: %s", out)
	}

}

// https://gist.github.com/alexisrobert/982674.
// Copyright (c) 2010-2014 Alexis ROBERT <alexis.robert@gmail.com>.
const dirlisting_tpl = `<?xml version="1.0" encoding="iso-8859-1"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en">
<!-- Modified from lighttpd directory listing -->
<head>
<title>Index of {{.Name}}</title>
<style type="text/css">
a, a:active {text-decoration: none; color: blue;}
a:visited {color: #48468F;}
a:hover, a:focus {text-decoration: underline; color: red;}
body {background-color: #F5F5F5;}
h2 {margin-bottom: 12px;}
table {margin-left: 12px;}
th, td { font: 90% monospace; text-align: left;}
th { font-weight: bold; padding-right: 14px; padding-bottom: 3px;}
td {padding-right: 14px;}
td.s, th.s {text-align: right;}
div.list { background-color: white; border-top: 1px solid #646464; border-bottom: 1px solid #646464; padding-top: 10px; padding-bottom: 14px;}
div.foot { font: 90% monospace; color: #787878; padding-top: 4px;}
</style>
</head>
<body>
<h2>Index of {{.Name}}</h2>
<div class="list">
<table summary="Directory Listing" cellpadding="0" cellspacing="0">
<thead><tr><th class="n">Name</th><th>Last Modified</th><th>Size (bytes)</th><th class="dl">Options</th></tr></thead>
<tbody>
{{range .Children_files}}
<tr><td class="n"><a href="/logs/{{.Path}}">{{.Name}}</a></td><td>{{.Mtime}}</td><td>{{.Size}}</td><td class="dl"><a href="/logs/{{.Path}}">Download</a></td></tr>
{{end}}
</tbody>
</table>
</div>
<div class="foot">{{.ServerUA}}</div>
</body>
</html>`

type fileInfo struct {
	Name  string
	Path  string
	Mtime string
	Size  string
}

// Manages directory listings
type dirlisting struct {
	Name           string
	Children_files []fileInfo
	ServerUA       string
}

//FIXME: This needs to be switched to show a "sessions log" from the sqlite database.
func viewLogs(w http.ResponseWriter, r *http.Request) {
	urlpath := strings.TrimPrefix(r.URL.Path, "/logs/")
	path := "/var/log/" + urlpath
	finfo, err := os.Stat(path)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Failed to open %s: %s", path, err.Error())))
		return
	}

	if !finfo.IsDir() {
		http.ServeFile(w, r, path)
		return
	}
	
	names, err := ioutil.ReadDir(path)
	if err != nil {
		return
	}	

	fi := make([]fileInfo, 0)
	for _, val := range names {
		if val.Name()[0] == '.' {
			continue
		} // Remove hidden files from listing
		
		if val.IsDir() {
			mtime := val.ModTime().Format("2006-Jan-02 15:04:05")
			sz := ""
			fi = append(fi, fileInfo{Name: val.Name() + "/", Path: urlpath + "/" + val.Name(), Mtime: mtime, Size: sz})
		} else {
			mtime := val.ModTime().Format("2006-Jan-02 15:04:05")
			sz := humanize.Comma(val.Size())
			fi = append(fi, fileInfo{Name: val.Name(), Path: urlpath + "/" + val.Name(), Mtime: mtime, Size: sz})
		}
	}

	tpl, err := template.New("tpl").Parse(dirlisting_tpl)
	if err != nil {
		return
	}
	data := dirlisting{Name: r.URL.Path, ServerUA: "Stratux " + stratuxVersion + "/" + stratuxBuild,
		Children_files: fi}

	err = tpl.Execute(w, data)
	if err != nil {
		log.Printf("viewLogs() error: %s\n", err.Error())
	}

}

func connectMbTilesArchive(path string) (*sql.DB, map[string]string, error) {
	mbtileCacheLock.Lock()
	defer mbtileCacheLock.Unlock()
	if conn, ok := mbtileConnectionCache[path]; ok {
		if !conn.IsOutdated() {
			return conn.Conn, conn.Metadata, nil
		}
		log.Printf("Reloading MBTiles " + path)
	}

	conn, err := sql.Open("sqlite3", "file:" + path + "?mode=ro")
	if err != nil {
		return nil, nil, err
	}
	cacheEntry := NewMbTileConnectionCacheEntry(path, conn)
	cacheEntry.Metadata = readMbTilesMetadata(path, conn)
	if cacheEntry != nil {
		mbtileConnectionCache[path] = *cacheEntry
	}
	return conn, cacheEntry.Metadata, nil
}

func tileToDegree(z, x, y int) (lon, lat float64) {
	// osm-like schema:
	y = (1 << z) - y - 1
	n := math.Pi - 2.0*math.Pi*float64(y)/math.Exp2(float64(z))
	lat = 180.0 / math.Pi * math.Atan(0.5*(math.Exp(n)-math.Exp(-n)))
	lon = float64(x)/math.Exp2(float64(z))*360.0 - 180.0
	return lon, lat
}

func readMbTilesMetadata(fname string, db *sql.DB) map[string]string {
	rows, err := db.Query(`SELECT name, value FROM metadata 
		UNION SELECT 'minzoom', min(zoom_level) FROM tiles WHERE NOT EXISTS (SELECT * FROM metadata WHERE name='minzoom' and value is not null and value != '')
		UNION SELECT 'maxzoom', max(zoom_level) FROM tiles WHERE NOT EXISTS (SELECT * FROM metadata WHERE name='maxzoom' and value is not null and value != '')`);
	if err != nil {
		log.Printf("SQLite read error %s: %s", fname, err.Error())
		return nil
	}
	defer rows.Close()
	meta := make(map[string]string)
	for rows.Next() {
		var name, val string
		rows.Scan(&name, &val)
		if len(val) > 0 {
			meta[name] = val
		}
	}
	// determine extent of layer if not given.. Openlayers kinda needs this, or it can happen that it tries to do
	// a billion request do down-scale high-res pngs that aren't even there (i.e. all 404s)
	if _, ok := meta["bounds"]; !ok {
		maxZoomInt, _ := strconv.ParseInt(meta["maxzoom"], 10, 32)
		rows, err = db.Query("SELECT min(tile_column), min(tile_row), max(tile_column), max(tile_row) FROM tiles WHERE zoom_level=?", maxZoomInt)
		if err != nil {
			log.Printf("SQLite read error %s: %s", fname, err.Error())
			return nil
		}
		rows.Next()
		var xmin, ymin, xmax, ymax int
		rows.Scan(&xmin, &ymin, &xmax, &ymax)
		lonmin, latmin := tileToDegree(int(maxZoomInt), xmin, ymin)
		lonmax, latmax := tileToDegree(int(maxZoomInt), xmax+1, ymax+1)
		meta["bounds"] = fmt.Sprintf("%f,%f,%f,%f", lonmin, latmin, lonmax, latmax)
	}

	// check if it is vectortiles and we have a style, then add the URL to metadata...
	if format, ok := meta["format"]; ok && format == "pbf" {
		_, file := filepath.Split(fname)
		if _, err := os.Stat(STRATUX_HOME + "/mapdata/styles/" + file + "/style.json"); err == nil {
			// We found a style!
			meta["stratux_style_url"] = "/mapdata/styles/" + file + "/style.json"
		}
		
	}
	return meta
}

// Scans mapdata dir for all .db and .mbtiles files and returns json representation of all metadata values
func handleTilesets(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(STRATUX_HOME + "/mapdata/");
	if err != nil {
		log.Printf("handleTilesets() error: %s\n", err.Error())
		http.Error(w, err.Error(), 500)
	}
	result := make(map[string]map[string]string, 0)
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if strings.HasSuffix(f.Name(), ".mbtiles") || strings.HasSuffix(f.Name(), ".db") {
			_, meta, err := connectMbTilesArchive(STRATUX_HOME + "/mapdata/" + f.Name())
			if err != nil {
				log.Printf("SQLite open " + f.Name() + " failed: %s", err.Error())
				continue
			}
			result[f.Name()] = meta
		}
	}
	resJson, _ := json.Marshal(result)
	w.Write(resJson)
}

func loadTile(fname string, z, x, y int) ([]byte, error) {
	db, meta, err := connectMbTilesArchive(STRATUX_HOME + "/mapdata/" + fname)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query("SELECT tile_data FROM tiles WHERE zoom_level=? AND tile_column=? AND tile_row=?", z, x, y)
	if err != nil {
		log.Printf("Failed to query mbtiles: %s", err.Error())
		return nil, nil
	}
	
	defer rows.Close()
	for rows.Next() {
		var res []byte
		rows.Scan(&res)
		// sometimes pbfs are gzipped...
		if format, ok := meta["format"]; ok && format == "pbf" && len(res) >= 2 && res[0] == 0x1f && res[1] == 0x8b {
			reader := bytes.NewReader(res)
			gzreader, _ := gzip.NewReader(reader)
			unzipped, err := ioutil.ReadAll(gzreader)
			if err != nil {
				log.Printf("Failed to unzip gzipped PBF data")
				return nil, nil
			}
			res = unzipped
		}
		return res, nil
	}
	return nil, nil
}

func handleTile(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.RequestURI, "/")
	if len(parts) < 4 {
		return
	}
	idx := len(parts) - 1
	y, err := strconv.Atoi(strings.Split(parts[idx], ".")[0])
	if err != nil {
		http.Error(w, "Failed to parse y", 500)
		return
	}
	idx--
	x, _ := strconv.Atoi(parts[idx])
	idx--
	z, _ := strconv.Atoi(parts[idx])
	idx--
	file, _ := url.QueryUnescape(parts[idx])
	tileData, err := loadTile(file, z, x, y)
	if err != nil {
		http.Error(w, err.Error(), 500)
	} else if tileData == nil {
		http.Error(w, "Tile not found", 404)
	} else {
		w.Write(tileData)
	}
}

func managementInterface() {
	weatherUpdate = NewUIBroadcaster()
	trafficUpdate = NewUIBroadcaster()
	radarUpdate = NewUIBroadcaster()
	situationUpdate = NewUIBroadcaster()
	weatherRawUpdate = NewUIBroadcaster()
	gdl90Update = NewUIBroadcaster()

	http.HandleFunc("/", defaultServer)
	//http.Handle("/logs/", http.StripPrefix("/logs/", http.FileServer(http.Dir("/var/log"))))
	http.Handle("/mapdata/styles/", http.StripPrefix("/mapdata/styles/", http.FileServer(http.Dir(STRATUX_HOME + "/mapdata/styles"))))
	http.HandleFunc("/logs/", viewLogs)

	http.HandleFunc("/gdl90",
		func(w http.ResponseWriter, req *http.Request) {
			s := websocket.Server{
				Handler: websocket.Handler(handleGDL90WS)}
			s.ServeHTTP(w, req)
		})
	http.HandleFunc("/status",
		func(w http.ResponseWriter, req *http.Request) {
			s := websocket.Server{
				Handler: websocket.Handler(handleStatusWS)}
			s.ServeHTTP(w, req)
		})
	http.HandleFunc("/situation",
		func(w http.ResponseWriter, req *http.Request) {
			s := websocket.Server{
				Handler: websocket.Handler(handleSituationWS)}
			s.ServeHTTP(w, req)
		})
	http.HandleFunc("/weather",
		func(w http.ResponseWriter, req *http.Request) {
			s := websocket.Server{
				Handler: websocket.Handler(handleWeatherWS)}
			s.ServeHTTP(w, req)
		})
	http.HandleFunc("/traffic",
		func(w http.ResponseWriter, req *http.Request) {
			s := websocket.Server{
				Handler: websocket.Handler(handleTrafficWS)}
			s.ServeHTTP(w, req)
		})
	http.HandleFunc("/radar",
		func(w http.ResponseWriter, req *http.Request) {
			s := websocket.Server{
				Handler: websocket.Handler(handleRadarWS)}
			s.ServeHTTP(w, req)
		})


	http.HandleFunc("/jsonio",
		func(w http.ResponseWriter, req *http.Request) {
			s := websocket.Server{
				Handler: websocket.Handler(handleJsonIo)}
			s.ServeHTTP(w, req)
		})

	http.HandleFunc("/getStatus", handleStatusRequest)
	http.HandleFunc("/getSituation", handleSituationRequest)
	http.HandleFunc("/getTowers", handleTowersRequest)
	http.HandleFunc("/getSatellites", handleSatellitesRequest)
	http.HandleFunc("/getSettings", handleSettingsGetRequest)
	http.HandleFunc("/setSettings", handleSettingsSetRequest)
	http.HandleFunc("/restart", handleRestartRequest)
	http.HandleFunc("/shutdown", handleShutdownRequest)
	http.HandleFunc("/reboot", handleRebootRequest)
	http.HandleFunc("/getClients", handleClientsGetRequest)
	http.HandleFunc("/updateUpload", handleUpdatePostRequest)
	http.HandleFunc("/roPartitionRebuild", handleroPartitionRebuild)
	http.HandleFunc("/develmodetoggle", handleDevelModeToggle)
	http.HandleFunc("/orientAHRS", handleOrientAHRS)
	http.HandleFunc("/calibrateAHRS", handleCalibrateAHRS)
	http.HandleFunc("/cageAHRS", handleCageAHRS)
	http.HandleFunc("/resetGMeter", handleResetGMeter)
	http.HandleFunc("/deletelogfile", handleDeleteLogFile)
	http.HandleFunc("/downloadlog", handleDownloadLogRequest)
	http.HandleFunc("/deleteahrslogfiles", handleDeleteAHRSLogFiles)
	http.HandleFunc("/downloadahrslogs", handleDownloadAHRSLogsRequest)
	http.HandleFunc("/downloaddb", handleDownloadDBRequest)
	http.HandleFunc("/tiles/tilesets", handleTilesets)
	http.HandleFunc("/tiles/", handleTile)

	var addr string
	if common.IsRunningAsRoot() {
		addr = managementAddr
	} else {
		addr = ":8000" // Make sure we can run without root priviledges on different port
	}

	if err :=http.ListenAndServe(addr, nil); err != nil {
		log.Printf("managementInterface ListenAndServe: %s\n", err.Error())
	}
}
