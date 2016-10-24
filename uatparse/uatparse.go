package uatparse

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
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
	FISB_month   uint32
	FISB_day     uint32
	FISB_hours   uint32
	FISB_minutes uint32
	FISB_seconds uint32

	FISB_length uint32

	frame_length uint32
	Frame_type   uint32

	Product_id uint32
	// Text data, if applicable.
	Text_data []string

	// Flags.
	a_f bool
	g_f bool
	p_f bool
	s_f bool //TODO: Segmentation.

	// For AIRMET/NOTAM.
	//FIXME: Temporary.
	Points             []GeoPoint
	ReportNumber       uint16
	ReportYear         uint16
	LocationIdentifier string
	RecordFormat       uint8
	ReportStart        string
	ReportEnd          string

	// For NEXRAD.
	NEXRAD []NEXRADBlock
}

type UATMsg struct {
	// Metadata from demodulation.
	RS_Err         int
	SignalStrength int
	msg            []byte
	decoded        bool
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

// Decodes the time format and aligns 'FISB_data' accordingly.
//TODO: Make a new "FISB Time" structure that also encodes the type of timestamp received.
//TODO: pass up error.
func (f *UATFrame) decodeTimeFormat() {
	if len(f.Raw_data) < 3 {
		return // Can't determine time format.
	}

	t_opt := ((uint32(f.Raw_data[1]) & 0x01) << 1) | (uint32(f.Raw_data[2]) >> 7)

	var fisb_data []byte
	switch t_opt {
	case 0: // Hours, Minutes.
		if f.frame_length < 4 {
			return
		}
		f.FISB_hours = (uint32(f.Raw_data[2]) & 0x7c) >> 2
		f.FISB_minutes = ((uint32(f.Raw_data[2]) & 0x03) << 4) | (uint32(f.Raw_data[3]) >> 4)
		f.FISB_length = f.frame_length - 4
		fisb_data = f.Raw_data[4:]
	case 1: // Hours, Minutes, Seconds.
		if f.frame_length < 5 {
			return
		}
		f.FISB_hours = (uint32(f.Raw_data[2]) & 0x7c) >> 2
		f.FISB_minutes = ((uint32(f.Raw_data[2]) & 0x03) << 4) | (uint32(f.Raw_data[3]) >> 4)
		f.FISB_seconds = ((uint32(f.Raw_data[3]) & 0x0f) << 2) | (uint32(f.Raw_data[4]) >> 6)
		f.FISB_length = f.frame_length - 5
		fisb_data = f.Raw_data[5:]
	case 2: // Month, Day, Hours, Minutes.
		if f.frame_length < 5 {
			return
		}
		f.FISB_month = (uint32(f.Raw_data[2]) & 0x78) >> 3
		f.FISB_day = ((uint32(f.Raw_data[2]) & 0x07) << 2) | (uint32(f.Raw_data[3]) >> 6)
		f.FISB_hours = (uint32(f.Raw_data[3]) & 0x3e) >> 1
		f.FISB_minutes = ((uint32(f.Raw_data[3]) & 0x01) << 5) | (uint32(f.Raw_data[4]) >> 3)
		f.FISB_length = f.frame_length - 5
		fisb_data = f.Raw_data[5:]
	case 3: // Month, Day, Hours, Minutes, Seconds.
		if f.frame_length < 6 {
			return
		}
		f.FISB_month = (uint32(f.Raw_data[2]) & 0x78) >> 3
		f.FISB_day = ((uint32(f.Raw_data[2]) & 0x07) << 2) | (uint32(f.Raw_data[3]) >> 6)
		f.FISB_hours = (uint32(f.Raw_data[3]) & 0x3e) >> 1
		f.FISB_minutes = ((uint32(f.Raw_data[3]) & 0x01) << 5) | (uint32(f.Raw_data[4]) >> 3)
		f.FISB_seconds = ((uint32(f.Raw_data[4]) & 0x03) << 3) | (uint32(f.Raw_data[5]) >> 5)
		f.FISB_length = f.frame_length - 6
		fisb_data = f.Raw_data[6:]
	default:
		return // Should never reach this.
	}

	f.FISB_data = fisb_data

	if (uint16(f.Raw_data[1]) & 0x02) != 0 {
		f.s_f = true // Default false.
	}
}

// Format newlines.
func formatDLACData(p string) []string {
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
	return ret
}

// Whole frame contents is DLAC encoded text.
func (f *UATFrame) decodeTextFrame() {
	if len(f.FISB_data) < int(f.FISB_length) {
		return
	}

	p := dlac_decode(f.FISB_data, f.FISB_length)

	f.Text_data = formatDLACData(p)
}

// Gets month, day, hours, minutes.
// Formats into a string.
func airmetParseDate(b []byte, date_time_format uint8) string {
	switch date_time_format {
	case 0: // No date/time used.
		return ""
	case 1: // Month, Day, Hours, Minutes.
		month := uint8(b[0])
		day := uint8(b[1])
		hours := uint8(b[2])
		minutes := uint8(b[3])
		return fmt.Sprintf("%02d-%02d %02d:%02d", month, day, hours, minutes)
	case 2: // Day, Hours, Minutes.
		day := uint8(b[0])
		hours := uint8(b[1])
		minutes := uint8(b[2])
		return fmt.Sprintf("%02d %02d:%02d", day, hours, minutes)
	case 3: // Hours, Minutes.
		hours := uint8(b[0])
		minutes := uint8(b[1])
		return fmt.Sprintf("%02d:%02d", hours, minutes)
	}

	return ""
}

func airmetLatLng(lat_raw, lng_raw int32, alt bool) (float64, float64) {
	fct := float64(0.000687)
	if alt {
		fct = float64(0.001373)
	}
	lat := fct * float64(lat_raw)
	lng := fct * float64(lng_raw)
	if lat > 90.0 {
		lat = lat - 180.0
	}
	if lng > 180.0 {
		lng = lng - 360.0
	}
	return lat, lng
}

//TODO: Ignoring flags (segmentation, etc.)
// Aero_FISB_ProdDef_Rev4.pdf
// Decode product IDs 8-13.
func (f *UATFrame) decodeAirmet() {
	// APDU header: 48 bits  (3-3) - assume no segmentation.

	record_format := (uint8(f.FISB_data[0]) & 0xF0) >> 4
	f.RecordFormat = record_format
	fmt.Fprintf(ioutil.Discard, "record_format=%d\n", record_format)
	product_version := (uint8(f.FISB_data[0]) & 0x0F)
	fmt.Fprintf(ioutil.Discard, "product_version=%d\n", product_version)
	record_count := (uint8(f.FISB_data[1]) & 0xF0) >> 4
	fmt.Fprintf(ioutil.Discard, "record_count=%d\n", record_count)
	location_identifier := dlac_decode(f.FISB_data[2:], 3)
	fmt.Fprintf(ioutil.Discard, "%s\n", hex.Dump(f.FISB_data))
	f.LocationIdentifier = location_identifier
	fmt.Fprintf(ioutil.Discard, "location_identifier=%s\n", location_identifier)
	record_reference := (uint8(f.FISB_data[5])) //FIXME: Special values. 0x00 means "use location_identifier". 0xFF means "use different reference". (4-3).
	fmt.Fprintf(ioutil.Discard, "record_reference=%d\n", record_reference)
	// Not sure when this is even used.
	// rwy_designator := (record_reference & FC) >> 4
	// parallel_rwy_designator := record_reference & 0x03 // 0 = NA, 1 = R, 2 = L, 3 = C (Figure 4-2).

	//FIXME: Assume one record.
	if record_count != 1 {
		fmt.Fprintf(ioutil.Discard, "record_count=%d, != 1\n", record_count)
		return
	}
	/*
		0 - No data
		1 - Unformatted ASCII Text
		2 - Unformatted DLAC Text
		3 - Unformatted DLAC Text w/ dictionary
		4 - Formatted Text using ASN.1/PER
		5-7 - Future Use
		8 - Graphical Overlay
		9-15 - Future Use
	*/
	switch record_format {
	case 2:
		record_length := (uint16(f.FISB_data[6]) << 8) | uint16(f.FISB_data[7])
		if len(f.FISB_data)-int(record_length) < 6 {
			fmt.Fprintf(ioutil.Discard, "FISB record not long enough: record_length=%d, len(f.FISB_data)=%d\n", record_length, len(f.FISB_data))
			return
		}
		fmt.Fprintf(ioutil.Discard, "record_length=%d\n", record_length)
		// Report identifier = report number + report year.
		report_number := (uint16(f.FISB_data[8]) << 6) | ((uint16(f.FISB_data[9]) & 0xFC) >> 2)
		f.ReportNumber = report_number
		fmt.Fprintf(ioutil.Discard, "report_number=%d\n", report_number)
		report_year := ((uint16(f.FISB_data[9]) & 0x03) << 5) | ((uint16(f.FISB_data[10]) & 0xF8) >> 3)
		f.ReportYear = report_year
		fmt.Fprintf(ioutil.Discard, "report_year=%d\n", report_year)
		report_status := (uint8(f.FISB_data[10]) & 0x04) >> 2 //TODO: 0 = cancelled, 1 = active.
		fmt.Fprintf(ioutil.Discard, "report_status=%d\n", report_status)
		fmt.Fprintf(ioutil.Discard, "record_length=%d,len=%d\n", record_length, len(f.FISB_data))
		text_data_len := record_length - 5
		text_data := dlac_decode(f.FISB_data[11:], uint32(text_data_len))
		fmt.Fprintf(ioutil.Discard, "text_data=%s\n", text_data)
		f.Text_data = formatDLACData(text_data)
	case 8:
		// (6-1). (6.22 - Graphical Overlay Record Format).
		record_data := f.FISB_data[6:] // Start after the record header.
		record_length := (uint16(record_data[0]) << 2) | ((uint16(record_data[1]) & 0xC0) >> 6)
		fmt.Fprintf(ioutil.Discard, "record_length=%d\n", record_length)
		// Report identifier = report number + report year.
		report_number := ((uint16(record_data[1]) & 0x3F) << 8) | uint16(record_data[2])
		f.ReportNumber = report_number
		fmt.Fprintf(ioutil.Discard, "report_number=%d\n", report_number)
		report_year := (uint16(record_data[3]) & 0xFE) >> 1
		f.ReportYear = report_year
		fmt.Fprintf(ioutil.Discard, "report_year=%d\n", report_year)
		overlay_record_identifier := ((uint8(record_data[4]) & 0x1E) >> 1) + 1 // Document instructs to add 1.
		fmt.Fprintf(ioutil.Discard, "overlay_record_identifier=%d\n", overlay_record_identifier)
		object_label_flag := uint8(record_data[4] & 0x01)
		fmt.Fprintf(ioutil.Discard, "object_label_flag=%d\n", object_label_flag)

		if object_label_flag == 0 { // Numeric index.
			object_label := (uint8(record_data[5]) << 8) | uint8(record_data[6])
			record_data = record_data[7:]
			fmt.Fprintf(ioutil.Discard, "object_label=%d\n", object_label)
		} else {
			object_label := dlac_decode(record_data[5:], 9)
			record_data = record_data[14:]
			fmt.Fprintf(ioutil.Discard, "object_label=%s\n", object_label)
		}

		element_flag := (uint8(record_data[0]) & 0x80) >> 7
		fmt.Fprintf(ioutil.Discard, "element_flag=%d\n", element_flag)
		qualifier_flag := (uint8(record_data[0]) & 0x40) >> 6
		fmt.Fprintf(ioutil.Discard, "qualifier_flag=%d\n", qualifier_flag)
		param_flag := (uint8(record_data[0]) & 0x20) >> 5
		fmt.Fprintf(ioutil.Discard, "param_flag=%d\n", param_flag)
		object_element := uint8(record_data[0]) & 0x1F
		fmt.Fprintf(ioutil.Discard, "object_element=%d\n", object_element)

		object_type := (uint8(record_data[1]) & 0xF0) >> 4
		fmt.Fprintf(ioutil.Discard, "object_type=%d\n", object_type)

		object_status := uint8(record_data[1]) & 0x0F
		fmt.Fprintf(ioutil.Discard, "object_status=%d\n", object_status)

		//FIXME
		if qualifier_flag == 0 { //TODO: Check.
			record_data = record_data[2:]
		} else {
			object_qualifier := (uint32(record_data[2]) << 16) | (uint32(record_data[3]) << 8) | uint32(record_data[4])
			fmt.Fprintf(ioutil.Discard, "object_qualifier=%d\n", object_qualifier)
			fmt.Fprintf(ioutil.Discard, "%02x%02x%02x\n", record_data[2], record_data[3], record_data[4])
			record_data = record_data[5:]
		}
		//FIXME
		//if param_flag == 0 { //TODO: Check.
		//	record_data = record_data[2:]
		//} else {
		//	//TODO.
		//	//			record_data = record_data[4:]
		//}

		record_applicability_options := (uint8(record_data[0]) & 0xC0) >> 6
		fmt.Fprintf(ioutil.Discard, "record_applicability_options=%d\n", record_applicability_options)
		date_time_format := (uint8(record_data[0]) & 0x30) >> 4
		fmt.Fprintf(ioutil.Discard, "date_time_format=%d\n", date_time_format)
		geometry_overlay_options := uint8(record_data[0]) & 0x0F
		fmt.Fprintf(ioutil.Discard, "geometry_overlay_options=%d\n", geometry_overlay_options)

		overlay_operator := (uint8(record_data[1]) & 0xC0) >> 6
		fmt.Fprintf(ioutil.Discard, "overlay_operator=%d\n", overlay_operator)

		overlay_vertices_count := (uint8(record_data[1]) & 0x3F) + 1 // Document instructs to add 1. (6.20).
		fmt.Fprintf(ioutil.Discard, "overlay_vertices_count=%d\n", overlay_vertices_count)

		// Parse all of the dates.
		switch record_applicability_options {
		case 0: // No times given. UFN.
			record_data = record_data[2:]
		case 1: // Start time only. WEF.
			f.ReportStart = airmetParseDate(record_data[2:], date_time_format)
			record_data = record_data[6:]
		case 2: // End time only. TIL.
			f.ReportEnd = airmetParseDate(record_data[2:], date_time_format)
			record_data = record_data[6:]
		case 3: // Both start and end times. WEF.
			f.ReportStart = airmetParseDate(record_data[2:], date_time_format)
			f.ReportEnd = airmetParseDate(record_data[6:], date_time_format)
			record_data = record_data[10:]
		}

		// Now we have the vertices.
		switch geometry_overlay_options {
		case 3: // Extended Range 3D Polygon (MSL).
			points := make([]GeoPoint, 0) // Slice containing all of the points.
			fmt.Fprintf(ioutil.Discard, "%d\n", len(record_data))
			for i := 0; i < int(overlay_vertices_count); i++ {
				lng_raw := (int32(record_data[6*i]) << 11) | (int32(record_data[6*i+1]) << 3) | (int32(record_data[6*i+2]) & 0xE0 >> 5)
				lat_raw := ((int32(record_data[6*i+2]) & 0x1F) << 14) | (int32(record_data[6*i+3]) << 6) | ((int32(record_data[6*i+4]) & 0xFC) >> 2)
				alt_raw := ((int32(record_data[6*i+4]) & 0x03) << 8) | int32(record_data[6*i+5])

				fmt.Fprintf(ioutil.Discard, "lat_raw=%d, lng_raw=%d, alt_raw=%d\n", lat_raw, lng_raw, alt_raw)
				lat, lng := airmetLatLng(lat_raw, lng_raw, false)

				alt := alt_raw * 100
				fmt.Fprintf(ioutil.Discard, "lat=%f,lng=%f,alt=%d\n", lat, lng, alt)
				fmt.Fprintf(ioutil.Discard, "coord:%f,%f\n", lat, lng)
				var point GeoPoint
				point.Lat = lat
				point.Lon = lng
				point.Alt = alt
				points = append(points, point)
				f.Points = points
			}
		case 9: // Extended Range 3D Point (AGL). p.47.
			if len(record_data) < 6 {
				fmt.Fprintf(ioutil.Discard, "invalid data: Extended Range 3D Point. Should be 6 bytes; % seen.\n", len(record_data))
			} else {
				lng_raw := (int32(record_data[0]) << 11) | (int32(record_data[1]) << 3) | (int32(record_data[2]) & 0xE0 >> 5)
				lat_raw := ((int32(record_data[2]) & 0x1F) << 14) | (int32(record_data[3]) << 6) | ((int32(record_data[4]) & 0xFC) >> 2)
				alt_raw := ((int32(record_data[4]) & 0x03) << 8) | int32(record_data[5])

				fmt.Fprintf(ioutil.Discard, "lat_raw=%d, lng_raw=%d, alt_raw=%d\n", lat_raw, lng_raw, alt_raw)
				lat, lng := airmetLatLng(lat_raw, lng_raw, false)

				alt := alt_raw * 100
				fmt.Fprintf(ioutil.Discard, "lat=%f,lng=%f,alt=%d\n", lat, lng, alt)
				fmt.Fprintf(ioutil.Discard, "coord:%f,%f\n", lat, lng)
				var point GeoPoint
				point.Lat = lat
				point.Lon = lng
				point.Alt = alt
				f.Points = []GeoPoint{point}
			}
		case 7, 8: // Extended Range Circular Prism (7 = MSL, 8 = AGL)
			if len(record_data) < 14 {
				fmt.Fprintf(ioutil.Discard, "invalid data: Extended Range Circular Prism. Should be 14 bytes; % seen.\n", len(record_data))
			} else {

				lng_bot_raw := (int32(record_data[0]) << 10) | (int32(record_data[1]) << 2) | (int32(record_data[2]) & 0xC0 >> 6)
				lat_bot_raw := ((int32(record_data[2]) & 0x3F) << 12) | (int32(record_data[3]) << 4) | ((int32(record_data[4]) & 0xF0) >> 4)
				lng_top_raw := ((int32(record_data[4]) & 0x0F) << 14) | (int32(record_data[5]) << 6) | ((int32(record_data[6]) & 0xFC) >> 2)
				lat_top_raw := ((int32(record_data[6]) & 0x03) << 16) | (int32(record_data[7]) << 8) | int32(record_data[8])

				alt_bot_raw := (int32(record_data[9]) & 0xFE) >> 1
				alt_top_raw := ((int32(record_data[9]) & 0x01) << 6) | ((int32(record_data[10]) & 0xFC) >> 2)

				r_lng_raw := ((int32(record_data[10]) & 0x03) << 7) | ((int32(record_data[11]) & 0xFE) >> 1)
				r_lat_raw := ((int32(record_data[11]) & 0x01) << 8) | int32(record_data[12])
				alpha := int32(record_data[13])

				lat_bot, lng_bot := airmetLatLng(lat_bot_raw, lng_bot_raw, true)
				lat_top, lng_top := airmetLatLng(lat_top_raw, lng_top_raw, true)

				alt_bot := alt_bot_raw * 5
				alt_top := alt_top_raw * 500

				r_lng := float64(r_lng_raw) * float64(0.2)
				r_lat := float64(r_lat_raw) * float64(0.2)

				fmt.Fprintf(ioutil.Discard, "lat_bot, lng_bot = %f, %f\n", lat_bot, lng_bot)
				fmt.Fprintf(ioutil.Discard, "lat_top, lng_top = %f, %f\n", lat_top, lng_top)

				if geometry_overlay_options == 8 {
					fmt.Fprintf(ioutil.Discard, "alt_bot, alt_top = %d AGL, %d AGL\n", alt_bot, alt_top)
				} else {
					fmt.Fprintf(ioutil.Discard, "alt_bot, alt_top = %d MSL, %d MSL\n", alt_bot, alt_top)
				}
				fmt.Fprintf(ioutil.Discard, "r_lng, r_lat = %f, %f\n", r_lng, r_lat)

				fmt.Fprintf(ioutil.Discard, "alpha=%d\n", alpha)
			}
		default:
			fmt.Fprintf(ioutil.Discard, "unknown geometry: %d\n", geometry_overlay_options)
		}
	//case 1: // Unformatted ASCII Text.
	default:
		fmt.Fprintf(ioutil.Discard, "unknown record format: %d\n", record_format)
	}
	fmt.Fprintf(ioutil.Discard, "\n\n\n")
}

func (f *UATFrame) decodeInfoFrame() {

	if len(f.Raw_data) < 2 {
		return // Can't determine Product_id.
	}

	f.Product_id = ((uint32(f.Raw_data[0]) & 0x1f) << 6) | (uint32(f.Raw_data[1]) >> 2)

	if f.Frame_type != 0 {
		return // Not FIS-B.
	}

	f.decodeTimeFormat()

	switch f.Product_id {
	case 413:
		f.decodeTextFrame()
		/*
			case 8, 11, 13:
				f.decodeAirmet()
		*/
	case 63, 64:
		f.decodeNexradFrame()

	default:
		fmt.Fprintf(ioutil.Discard, "don't know what to do with product id: %d\n", f.Product_id)
	}

	//	logger.Printf("pos=%d,len=%d,t_opt=%d,product_id=%d, time=%d:%d\n", frame_start, frame_len, t_opt, product_id, fisb_hours, fisb_minutes)
}

func (u *UATMsg) DecodeUplink() error {
	//	position_valid := (uint32(frame[5]) & 0x01) != 0
	frame := u.msg

	if len(frame) < UPLINK_FRAME_DATA_BYTES {
		return errors.New(fmt.Sprintf("DecodeUplink: short read (%d).", len(frame)))
	}

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

		if frame_length == 0 { // Empty frame. Quit here.
			break
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

	if len(x) < 2 {
		return ret, errors.New(fmt.Sprintf("New UATMsg: Invalid format (%s).", buf))
	}

	/*
		Parse _;rs=?;ss=? - if available.
			RS_Err         int
			SignalStrength int
	*/
	ret.SignalStrength = -1
	ret.RS_Err = -1
	for _, f := range x[1:] {
		x2 := strings.Split(f, "=")
		if len(x2) != 2 {
			continue
		}
		i, err := strconv.Atoi(x2[1])
		if err != nil {
			continue
		}
		if x2[0] == "ss" {
			ret.SignalStrength = i
		} else if x2[0] == "rs" {
			ret.RS_Err = i
		}
	}
	s := x[0]

	// Only want "long" uplink messages.
	if (len(s)-1)%2 != 0 || (len(s)-1)/2 != UPLINK_FRAME_DATA_BYTES {
		return ret, errors.New(fmt.Sprintf("New UATMsg: short read (%d).", len(s)))
	}

	if s[0] != '+' { // Only want + ("Uplink") messages currently. - (Downlink) or messages that start with other are discarded.
		return ret, errors.New("New UATMsg: expecting uplink frame.")
	}

	s = s[1:] // Remove the preceding '+' or '-' character.

	// Convert the hex string into a byte array.
	frame := make([]byte, UPLINK_FRAME_DATA_BYTES)
	hex.Decode(frame, []byte(s))
	ret.msg = frame

	return ret, nil
}
