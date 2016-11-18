package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/net/websocket"
	"net"
	"os"
	"time"
)

const (
	configLocation = "./gdl90_websocket.conf"
)

type dumpdata struct {
	data string
}

type HostsType struct {
	Address string
	Port    int
}

type settings struct {
	Source string
	//Hosts   []HostsType
	Host string
	Port int
}

var globalSettings settings
var Verbose bool

func defaultSettings(confFile string) {
	globalSettings.Source = "192.168.0.1"
	/*
		    globalSettings.Hosts = []HostsType{
				{ Address: "192.168.0.1", Port: 4000 },
			}
	*/
	globalSettings.Host = "192.168.0.1"
	globalSettings.Port = 4000
	saveSettings(confFile)
}

func readSettings(confFile string) {
	fd, err := os.Open(confFile)
	if err != nil {
		fmt.Printf("can't read settings %s: %s\n", confFile, err.Error())
		defaultSettings(confFile)
		return
	}
	defer fd.Close()
	buf := make([]byte, 1024)
	count, err := fd.Read(buf)
	if err != nil {
		fmt.Printf("can't read settings %s: %s\n", confFile, err.Error())
		defaultSettings(confFile)
		return
	}
	var newSettings settings
	err = json.Unmarshal(buf[0:count], &newSettings)
	if err != nil {
		fmt.Printf("can't read settings %s: %s\n", confFile, err.Error())
		defaultSettings(confFile)
		return
	}
	globalSettings = newSettings
	fmt.Printf("read in settings.\n")
}

func saveSettings(confFile string) {
	fd, err := os.OpenFile(configLocation, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0644))
	if err != nil {
		err_ret := fmt.Errorf("can't save settings %s: %s", configLocation, err.Error())
		fmt.Printf("%s\n", err_ret.Error())
		return
	}
	defer fd.Close()
	jsonSettings, _ := json.Marshal(&globalSettings)
	fd.Write(jsonSettings)
	fmt.Printf("wrote settings.\n")
}

func Usage() {
    fmt.Printf(`Usage:
The configuration file contains the source and destination IP and port. Edit
the configuration file directly to modify the source and destination hosts

    `)
}

func main() {
	confFile := flag.String("config", configLocation, "Specify custom config file")
	Verb := flag.Bool("verbose", false, "Set verbose - show received messages")
    HelpFlag := flag.Bool("info", false, "Show detailed help")
	flag.Parse()
	Verbose = *Verb
    if *HelpFlag {
        Usage()
        os.Exit(0)
    }
	readSettings(*confFile)
	initWebsocketClient(globalSettings.Source)
}

func initWebsocketClient(srcHost string) {

	fmt.Printf("Starting Client, trying to connect to %s\n", srcHost)
	ws, err := websocket.Dial(fmt.Sprintf("ws://%s/gdl90", srcHost), "", fmt.Sprintf("http://%s/gdl90", srcHost))
	if err != nil {
		fmt.Printf("Dial failed: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("Connected to %s\n", srcHost)

	dest := fmt.Sprintf("%s:%d", globalSettings.Host, globalSettings.Port)
	fmt.Printf("Destination %s\n", dest)
	udp, err := net.Dial("udp", dest)
	if err != nil {
		fmt.Printf("Dial failed: %s\n", err.Error())
	}

	incomingMessages := make(chan string)
	go readClientMessages(ws, incomingMessages)
	i := 0
	for {
		select {
		case <-time.After(time.Duration(2e9)):
			i++
		case message := <-incomingMessages:
			var d []byte
			err := json.Unmarshal([]byte(message), &d)
			if err != nil {
				fmt.Printf("json.Unmarshal(): %s\n", err.Error())
				continue
			}
			if Verbose {
				fmt.Printf(hex.Dump(d))
			}
			_, err2 := udp.Write(d)
			if err2 != nil {
				fmt.Printf("Write failed: %s\n", err2.Error())
			}
		}
	}
}

func readClientMessages(ws *websocket.Conn, incomingMessages chan string) {
	for {
		var message string
		// err := websocket.JSON.Receive(ws, &message)
		err := websocket.Message.Receive(ws, &message)
		if err != nil {
			fmt.Printf("Error::: %s\n", err.Error())
			return
		}
		incomingMessages <- message
	}
}
