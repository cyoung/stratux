package main

//usage examples
//export_datalog > multi.kml
//export_datalog -target=ownship > myflight.kml

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"github.com/twpayne/go-kml"
	"image/color"
	"fmt"
	"flag"
	"io/ioutil"
	"bytes"
	"time"
	"math/rand"
	"runtime/pprof"
)


type traffic_position struct{

	reg string
	tail string
	targettype int
	coordinates []kml.Coordinate
}

type traffic_position_alt struct{

	reg string
	tail string
	targettype int
	coordinates []kml.Coordinate
	min_alt float64
	max_alt float64
}

type traffic_position_time struct{

	reg string
	tail string
	targettype int
	coordinates []kml.Coordinate
	times []time.Time
}

var dataLogFilef string

const (
	dataLogFile    = "/var/log/stratux.sqlite"
	gpsLogPath     = "/var/log/"
)

func check(e error) {
    if e != nil {
        panic(e)
    }
}

func writeFile(name string, content *kml.CompoundElement){
	buf := new(bytes.Buffer)
	content.WriteIndent(buf, "", "  ")
	err := ioutil.WriteFile(fmt.Sprintf("%s%s.kml", gpsLogPath, name), buf.Bytes(), 0644)
    	check(err)
}

func writeKML(coordinates []kml.Coordinate) {
	k := kml.KML(
		kml.Placemark(
			kml.Name("OwnShip"),
			kml.Description("GPS data from Stratux Replaylog database mySituation Table"),
			kml.Style("ownShip",
				kml.LineStyle(kml.Color(color.White)),
			),
			kml.LineString(
				kml.AltitudeMode("absolute"),
				kml.Extrude(true),
				kml.Tessellate(true),
				kml.Coordinates(coordinates ...),
			),
		),
	)
	if err := k.WriteIndent(os.Stdout, "", "  "); err != nil {
		log.Fatal(err)
	}
}

func writeKML_multi(traffic_data []traffic_position) {
	k := kml.KML()
	d := kml.Document()
	ownnship_style := kml.Style("ownship",
		kml.LineStyle(kml.Color(color.RGBA{uint8(255), uint8(0), uint8(0), uint8(140)}),
			kml.Width(10)),
			kml.PolyStyle(kml.Color(color.RGBA{uint8(255), uint8(0), uint8(0), uint8(100)})))
	d.Add(ownnship_style)
	es_style := kml.Style("1090es",
		kml.LineStyle(kml.Color(color.RGBA{uint8(0), uint8(0), uint8(255), uint8(140)}),
			kml.Width(10)),
			kml.PolyStyle(kml.Color(color.RGBA{uint8(0), uint8(0), uint8(255), uint8(100)})))
	d.Add(ownnship_style)
	d.Add(es_style)
	UAT_style := kml.Style("UAT",
		kml.LineStyle(kml.Color(color.RGBA{uint8(0), uint8(255), uint8(0), uint8(140)}),
			kml.Width(10)),
			kml.PolyStyle(kml.Color(color.RGBA{uint8(0), uint8(255), uint8(0), uint8(100)})))
	d.Add(UAT_style)
	for _, traffic_pos := range traffic_data {
		if traffic_pos.tail == "" {
			continue
		}
		var style *kml.SharedElement
		switch traffic_pos.targettype {
		case 99:
			style = ownnship_style
		case 1:
			style = es_style
		default:
			style = UAT_style
		}
		d.Add(kml.Placemark(
				kml.Name(fmt.Sprintf("%s - %s", traffic_pos.tail, traffic_pos.reg)),
				kml.StyleURL(style),
				kml.LineString(
					kml.AltitudeMode("absolute"),
					kml.Extrude(true),
					kml.Tessellate(true),
					kml.Coordinates(traffic_pos.coordinates ...),


				),
			))
	}
	k.Add(d)
	writeFile("multi", k)
}

