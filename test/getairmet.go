package main

import (
	"../uatparse"
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gansidui/geohash"
	"os"
	"strconv"
	"strings"
)

type UATFrame struct {
	FISB_month   uint32
	FISB_day     uint32
	FISB_hours   uint32
	FISB_minutes uint32
	FISB_seconds uint32

	Product_id uint32
	// Text data, if applicable.
	Text_data []string

	// For AIRMET/NOTAM.
	//FIXME: Temporary.
	Points             map[string][]uatparse.GeoPoint
	ReportNumber       uint16
	ReportYear         uint16
	LocationIdentifier string
	RecordFormat       uint8
	ReportStart        string
	ReportEnd          string
}

var reports map[string]UATFrame

func groupPoints(f *uatparse.UATFrame) map[string][]uatparse.GeoPoint {
	// Index all of the points by GeoHash. Group points together.
	res := make(map[string][]uatparse.GeoPoint)
	precision := 5 // 6 maybe, 0.000687.
	for _, p := range f.Points {
		hash, _ := geohash.Encode(p.Lat, p.Lon, precision)
		if r, ok := res[hash]; ok {
			r = append(r, p)
			res[hash] = r
		} else {
			res[hash] = []uatparse.GeoPoint{p}
		}
	}
	return res
}

func updateReport(f *uatparse.UATFrame) {
	if f.ReportNumber == 0 || f.ReportYear == 0 || f.RecordFormat == 0 {
		return
	}
	s := strconv.Itoa(int(f.ReportNumber)) + "-" + strconv.Itoa(int(f.ReportYear))
	f.LocationIdentifier = strings.Replace(f.LocationIdentifier, "\x03", "", -1)
	if len(f.Points) == 0 && len(f.Text_data) == 0 {
		return
	}
	if p, ok := reports[s]; ok {
		if len(f.Points) > 0 {
			p.Points = groupPoints(f)
			reports[s] = p
		}
		if len(f.Text_data) > 0 {
			p.Text_data = f.Text_data
			reports[s] = p
		}
	} else {
		var z UATFrame
		z.FISB_month = f.FISB_month
		z.FISB_day = f.FISB_day
		z.FISB_hours = f.FISB_hours
		z.FISB_minutes = f.FISB_minutes
		z.FISB_seconds = f.FISB_seconds
		z.Product_id = f.Product_id
		z.Points = groupPoints(f)
		z.ReportNumber = f.ReportNumber
		z.ReportYear = f.ReportYear
		z.LocationIdentifier = f.LocationIdentifier
		z.RecordFormat = f.RecordFormat
		z.ReportStart = f.ReportStart
		z.ReportEnd = f.ReportEnd

		reports[s] = z
	}
}

