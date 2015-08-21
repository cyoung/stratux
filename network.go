package main

import (
	"golang.org/x/exp/inotify"
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	dhcp_lease_file = "/var/lib/dhcp/dhcpd.leases"
)

var messageQueue chan []byte
var outSockets map[string]*net.UDPConn
var dhcpLeases map[string]string
var netMutex *sync.Mutex

// Read the "dhcpd.leases" file and parse out IP/hostname.
func getDHCPLeases() (map[string]string, error) {
	dat, err := ioutil.ReadFile(dhcp_lease_file)
	ret := make(map[string]string)
	if err != nil {
		return ret, err
	}
	lines := strings.Split(string(dat), "\n")
	open_block := false
	block_ip := ""
	for _, line := range lines {
		spaced := strings.Split(line, " ")
		if len(spaced) > 2 && spaced[0] == "lease" {
			open_block = true
			block_ip = spaced[1]
		} else if open_block && len(spaced) >= 4 && spaced[2] == "client-hostname" {
			hostname := strings.TrimRight(strings.TrimLeft(strings.Join(spaced[3:], " "), "\""), "\";")
			ret[block_ip] = hostname
			open_block = false
		}
	}
	return ret, nil
}

func sendToAllConnectedClients(msg []byte) {
	netMutex.Lock()
	defer netMutex.Unlock()
	for _, sock := range outSockets {
		sock.Write(msg)
	}
}

// Just returns the number of DHCP leases for now.
func getNetworkStats() int {
	return len(dhcpLeases)
}

// See who has a DHCP lease and make a UDP connection to each of them.
func refreshConnectedClients() {
	netMutex.Lock()
	defer netMutex.Unlock()
	validConnections := make(map[string]bool)
	t, err := getDHCPLeases()
	if err != nil {
		log.Printf("getDHCPLeases(): %s\n", err.Error())
		return
	}
	dhcpLeases = t
	// Client connected that wasn't before.
	for ip, hostname := range dhcpLeases {
		for _, port := range globalSettings.GDLOutputPorts {
			ipAndPort := ip + ":" + strconv.Itoa(int(port))
			if _, ok := outSockets[ipAndPort]; !ok {
				log.Printf("client connected: %s:%d (%s).\n", ip, port, hostname)
				addr, err := net.ResolveUDPAddr("udp", ipAndPort)
				if err != nil {
					log.Printf("ResolveUDPAddr(%s): %s\n", ipAndPort, err.Error())
					continue
				}
				outConn, err := net.DialUDP("udp", nil, addr)
				if err != nil {
					log.Printf("DialUDP(%s): %s\n", ipAndPort, err.Error())
					continue
				}
				outSockets[ipAndPort] = outConn
			}
			validConnections[ipAndPort] = true
		}
	}
	// Client that was connected before that isn't.
	for ipAndPort, conn := range outSockets {
		if _, ok := validConnections[ipAndPort]; !ok {
			log.Printf("removed connection %s.\n", ipAndPort)
			conn.Close()
			delete(outSockets, ipAndPort)
		}
	}
}

func messageQueueSender() {
	secondTimer := time.NewTicker(1 * time.Second)
	for {
		select {
		case msg := <-messageQueue:
			sendToAllConnectedClients(msg)
		case <-secondTimer.C:
			getNetworkStats()
		}

	}
}

func monitorDHCPLeases() {
	watcher, err := inotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	err = watcher.AddWatch(dhcp_lease_file, inotify.IN_CLOSE_WRITE)
	if err != nil {
		log.Fatal(err)
	}
	for {
		select {
		case <-watcher.Event:
			log.Println("file modified, attempting to refresh DHCP")
			refreshConnectedClients()
		case err := <-watcher.Error:
			log.Println("error with DHCP file system watcher:", err)
		}
	}
}

func sendMsg(msg []byte) {
	messageQueue <- msg
}

func initNetwork() {
	messageQueue = make(chan []byte, 1024) // Buffered channel, 1024 messages.
	outSockets = make(map[string]*net.UDPConn)
	netMutex = &sync.Mutex{}
	refreshConnectedClients()
	go monitorDHCPLeases()
	go messageQueueSender()
}