func writeAltKML(traffic_data []traffic_position_alt) {
	k := kml.GxKML()
	d := kml.Document()
	ownnship_style := kml.Style("ownship",
		kml.LineStyle(kml.Color(color.RGBA{uint8(255), uint8(0), uint8(0), uint8(140)}),
			kml.Width(10)),
			kml.PolyStyle(kml.Color(color.RGBA{uint8(255), uint8(0), uint8(0), uint8(100)})))
	d.Add(ownnship_style)
	es_style := kml.Style("1090es",
		kml.LineStyle(kml.Color(color.RGBA{uint8(0), uint8(0), uint8(255), uint8(140)}),
			kml.Width(1)),
			kml.PolyStyle(kml.Color(color.RGBA{uint8(0), uint8(0), uint8(255), uint8(100)})))
	d.Add(ownnship_style)
	d.Add(es_style)
	UAT_style := kml.Style("UAT",
		kml.LineStyle(kml.Color(color.RGBA{uint8(0), uint8(255), uint8(0), uint8(140)}),
			kml.Width(10)),
			kml.PolyStyle(kml.Color(color.RGBA{uint8(0), uint8(255), uint8(0), uint8(100)})))
	d.Add(UAT_style)
	for _, traffic_pos := range traffic_data {
		if traffic_pos.tail == ""|| traffic_pos.reg == "" {
			continue
		}
		GxTrack := kml.GxTrack(kml.AltitudeMode("absolute"),
					kml.Extrude(false),
					kml.Tessellate(false),)
		//kml.When(time.Date(2010, 5, 28, 2, 2, 56, 0, time.UTC)),
		//kml.GxCoord(kml.Coordinate{-122.207881, 37.371915, 156.000000}),
		start_alt := time.Date(2016, 5, 28, 0, 0, 0, 0, time.UTC)
		start_alt = start_alt.Add(time.Duration(traffic_pos.min_alt)*time.Hour)
		var length_of_data int = len(traffic_pos.coordinates)*100
		for _,coordinate := range traffic_pos.coordinates {
			GxTrack.Add(kml.When(start_alt))
			start_alt = start_alt.Add(time.Duration(length_of_data)*time.Second)
			GxTrack.Add(kml.GxCoord(coordinate))
		}

		placemark := kml.Placemark(
				kml.Name(fmt.Sprintf("%s - %s", traffic_pos.tail, traffic_pos.reg)),
				kml.Style("randrom",
					kml.LineStyle(kml.Color(color.RGBA{uint8(rand.Intn(255)), uint8(rand.Intn(255)),
										uint8(rand.Intn(255)), uint8(255)}),
			kml.Width(10)),
			kml.PolyStyle(kml.Color(color.RGBA{uint8(0), uint8(255), uint8(0), uint8(100)}))),
		)
		placemark.Add(GxTrack)
		d.Add(placemark)
	}
	k.Add(d)
	writeFile("alt", k)
}

func writeTimeKML(traffic_data traffic_position_times) {
	k := kml.GxKML()
	d := kml.Document()
	ownnship_style := kml.Style("ownship",
		kml.LineStyle(kml.Color(color.RGBA{uint8(255), uint8(0), uint8(0), uint8(140)}),
			kml.Width(10)),
			kml.PolyStyle(kml.Color(color.RGBA{uint8(255), uint8(0), uint8(0), uint8(100)})))
	d.Add(ownnship_style)
	es_style := kml.Style("1090es",
		kml.LineStyle(kml.Color(color.RGBA{uint8(0), uint8(0), uint8(255), uint8(140)}),
			kml.Width(1)),
			kml.PolyStyle(kml.Color(color.RGBA{uint8(0), uint8(0), uint8(255), uint8(100)})))
	d.Add(ownnship_style)
	d.Add(es_style)
	UAT_style := kml.Style("UAT",
		kml.LineStyle(kml.Color(color.RGBA{uint8(0), uint8(255), uint8(0), uint8(140)}),
			kml.Width(10)),
			kml.PolyStyle(kml.Color(color.RGBA{uint8(0), uint8(255), uint8(0), uint8(100)})))
	d.Add(UAT_style)
	for traffic_pos, _ := range traffic_data {
		if traffic_data[traffic_pos].tail == ""|| traffic_data[traffic_pos].reg == "" {
			continue
		}
		GxTrack := kml.GxTrack(kml.AltitudeMode("absolute"),
					kml.Extrude(false),
					kml.Tessellate(false),)
		//kml.When(time.Date(2010, 5, 28, 2, 2, 56, 0, time.UTC)),
		//kml.GxCoord(kml.Coordinate{-122.207881, 37.371915, 156.000000}),
		for index,coordinate := range traffic_data[traffic_pos].coordinates {
			GxTrack.Add(kml.When(traffic_data[traffic_pos].times[index]))
			GxTrack.Add(kml.GxCoord(coordinate))
		}

		placemark := kml.Placemark(
				kml.Name(fmt.Sprintf("%s - %s", traffic_data[traffic_pos].tail, traffic_data[traffic_pos].reg)),
				kml.Style("randrom",
					kml.LineStyle(kml.Color(color.RGBA{uint8(rand.Intn(255)), uint8(rand.Intn(255)),
										uint8(rand.Intn(255)), uint8(255)}),
			kml.Width(10)),
			kml.PolyStyle(kml.Color(color.RGBA{uint8(0), uint8(255), uint8(0), uint8(100)}))),
		)
		placemark.Add(GxTrack)
		d.Add(placemark)
	}
	k.Add(d)
	writeFile("time", k)
}

