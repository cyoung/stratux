/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New"" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	network.go: Client networking routines, DHCP lease monitoring, queue management, ICMP monitoring.
*/

package main

import (
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"io/ioutil"
	"log"
	"math"
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
	LastUnreachable time.Time // Last time the device sent an ICMP Unreachable packet.
	nextMessageTime time.Time // The next time that the device is "able" to receive a message.
	numOverflows    uint32    // Number of times the queue has overflowed - for calculating the amount to chop off from the queue.
	SleepFlag       bool      // Whether or not this client has been marked as sleeping - only used for debugging (relies on messages being sent to update this flag in sendToAllConnectedClients()).
}

var messageQueue chan networkMessage
var outSockets map[string]networkConnection
var dhcpLeases map[string]string
var netMutex *sync.Mutex

var totalNetworkMessagesSent uint32

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
	if !ok || stratuxClock.Since(lastPing) > (10*time.Second) {
		return true
	}
	if stratuxClock.Since(outSockets[k].LastUnreachable) < (5 * time.Second) {
		return true
	}
	return false
}

// Throttle mode for testing port open and giving some start-up time to the app.
// Throttling is 0.1% data rate for first 15 seconds.
func isThrottled(k string) bool {
	return (rand.Int()%1000 != 0) && stratuxClock.Since(outSockets[k].LastUnreachable) < (15*time.Second)
}

func sendToAllConnectedClients(msg networkMessage) {
	netMutex.Lock()
	defer netMutex.Unlock()
	for k, netconn := range outSockets {
		sleepFlag := isSleeping(k)

		netconn.SleepFlag = sleepFlag
		outSockets[k] = netconn

		// Check if this port is able to accept the type of message we're sending.
		if (netconn.Capability & msg.msgType) == 0 {
			continue
		}
		// Send non-queueable messages immediately, or discard if the client is in sleep mode.

		if !sleepFlag {
			netconn.numOverflows = 0 // Reset the overflow counter whenever the client is not sleeping so that we're not penalizing future sleepmodes.
		}

		if !msg.queueable {
			if sleepFlag {
				continue
			}
			netconn.Conn.Write(msg.msg) // Write immediately.
			totalNetworkMessagesSent++
			globalStatus.NetworkDataMessagesSent++
			globalStatus.NetworkDataMessagesSentNonqueueable++
			globalStatus.NetworkDataBytesSent += uint64(len(msg.msg))
			globalStatus.NetworkDataBytesSentNonqueueable += uint64(len(msg.msg))
		} else {
			// Queue the message if the message is "queueable".
			if len(netconn.messageQueue) >= maxUserMsgQueueSize { // Too many messages queued? Drop the oldest.
				log.Printf("%s:%d - message queue overflow.\n", netconn.Ip, netconn.Port)
				netconn.numOverflows++
				s := 2 * netconn.numOverflows // Double the amount we chop off on each overflow.
				if int(s) >= len(netconn.messageQueue) {
					netconn.messageQueue = make([][]byte, 0)
				} else {
					netconn.messageQueue = netconn.messageQueue[s:]
				}
			}
			netconn.messageQueue = append(netconn.messageQueue, msg.msg) // each netconn.messageQueue is therefore an array (well, a slice) of formatted GDL90 messages
			outSockets[k] = netconn
		}
	}
}

