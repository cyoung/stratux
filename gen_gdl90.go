package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// http://www.faa.gov/nextgen/programs/adsb/wsa/media/GDL90_Public_ICD_RevA.PDF

const (
	ipadAddr                = "192.168.10.255:4000" // Port 4000 for FreeFlight RANGR.
	maxDatagramSize         = 8192
	UPLINK_BLOCK_DATA_BITS  = 576
	UPLINK_BLOCK_BITS       = (UPLINK_BLOCK_DATA_BITS + 160)
	UPLINK_BLOCK_DATA_BYTES = (UPLINK_BLOCK_DATA_BITS / 8)
	UPLINK_BLOCK_BYTES      = (UPLINK_BLOCK_BITS / 8)

	UPLINK_FRAME_BLOCKS     = 6
	UPLINK_FRAME_DATA_BITS  = (UPLINK_FRAME_BLOCKS * UPLINK_BLOCK_DATA_BITS)
	UPLINK_FRAME_BITS       = (UPLINK_FRAME_BLOCKS * UPLINK_BLOCK_BITS)
	UPLINK_FRAME_DATA_BYTES = (UPLINK_FRAME_DATA_BITS / 8)
	UPLINK_FRAME_BYTES      = (UPLINK_FRAME_BITS / 8)

	// assume 6 byte frames: 2 header bytes, 4 byte payload
	// (TIS-B heartbeat with one address, or empty FIS-B APDU)
	UPLINK_MAX_INFO_FRAMES = (424 / 6)
)

var Crc16Table [256]uint16
var outConn *net.UDPConn

// Construct the CRC table. Adapted from FAA ref above.
func crcInit() {
	var i uint16
	var bitctr uint16
	var crc uint16
	for i = 0; i < 256; i++ {
		crc = (i << 8)
		for bitctr = 0; bitctr < 8; bitctr++ {
			z := uint16(0)
			if (crc & 0x8000) != 0 {
				z = 0x1021
			}
			crc = (crc << 1) ^ z
		}
		Crc16Table[i] = crc
	}
}

// Compute CRC. Adapted from FAA ref above.
func crcCompute(data []byte) uint16 {
	ret := uint16(0)
	for i := 0; i < len(data); i++ {
		ret = Crc16Table[ret>>8] ^ (ret << 8) ^ uint16(data[i])
	}
	return ret
}

func prepareMessage(data []byte) []byte {
	tmp := []byte{0x7E} // Flag start.
	// Compute CRC before modifying the message.
	crc := crcCompute(data)

	// Copy the message over, escaping 0x7E (Flag Byte) and 0x7D (Control-Escape).
	for i := 0; i < len(data); i++ {
		mv := data[i]
		if (mv == 0x7E) || (mv == 0x7D) {
			mv = mv ^ 0x20
			tmp = append(tmp, 0x7D)
		}
		tmp = append(tmp, mv)
	}

	// Add the two CRC16 bytes.
	tmp = append(tmp, byte(crc&0xFF))
	tmp = append(tmp, byte(crc>>8))

	tmp = append(tmp, 0x7E) // Flag end.

	return tmp
}

func makeHeartbeat() []byte {
	msg := make([]byte, 7)
	// See p.10.
	msg[0] = 0x00 // Message type "Heartbeat".
	msg[1] = 0x01 // "UAT Initialized".
	nowUTC := time.Now().UTC()
	// Seconds since 0000Z.
	midnightUTC := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC)
	secondsSinceMidnightUTC := uint32(nowUTC.Sub(midnightUTC).Seconds())

	msg[2] = byte((secondsSinceMidnightUTC >> 16) << 7)
	msg[3] = byte((secondsSinceMidnightUTC & 0xFF))
	msg[4] = byte((secondsSinceMidnightUTC & 0xFFFF) >> 8)

	// TODO. Number of uplink messages. See p.12.
	// msg[5]
	// msg[6]

	return prepareMessage(msg)
}

func relayUplinkMessage(msg []byte) {
	ret := make([]byte, len(msg)+4)
	// See p.15.
	ret[0] = 0x07 // Uplink message ID.
	ret[1] = 0x00 //TODO: Time.
	ret[2] = 0x00 //TODO: Time.
	ret[3] = 0x00 //TODO: Time.

	for i := 0; i < len(msg); i++ {
		ret[i+4] = msg[i]
	}

	outConn.Write(prepareMessage(ret))
}

func heartBeatSender() {
	for {
		outConn.Write(makeHeartbeat())
		time.Sleep(1 * time.Second)
	}
}

func parseInput(buf string) []byte {
	buf = strings.Trim(buf, "\r\n") // Remove newlines.
	x := strings.Split(buf, ";")    // We want to discard everything before the first ';'.
	if len(x) == 0 {
		return nil
	}
	s := x[0]
	if len(s) == 0 {
		return nil
	}
	if s[0] != '+' {
		return nil // Only want + ("Uplink") messages currently. - (Downlink) or messages that start with other are discarded.
	}

	s = s[1:]

	if len(s)%2 != 0 { // Bad format.
		return nil
	}

	if len(s)/2 != UPLINK_FRAME_DATA_BYTES {
		fmt.Printf("UPLINK_FRAME_DATA_BYTES=%d, len(s)=%d\n", UPLINK_FRAME_DATA_BYTES, len(s))
		//		panic("Error")
		return nil
	}

	// Now, begin converting the string into a byte array.
	frame := make([]byte, UPLINK_FRAME_DATA_BYTES)
	hex.Decode(frame, []byte(s))

	return frame
}

func main() {
	crcInit() // Initialize CRC16 table.

	// Open UDP port to send out the messages.
	addr, err := net.ResolveUDPAddr("udp", ipadAddr)
	if err != nil {
		panic(err)
	}
	outConn, err = net.DialUDP("udp", nil, addr)

	// Start the heartbeat message loop in the background, once per second.
	go heartBeatSender()

	reader := bufio.NewReader(os.Stdin)

	for {
		buf, _ := reader.ReadString('\n')
		o := parseInput(buf)
		if o != nil {
			relayUplinkMessage(o)
		}
	}

}
