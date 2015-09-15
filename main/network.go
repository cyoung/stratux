package main

import (
	"golang.org/x/net/icmp"
	"golang.org/x/net/internal/iana"
	"golang.org/x/net/ipv4"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
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
	Conn         *net.UDPConn
	Ip           string
	Port         uint32
	Capability   uint8
	messageQueue [][]byte // Device message queue.
	/*
		Sleep mode/throttle variables. "sleep mode" is actually now just a very reduced packet rate, since we don't know positively
		 when a client is ready to accept packets - we just assume so if we don't receive ICMP Unreachable packets in 5 secs.
	*/
	lastUnreachable time.Time // Last time the device sent an ICMP Unreachable packet.
	nextMessageTime time.Time // The next time that the device is "able" to receive a message.
}

var messageQueue chan networkMessage
var outSockets map[string]networkConnection
var dhcpLeases map[string]string
var netMutex *sync.Mutex

var pingResponse map[string]time.Time // Last time an IP responded to an "echo" response.

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

func isSleeping(k string) bool {
	ipAndPort := strings.Split(k, ":")
	lastPing, ok := pingResponse[ipAndPort[0]]
	// No ping response. Assume disconnected/sleeping device.
	if !ok || time.Since(lastPing) > (10*time.Second) {
		return true
	}
	if time.Since(outSockets[k].lastUnreachable) < (5 * time.Second) {
		return true
	}
	return false
}

// Throttle mode for testing port open and giving some start-up time to the app.
// Throttling is 0.1% data rate for first 15 seconds.
func isThrottled(k string) bool {
	return (rand.Int()%1000 != 0) && time.Since(outSockets[k].lastUnreachable) < (15*time.Second)
}

func sendToAllConnectedClients(msg networkMessage) {
	netMutex.Lock()
	defer netMutex.Unlock()
	for k, netconn := range outSockets {
		// Check if this port is able to accept the type of message we're sending.
		if (netconn.Capability & msg.msgType) == 0 {
			continue
		}
		// Send non-queueable messages immediately, or discard if the client is in sleep mode.
		if !msg.queueable {
			if !isSleeping(k) {
				netconn.Conn.Write(msg.msg) // Write immediately.
			} else {
				log.Printf("sleepy %s\n", k)
			}
		} else {
			// Queue the message if the message is "queueable".
			if len(netconn.messageQueue) >= maxUserMsgQueueSize { // Too many messages queued? Drop the oldest.
				log.Printf("%s:%d - message queue overflow.\n", netconn.Ip, netconn.Port)
				netconn.messageQueue = netconn.messageQueue[1 : maxUserMsgQueueSize-1]
			}
			netconn.messageQueue = append(netconn.messageQueue, msg.msg)
			outSockets[k] = netconn
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
				outSockets[ipAndPort] = networkConnection{Conn: outConn, Ip: ip, Port: networkOutput.Port, Capability: networkOutput.Capability, messageQueue: newq}
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

func messageQueueSender() {
	secondTimer := time.NewTicker(5 * time.Second)
	queueTimer := time.NewTicker(400 * time.Microsecond) // 2500 msg/sec
	for {
		select {
		case msg := <-messageQueue:
			sendToAllConnectedClients(msg)
		case <-queueTimer.C:
			netMutex.Lock()
			for k, netconn := range outSockets {
				if len(netconn.messageQueue) > 0 && !isSleeping(k) && !isThrottled(k) {
					tmpConn := netconn
					tmpConn.Conn.Write(tmpConn.messageQueue[0])
					tmpConn.messageQueue = tmpConn.messageQueue[1:]
					outSockets[k] = tmpConn
				}
			}
			netMutex.Unlock()
		case <-secondTimer.C:
			getNetworkStats()
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

func icmpEchoSender(c *icmp.PacketConn) {
	timer := time.NewTicker(5 * time.Second)
	for {
		<-timer.C
		// Collect IPs.
		ips := make(map[string]bool)
		for k, _ := range outSockets {
			ipAndPort := strings.Split(k, ":")
			ips[ipAndPort[0]] = true
		}
		// Send to all IPs.
		for ip, _ := range ips {
			wm := icmp.Message{
				Type: ipv4.ICMPTypeEcho, Code: 0,
				Body: &icmp.Echo{
					ID: os.Getpid() & 0xffff, Seq: 1,
					Data: []byte("STRATUX"),
				},
			}
			wb, err := wm.Marshal(nil)
			if err != nil {
				log.Printf("couldn't send ICMP Echo: %s\n", err.Error())
				continue
			}
			if _, err := c.WriteTo(wb, &net.IPAddr{IP: net.ParseIP(ip)}); err != nil {
				log.Printf("couldn't send ICMP Echo: %s\n", err.Error())
				continue
			}
		}
	}
}

// Monitor clients going in/out of sleep mode via ICMP unreachable packets.
func sleepMonitor() {
	c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		log.Printf("error listening for udp - sending data to all ports for all connected clients. err: %s", err)
		return
	}
	go icmpEchoSender(c)
	defer c.Close()
	for {
		buf := make([]byte, 1500)
		n, peer, err := c.ReadFrom(buf)
		if err != nil {
			log.Printf("%s\n", err.Error())
			continue
		}
		msg, err := icmp.ParseMessage(iana.ProtocolICMP, buf[:n])
		if err != nil {
			continue
		}

		ip := peer.String()

		// Look for echo replies, mark it as received.
		if msg.Type == ipv4.ICMPTypeEchoReply {
			pingResponse[ip] = time.Now()
			continue // No further processing needed.
		}

		// Only deal with ICMP Unreachable packets (since that's what iOS and Android seem to be sending whenever the apps are not available).
		if msg.Type != ipv4.ICMPTypeDestinationUnreachable {
			continue
		}
		// Packet parsing.
		mb, err := msg.Body.Marshal(iana.ProtocolICMP)
		if err != nil {
			continue
		}
		if len(mb) < 28 {
			continue
		}

		// The unreachable port.
		port := (uint16(mb[26]) << 8) | uint16(mb[27])
		ipAndPort := ip + ":" + strconv.Itoa(int(port))

		netMutex.Lock()
		p, ok := outSockets[ipAndPort]
		if !ok {
			// Can't do anything, the client isn't even technically connected.
			netMutex.Unlock()
			continue
		}
		p.lastUnreachable = time.Now()
		outSockets[ipAndPort] = p
		netMutex.Unlock()
	}
}

func initNetwork() {
	messageQueue = make(chan networkMessage, 1024) // Buffered channel, 1024 messages.
	outSockets = make(map[string]networkConnection)
	pingResponse = make(map[string]time.Time)
	netMutex = &sync.Mutex{}
	refreshConnectedClients()
	go monitorDHCPLeases()
	go messageQueueSender()
	go sleepMonitor()
}
