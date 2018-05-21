package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/uavionix/serial"
	"log"
	"math"
)

var radioSerialConfig *serial.Config
var radioSerialPort *serial.Port

func initUATRadioSerial() error {
	// Initialize port at 2Mbaud.
	radioSerialConfig = &serial.Config{Name: "/dev/ttyACM0", Baud: 2000000}
	p, err := serial.OpenPort(radioSerialConfig)
	if err != nil {
		log.Printf("serial port err: %s\n", err.Error())
		return errors.New("serial port failed to initialize")
	}

	radioSerialPort = p

	// Start a goroutine to watch the serial port.
	go radioSerialPortReader()
	return nil
}

/*
	radioSerialPortReader().
	 Loop to read data from the radio serial port.
*/
var radioMagic = []byte{0x0a, 0xb0, 0xcd, 0xe0}

func radioSerialPortReader() {
	tmpBuf := make([]byte, 1024) // Read buffer.
	var buf []byte               // Message buffer.
	if radioSerialPort == nil {
		return
	}
	defer radioSerialPort.Close()
	for {
		n, err := radioSerialPort.Read(tmpBuf)
		if err != nil {
			log.Printf("serial port err, shutting down radio: %s\n", err.Error())
			return
		}
		buf = append(buf, tmpBuf[:n]...)
		bufLen := len(buf)
		var finBuf []byte   // Truncated buffer, with processed messages extracted.
		var numMessages int // Number of messages extracted.
		// Search for a suitable message to extract.
		for i := 0; i < bufLen-6; i++ {
			if (buf[i] == radioMagic[0]) && (buf[i+1] == radioMagic[1]) && (buf[i+2] == radioMagic[2]) && (buf[i+3] == radioMagic[3]) {
				// Found the magic sequence. Get the length.
				msgLen := int(uint16(buf[i+4])+(uint16(buf[i+5])<<8)) + 6 // 6 bytes for rs_errors, RSSI, and TS.
				// Check if we've read enough to finish this message.
				if bufLen < i+6+msgLen {
					break // Wait for more of the message to come in.
				}
				// Message is long enough.
				processRadioMessage(buf[i+6 : i+6+msgLen])
				// Remove everything in the buffer before this message.
				finBuf = buf[i+6+msgLen:]
				numMessages++
			}
		}
		if numMessages > 0 {
			buf = finBuf
		}
	}
}

/*
	processRadioMessage().
	 Processes a single message from the radio. De-interleaves (when necessary), checks Reed-Solomon, passes to main process.
*/

func processRadioMessage(msg []byte) {
	log.Printf("processRadioMessage(): %d %s\n", len(msg), hex.EncodeToString(msg))

	// rs_errors, RSSI, and message timestamp are prepended to the actual packet.

	// rs_errors.
	rs_errors := int8(msg[0])

	// RSSI
	rssiRaw := int8(msg[1])
	rssiAdjusted := int16(rssiRaw) - 132 // -132 dBm, calculated minimum RSSI.
	l := math.Pow(10.0, float64(rssiAdjusted)/20.0)
	rssiDump978 := int16(1000.0 * l)

	// Timestamp.
	//ts := uint32(msg[2]) + (uint32(msg[3]) << 8) + (uint32(msg[4]) << 16) + (uint32(msg[5]) << 24) // Timestamp. Currently unused.

	msg = msg[6:]

	var toRelay string
	if len(msg) < 432 {
		toRelay = "-" // Downlink.
	} else {
		toRelay = "+" // Uplink.
	}

	toRelay += fmt.Sprintf("%s;ss=%d;", hex.EncodeToString(msg), rssiDump978)
	log.Printf("rs_errors=%d, msg=%s\n", rs_errors, toRelay)

	if len(toRelay) > 0 {
		o, msgtype := parseInput(toRelay)
		if o != nil && msgtype != 0 {
			relayMessage(msgtype, o)
		}
	}
}
