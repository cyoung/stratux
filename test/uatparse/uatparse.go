package uatparse

import (
	"encoding/hex"
	"errors"
	"fmt"
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

type UATFrame struct {
	Raw_data     []byte
	FISB_data    []byte
	FISB_hours   uint32
	FISB_minutes uint32

	frame_length uint32
	Frame_type   uint32

	Product_id uint32
	// Text data, if applicable.
	Text_data []string
}

type UATMsg struct {
	msg     []byte
	decoded bool
	// Station location for uplink frames, aircraft position for downlink frames.
	Lat    float64
	Lon    float64
	Frames []*UATFrame
}

func dlac_decode(data []byte, data_len uint32) string {
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

func (f *UATFrame) decodeInfoFrame() {

	f.Product_id = ((uint32(f.Raw_data[0]) & 0x1f) << 6) | (uint32(f.Raw_data[1]) >> 2)

	if f.Frame_type != 0 {
		return // Not FIS-B.
	}
	if f.frame_length < 4 {
		return // Too short for FIS-B.
	}

	t_opt := ((uint32(f.Raw_data[1]) & 0x01) << 1) | (uint32(f.Raw_data[2]) >> 7)

	if t_opt != 0 { //FIXME.
		//		fmt.Printf("don't know time format %d\n", t_opt)
		//		panic("time format")
		return
	}

	f.FISB_hours = (uint32(f.Raw_data[2]) & 0x7c) >> 2
	f.FISB_minutes = ((uint32(f.Raw_data[2]) & 0x03) << 4) | (uint32(f.Raw_data[3]) >> 4)

	fisb_length := f.frame_length - 4
	fisb_data := f.Raw_data[4:]
	f.FISB_data = fisb_data

	if f.Product_id != 413 { // Doesn't have text data that we'd be interested in, so we're done.
		return
	}

	p := dlac_decode(fisb_data, fisb_length)
	ret := make([]string, 0)
	for {
		pos := strings.Index(p, "\x1E")
		if pos == -1 {
			pos = strings.Index(p, "\x03")
			if pos == -1 {
				ret = append(ret, p)
				break
			}
		}
		ret = append(ret, p[:pos])
		p = p[pos+1:]
	}

	f.Text_data = ret

	//	logger.Printf("pos=%d,len=%d,t_opt=%d,product_id=%d, time=%d:%d\n", frame_start, frame_len, t_opt, product_id, fisb_hours, fisb_minutes)
}

func (u *UATMsg) DecodeUplink() error {
	//	position_valid := (uint32(frame[5]) & 0x01) != 0
	frame := u.msg

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

	u.Lat = lat
	u.Lon = lon

	//	utc_coupled := (uint32(frame[6]) & 0x80) != 0
	app_data_valid := (uint32(frame[6]) & 0x20) != 0
	//	slot_id := uint32(frame[6]) & 0x1f
	//	tisb_site_id := uint32(frame[7]) >> 4

	//	logger.Printf("position_valid=%t, %.04f, %.04f, %t, %t, %d, %d\n", position_valid, lat, lon, utc_coupled, app_data_valid, slot_id, tisb_site_id)

	if !app_data_valid {
		return nil // Not sure when this even happens?
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

		data = data[2 : frame_length+2]

		thisFrame := new(UATFrame)
		thisFrame.Raw_data = data
		thisFrame.frame_length = frame_length
		thisFrame.Frame_type = frame_type

		thisFrame.decodeInfoFrame()

		// Save the decoded frame.
		u.Frames = append(u.Frames, thisFrame)

		pos = pos + int(frame_length)
	}

	u.decoded = true
	return nil
}

/*
	Aggregate all of the text rates across the frames in the message and return as an array.
*/

func (u *UATMsg) GetTextReports() ([]string, error) {
	ret := make([]string, 0)
	if !u.decoded {
		err := u.DecodeUplink()
		if err != nil {
			return ret, err
		}
	}

	for _, f := range u.Frames {
		for _, m := range f.Text_data {
			if len(m) > 0 {
				ret = append(ret, m)
			}
		}
	}

	return ret, nil
}

/*
	Parse out the message from the "dump978" output format.
*/

func New(buf string) (*UATMsg, error) {
	ret := new(UATMsg)

	buf = strings.Trim(buf, "\r\n") // Remove newlines.
	x := strings.Split(buf, ";")    // We want to discard everything before the first ';'.

	s := x[0]

	// Only want "long" uplink messages.
	if (len(s)-1)%2 != 0 || (len(s)-1)/2 != UPLINK_FRAME_DATA_BYTES {
		return ret, errors.New(fmt.Sprintf("New UATMsg: short read (%d).", len(s)))
	}

	if s[0] != '+' { // Only want + ("Uplink") messages currently. - (Downlink) or messages that start with other are discarded.
		return ret, errors.New("New UATMsg: expecting uplink frames.")
	}

	s = s[1:]

	// Convert the hex string into a byte array.
	frame := make([]byte, UPLINK_FRAME_DATA_BYTES)
	hex.Decode(frame, []byte(s))
	ret.msg = frame

	return ret, nil
}
