package main

import (
	"bufio"
	"fmt"
	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/plotutil"
	"github.com/gonum/plot/vg"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type XY struct {
	X float64
	Y float64
}

func imageWriter() {
	for {
		p, err := plot.New()
		if err != nil {
			panic(err)
		}
		p.Title.Text = "AHRS Plot"
		p.X.Label.Text = "Roll"
		p.Y.Label.Text = "Pitch"

		file, err := os.Open("ahrs_table.log")
		if err != nil {
			panic(err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)

		vals := make([]XY, 0)
		for scanner.Scan() {
			l := scanner.Text()
			x := strings.Split(l, ",")
			if len(x) < 3 {
				continue
			}
			roll, err := strconv.ParseFloat(x[0], 64)
			if err != nil {
				continue
			}
			pitch, err := strconv.ParseFloat(x[1], 64)
			if err != nil {
				continue
			}
			v := XY{X: roll, Y: pitch}
			vals = append(vals, v)
		}
		vals_XY := make(plotter.XYs, len(vals))
		for i := 0; i < len(vals); i++ {
			vals_XY[i].X = vals[i].X
			vals_XY[i].Y = vals[i].Y
		}
		err = plotutil.AddScatters(p, "First", vals_XY)
		if err != nil {
			panic(err)
		}
		if err := p.Save(8*vg.Inch, 8*vg.Inch, "out.png"); err != nil {
			panic(err)
		}
		time.Sleep(1000 * time.Millisecond)
	}
}

func main() {
	go imageWriter()
	http.Handle("/", http.FileServer(http.Dir(".")))
	err := http.ListenAndServe(":8080", nil)

	if err != nil {
		fmt.Printf("managementInterface ListenAndServe: %s\n", err.Error())
	}

	for {
		time.Sleep(1 * time.Second)
	}

}