func dataLogReader(db *sql.DB, query string)(rows *sql.Rows){
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	return rows

}

func readCoordinates(db *sql.DB, query string) (coordinates []kml.Coordinate){
	rows := dataLogReader(db, query)
	defer rows.Close()
	//Coords structure https://play.golang.org/p/RLergI3WyN
	for rows.Next() {
		var lat, lng, alt float64
		rows.Scan( &lng, &lat, &alt)
		coordinates = append(coordinates, kml.Coordinate{lng,lat,alt/3.28084})
	}
	return coordinates

}

func get_traffic_list(db *sql.DB, query string) (traffic_list []string){
	rows := dataLogReader(db, query)
	defer rows.Close()
	//Coords structure https://play.golang.org/p/RLergI3WyN
	for rows.Next() {
		var Reg string
		rows.Scan( &Reg)
		traffic_list = append(traffic_list, Reg)
	}
	return traffic_list

}

func build_traffic_coords(db *sql.DB, traffic_list []string) (data []traffic_position) {
	for _, traffic := range traffic_list {
		rows := dataLogReader(db, build_query("traffic", traffic))
		defer rows.Close()
		coordinates := []kml.Coordinate{}
		var tail, reg, fancy_name string
		var targettype int
		var fancy_name_found bool
		for rows.Next() {
			var lat, lng, alt float64
			rows.Scan(&reg, &tail, &targettype, &lng, &lat, &alt)
			if !fancy_name_found && tail != reg{
				fancy_name = tail
				fancy_name_found = true
			}
			coordinates = append(coordinates, kml.Coordinate{lng, lat, alt / 3.28084})
		}
		if fancy_name_found{
			data = append(data, traffic_position{reg: reg, tail: fancy_name, targettype: targettype, coordinates: coordinates})
		} else {
			data = append(data, traffic_position{reg: reg, tail: tail, targettype: targettype, coordinates: coordinates})
		}
	}
	return data
}

func build_traffic_coords_alt(db *sql.DB, traffic_list []string) (data []traffic_position_alt) {
	for _, traffic := range traffic_list {
		rows := dataLogReader(db, build_query("traffic", traffic))
		defer rows.Close()
		coordinates := []kml.Coordinate{}
		var tail, reg, fancy_name string
		var targettype int
		var fancy_name_found bool
		var minimum_alt, maximum_alt float64
		for rows.Next() {
			var lat, lng, alt float64
			rows.Scan(&reg, &tail, &targettype, &lng, &lat, &alt)
			if minimum_alt == 0 || alt < minimum_alt {
				minimum_alt = alt / 3.28084
			}
			if maximum_alt == 0 || alt > maximum_alt {
				maximum_alt = alt / 3.28084
			}
			if !fancy_name_found && tail != reg{
				fancy_name = tail
				fancy_name_found = true
			}
			coordinates = append(coordinates, kml.Coordinate{lng, lat, alt / 3.28084})
		}
		if fancy_name_found{
			data = append(data, traffic_position_alt{reg: reg, tail: fancy_name, targettype: targettype,
				coordinates: coordinates, min_alt: minimum_alt, max_alt: maximum_alt})
		} else {
			data = append(data, traffic_position_alt{reg: reg, tail: tail, targettype: targettype,
				coordinates: coordinates, min_alt: minimum_alt, max_alt: maximum_alt})
		}
	}
	return data
}
type traffic_position_times map[string]*traffic_position_time
func build_traffic_coords_time(db *sql.DB, traffic_list []string) (d traffic_position_times) {
	rows := dataLogReader(db, build_query("traffic_time", "foo"))
	defer rows.Close()
	//coordinates := []kml.Coordinate{}
	//times := []time.Time{}
	var tail, reg string
	var targettype int
	var good_previous_time = time.Time{}
	d = make(traffic_position_times)
	for rows.Next() {
		var lat, lng, alt float64
		var GPSClock_value string
		rows.Scan(&reg, &tail, &targettype, &lng, &lat, &alt, &GPSClock_value)
		if _ , missing:=d[reg]; !missing {
			d[reg] = &traffic_position_time{reg: reg, tail: tail, targettype: targettype}
		}
		if tail != reg {
			d[reg].tail = tail
		}
		//fmt.Println(d[reg].coordinates)
		d[reg].coordinates = append(d[reg].coordinates, kml.Coordinate{lng, lat, alt / 3.28084})
		//fmt.Println(d[reg].coordinates)
		time_obj, err := time.Parse("2006-01-02 15:04:05 -0700 MST", GPSClock_value)
		if err != nil {
			if err != nil {
				fmt.Printf("%s - %s: %s \n%s\n", reg, tail, GPSClock_value, err)
				time_obj = good_previous_time.Add(time.Duration(1) * time.Second / 10)
			}
		}
		d[reg].times = append(d[reg].times, time_obj)
	}
	return d
}

