package main

import (
	"encoding/hex"
	"errors"
	"github.com/uavionix/serial"
	"log"
	"unsafe"
)

/*

#cgo LDFLAGS: -ldump978

#include <stdint.h>
#include "../dump978/fec.h"

*/
import "C"

var radioSerialConfig *serial.Config
var radioSerialPort *serial.Port

func initUATRadioSerial() error {
	// Init for FEC routines.
	C.init_fec()
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
				msgLen := int(uint16(buf[i+4]) + (uint16(buf[i+5]) << 8))
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
	var toRelay string
	switch len(msg) {
	case 552:
		to := make([]byte, 552)
		var rs_errors int
		i := int(C.correct_uplink_frame((*C.uint8_t)(unsafe.Pointer(&msg[0])), (*C.uint8_t)(unsafe.Pointer(&to[0])), (*C.int)(unsafe.Pointer(&rs_errors))))
		toRelay = "+" + hex.EncodeToString(to[:432]) + ";"
		log.Printf("i=%d, rs_errors=%d, msg=%s\n", i, rs_errors, toRelay)
	case 48:
		to := make([]byte, 48)
		copy(to, msg)
		var rs_errors int
		i := int(C.correct_adsb_frame((*C.uint8_t)(unsafe.Pointer(&to[0])), (*C.int)(unsafe.Pointer(&rs_errors))))
		if i == 1 {
			// Short ADS-B frame.
			toRelay = "-" + hex.EncodeToString(to[:18]) + ";"
			log.Printf("i=%d, rs_errors=%d, msg=%s\n", i, rs_errors, toRelay)
		} else if i == 2 {
			// Long ADS-B frame.
			toRelay = "-" + hex.EncodeToString(to[:34]) + ";"
			log.Printf("i=%d, rs_errors=%d, msg=%s\n", i, rs_errors, toRelay)
		} else {
			log.Printf("i=%d, rs_errors=%d, msg=%s\n", i, rs_errors, hex.EncodeToString(to))
		}
	default:
		log.Printf("processRadioMessage(): unhandled message size %d\n", len(msg))
	}

	if len(toRelay) > 0 {
		o, msgtype := parseInput(toRelay)
		if o != nil && msgtype != 0 {
			relayMessage(msgtype, o)
		}
	}
}
