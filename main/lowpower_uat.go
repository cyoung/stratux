package main

import (
	"encoding/hex"
	"fmt"
	"github.com/stratux/serial"
	"log"
	"os"
	"time"
	"unsafe"
)

/*

#cgo LDFLAGS: -ldump978 -lm

#include <stdint.h>
#include "../dump978/fec.h"

*/
import "C"

var radioSerialConfig *serial.Config
var radioSerialPort *serial.Port

func initUATRadioSerial() error {
	// Init for FEC routines.
	C.init_fec()
	go func() {
		watchTicker := time.NewTicker(1 * time.Second)
		for {
			<-watchTicker.C
			// Watch for the radio or change in settings.
			if !globalSettings.UAT_Enabled || globalStatus.UATRadio_connected {
				// UAT not enabled or radio already set up. Continue.
				continue
			}
			if _, err := os.Stat("/dev/uatradio"); err != nil {
				// Device not connected.
				continue
			}
			log.Printf("===== UAT Device Name  : UATRadio v1.0 =====\n")

			// Initialize port at 2Mbaud.
			radioSerialConfig = &serial.Config{Name: "/dev/uatradio", Baud: 2000000}
			p, err := serial.OpenPort(radioSerialConfig)
			if err != nil {
				log.Printf("\tUAT Open Failed: %s\n", err.Error())
				continue
			}

			log.Printf("\tUATRadio init success.\n")

			radioSerialPort = p
			globalStatus.UATRadio_connected = true

			// Start a goroutine to watch the serial port.
			go radioSerialPortReader(radioSerialPort)
		}
	}()
	return nil
}

/*
	radioSerialPortReader().
	 Loop to read data from the radio serial port.
*/
var radioMagic = []byte{0x0a, 0xb0, 0xcd, 0xe0}

func radioSerialPortReader(serialPort *serial.Port) {
	defer func() {
		globalStatus.UATRadio_connected = false
		serialPort.Close()
	}()
	tmpBuf := make([]byte, 1024) // Read buffer.
	var buf []byte               // Message buffer.
	for {
		n, err := serialPort.Read(tmpBuf)
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
				msgLen := int(uint16(buf[i+4])+(uint16(buf[i+5])<<8)) + 5 // 5 bytes for RSSI and TS.
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
	// RSSI and message timestamp are prepended to the actual packet.

	// RSSI
	rssiRaw := int8(msg[0])
	//rssiAdjusted := int16(rssiRaw) - 132 // -132 dBm, calculated minimum RSSI.
	//rssiDump978 := int16(1000 * (10 ^ (float64(rssiAdjusted) / 20)))
	rssiDump978 := rssiRaw

	//_ := uint32(msg[1]) + (uint32(msg[2]) << 8) + (uint32(msg[3]) << 16) + (uint32(msg[4]) << 24) // Timestamp. Currently unused.

	msg = msg[5:]

	var toRelay string
	var rs_errors int
	switch len(msg) {
	case 552:
		to := make([]byte, 552)
		C.correct_uplink_frame((*C.uint8_t)(unsafe.Pointer(&msg[0])), (*C.uint8_t)(unsafe.Pointer(&to[0])), (*C.int)(unsafe.Pointer(&rs_errors)))
		toRelay = fmt.Sprintf("+%s;ss=%d;", hex.EncodeToString(to[:432]), rssiDump978)
	case 48:
		to := make([]byte, 48)
		copy(to, msg)
		i := int(C.correct_adsb_frame((*C.uint8_t)(unsafe.Pointer(&to[0])), (*C.int)(unsafe.Pointer(&rs_errors))))
		if i == 1 {
			// Short ADS-B frame.
			toRelay = fmt.Sprintf("-%s;ss=%d;", hex.EncodeToString(to[:18]), rssiDump978)
		} else if i == 2 {
			// Long ADS-B frame.
			toRelay = fmt.Sprintf("-%s;ss=%d;", hex.EncodeToString(to[:34]), rssiDump978)
		}
	default:
		log.Printf("processRadioMessage(): unhandled message size %d\n", len(msg))
	}

	if len(toRelay) > 0 && rs_errors != 9999 {
		o, msgtype := parseInput(toRelay)
		if o != nil && msgtype != 0 {
			relayMessage(msgtype, o)
		}
	}
}
