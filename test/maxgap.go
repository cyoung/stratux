package main


import (
	"fmt"
//	"time"
	"../uatparse"
	"os"
	"bufio"
	"strings"
	"unicode"
	"strconv"
	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/plotutil"
	"github.com/gonum/plot/vg"
	"sort"
)

const (
	UPLINK_FRAME_DATA_BYTES = 432
)


/*

From AC 00-45G [http://www.faa.gov/documentLibrary/media/Advisory_Circular/AC_00-45G_CHG_1-2.pdf]

1.3.7.1 Flight Information Service – Broadcast (FIS-B)

Table 1-1. FIS-B Over UAT Product Update and Transmission Intervals:

Product											FIS-B Over UAT Service Update Intervals(1)				FIS-B Service Transmission Intervals(2)
AIRMET											As available											5 minutes
Convective SIGMET								As available											5 minutes
METARs / SPECIs									1 minute / as available									5 minutes
NEXRAD Composite Reflectivity (CONUS)			15 minutes												15 minutes
NEXRAD Composite Reflectivity (Regional)		5 minutes												2.5 minutes
NOTAMs-D/FDC/TFR								As available											10 minutes
PIREP											As available											10 minutes
SIGMET											As available											5 minutes
Special Use Airspace Status						As available											10 minutes
TAF/AMEND										8 hours/as available									10 minutes
Temperature Aloft								12 hours												10 minutes
Winds Aloft										12 hours												10 minutes


(1) The Update Interval is the rate at which the product data is available from the source.
(2) The Transmission Interval is the amount of time within which a new or updated product transmission must be
  completed and the rate or repetition interval at which the product is rebroadcast.

*/

func append_metars(rawUplinkMessage string, curMetars []string)  []string {
	ret := curMetars

	uatMsg, err := uatparse.New(rawUplinkMessage)
	if err != nil {
		return ret
	}
//fmt.Printf("*************************\n")
	metars, _ := uatMsg.GetTextReports()
	for _, v := range metars {
//fmt.Printf("EE: %s\n", v)
		vSplit := strings.Split(v, " ")
		if vSplit[0] != "METAR" || len(vSplit) < 3 { // Only looking for METARs.
			continue
		}
		ret = append(ret, v)
	}
//fmt.Printf("=========================\n")
	
	return ret
}


/*
	Average number of METARs received for an airport for which you first received a METAR in the first 5 minutes, over 10 minutes. Divided by two.
*/

func metar_qos_one_period(a, b []string) float64 {
	numMetarByIdent := make(map[string]uint)
	for _, v := range a {
		vSplit := strings.Split(v, " ")
		numMetarByIdent[vSplit[1]]++
	}
	// b is treated differently - new airports in b aren't counted.
	for _, v := range b {
		vSplit := strings.Split(v, " ")
		if _, ok := numMetarByIdent[vSplit[1]]; ok {
			numMetarByIdent[vSplit[1]]++
		}
	}
	// Final count.
	ret := float64(0.0)
	for _, num := range numMetarByIdent {
		ret += float64(num)
	}
	if len(numMetarByIdent) > 0 {
		ret = ret / float64(2 * len(numMetarByIdent))
	}
	return ret
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("%s <replay log>\n", os.Args[0])
		return
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("error opening '%s': %s\n", os.Args[1], err.Error())
		return
	}
	rdr := bufio.NewReader(f)

	// For now, "windows" are 5 minute intervals.
	qos := make(map[int64]float64) // window number -> qos value
	curWindow := int64(0)
	windowOffset := int64(0)
	metarsByWindow := make(map[int64][]string)
	for {
		buf, err := rdr.ReadString('\n')
		if err != nil {
			break
		}
		buf = strings.TrimFunc(buf, func(r rune) bool {return unicode.IsControl(r)})
		linesplit := strings.Split(buf, ",")
		if len(linesplit) < 2 { // Blank line or invalid.
			continue
		}
		if linesplit[0] == "START" { // Reset ticker, new start.
			//TODO: Support multiple sessions.
			// Reset the counters, new session.
//			qos = make(map[uint]float64)
//			curWindowMetars = make([]string, 0)
//			curWindow = 0
			windowOffset = curWindow
		} else { // If it's not "START", then it's a tick count.
			i, err := strconv.ParseInt(linesplit[0], 10, 64)
			if err != nil {
				fmt.Printf("invalid tick: '%s'\n\n\n%s\n", linesplit[0], buf)
				continue
			}

			// Window number in current session.
			wnum := int64(i / (5 * 60 * 1000000000))
//			fmt.Printf("%d\n", curWindow)
			if wnum + windowOffset != curWindow { // Switched over.
				curWindow = wnum + windowOffset
				beforeLastWindowMetars, ok := metarsByWindow[curWindow - 2]
				lastWindowMetars, ok2 := metarsByWindow[curWindow - 1]
				if ok && ok2 {
			//		fmt.Printf("%v\n\n\nheyy\n\n%v\n", beforeLastWindowMetars, lastWindowMetars)
					qos[curWindow - 1] = metar_qos_one_period(beforeLastWindowMetars, lastWindowMetars)
					fmt.Printf("qos=%f\n", qos[curWindow - 1])
					delete(metarsByWindow, curWindow - 2)
					delete(metarsByWindow, curWindow - 1)
				}
			}
			metarsByWindow[curWindow] = append_metars(linesplit[1], metarsByWindow[curWindow])
		}
	}

	// Make graph.
	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	p.Title.Text = "Stratux FIS-B QoS vs. Time"
	p.X.Label.Text = "5 min intervals"
	p.Y.Label.Text = "QoS"

	// Loop through an ordered list of the periods, so that the line connects the right dots.
	var keys []int
	for k := range qos {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	pts := make(plotter.XYs, len(qos))
	i := 0
	for _,k := range keys {
		v := qos[int64(k)]
		fmt.Printf("%d, %f\n", k, v)
		pts[i].X = float64(k)
		pts[i].Y = v
		i++
	}

	err = plotutil.AddLinePoints(p, "UAT", pts)
	if err != nil {
		panic(err)
	}
	if err := p.Save(4 * vg.Inch, 4 * vg.Inch, "qos.png"); err != nil {
		panic(err)
	}
}