func build_query(target_type, target_id string)(string){
	switch {
	    case "ownship" == target_type:
		return "select Lng, Lat, Alt from mySituation"
	    case "traffic" == target_type:
		return fmt.Sprintf("select Reg, Tail, TargetType, Lng, Lat, Alt FROM traffic WHERE Reg = '%s'", target_id)
	    case "traffic_time" == target_type:
		return "select traffic.Reg, traffic.Tail, traffic.TargetType, traffic.Lng, traffic.Lat, " +
			"traffic.Alt, timestamp.GPSClock_value FROM traffic " +
			"INNER JOIN timestamp ON traffic.timestamp_id=timestamp.id"
	    case "traffic_list" == target_type:
		return "select DISTINCT  Reg FROM traffic"
	    case "towers" == target_type:
		return "select Lng, Lat, Alt FROM traffic WHERE Reg = 'N746FD'"
	    }
	return "select Lng, Lat, Alt from mySituation"
}


func main() {
	targetPTR := flag.String("target", "", "ownship, traffic, traffic_list, towers")
	target_idPTR := flag.String("id", "", "a string containing Reg number")
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
		    log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	dataLogFilef = dataLogFile

	if _, err := os.Stat(dataLogFilef); os.IsNotExist(err) {
		log.Printf("No database exists at '%s', record a replay log first.\n", dataLogFilef)
	}
	db, err := sql.Open("sqlite3", dataLogFilef)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	//db.Exec("create virtual table repo using github(id, full_name, description, html_url)")
	//[id LastFixSinceMidnightUTC Lat Lng Quality HeightAboveEllipsoid GeoidSep Satellites
	// SatellitesTracked SatellitesSeen Accuracy NACp Alt AccuracyVert GPSVertVel
	// LastFixLocalTime TrueCourse GroundSpeed LastGroundTrackTime GPSTime LastGPSTimeTime
	// LastValidNMEAMessageTime LastValidNMEAMessage Temp Pressure_alt
	// LastTempPressTime Pitch Roll Gyro_heading LastAttitudeTime timestamp_id]
	query := build_query(*targetPTR, *target_idPTR)
	if *targetPTR != "traffic_list" && *targetPTR !="" {
		coordinates := readCoordinates(db, query)
		writeKML(coordinates)
	}
	if *targetPTR == "traffic_list" {
		fmt.Printf("%s", get_traffic_list(db, query))
	}
	if *targetPTR == "" && *targetPTR =="" {
		/*//ownship + traffic in flat KML file
		ownship_coords := readCoordinates(db, build_query("ownship", ""))
		traffic_coords := build_traffic_coords(db, get_traffic_list(db, build_query("traffic_list", "")))
		traffic_coords = append(traffic_coords, traffic_position{reg: "ownship", tail: "ownship", targettype: 99, coordinates: ownship_coords})
		writeKML_multi(traffic_coords)
		//Filter based on Minimum Altitude
		traffic_coords_alt := build_traffic_coords_alt(db, get_traffic_list(db, build_query("traffic_list", "")))
		writeAltKML(traffic_coords_alt)*/
		//Filter based on GPS Time of target
		traffic_coords_time := build_traffic_coords_time(db, get_traffic_list(db, build_query("traffic_list", "")))
		writeTimeKML(traffic_coords_time)
	}
}
