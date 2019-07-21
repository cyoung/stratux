/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	network.go: Client networking routines, DHCP lease monitoring, queue management, ICMP monitoring.
*/

package main

import (
	"github.com/tarm/serial"
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
	Conn            *net.UDPConn
	Ip              string
	Port            uint32
	Capability      uint8
	messageQueue    [][]byte // Device message queue.
	MessageQueueLen int      // Length of the message queue. For debugging.
	/*
		Sleep mode/throttle variables. "sleep mode" is actually now just a very reduced packet rate, since we don't know positively
		 when a client is ready to accept packets - we just assume so if we don't receive ICMP Unreachable packets in 5 secs.
	*/
	LastUnreachable time.Time // Last time the device sent an ICMP Unreachable packet.
	nextMessageTime time.Time // The next time that the device is "able" to receive a message.
	numOverflows    uint32    // Number of times the queue has overflowed - for calculating the amount to chop off from the queue.
	SleepFlag       bool      // Whether or not this client has been marked as sleeping - only used for debugging (relies on messages being sent to update this flag in sendToAllConnectedClients()).
	FFCrippled      bool
}

type serialConnection struct {
	DeviceString string
	Baud         int
	serialPort   *serial.Port
}

var messageQueue chan networkMessage

var outSockets map[string]networkConnection
var dhcpLeases map[string]string
var pingResponse map[string]time.Time // Last time an IP responded to an "echo" response.
var netMutex *sync.Mutex              // netMutex needs to be locked before accessing dhcpLeases, pingResponse, and outSockets and calling isSleeping() and isThrottled().

var totalNetworkMessagesSent uint32

const (
	NETWORK_GDL90_STANDARD = 1
	NETWORK_AHRS_FFSIM     = 2
	NETWORK_AHRS_GDL90     = 4
	NETWORK_FLARM_NMEA     = 8
	dhcp_lease_file        = "/var/lib/dhcp/dhcpd.leases"
	dhcp_lease_dir         = "/var/lib/dhcp"
	extra_hosts_file       = "/etc/stratux-static-hosts.conf"
)

var dhcpLeaseDirectoryLastTest time.Time // Last time fsWriteTest() was run on the DHCP lease directory.

// Read the "dhcpd.leases" file and parse out IP/hostname.
func getDHCPLeases() (map[string]string, error) {
	// Do a write test. Even if we are able to read the file, it may be out of date because there's a fs write issue.
	// Only perform the test once every 5 minutes to minimize writes.
	if stratuxClock.Since(dhcpLeaseDirectoryLastTest) >= 5*time.Minute {
		err := fsWriteTest(dhcp_lease_dir)
		if err != nil {
			addSingleSystemErrorf("fs-write", "Write error on '%s', your EFB may have issues receiving weather and traffic.", dhcp_lease_dir)
		}
		dhcpLeaseDirectoryLastTest = stratuxClock.Time
	}
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

	// Add IP's set through the settings page
	if globalSettings.StaticIps != nil {
		for _, ip := range globalSettings.StaticIps {
			ret[ip] = ""
		}
	}

	// Added the ability to have static IP hosts stored in /etc/stratux-static-hosts.conf

	dat2, err := ioutil.ReadFile(extra_hosts_file)
	if err != nil {
		return ret, nil
	}

	iplines := strings.Split(string(dat2), "\n")
	block_ip2 := ""
	for _, ipline := range iplines {
		spacedip := strings.Split(ipline, " ")
		if len(spacedip) == 2 {
			// The ip is in block_ip2
			block_ip2 = spacedip[0]
			// the hostname is here
			ret[block_ip2] = spacedip[1]
		}
	}

	return ret, nil
}

