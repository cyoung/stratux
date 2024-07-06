/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	network.go: Client networking routines, DHCP lease monitoring, queue management, ICMP monitoring.
*/

package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tarm/serial"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"tinygo.org/x/bluetooth"
)


var clientConnections map[string]connection // UDP out, TCP out, serial out

var dhcpLeases map[string]string
var networkGDL90Chan chan []byte      // For gdl90 web socket
var netMutex *sync.Mutex              // netMutex needs to be locked before accessing dhcpLeases, pingResponse, and outSockets and calling isSleeping() and isThrottled().

var totalNetworkMessagesSent uint32

const (
	NETWORK_GDL90_STANDARD = 1
	NETWORK_AHRS_FFSIM     = 2
	NETWORK_AHRS_GDL90     = 4
	NETWORK_FLARM_NMEA     = 8
	NETWORK_POSITION_FFSIM = 16
	dhcp_lease_file        = "/var/lib/misc/dnsmasq.leases"
	dhcp_lease_dir         = "/var/lib/misc/"
	extra_hosts_file       = "/etc/stratux-static-hosts.conf"
)

var dhcpLeaseDirectoryLastTest time.Time // Last time fsWriteTest() was run on the DHCP lease directory.

// Read the "dnsmasq.leases" file and parse out IP/hostname.
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

	if err == nil {
		lines := strings.Split(string(dat), "\n")
		for _, line := range lines {
			spaced := strings.Split(line, " ")
			if len(spaced) >= 4 {
				ip := spaced[2]
				host := spaced[3]
				if host == "*" {
					host = ""
				}
				ret[ip] = host
			}
		}
	}

	// Add IP's set through the settings page
	if globalSettings.StaticIps != nil {
		for _, ip := range globalSettings.StaticIps {
			ret[ip] = ""
		}
	}

	// Check kernel ARP table - useful when in client mode. We skip reverse hostname lookup since it can be very slow..
	dat2, err := ioutil.ReadFile("/proc/net/arp")
	if err != nil {
		return ret, nil
	}
	iplines := strings.Split(string(dat2), "\n")
	for _, ipline := range iplines {
		spacedip := strings.Split(ipline, " ")
		if len(spacedip) > 1 {
			ip := spacedip[0]
			if _, contained := ret[ip]; !contained && ip[0] >= '0' && ip[0] <= '2' {
				ret[ip] = ""
			}
		}
	}

	// Added the ability to have static IP hosts stored in /etc/stratux-static-hosts.conf

	dat3, err := ioutil.ReadFile(extra_hosts_file)
	if err != nil {
		return ret, nil
	}

	iplines = strings.Split(string(dat3), "\n")
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

func networkOutWatcher() {
	for {
		ch := <-networkGDL90Chan
		gdl90Update.SendJSON(ch)
	}
}

// Monitor serial output channel, send to serial port.
func serialOutWatcher() {
	// Check every 30 seconds for a serial output device.
	serialTicker := time.NewTicker(10 * time.Second)

	//FIXME: This is temporary. Only one serial output device for each protocol for now.
	serialDevs := make([]string, 0)
	for i := 0; i < 10; i++ {
		serialDevs = append(serialDevs, fmt.Sprintf("/dev/serialout%d", i))
		serialDevs = append(serialDevs, fmt.Sprintf("/dev/serialout_nmea%d", i))
	}

	for {
		select {
		case <-serialTicker.C:
			for _, serialDev := range serialDevs {
				if _, err := os.Stat(serialDev); !os.IsNotExist(err) { // Check if the device file exists.
					var config serialConnection

					// Master is globalSettings.SerialOutputs. Once we connect to one, it will be copied to the active connections map
					if val, ok := globalSettings.SerialOutputs[serialDev]; !ok {
						proto := uint8(NETWORK_GDL90_STANDARD)
						if strings.Contains(serialDev, "_nmea") {
							proto = NETWORK_FLARM_NMEA
						}
						if globalSettings.SerialOutputs == nil {
							globalSettings.SerialOutputs = make(map[string]serialConnection)
						}
						globalSettings.SerialOutputs[serialDev] = serialConnection{DeviceString: serialDev, Baud: 38400, Capability: proto, Queue: NewMessageQueue(1024)}
						log.Printf("detected new serial output, setting up now: %s. Default baudrate 38400.\n", serialDev)
						config = globalSettings.SerialOutputs[serialDev]

						saveSettings()
					} else {
						config = val
						if config.Capability == 0 {
							config.Capability = NETWORK_GDL90_STANDARD // Fix old serial conns that didn't have protocol set
						}
					}

					netMutex.Lock()

					needsConnect := true
					if activeConn, ok := clientConnections[serialDev]; ok {
						if !activeConn.IsSleeping() {
							needsConnect = false
						} else {
							go activeConn.Close() // expired/unplugged? async because it might lock..
							delete(clientConnections, serialDev)
						}
					}

					if needsConnect {
						cfg := &serial.Config{Name: config.DeviceString, Baud: config.Baud}
						p, err := serial.OpenPort(cfg)
						if err != nil {
							log.Printf("serialout port (%s) err: %s\n", config.DeviceString, err.Error())
						} else {
							log.Printf("opened serialout: Name: %s, Baud: %d\n", config.DeviceString, config.Baud)
							// Save the serial port connection.
							tmp := config
							tmp.serialPort = p
							clientConnections[serialDev] = &tmp
							go connectionWriter(&tmp)
						}
					}
					netMutex.Unlock()
				}
			}
		}
	}
}

