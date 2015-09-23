package main

import (
	"fmt"
	"./uatparse"
	"strconv"
	"os"
	"bufio"
	)


// Most adapted from extract_nexrad.c

const (
	BLOCK_WIDTH = float64(48.0/60.0)
	WIDE_BLOCK_WIDTH = float64(96.0/60.0)
	BLOCK_HEIGHT = float64(4.0/60.0)
	BLOCK_THRESHOLD = 405000
	BLOCKS_PER_RING = 450
)

type NEXRADFrame struct {
	radar_type uint32
	ts string
	scale int
	latNorth float64
	lonWest float64
	height float64
	width float64
	intensity []uint8 // Really only 4-bit values.
}


func block_location(block_num int, ns_flag bool, scale_factor int) (float64, float64, float64, float64) {
	var realScale float64
	if scale_factor == 1 {
		realScale = float64(5.0)
	} else if scale_factor == 2 {
		realScale = float64(9.0)
	} else {
		realScale = float64(1.0)
	}

	if block_num >= BLOCK_THRESHOLD {
		block_num = block_num & ^1
	}

	raw_lat := float64(BLOCK_HEIGHT * float64(int(float64(block_num) / float64(BLOCKS_PER_RING))))
	raw_lon := float64(block_num % BLOCKS_PER_RING) * BLOCK_WIDTH

	var lonSize float64
	if block_num >= BLOCK_THRESHOLD {
		lonSize = WIDE_BLOCK_WIDTH * realScale
	} else {
		lonSize = BLOCK_WIDTH * realScale
	}

	latSize := BLOCK_HEIGHT * realScale

	if ns_flag { // Southern hemisphere.
		raw_lat = 0 - raw_lat
	} else {
		raw_lat = raw_lat + BLOCK_HEIGHT
	}
	
	if raw_lon > 180.0 {
		raw_lon = raw_lon - 360.0
	}

	return raw_lat, raw_lon, latSize, lonSize

}

func decode_nexrad(f *uatparse.UATFrame) []NEXRADFrame {
	ret := make([]NEXRADFrame, 0)

	rle_flag := (uint32(f.FISB_data[0]) & 0x80) != 0
	ns_flag := (uint32(f.FISB_data[0]) & 0x40) != 0
	block_num := ((int(f.FISB_data[0]) & 0x0f) << 16) | (int(f.FISB_data[1]) << 8) | (int(f.FISB_data[2]))
	scale_factor := (int(f.FISB_data[0]) & 0x30) >> 4

	if rle_flag { // Single bin, RLE encoded.
		lat, lon, h, w := block_location(block_num, ns_flag, scale_factor)
		var tmp NEXRADFrame
		tmp.radar_type = f.Product_id
		tmp.ts = strconv.Itoa(int(f.FISB_hours)) + ":" + strconv.Itoa(int(f.FISB_minutes))
		tmp.scale = scale_factor
		tmp.latNorth = lat
		tmp.lonWest = lon
		tmp.height = h
		tmp.width = w
		tmp.intensity = make([]uint8, 0)

		intensityData := f.FISB_data[3:]
		for _, v := range intensityData {
			intensity := uint8(v) & 0x7;
			runlength := (uint8(v) >> 3) + 1
			for runlength > 0 {
				tmp.intensity = append(tmp.intensity, intensity)
				runlength--
			}
		}
		ret = append(ret, tmp)
	} else {
		var row_start int
		var row_size int
		if block_num >= 405000 {
			row_start = block_num - ((block_num - 405000) % 225)
			row_size = 225
		} else {
			row_start = block_num - (block_num % 450)
			row_size = 450
		}

		row_offset := block_num - row_start

		L := int(f.FISB_data[3] & 15)
		for i := 0; i < L; i++ {
			var bb int
			if i == 0 {
				bb = (int(f.FISB_data[3]) & 0xF0) | 0x08
			} else {
				bb = int(f.FISB_data[i+3])
			}

			for j := 0; j < 8; j++ {
				if bb & (1 << uint(j)) != 0 {
					row_x := (row_offset + 8*i + j - 3) % row_size
					bn := row_start + row_x
					lat, lon, h, w := block_location(bn, ns_flag, scale_factor)
					var tmp NEXRADFrame
					tmp.radar_type = f.Product_id
					tmp.ts = strconv.Itoa(int(f.FISB_hours)) + ":" + strconv.Itoa(int(f.FISB_minutes))
					tmp.scale = scale_factor
					tmp.latNorth = lat
					tmp.lonWest = lon
					tmp.height = h
					tmp.width = w
					tmp.intensity = make([]uint8, 0)
					for k := 0; k < 128; k++ {
						z := uint8(0)
						if f.Product_id == 64 {
							z = 1
						}
						tmp.intensity = append(tmp.intensity, z)
					}
					ret = append(ret, tmp)
				}
			}
		}
	}
	return ret
}

func parseInput(buf string) []NEXRADFrame {
	ret := make([]NEXRADFrame, 0)

	uatmsg, err := uatparse.New(buf)
	if err != nil {
		return ret
	}

	uatmsg.DecodeUplink()

	for _, uatframe := range uatmsg.Frames {
		if uatframe.Product_id != 63 && uatframe.Product_id != 64 { // It's neither CONUS nor Regional NEXRAD.
			continue
		}
		p := decode_nexrad(uatframe)
		if p != nil {
			ret = append(ret, p...)
			fmt.Printf("%s", buf)
		}
	}
	return ret
}

// Range is 0 to 360.
func oclock(ang float64) uint8 {
	if ang > 345 || ang <= 15 {
		return 12
	} else if ang > 15 && ang <= 45 {
		return 1
	} else if ang > 45 && ang <= 75 {
		return 2
	} else if ang > 75 && ang <= 105 {
		return 3
	} else if ang > 105 && ang <= 135 {
		return 4
	} else if ang > 135 && ang <= 165 {
		return 5
	} else if ang > 165 && ang <= 195 {
		return 6
	} else if ang > 195 && ang <= 225 {
		return 7
	} else if ang > 225 && ang <= 255 {
		return 8
	} else if ang > 255 && ang <= 285 {
		return 9
	} else if ang > 285 && ang <= 315 {
		return 10
	} else if ang > 315 && ang <= 345 {
		return 11
	}
	return 0
}

func main() {

	if len(os.Args) < 2 {
		fmt.Printf("%s <uat log> <lat> <lon> <hdg>\n", os.Args[0])
		return
	}

	fd, err := os.Open(os.Args[1])

	if err != nil {
		fmt.Printf("can't open file: %s\n", err.Error())
		return
	}

	reader := bufio.NewReader(fd)

	for {
		buf, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		parseInput(buf)
	}

	// Do processing.



}