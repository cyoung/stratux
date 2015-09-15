package main


import (
	"fmt"
//	"time"
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

	// For now, "windows" are 1 minute intervals.
	ppm := make(map[int64]int64) // window number -> pkts
	curWindow := int64(0)
	windowOffset := int64(0)
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
			windowOffset = curWindow
		} else { // If it's not "START", then it's a tick count.
			i, err := strconv.ParseInt(linesplit[0], 10, 64)
			if err != nil {
				fmt.Printf("invalid tick: '%s'\n\n\n%s\n", linesplit[0], buf)
				continue
			}

			// Window number in current session.
			wnum := int64(i / (60 * 1000000000))
//			fmt.Printf("%d\n", curWindow)
			if wnum + windowOffset != curWindow { // Switched over.
				curWindow = wnum + windowOffset
				fmt.Printf("ppm=%d\n", ppm[curWindow - 1])
			}
			ppm[curWindow]++
		}
	}

	// Make graph.
	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	p.Title.Text = "Stratux PPM vs. Time"
	p.X.Label.Text = "1 min intervals"
	p.Y.Label.Text = "PPM"

	// Loop through an ordered list of the periods, so that the line connects the right dots.
	var keys []int
	for k := range ppm {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	pts := make(plotter.XYs, len(ppm))
	i := 0
	for _,k := range keys {
		v := ppm[int64(k)]
		fmt.Printf("%d, %d\n", k, v)
		pts[i].X = float64(k)
		pts[i].Y = float64(v)
		i++
	}

	err = plotutil.AddLinePoints(p, "UAT", pts)
	if err != nil {
		panic(err)
	}
	if err := p.Save(4 * vg.Inch, 4 * vg.Inch, "ppm.png"); err != nil {
		panic(err)
	}
}