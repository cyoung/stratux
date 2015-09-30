package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
)

const (
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

	dlac_alpha = "\x03ABCDEFGHIJKLMNOPQRSTUVWXYZ\x1A\t\x1E\n| !\"#$%&'()*+,-./0123456789:;<=>?"
)

var logger *log.Logger
var buf bytes.Buffer

func dlac_decode(data []byte, data_len uint32) string {
	//	fmt.Printf("dlac on %s\n", hex.Dump(data))
	step := 0
	tab := false
	ret := ""
	for i := uint32(0); i < data_len; i++ {
		var ch uint32
		switch step {
		case 0:
			ch = uint32(data[i+0]) >> 2
		case 1:
			ch = ((uint32(data[i-1]) & 0x03) << 4) | (uint32(data[i+0]) >> 4)
		case 2:
			ch = ((uint32(data[i-1]) & 0x0f) << 2) | (uint32(data[i+0]) >> 6)
			i = i - 1
		case 3:
			ch = uint32(data[i+0]) & 0x3f
		}
		if tab {
			for ch > 0 {
				ret += " "
				ch--
			}
			tab = false
		} else if ch == 28 { // tab
			tab = true
		} else {
			ret += string(dlac_alpha[ch])
		}
		step = (step + 1) % 4
	}
	return ret
}

func decodeInfoFrame(frame []byte, frame_start int, frame_len uint32, frame_type uint32) {
	data := frame[frame_start : frame_start+int(frame_len)]

	if frame_type != 0 {
		return // Not FIS-B.
	}
	if frame_len < 4 {
		return // Too short for FIS-B.
	}

	t_opt := ((uint32(data[1]) & 0x01) << 1) | (uint32(data[2]) >> 7)
	product_id := ((uint32(data[0]) & 0x1f) << 6) | (uint32(data[1]) >> 2)
	//	fmt.Printf("%d %d\n", data[0], data[1])
	if product_id != 413 { // FIXME.
		return
	}

	if t_opt != 0 { //FIXME.
		fmt.Printf("don't know time format %d\n", t_opt)
		panic("time format")
	}

	fisb_hours := (uint32(data[2]) & 0x7c) >> 2
	fisb_minutes := ((uint32(data[2]) & 0x03) << 4) | (uint32(data[3]) >> 4)
	fisb_length := frame_len - 4
	fisb_data := data[4:]

	p := dlac_decode(fisb_data, fisb_length)
	fmt.Printf("%v\n", p)

	logger.Printf("pos=%d,len=%d,t_opt=%d,product_id=%d, time=%d:%d\n", frame_start, frame_len, t_opt, product_id, fisb_hours, fisb_minutes)
}

func decodeUplink(frame []byte) {
	position_valid := (uint32(frame[5]) & 0x01) != 0
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

	utc_coupled := (uint32(frame[6]) & 0x80) != 0
	app_data_valid := (uint32(frame[6]) & 0x20) != 0
	slot_id := uint32(frame[6]) & 0x1f
	tisb_site_id := uint32(frame[7]) >> 4

	logger.Printf("position_valid=%t, %.04f, %.04f, %t, %t, %d, %d\n", position_valid, lat, lon, utc_coupled, app_data_valid, slot_id, tisb_site_id)

	if !app_data_valid {
		return // Not sure when this even happens?
	}

	app_data := frame[8:432]
	num_info_frames := 0
	pos := 0
	total_len := len(app_data)
	for (num_info_frames < UPLINK_MAX_INFO_FRAMES) && (pos+2 <= total_len) {
		data := app_data[pos:]
		frame_length := (uint32(data[0]) << 1) | (uint32(data[1]) >> 7)
		frame_type := uint32(data[1]) & 0x0f
		if pos+int(frame_length) > total_len {
			break // Overrun?
		}
		if frame_length == 0 && frame_type == 0 {
			break // No more frames.
		}
		pos = pos + 2
		decodeInfoFrame(app_data, pos, frame_length, frame_type)
		pos = pos + int(frame_length)
	}
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
	logger.Println(buf)
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
	logger.Printf("%v\n", frame)

	// Decode the frame.
	decodeUplink(frame)
}

func main() {
	logger = log.New(&buf, "logger: ", log.Lshortfile)
	reader := bufio.NewReader(os.Stdin)
	for {
		buf, err := reader.ReadString('\n')
		if err != nil { // All done.
			break
		}
		parseInput(buf)
	}
}