// TCP port 2000 for Airconnect-like NMEA-Out
func tcpNMEAOutListener() {
	ln, err := net.Listen("tcp", ":2000")
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		key := "TCP:" + conn.RemoteAddr().String()

		tcpConn := &tcpConnection{
			conn.(*net.TCPConn),
			NewMessageQueue(1024),
			NETWORK_FLARM_NMEA,
			key,
		}
		clientConnections[tcpConn.GetConnectionKey()] = tcpConn
		go connectionWriter(tcpConn)
	}
}



/* Server that can be used to feed NMEA data to, e.g. to connect OGN Tracker wirelessly */
func tcpNMEAInListener() {
	ln, err := net.Listen("tcp", ":30011")
	if err != nil {
		log.Printf(err.Error())
		return
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf(err.Error())
			continue
		}
		go handleNmeaInConnection(conn)
	}	
}
func handleNmeaInConnection(c net.Conn) {
	defer c.Close()
	reader := bufio.NewReader(c)
	// Set to fixed GPS_TYPE_NETWORK in the beginning, to override previous detected NMEA types
	globalStatus.GPS_detected_type = GPS_TYPE_NETWORK
	globalStatus.GPS_NetworkRemoteIp = strings.Split(c.RemoteAddr().String(), ":")[0]
	for {
		globalStatus.GPS_connected = true
		// Keep detected protocol, only ensure type=network
		globalStatus.GPS_detected_type = GPS_TYPE_NETWORK | (globalStatus.GPS_detected_type & 0xf0)
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		processNMEALine(line)
	}
	globalStatus.GPS_connected = false
	globalStatus.GPS_detected_type = 0
	globalStatus.GPS_NetworkRemoteIp = ""
}

// Returns the number of DHCP leases and prints queue lengths.
func getNetworkStats() {
	timer := time.NewTicker(15 * time.Second)

	for {
		<-timer.C
		netMutex.Lock()
		

		var numNonSleepingClients uint

		for k, conn := range clientConnections {
			// only use net conns
			netconn, ok := conn.(*networkConnection)
			if netconn == nil || !ok {
				continue
			}

			if globalSettings.DEBUG {
				queueBytes := 0
				queueDump := netconn.Queue.GetQueueDump(true)
				for _, msg := range queueDump {
					queueBytes += len(msg.([]byte))
				}
				log.Printf("On  %s:%d,  Queue length = %d messages / %d bytes\n", netconn.Ip, netconn.Port, len(queueDump), queueBytes)
			}
			ipAndPort := strings.Split(k, ":")
			if len(ipAndPort) != 2 {
				continue
			}
			// Don't count the ping time if it is the same as stratuxClock epoch.
			// If the client has responded to a ping in the last 15 minutes, count it as "connected" or "recent".
			if !netconn.LastPingResponse.IsZero() && stratuxClock.Since(netconn.LastPingResponse) < 15*time.Minute {
				numNonSleepingClients++
			}
		}

		globalStatus.Connected_Users = numNonSleepingClients

		netMutex.Unlock()
	}
}

// See who has a DHCP lease and make a UDP connection to each of them.
func refreshConnectedClients() {
	validConnections := make(map[string]bool)
	t, err := getDHCPLeases()
	if err != nil {
		log.Printf("getDHCPLeases(): %s\n", err.Error())
		return
	}
	netMutex.Lock()
	defer netMutex.Unlock()

	dhcpLeases = t
	// Client connected that wasn't before.
	for ip, hostname := range dhcpLeases {
		for _, networkOutput := range globalSettings.NetworkOutputs {
			ipAndPort := ip + ":" + strconv.Itoa(int(networkOutput.Port))
			if _, ok := clientConnections[ipAndPort]; !ok {
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
				clientConnections[ipAndPort] = &networkConnection{
					Conn: outConn,
					Ip: ip,
					Port: networkOutput.Port,
					Capability: networkOutput.Capability,
					Queue: NewMessageQueue(1024),
				}
				go connectionWriter(clientConnections[ipAndPort])
			}
			validConnections[ipAndPort] = true
		}
	}
	// Client that was connected before that isn't.
	for ipAndPort, netconn := range clientConnections {
		if conn, ok := netconn.(*networkConnection); ok {
			if _, valid := validConnections[ipAndPort]; !valid {
				log.Printf("removed connection %s.\n", ipAndPort)
				conn.Queue.Close()
				conn.Conn.Close()
				delete(clientConnections, ipAndPort)
			}
		}
	}
}

