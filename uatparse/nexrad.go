package uatparse

import ()

const (
	BLOCK_WIDTH      = float64(48.0 / 60.0)
	WIDE_BLOCK_WIDTH = float64(96.0 / 60.0)
	BLOCK_HEIGHT     = float64(4.0 / 60.0)
	BLOCK_THRESHOLD  = 405000
	BLOCKS_PER_RING  = 450
)

type NEXRADBlock struct {
	Radar_Type uint32
	Scale      int
	LatNorth   float64
	LonWest    float64
	Height     float64
	Width      float64
	Intensity  []uint16 // Really only 4-bit values, but using this as a hack for the JSON encoding.
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

	raw_lat := float64(BLOCK_HEIGHT * float64(int(float64(block_num)/float64(BLOCKS_PER_RING))))
	raw_lon := float64(block_num%BLOCKS_PER_RING) * BLOCK_WIDTH

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

func (f *UATFrame) decodeNexradFrame() {
	if len(f.FISB_data) < 4 { // Short read.
		return
	}

	rle_flag := (uint32(f.FISB_data[0]) & 0x80) != 0
	ns_flag := (uint32(f.FISB_data[0]) & 0x40) != 0
	block_num := ((int(f.FISB_data[0]) & 0x0f) << 16) | (int(f.FISB_data[1]) << 8) | (int(f.FISB_data[2]))
	scale_factor := (int(f.FISB_data[0]) & 0x30) >> 4

	if rle_flag { // Single bin, RLE encoded.
		lat, lon, h, w := block_location(block_num, ns_flag, scale_factor)
		var tmp NEXRADBlock
		tmp.Radar_Type = f.Product_id
		tmp.Scale = scale_factor
		tmp.LatNorth = lat
		tmp.LonWest = lon
		tmp.Height = h
		tmp.Width = w
		tmp.Intensity = make([]uint16, 0)

		intensityData := f.FISB_data[3:]
		for _, v := range intensityData {
			intensity := uint16(v) & 0x7
			runlength := (uint16(v) >> 3) + 1
			for runlength > 0 {
				tmp.Intensity = append(tmp.Intensity, intensity)
				runlength--
			}
		}
		f.NEXRAD = []NEXRADBlock{tmp}
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

		if len(f.FISB_data) < L+3 { // Short read.
			return
		}

		for i := 0; i < L; i++ {
			var bb int
			if i == 0 {
				bb = (int(f.FISB_data[3]) & 0xF0) | 0x08
			} else {
				bb = int(f.FISB_data[i+3])
			}

			for j := 0; j < 8; j++ {
				if bb&(1<<uint(j)) != 0 {
					row_x := (row_offset + 8*i + j - 3) % row_size
					bn := row_start + row_x
					lat, lon, h, w := block_location(bn, ns_flag, scale_factor)
					var tmp NEXRADBlock
					tmp.Radar_Type = f.Product_id
					tmp.Scale = scale_factor
					tmp.LatNorth = lat
					tmp.LonWest = lon
					tmp.Height = h
					tmp.Width = w
					tmp.Intensity = make([]uint16, 0)
					for k := 0; k < 128; k++ {
						z := uint16(0)
						if f.Product_id == 64 {
							z = 1
						}
						tmp.Intensity = append(tmp.Intensity, z)
					}
					f.NEXRAD = append(f.NEXRAD, tmp)
				}
			}
		}
	}

}