/*
	isSleeping().
	 Check if a client identifier 'ip:port' is in either a sleep or active state.
	 ***WARNING***: netMutex must be locked before calling this function.
*/
func isSleeping(k string) bool {
	// Unable to listen to ICMP without root - send to everything. Just for debugging.
	if isX86DebugMode() {
		return false
	}
	ipAndPort := strings.Split(k, ":")
	// No ping response. Assume disconnected/sleeping device.
	if lastPing, ok := pingResponse[ipAndPort[0]]; !ok || stratuxClock.Since(lastPing) > (10*time.Second) {
		return true
	}
	if stratuxClock.Since(outSockets[k].LastUnreachable) < (5 * time.Second) {
		return true
	}
	return false
}

/*
	isThrottled().
	 Checks if a client identifier 'ip:port' is throttled.
	 Throttle mode is for testing port open and giving some start-up time to the app.
	 Throttling is 0.1% data rate for first 15 seconds.
	 ***WARNING***: netMutex must be locked before calling this function.
*/
func isThrottled(k string) bool {
	return (rand.Int()%1000 != 0) && stratuxClock.Since(outSockets[k].LastUnreachable) < (15*time.Second)
}

func sendToAllConnectedClients(msg networkMessage) {
	if (msg.msgType & NETWORK_GDL90_STANDARD) != 0 {
		// It's a GDL90 message. Send to serial output channel (which may or may not cause something to happen).
		serialOutputChan <- msg.msg
		networkGDL90Chan <- msg.msg
	}

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

var serialOutputChan chan []byte
var networkGDL90Chan chan []byte

func networkOutWatcher() {
	for {
		ch := <-networkGDL90Chan
		gdl90Update.SendJSON(ch)
	}
}

// Monitor serial output channel, send to serial port.
func serialOutWatcher() {
	// Check every 30 seconds for a serial output device.
	serialTicker := time.NewTicker(30 * time.Second)

	serialDev := "/dev/serialout0" //FIXME: This is temporary. Only one serial output device for now.

	for {
		select {
		case <-serialTicker.C:
			if _, err := os.Stat(serialDev); !os.IsNotExist(err) { // Check if the device file exists.
				var thisSerialConn serialConnection
				// Check if we need to start handling a new device.
				if val, ok := globalSettings.SerialOutputs[serialDev]; !ok {
					newSerialOut := serialConnection{DeviceString: serialDev, Baud: 38400}
					log.Printf("detected new serial output, setting up now: %s. Default baudrate 38400.\n", serialDev)
					if globalSettings.SerialOutputs == nil {
						globalSettings.SerialOutputs = make(map[string]serialConnection)
					}
					globalSettings.SerialOutputs[serialDev] = newSerialOut
					saveSettings()
					thisSerialConn = newSerialOut
				} else {
					thisSerialConn = val
				}
				// Check if we need to open the connection now.
				if thisSerialConn.serialPort == nil {
					cfg := &serial.Config{Name: thisSerialConn.DeviceString, Baud: thisSerialConn.Baud}
					p, err := serial.OpenPort(cfg)
					if err != nil {
						log.Printf("serialout port (%s) err: %s\n", thisSerialConn.DeviceString, err.Error())
						break // We'll attempt again in 30 seconds.
					} else {
						log.Printf("opened serialout: Name: %s, Baud: %d\n", thisSerialConn.DeviceString, thisSerialConn.Baud)
					}
					// Save the serial port connection.
					thisSerialConn.serialPort = p
					globalSettings.SerialOutputs[serialDev] = thisSerialConn
				}
			}

		case b := <-serialOutputChan:
			if val, ok := globalSettings.SerialOutputs[serialDev]; ok {
				if val.serialPort != nil {
					_, err := val.serialPort.Write(b)
					if err != nil { // Encountered an error in writing to the serial port. Close it and set Serial_out_enabled.
						log.Printf("serialout (%s) port err: %s. Closing port.\n", val.DeviceString, err.Error())
						val.serialPort.Close()
						val.serialPort = nil
						globalSettings.SerialOutputs[serialDev] = val
					}
				}
			}
		}
	}
}

// Returns the number of DHCP leases and prints queue lengths.
func getNetworkStats() {

	netMutex.Lock()
	defer netMutex.Unlock()

	var numNonSleepingClients uint

	for k, netconn := range outSockets {
		queueBytes := 0
		for _, msg := range netconn.messageQueue {
			queueBytes += len(msg)
		}
		if globalSettings.DEBUG {
			log.Printf("On  %s:%d,  Queue length = %d messages / %d bytes\n", netconn.Ip, netconn.Port, len(netconn.messageQueue), queueBytes)
		}
		ipAndPort := strings.Split(k, ":")
		if len(ipAndPort) != 2 {
			continue
		}
		ip := ipAndPort[0]
		if pingRespTime, ok := pingResponse[ip]; ok {
			// Don't count the ping time if it is the same as stratuxClock epoch.
			// If the client has responded to a ping in the last 15 minutes, count it as "connected" or "recent".
			if !pingRespTime.Equal(time.Time{}) && stratuxClock.Since(pingRespTime) < 15*time.Minute {
				numNonSleepingClients++
			}
		}
	}

	globalStatus.Connected_Users = numNonSleepingClients
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
	secondTimer := time.NewTicker(15 * time.Second) // getNetworkStats().
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
				netconn.MessageQueueLen = len(netconn.messageQueue)
				outSockets[k] = netconn
			}

			if stratuxClock.Since(lastQueueTimeChange) >= 5*time.Second {
				var pd float64
				if averageSendableQueueSize > 0.0 && len(outSockets) > 0 {
					averageSendableQueueSize = averageSendableQueueSize / float64(len(outSockets)) // It's a total, not an average, up until this point.
					pd = math.Max(float64(1.0/750.0), float64(1.0/(4.0*averageSendableQueueSize))) // Say 250ms is enough to get through the whole queue.
				} else {
					pd = float64(0.1) // 100ms.
				}

				if globalSettings.DEBUG {
					log.Printf("Average sendable queue is %v messages. Changing queue timer to %f seconds\n", averageSendableQueueSize, pd)
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
		netMutex.Lock()
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
		netMutex.Unlock()
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
			netMutex.Lock()
			pingResponse[ip] = stratuxClock.Time
			netMutex.Unlock()
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

/*
	ffMonitor().
		Watches for "i-want-to-play-ffm-udp", "i-can-play-ffm-udp", and "i-cannot-play-ffm-udp" UDP messages broadcasted on
		 port 50113. Tags the client, issues a warning, and disables AHRS GDL90 output.

*/

func ffMonitor() {
	addr := net.UDPAddr{Port: 50113, IP: net.ParseIP("0.0.0.0")}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Printf("ffMonitor(): error listening on port 50113: %s\n", err.Error())
		return
	}
	defer conn.Close()
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
		if strings.HasPrefix(s, "i-want-to-play-ffm-udp") || strings.HasPrefix(s, "i-can-play-ffm-udp") || strings.HasPrefix(s, "i-cannot-play-ffm-udp") {
			p.FFCrippled = true
			//FIXME: AHRS output doesn't need to be disabled globally, just on the ForeFlight client IPs.
			addSingleSystemErrorf("ff-warn", "Stratux is not supported by your EFB app. Your EFB app is known to regularly make changes that cause compatibility issues with Stratux. See the README for a list of apps that officially support Stratux.")
		}
		outSockets[ffIpAndPort] = p
		netMutex.Unlock()
	}
}

func initNetwork() {
	messageQueue = make(chan networkMessage, 1024) // Buffered channel, 1024 messages.
	serialOutputChan = make(chan []byte, 1024)     // Buffered channel, 1024 GDL90 messages.
	networkGDL90Chan = make(chan []byte, 1024)
	outSockets = make(map[string]networkConnection)
	pingResponse = make(map[string]time.Time)
	netMutex = &sync.Mutex{}
	refreshConnectedClients()
	go monitorDHCPLeases()
	go messageQueueSender()
	go sleepMonitor()
	go networkStatsCounter()
	go serialOutWatcher()
	go networkOutWatcher()
	go tcpNMEAListener()
}