func parseBleUuid(uuidStr string) (uuid bluetooth.UUID) {
	if len(uuidStr) == 4 {
		// Assume hex 16 bit
		var val uint64
		val, _ = strconv.ParseUint(uuidStr, 16, 16)
		uuid = bluetooth.New16BitUUID(uint16(val))
	} else {
		uuid, _ = bluetooth.ParseUUID(uuidStr)
	}
	return
}

var bleAdapter = bluetooth.DefaultAdapter
func initBluetooth() {
	if len(globalSettings.BleOutputs) == 0 {
		return
	}
	if err := bleAdapter.Enable(); err != nil {
		addSingleSystemErrorf("BLE", "Failed to init BLE adapter: %s", err.Error())
		return
	}
	services := []bluetooth.UUID{}
	for _, conn := range globalSettings.BleOutputs {
		services = append(services, parseBleUuid(conn.UUIDService))
	}

	adv := bleAdapter.DefaultAdvertisement()
	adv.Configure(bluetooth.AdvertisementOptions{
		LocalName: globalSettings.WiFiSSID,
		ServiceUUIDs: services,
	})
	if err := adv.Start(); err != nil {
		addSingleSystemErrorf("BLE", "BLE Advertising failed to start: %s", err.Error())
	}
	// TODO: not working if we have multiple GATTs in one service
	for i := range globalSettings.BleOutputs {
		conn := &globalSettings.BleOutputs[i]
		err := bleAdapter.AddService(&bluetooth.Service{
			UUID: parseBleUuid(conn.UUIDService),
			Characteristics: []bluetooth.CharacteristicConfig {
				{
					Handle: &conn.Characteristic,
					UUID:   parseBleUuid(conn.UUIDGatt),
					Value:  []byte{},
					Flags:  bluetooth.CharacteristicNotifyPermission | bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
					//WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					//	log.Printf("client %d: received %s", client, string(value))
					//},
				},
			},

		})
		if err != nil {
			log.Printf("Failed to bring up BLE Gatt Service %s: %s", conn.UUIDService, err.Error())
			continue
		}
		netMutex.Lock()
		clientConnections[conn.GetConnectionKey()] = conn
		go connectionWriter(conn)
		netMutex.Unlock()
	}
}

func onConnectionClosed(conn connection) {
	if conn == nil {
		return
	}
	netMutex.Lock()
	key := conn.GetConnectionKey()
	delete(clientConnections, key)
	netMutex.Unlock()
}


func collectMessages(conn connection) []byte {
	data := make([]byte, 0)
	maxMsgLen := 0
	for {
		if conn.IsSleeping() || conn.IsThrottled() {
			_, prio := conn.MessageQueue().PeekFirst()
			if conn.IsSleeping() && prio > -10 {
				// If we are sleeping, only send heartbeats do detect a client becoming available
				return data
			}
			if conn.IsThrottled() && prio > 0 {
				// if throttled, only send important stuff (position, status, crucial traffic)
				return data
			}
		}

		newData, _ := conn.MessageQueue().PopFirst()
		if newData == nil {
			return data // no more data to send
		}
		msg := newData.([]byte)
		data = append(data, msg...)
		// So we can estimate if another message fits in
		if len(msg) > maxMsgLen {
			maxMsgLen = len(data)
		}
		if len(data) + maxMsgLen > conn.GetDesiredPacketSize() {
			// Probably can't fit in another message
			return data
		}
	}
}

func connectionWriter(connection connection) {
	for {
		queue := connection.MessageQueue()
		<-queue.DataAvailable
		for {
			if queue.Closed {
				return
			}

			// Try to send around 1kb of data per packet to reduce IOPS when queue is full
			msg := collectMessages(connection)
			if msg == nil || len(msg) == 0 {
				break // Wait for next time that the DataAvailable channel has more for us
			}
			//fmt.Printf("Sending message bytes %d\n", len(msg))
			written := 0
			for written < len(msg) {
				writtenNow, err := connection.Writer().Write(msg)
				written += writtenNow
				if err != nil {
					connection.OnError(err)
					break
				}
			}
			totalNetworkMessagesSent++
			globalStatus.NetworkDataMessagesSent++
			globalStatus.NetworkDataBytesSent += uint64(written)
			//time.Sleep(532 * time.Millisecond)
		}
	}
}


