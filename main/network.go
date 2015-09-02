package main

import (
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type networkMessage struct {
	msg       []byte
	msgType   uint8
	queueable bool
	ts        time.Time
}

type networkConnection struct {
	Conn       *net.UDPConn
	Ip         string
	Port       uint32
	Capability uint8
	sleepMode  bool     // Device is not able to receive messages currently.
	sleepQueue [][]byte // Device message queue.
}

var messageQueue chan networkMessage
var outSockets map[string]networkConnection
var dhcpLeases map[string]string
var netMutex *sync.Mutex

const (
	NETWORK_GDL90_STANDARD = 1
	NETWORK_AHRS_FFSIM     = 2
	NETWORK_AHRS_GDL90     = 4
	dhcp_lease_file        = "/var/lib/dhcp/dhcpd.leases"
)

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
		} else if open_block && strings.HasPrefix(spaced[0], "}") { // No hostname.
			open_block = false
			ret[block_ip] = ""
		}
	}
	return ret, nil
}

func sendToAllConnectedClients(msg networkMessage) {
	netMutex.Lock()
	defer netMutex.Unlock()
	for k, netconn := range outSockets {
		if (netconn.Capability & msg.msgType) != 0 { // Check if this port is able to accept the type of message we're sending.
			// Check if the client is in sleep mode.
			if !netconn.sleepMode { // Write immediately.
				netconn.Conn.Write(msg.msg)
			} else if msg.queueable { // Queue the message if the message is "queueable". Discard otherwise.
				if len(netconn.sleepQueue) >= maxUserMsgQueueSize { // Too many messages queued? Client has been asleep for too long. Drop the oldest.
					log.Printf("%s:%d - message queue overflow.\n", netconn.Ip, netconn.Port)
					netconn.sleepQueue = netconn.sleepQueue[1:maxUserMsgQueueSize-1]
				}
				netconn.sleepQueue = append(netconn.sleepQueue, msg.msg)
				outSockets[k] = netconn
			}
		}
	}
}

// Just returns the number of DHCP leases for now.
func getNetworkStats() uint {
	ret := uint(len(dhcpLeases))
	globalStatus.Connected_Users = ret
	return ret
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
		for _, networkOutput := range globalSettings.NetworkOutputs {
			ipAndPort := ip + ":" + strconv.Itoa(int(networkOutput.Port))
			if _, ok := outSockets[ipAndPort]; !ok {
				log.Printf("client connected: %s:%d (%s).\n", ip, networkOutput.Port, hostname)
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
				newq := make([][]byte, 0)
				sleepMode := false
				if networkOutput.Port == 4000 { // Start off FF in sleep mode until something is received.
					sleepMode = true
				}
				outSockets[ipAndPort] = networkConnection{Conn: outConn, Ip: ip, Port: networkOutput.Port, Capability: networkOutput.Capability, sleepMode: sleepMode, sleepQueue: newq}
			}
			validConnections[ipAndPort] = true
		}
	}
	// Client that was connected before that isn't.
	for ipAndPort, conn := range outSockets {
		if _, ok := validConnections[ipAndPort]; !ok {
			log.Printf("removed connection %s.\n", ipAndPort)
			conn.Conn.Close()
			delete(outSockets, ipAndPort)
		}
	}
}

func checkMessageQueues() {
	netMutex.Lock()
	defer netMutex.Unlock()
	for k, netconn := range outSockets {
		if len(netconn.sleepQueue) > 0 && !netconn.sleepMode {
			// Empty the sleep queue.
			tmpQueue := netconn.sleepQueue
			tmpConn := netconn.Conn
			go func() {
				time.Sleep(5 * time.Second) // Give it time to start listening, some apps send the "woke up" message too quickly.
				for _, msg := range tmpQueue {
					tmpConn.Write(msg)
					time.Sleep(50 * time.Millisecond) // Slow down the sending, 20/sec.
				}
			}()
			netconn.sleepQueue = make([][]byte, 0)
			outSockets[k] = netconn
			log.Printf("%s - emptied %d in queue.\n", k, len(tmpQueue))
		}
	}
}

func messageQueueSender() {
	secondTimer := time.NewTicker(5 * time.Second)
	for {
		select {
		case msg := <-messageQueue:
			sendToAllConnectedClients(msg)
		case <-secondTimer.C:
			getNetworkStats()
			checkMessageQueues()
		}
	}
}

func sendMsg(msg []byte, msgType uint8, queueable bool) {
	messageQueue <- networkMessage{msg: msg, msgType: msgType, queueable: queueable, ts: time.Now()}
}

func sendGDL90(msg []byte, queueable bool) {
	sendMsg(msg, NETWORK_GDL90_STANDARD, queueable)
}

func monitorDHCPLeases() {
	//TODO: inotify or dhcp event hook.
	timer := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-timer.C:
			refreshConnectedClients()
		}
	}
}

// Monitor clients going in/out of sleep mode. This will be different for different apps.
func sleepMonitor() {
	// FF sleep mode.
	addr := net.UDPAddr{Port: 50113, IP: net.ParseIP("0.0.0.0")}
	conn, err := net.ListenUDP("udp", &addr)
	defer conn.Close()
	if err != nil {
		log.Printf("err: %s\n", err.Error())
		log.Printf("error listening on port 50113 (FF comm) - assuming FF is always awake (if connected).\n")
		return
	}
	for {
		buf := make([]byte, 1024)
		n, addr, err := conn.ReadFrom(buf)
		ipAndPort := strings.Split(addr.String(), ":")
		ip := ipAndPort[0]
		if err != nil {
			log.Printf("err: %s\n", err.Error())
			return
		}
		// Got message, check if it's in the correct format.
		if n < 3 || buf[0] != 0xFF || buf[1] != 0xFE {
			continue
		}
		s := string(buf[2:n])
		s = strings.Replace(s, "\x00", "", -1)
		ffIpAndPort := ip + ":4000"
		netMutex.Lock()
		p, ok := outSockets[ffIpAndPort]
		if !ok {
			// Can't do anything, the client isn't even technically connected.
			netMutex.Unlock()
			continue
		}
		if strings.HasPrefix(s, "i-want-to-play-ffm-udp") || strings.HasPrefix(s, "i-can-play-ffm-udp") {
			if p.sleepMode {
				log.Printf("%s - woke up\n", ffIpAndPort)
				p.sleepMode = false
			}
		} else if strings.HasPrefix(s, "i-cannot-play-ffm-udp") {
			if !p.sleepMode {
				log.Printf("%s - went to sleep\n", ffIpAndPort)
				p.sleepMode = true
			}
		}
		outSockets[ffIpAndPort] = p
		netMutex.Unlock()
	}
}

func initNetwork() {
	messageQueue = make(chan networkMessage, 1024) // Buffered channel, 1024 messages.
	outSockets = make(map[string]networkConnection)
	netMutex = &sync.Mutex{}
	refreshConnectedClients()
	go monitorDHCPLeases()
	go messageQueueSender()
	go sleepMonitor()
}
