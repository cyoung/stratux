package main

import (
	"fmt"
	"../uatparse"
	"strconv"
	"os"
	"bufio"
	"github.com/kellydunn/golang-geo"
	"math"
	)


// Most adapted from extract_nexrad.c

const (
	BLOCK_WIDTH = float64(48.0/60.0)
	WIDE_BLOCK_WIDTH = float64(96.0/60.0)
	BLOCK_HEIGHT = float64(4.0/60.0)
	BLOCK_THRESHOLD = 405000
	BLOCKS_PER_RING = 450

	WARN_DIST = float64(18.52) // kilometers (10 nm).
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

func intensityToText(intensity uint8) string {
	if intensity >= 0 && intensity < 3 {
		return "light"
	} else if intensity >= 3 && intensity < 6 {
		return "moderate"
	} else if intensity == 6 {
		return "heavy"
	} else if intensity == 7 {
		return "very heavy"
	}
	return ""
}

func fixHeading(hdg float64) float64 {
	if hdg < 0 {
		return float64(hdg + 360)
	}
	if hdg >= 360 {
		return float64(hdg - 360)
	}
	return float64(hdg)
}

func scanNEXRAD(poly *geo.Polygon, frame NEXRADFrame) (*geo.Point, uint8) {
	var retpt *geo.Point
	var maxIntensity uint8
	for y := 0; y < 4; y++ {
		for x := 0; x < 32; x++ {
			intensity := frame.intensity[x + 32*y]
			lat := frame.latNorth - (float64(y) * (frame.height)/float64(4.0))
			lon := frame.lonWest + (float64(x) * (frame.width)/float64(32.0))
			pt := geo.NewPoint(lat, lon)
			if !poly.Contains(pt) { // Doesn't contain this point - skip.
				continue
			}
			if intensity > maxIntensity {
				retpt = pt
				maxIntensity = intensity
			}
		}
	}
	return retpt, maxIntensity
}

func main() {
	if len(os.Args) < 5 {
		fmt.Printf("%s <uat log> <lat> <lon> <hdg>\n", os.Args[0])
		return
	}

	fd, err := os.Open(os.Args[1])

	if err != nil {
		fmt.Printf("can't open file: %s\n", err.Error())
		return
	}

	hdg, err := strconv.Atoi(os.Args[4])
	if err != nil || hdg > 360 || hdg < 0 {
		fmt.Printf("invalid heading: %s\n", os.Args[4])
		return
	}
	lat, err := strconv.ParseFloat(os.Args[2], 64)
	if err != nil {
		fmt.Printf("invalid lat: %s\n", os.Args[2])
		return
	}
	lon, err := strconv.ParseFloat(os.Args[3], 64)
	if err != nil {
		fmt.Printf("invalid lon: %s\n", os.Args[3])
		return
	}

	hdgFloat := float64(hdg)

	frames := make([]NEXRADFrame, 0)

	reader := bufio.NewReader(fd)

	for {
		buf, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		o := parseInput(buf)
		frames = append(frames, o...)
	}

	// Do processing.
	myPos := geo.NewPoint(lat, lon)
	// We'll now draw a rectangle 20nm wide by 10nm tall in space, with the aircraft at the center of the bottom edge.
	// This gives 180 degrees of "visibility" for a decent sized area.
	nineOClock := fixHeading(hdgFloat - 90.0)
	threeOClock := fixHeading(hdgFloat + 90.0)

//	fmt.Printf("myPos=%v\n", myPos)

	leftBottom := myPos.PointAtDistanceAndBearing(WARN_DIST, nineOClock)
	rightBottom := myPos.PointAtDistanceAndBearing(WARN_DIST, threeOClock)

//	fmt.Printf("nineOClock=%f [leftBottom=%v], threeOClock=%f [rightBottom=%v]\n", nineOClock, leftBottom, threeOClock, rightBottom)

	hypDist := math.Sqrt2 * WARN_DIST
	leftTopHdg := fixHeading(hdgFloat - 45.0)
	rightTopHdg := fixHeading(hdgFloat + 45.0)

	leftTop := myPos.PointAtDistanceAndBearing(hypDist, leftTopHdg)
	rightTop := myPos.PointAtDistanceAndBearing(hypDist, rightTopHdg)

//	fmt.Printf("leftTopHdg=%f [leftTop=%v], rightTopHdg=%f [rightTop=%v]\n", leftTopHdg, leftTop, rightTopHdg, rightTop)

	points := []*geo.Point{leftTop, rightTop, rightBottom, leftBottom, leftTop}
	poly := geo.NewPolygon(points)


	var maxpt *geo.Point
	var maxIntensity uint8

	for _, frame := range frames {
		//FIXME: Scans the whole map.
		thisMaxpt, thisMaxIntensity := scanNEXRAD(poly, frame)
		if thisMaxIntensity > maxIntensity {
			maxpt = thisMaxpt
			maxIntensity = thisMaxIntensity
		}
	}

//	fmt.Printf("maxes: %d %v\n", maxIntensity, maxpt)


	if maxIntensity > 0 && maxpt != nil {
		desc := intensityToText(maxIntensity)
		direction := fixHeading(myPos.BearingTo(maxpt))
		relativeDirection := fixHeading(direction - hdgFloat)
//		fmt.Printf("direction=%f, relativeDirection=%f\n", direction, relativeDirection)
		directionDesc := oclock(relativeDirection)
		dist := myPos.GreatCircleDistance(maxpt) * float64(0.539957) // Convert km -> nm.
		fmt.Printf("%s precip %d o'clock, %0.1f nm.\n", desc, directionDesc, dist)
	} else {
		fmt.Printf("no precip.\n")
	}

}