func sendMsg(msg []byte, msgType uint8, maxAge time.Duration, priority int32) {
	if (msgType & NETWORK_GDL90_STANDARD) != 0 {
		// It's a GDL90 message - do ui broadcast.
		networkGDL90Chan <- msg
	}

	netMutex.Lock()
	defer netMutex.Unlock()

	// Push to all UDP, TCP, Serial connections if they support the message
	for _, conn := range clientConnections {
		// Check if this port is able to accept the type of message we're sending.
		if (conn.Capabilities() & msgType) == 0 {
			continue
		}
		conn.MessageQueue().Put(priority, maxAge, msg)
	}
}

func sendGDL90(msg []byte, maxAge time.Duration, priority int32) {
	sendMsg(msg, NETWORK_GDL90_STANDARD, maxAge, priority)
}

func sendXPlane(msg []byte, maxAge time.Duration, priority int32) {
	sendMsg(msg, NETWORK_POSITION_FFSIM, maxAge, priority)
}

func sendNetFLARM(msg string, maxAge time.Duration, priority int32) {
	sendMsg([]byte(msg), NETWORK_FLARM_NMEA, maxAge, priority)
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
		for k, conn := range clientConnections {
			if _, ok := conn.(*networkConnection); ok {
				ipAndPort := strings.Split(k, ":")
				ips[ipAndPort[0]] = true
			}
		}
		// Send to all IPs.
		for ip := range ips {
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

func getNetworkConn(ipAndPort string) *networkConnection {
	netMutex.Lock()
	defer netMutex.Unlock()
	if strings.Contains(ipAndPort, ":") {
		if conn, ok := clientConnections[ipAndPort]; ok {
			if netconn, ok := conn.(*networkConnection); ok {
				return netconn;
			}
		}
	}
	return nil
}

func getNetworkConnsByIp(ip string) []*networkConnection {
	conns := make([]*networkConnection, 0)
	// Search for any connection with the same IP to match ping responses
	for key, conn := range clientConnections {
		if netconn, ok := conn.(*networkConnection); ok {
			if strings.HasPrefix(key, ip) {
				conns = append(conns, netconn)
			}
		}
	}
	return conns
}

func getSerialConns() []*serialConnection {
	netMutex.Lock()
	defer netMutex.Unlock()
	conns := make([]*serialConnection, 0)
	for _, conn := range clientConnections {
		if s, ok := conn.(*serialConnection); ok {
			conns = append(conns, s)
		}
	}
	return conns
}

func closeSerial(dev string) {
	netMutex.Lock()
	defer netMutex.Unlock()
	if conn, ok := clientConnections[dev]; ok {
		delete(clientConnections, dev)
		go conn.Close()
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
			for _, conn := range getNetworkConnsByIp(ip) {
				conn.LastPingResponse = stratuxClock.Time
			}
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
		conn := getNetworkConn(ipAndPort)
		if conn != nil {
			conn.LastUnreachable = stratuxClock.Time
		}
	}
}

func networkStatsCounter() {
	timer := time.NewTicker(1 * time.Second)
	var previousNetworkMessagesSent, previousNetworkBytesSent uint64

	for {
		<-timer.C
		globalStatus.NetworkDataMessagesSentLastSec = globalStatus.NetworkDataMessagesSent - previousNetworkMessagesSent
		globalStatus.NetworkDataBytesSentLastSec = globalStatus.NetworkDataBytesSent - previousNetworkBytesSent

		// debug option. Uncomment to log per-second network statistics. Useful for debugging WiFi instability.
		//log.Printf("Network data messages sent: %d total, %d last second.  Network data bytes sent: %d total, %d last second.\n", globalStatus.NetworkDataMessagesSent, globalStatus.NetworkDataMessagesSentLastSec, globalStatus.NetworkDataBytesSent, globalStatus.NetworkDataBytesSentLastSec)

		previousNetworkMessagesSent = globalStatus.NetworkDataMessagesSent
		previousNetworkBytesSent = globalStatus.NetworkDataBytesSent
	}
}


func initNetwork() {
	networkGDL90Chan = make(chan []byte, 1024)
	clientConnections = make(map[string]connection)

	netMutex = &sync.Mutex{}
	refreshConnectedClients()
	go monitorDHCPLeases() // Checks for new UDP connections
	go sleepMonitor()
	go networkStatsCounter()
	go serialOutWatcher() // Check for new Serial connections
	go networkOutWatcher() // Pushes to websocket
	go tcpNMEAOutListener()
	go tcpNMEAInListener()
	go getNetworkStats()
	go initBluetooth()
}
