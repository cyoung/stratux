/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	network.go: Client networking routines, DHCP lease monitoring, queue management, ICMP monitoring.
*/

package main

import (
	"io"
	"log"
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/tarm/serial"
	"tinygo.org/x/bluetooth"
)

// Connection interface where we will write data - no matter if UDP, Serial or TCP
type connection interface {
	GetConnectionKey() string // e.g. "192.168.10.22:12345" for udp, "TCP:192.168.10.22:12345" for TCP, "/dev/serialout0" for serialout
	MessageQueue() *MessageQueue
	Writer()       io.Writer
	IsThrottled()  bool
	IsSleeping()   bool
	Capabilities() uint8
	GetDesiredPacketSize() int
	OnError(error)
	Close()
}

type networkConnection struct {
	Conn            *net.UDPConn
	Ip              string
	Port            uint32
	Capability      uint8
	Queue           *MessageQueue `json:"-"` // don't store in settings

	LastPingResponse time.Time // last time the client responded
	LastUnreachable time.Time // Last time the device sent an ICMP Unreachable packet.
	/*
		Sleep mode/throttle variables. "sleep mode" is actually now just a very reduced packet rate, since we don't know positively
		 when a client is ready to accept packets - we just assume so if we don't receive ICMP Unreachable packets in 5 secs.
	*/
	SleepFlag       bool      // Whether or not this client has been marked as sleeping - only used for debugging
}

func (conn *networkConnection) MessageQueue() *MessageQueue {
	if conn.Queue == nil {
		conn.Queue = NewMessageQueue(1024)
	}
	return conn.Queue
}
func (conn *networkConnection) Writer() io.Writer {
	return conn.Conn
}
/*
	isThrottled().
	 Checks if a client identifier 'ip:port' is throttled.
	 Throttle mode is for testing port open and giving some start-up time to the app.
	 Throttling means that we only send important packets for first 15 seconds (location, status, very close traffic).
*/
func (conn *networkConnection) IsThrottled() bool {
	return (rand.Int()%1000 != 0) && stratuxClock.Since(conn.LastUnreachable) < (15*time.Second)
}

/*
	isSleeping().
	 Check if a client identifier 'ip:port' is in either a sleep or active state.
*/
func (conn *networkConnection) IsSleeping() bool {
	// Unable to listen to ICMP without root - send to everything. Just for debugging.
	if isX86DebugMode() || globalSettings.NoSleep == true {
		return false
	}
	// No ping response. Assume disconnected/sleeping device.
	if conn.LastPingResponse.IsZero() || stratuxClock.Since(conn.LastPingResponse) > (10*time.Second) {
		conn.SleepFlag = true
	} else if stratuxClock.Since(conn.LastUnreachable) < (5 * time.Second) {
		conn.SleepFlag = true
	} else {
		conn.SleepFlag = false
	}
	return conn.SleepFlag
}

func (conn *networkConnection) Capabilities() uint8 {
	return conn.Capability
}

func (conn *networkConnection) GetDesiredPacketSize() int {
	if conn.Capabilities() & (NETWORK_POSITION_FFSIM | NETWORK_AHRS_FFSIM) > 0 {
		// Hack: some software doesn't handle X-Plane as a stream correctly, e.g. SkyDemon, and requires each message in a separate packet, or it will crash.
		return 1
	}
	return 1024
}

func (conn *networkConnection) OnError(err error) {
	// Ignore for UDP. We keep the socket always open and just try to push data
	//log.Printf("UDP Write error: %s", err.Error())
}

func (conn *networkConnection) Close() {
	// Ignore for UDP. We keep the socket always open and just try to push data
}

func (conn *networkConnection) GetConnectionKey() string {
	return conn.Ip + ":" + strconv.Itoa(int(conn.Port))
}



type serialConnection struct {
	DeviceString string
	Baud         int
	Capability   uint8
	serialPort   *serial.Port
	Queue        *MessageQueue `json:"-"` // don't store in settings
}

func (conn *serialConnection) MessageQueue() *MessageQueue {
	if conn.Queue == nil {
		conn.Queue = NewMessageQueue(1024)
	}
	return conn.Queue
}

func (conn *serialConnection) Writer() io.Writer {
	return conn.serialPort
}
func (conn *serialConnection) IsThrottled() bool {
	return false
}
func (conn *serialConnection) IsSleeping() bool {
	return conn.serialPort == nil
}

func (conn *serialConnection) Capabilities() uint8 {
	return conn.Capability
}

func (conn *serialConnection) GetDesiredPacketSize() int {
	return 128
}

func (conn *serialConnection) OnError(err error) {
	// Close connection and queue
	log.Printf("Serial connection %s closed: %s", conn.DeviceString, err.Error())
	conn.Close()
}

func (conn *serialConnection) Close() {
	if conn.serialPort != nil {
		conn.serialPort.Close()
		log.Printf("Closed serial port %s", conn.DeviceString)
		conn.Queue.Close()
		onConnectionClosed(conn)
	}
}

func (conn *serialConnection) GetConnectionKey() string {
	return conn.DeviceString
}

type tcpConnection struct {
	Conn         *net.TCPConn
	Queue        *MessageQueue `json:"-"`
	Capability   uint8
	Key          string
}

func (conn *tcpConnection) MessageQueue() *MessageQueue {
	if conn.Queue == nil {
		conn.Queue = NewMessageQueue(1024)
	}
	return conn.Queue
}

func (conn *tcpConnection) Writer() io.Writer {
	return conn.Conn
}
func (conn *tcpConnection) IsThrottled() bool {
	return false
}
func (conn *tcpConnection) IsSleeping() bool {
	return conn.Conn == nil
}
func (conn *tcpConnection) Capabilities() uint8 {
	return conn.Capability
}
func (conn *tcpConnection) GetDesiredPacketSize() int {
	return 512
}

func (conn *tcpConnection) OnError(err error) {
	// Close connection and queue
	if conn.Conn != nil {
		log.Printf("TCP connection %s closed: %s", conn.Conn.RemoteAddr(), err.Error())
		conn.Close()
	}
}

func (conn *tcpConnection) Close() {
	// Close connection and queue
	if conn.Conn != nil {
		conn.Conn.Close()
		conn.Conn = nil
		conn.Queue.Close()
		onConnectionClosed(conn)
	}
}


func (conn *tcpConnection) GetConnectionKey() string {
	return conn.Key
}



type bleConnection struct {
	Capability   uint8
	UUIDService  string // SoftRF: 0xFFE0
	UUIDGatt     string
	Characteristic bluetooth.Characteristic
	Queue        *MessageQueue `json:"-"` // don't store in settings
}

func (conn *bleConnection) MessageQueue() *MessageQueue {
	if conn.Queue == nil {
		conn.Queue = NewMessageQueue(1024)
	}
	return conn.Queue
}

func (conn *bleConnection) Writer() io.Writer {
	return conn
}

func (conn *bleConnection) Write(p []byte) (n int, err error) {
	return conn.Characteristic.Write(p)
}

func (conn *bleConnection) IsThrottled() bool {
	return false
}
func (conn *bleConnection) IsSleeping() bool {
	return false
}

func (conn *bleConnection) Capabilities() uint8 {
	return conn.Capability
}

func (conn *bleConnection) GetDesiredPacketSize() int {
	return 20 // TODO
}

func (conn *bleConnection) OnError(err error) {
	// Close connection and queue
	log.Printf("BLE Error %s", err.Error())
}

func (conn *bleConnection) Close() {
}

func (conn *bleConnection) GetConnectionKey() string {
	return conn.UUIDService
}