// Returns the number of DHCP leases and prints queue lengths
func getNetworkStats() uint {
	for _, netconn := range outSockets {
		queueBytes := 0
		for _, msg := range netconn.messageQueue {
			queueBytes += len(msg)
		}
		log.Printf("On  %s:%d,  Queue length = %d messages / %d bytes\n", netconn.Ip, netconn.Port, len(netconn.messageQueue), queueBytes)
	}

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
	secondTimer := time.NewTicker(15 * time.Second)
	queueTimer := time.NewTicker(100 * time.Millisecond)

	var lastQueueTimeChange time.Time // Reevaluate	send frequency every 5 seconds.
	for {
		select {
		case msg := <-messageQueue:
			sendToAllConnectedClients(msg)
		case <-queueTimer.C:
			netMutex.Lock()

			averageSendableQueueSize := float64(0.0)
			for k, netconn := range outSockets {
				if len(netconn.messageQueue) > 0 && !isSleeping(k) && !isThrottled(k) {
					averageSendableQueueSize += float64(len(netconn.messageQueue)) // Add num sendable messages.

					var queuedMsg []byte

					// Combine the first 256 entries in netconn.messageQueue to avoid flooding wlan0 with too many IOPS.
					// Need to play nice with non-queued messages, so this limits the number of entries to combine.
					// UAT uplink block is 432 bytes, so transmit block size shouldn't be larger than 108 KiB. 10 Mbps per device would therefore be needed to send within a 100 ms window.

					mqDepth := len(netconn.messageQueue)
					if mqDepth > 256 {
						mqDepth = 256
					}

					for j := 0; j < mqDepth; j++ {
						queuedMsg = append(queuedMsg, netconn.messageQueue[j]...)
					}

					/*
						for j, _ := range netconn.messageQueue {
							queuedMsg = append(queuedMsg, netconn.messageQueue[j]...)
						}
					*/

					netconn.Conn.Write(queuedMsg)
					totalNetworkMessagesSent++
					globalStatus.NetworkDataMessagesSent++
					globalStatus.NetworkDataBytesSent += uint64(len(queuedMsg))

					//netconn.messageQueue = [][]byte{}
					if mqDepth < len(netconn.messageQueue) {
						netconn.messageQueue = netconn.messageQueue[mqDepth:]
					} else {
						netconn.messageQueue = [][]byte{}
					}
					outSockets[k] = netconn

					/*
						tmpConn := netconn
						tmpConn.Conn.Write(tmpConn.messageQueue[0])
						totalNetworkMessagesSent++
						globalStatus.NetworkDataMessagesSent++
						globalStatus.NetworkDataBytesSent += uint64(len(tmpConn.messageQueue[0]))
						tmpConn.messageQueue = tmpConn.messageQueue[1:]
						outSockets[k] = tmpConn
					*/
				}
			}

			if stratuxClock.Since(lastQueueTimeChange) >= 5*time.Second {
				var pd float64
				if averageSendableQueueSize > 0.0 && len(outSockets) > 0 {
					averageSendableQueueSize = averageSendableQueueSize / float64(len(outSockets)) // It's a total, not an average, up until this point.
					pd = math.Max(float64(1.0/750.0), float64(1.0/(4.0*averageSendableQueueSize))) // Say 250ms is enough to get through the whole queue.
				} else {
					pd = float64(1.0 / 0.1) // 100ms.
				}
				queueTimer.Stop()
				queueTimer = time.NewTicker(time.Duration(pd*1000000000.0) * time.Nanosecond)
				lastQueueTimeChange = stratuxClock.Time
			}
			netMutex.Unlock()
		case <-secondTimer.C:
			getNetworkStats()
		}
	}
}

func sendMsg(msg []byte, msgType uint8, queueable bool) {
	messageQueue <- networkMessage{msg: msg, msgType: msgType, queueable: queueable, ts: stratuxClock.Time}
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
			totalNetworkMessagesSent++
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
		msg, err := icmp.ParseMessage(1, buf[:n])
		if err != nil {
			continue
		}

		ip := peer.String()

		// Look for echo replies, mark it as received.
		if msg.Type == ipv4.ICMPTypeEchoReply {
			pingResponse[ip] = stratuxClock.Time
			continue // No further processing needed.
		}

		// Only deal with ICMP Unreachable packets (since that's what iOS and Android seem to be sending whenever the apps are not available).
		if msg.Type != ipv4.ICMPTypeDestinationUnreachable {
			continue
		}
		// Packet parsing.
		mb, err := msg.Body.Marshal(1)
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
		p.LastUnreachable = stratuxClock.Time
		outSockets[ipAndPort] = p
		netMutex.Unlock()
	}
}

func networkStatsCounter() {
	timer := time.NewTicker(1 * time.Second)
	var previousNetworkMessagesSent, previousNetworkBytesSent, previousNetworkMessagesSentNonqueueable, previousNetworkBytesSentNonqueueable uint64

	for {
		<-timer.C
		globalStatus.NetworkDataMessagesSentLastSec = globalStatus.NetworkDataMessagesSent - previousNetworkMessagesSent
		globalStatus.NetworkDataBytesSentLastSec = globalStatus.NetworkDataBytesSent - previousNetworkBytesSent
		globalStatus.NetworkDataMessagesSentNonqueueableLastSec = globalStatus.NetworkDataMessagesSentNonqueueable - previousNetworkMessagesSentNonqueueable
		globalStatus.NetworkDataBytesSentNonqueueableLastSec = globalStatus.NetworkDataBytesSentNonqueueable - previousNetworkBytesSentNonqueueable

		// debug option. Uncomment to log per-second network statistics. Useful for debugging WiFi instability.
		//log.Printf("Network data messages sent: %d total, %d last second.  Network data bytes sent: %d total, %d last second.\n", globalStatus.NetworkDataMessagesSent, globalStatus.NetworkDataMessagesSentLastSec, globalStatus.NetworkDataBytesSent, globalStatus.NetworkDataBytesSentLastSec)

		previousNetworkMessagesSent = globalStatus.NetworkDataMessagesSent
		previousNetworkBytesSent = globalStatus.NetworkDataBytesSent
		previousNetworkMessagesSentNonqueueable = globalStatus.NetworkDataMessagesSentNonqueueable
		previousNetworkBytesSentNonqueueable = globalStatus.NetworkDataBytesSentNonqueueable

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
	go networkStatsCounter()
}