func main() {
	reports = make(map[string]UATFrame)
	s := "+3cc0978aa66ca3a02100002d3f29688210000000ff0dc45e1e00000000efd305071c142d071d0300bef1e3f1900abdf823bc440abe9ee394a80ac088439a980abfefa3e45c0abef1e3f1900a248000353f6a002210000000ff003e51987c4d5060cb9cb1c30833df2c78cf87f2d74c307d77cf7c10893053857f1d70df2e72c70c1fc75c37cb9cb2cf07f3c707f3c707c17d97df7df780260000353f6a002210000000ff004146247c4d5060cb9cb1c30833df2cf3df07f2d35c307d77cf7d7b71e3881420f3417f1d70df2e72c70c1fc75c37cf0c35d797f0c307f2d707c17d97df7df780648000213c66b022102c45170000bec0487c38f50136d1202c4517bb0defcf0da0c77c79cb26a0844517830defcf0da01145e05176605f1b205f3b205f4b205f6b205176605e00943a0497660e52bf2dcc8013848145d9817c6c8145d980a80250e8145d98178013848125d98334afcb132c8145d981680250e8145d981780138480140322014e120497660cb74ac8145d981780250e8145d9810d2004e1205176605f68033131203075048013848020524890c1105120c75c37c77c79cb2b71d71c31df0cf0c054d47800;rs=31;"
	//	s := "+3c2643887cdca4802100002d3f29688210000000ff0dc45e1e00000000efd305071c142d071d0300bef1e3f1900abdf823bc440abe9ee394a80ac088439a980abfefa3e45c0abef1e3f1900a248000353f6a002210000000ff003e51987c4d5060cb9cb1c30833df2c78cf87f2d74c307d77cf7c10893053857f1d70df2e72c70c1fc75c37cb9cb2cf07f3c707f3c707c17d97df7df7806c80002d3f29682210000000ff00ce11787c04948d15480b0c8260cb8cb0d358032094e05c1832e32c34d5e04948d1548132454920605501148348063d280919281604c24481539424c832e70cf0c1e04948d154809192baeb8d3a024180d3e05c980931e1923cd833c138050558143e0cb039780430c8143e0d304d31600841a050f833c0538580948b8143e0d703858044cd7943e0cf04e014155e0c91e008c5e0c31c2f5894e008c5e0cd336040340ebc24ae8033ce11380458c407830c2dc336ae8033ce1138033ce507782644830cda814212560c3969e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000;rs=17;"

	//	s := "+3bb40f8953d8b1b0360000353f54002210000000ff006185947c4d5060cb9c70c30833df0d78d707f2e72d5f5df49fcf4c3105fc75c37cb9c70c307f1d70df3c30df0c1fc30c1fcb2c1f05f65f7f3d30c417d2cf4c3105f0545054825526604854c549616018f48315381448e1e0052141b2024e78006c80002d3f29682210000000ff00ce11787c04948d15480b0c8260cb8cb0d358032094e05c1832e32c34d5e04948d1548132454920605501148348063d280919281604c24481539424c832e70cf0c1e04948d154809192baeb8d3a024180d3e05c980931e1923cd833c138050558143e0cb039780430c8143e0d304d31600841a050f833c0538580948b8143e0d703858044cd7943e0cf04e014155e0c91e008c5e0c31c2f5894e008c5e0cd336040340ebc24ae8033ce11380458c407830c2dc336ae8033ce1138033ce507782644830cda814212560c3969e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000;rs=29;"

	//	s := "+3c2643887cdcb780480000213e955822102cc5c1000085bb887c38f50136d1202cc5c1bb0defc30ca0cb6c70d336a084c5c1830defc30ca03170603c24d48143d715280c1d48280534a0c72c34e30ca9834cb2c33c2ec303b0e36c38d38bb1c17828d2ee4e360160317069831c31d6ec46520a33cf1bb01948011cca603d55203c68131525890c5831d70df2db1c34cedc75c38c70c70d3378005700002d3f29688210000000ff28c4631e00000000efd317071c142d071d0300ce1a242695a4ce03a3e63da4cc01039cbda4ca5f633345a4c93ca31db1a4c899034aa1a4c8328379fda4c73fe39e15a4c75fa3d24da4cac703fe89a4ccf3840731a4ce1a242695a4ce1a24269540ce03a3e63d40cc01039cbd40ca5f63334540c93ca31db140c899034aa140c8328379fd40c73fe39e1540c75fa3d24d40cac703fe8940ccf384073140ce1a24269540000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000;"

	replayUATFilename := flag.Bool("stdin", false, "Read from stdin")

	flag.Parse()

	if *replayUATFilename == true {
		reader := bufio.NewReader(os.Stdin)
		for {
			text, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			msg, err := uatparse.New(text)
			if err != nil {
				//				fmt.Printf("err: %s\n", err.Error())
				//				return
				continue
			}
			msg.DecodeUplink()
			for _, frame := range msg.Frames {
				updateReport(frame)
			}
		}
	} else {

		msg, err := uatparse.New(s)

		if err != nil {
			fmt.Printf("err: %s\n", err.Error())
			return
		}

		msg.DecodeUplink()

		for _, frame := range msg.Frames {
			updateReport(frame)
		}
	}

	r := make([]UATFrame, 0)
	for _, p := range reports {
		if len(p.Points) > 0 && len(p.Text_data) > 0 {
			r = append(r, p)
		}
	}

	j, _ := json.Marshal(&r)
	fmt.Printf("%s\n", j)

}
