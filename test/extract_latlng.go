package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

const (
	UPLINK_FRAME_DATA_BYTES = 432
)

func decodeUplink(frame []byte) {
	raw_lat := (uint32(frame[0]) << 15) | (uint32(frame[1]) << 7) | (uint32(frame[2]) >> 1)

	raw_lon := ((uint32(frame[2]) & 0x01) << 23) | (uint32(frame[3]) << 15) | (uint32(frame[4]) << 7) | (uint32(frame[5]) >> 1)
	lat := float64(raw_lat) * 360.0 / 16777216.0
	lon := float64(raw_lon) * 360.0 / 16777216.0

	if lat > 90 {
		lat = lat - 180
	}
	if lon > 180 {
		lon = lon - 360
	}
	
	fmt.Printf("%.04f, %.04f\n", lat, lon)

}

func parseInput(buf string) {
	buf = strings.Trim(buf, "\r\n") // Remove newlines.
	x := strings.Split(buf, ";")    // We want to discard everything before the first ';'.
	if len(x) == 0 {
		return
	}
	s := x[0]
	if len(s) == 0 {
		return
	}
	if s[0] != '+' {
		return // Only want + ("Uplink") messages currently. - (Downlink) or messages that start with other are discarded.
	}

	s = s[1:]

	if len(s)%2 != 0 { // Bad format.
		return
	}

	if len(s)/2 != UPLINK_FRAME_DATA_BYTES {
		fmt.Printf("UPLINK_FRAME_DATA_BYTES=%d, len(s)=%d\n", UPLINK_FRAME_DATA_BYTES, len(s))
		panic("Error")
	}

	// Now, begin converting the string into a byte array.
	frame := make([]byte, UPLINK_FRAME_DATA_BYTES)
	hex.Decode(frame, []byte(s))

	// Decode the frame.
	decodeUplink(frame)
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	for {
		buf, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		parseInput(buf)
	}
}
