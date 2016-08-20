package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/twpayne/go-kml"
	"image/color"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"runtime/pprof"
	"time"
)

type traffic_map struct {
	reg              string
	tail             string
	target_type      int
	icao_address     uint32
	coordinates      []kml.Coordinate
	times            []time.Time
	minimum_altitude float64
	maximum_altitude float64
}

type traffic_maps map[string]*traffic_map

const (
	dataLogFile       = "/var/log/stratux.sqlite"
	gpsLogPath        = "/var/log/"
	StratuxTimeFormat = "2006-01-02 15:04:05 -0700 MST"
)

func writeFile(name string, content *kml.CompoundElement) {
	buf := new(bytes.Buffer)
	content.WriteIndent(buf, "", "  ")
	err := ioutil.WriteFile(fmt.Sprintf("%s%s.kml", gpsLogPath, name), buf.Bytes(), 0644)
	if err != nil {
		panic(err)
	}
}

func defaultKMLDocument() (document *kml.CompoundElement) {
	document = kml.Document()
	var ownship_color = kml.Color(color.RGBA{uint8(255), uint8(0), uint8(0), uint8(140)})
	var es_color = kml.Color(color.RGBA{uint8(0), uint8(0), uint8(255), uint8(140)})
	var UAT_color = kml.Color(color.RGBA{uint8(0), uint8(255), uint8(0), uint8(140)})
	ownship_style := kml.Style("ownship", kml.LineStyle(ownship_color, kml.Width(10)), kml.PolyStyle(ownship_color))
	document.Add(ownship_style)
	es_style := kml.Style("1090es", kml.LineStyle(es_color), kml.Width(1), kml.PolyStyle(es_color))
	document.Add(es_style)
	UAT_style := kml.Style("UAT", kml.LineStyle(UAT_color, kml.Width(10)), kml.PolyStyle(UAT_color))
	document.Add(UAT_style)
	return document
}

func defaultKMLPlacemark(details *traffic_map) (placemark *kml.CompoundElement) {
	var random_color = kml.Color(color.RGBA{uint8(rand.Intn(255)), uint8(rand.Intn(255)),
		uint8(rand.Intn(255)), uint8(255)})
	placemark = kml.Placemark(
		kml.Name(fmt.Sprintf("%s - %s", details.tail, details.reg)),
		kml.Style("randrom",
			kml.LineStyle(random_color, kml.Width(10)), kml.PolyStyle(random_color)),
	)
	return placemark
}

func defaultKMLGxTrack() (GxTrack *kml.CompoundElement) {
	GxTrack = kml.GxTrack(kml.AltitudeMode("absolute"),
		kml.Extrude(false),
		kml.Tessellate(false))
	return GxTrack
}

func defaultKMLFolders() (folders map[string]*kml.CompoundElement) {
	folders = make(map[string]*kml.CompoundElement)
	folders["ownship"] = kml.Folder(kml.Name("ownship"))
	folders["1090ES"] = kml.Folder(kml.Name("1090ES Traffic"))
	folders["UAT"] = kml.Folder(kml.Name("UAT Traffic"))
	return folders
}

func addToFolder(input_folders map[string]*kml.CompoundElement, traffic_pos traffic_map, placemark *kml.CompoundElement) (folders map[string]*kml.CompoundElement) {
	folders = input_folders
	switch traffic_pos.target_type {
	case 99:
		folders["ownship"].Add(placemark)
	case 1:
		folders["1090ES"].Add(placemark)
	default:
		folders["UAT"].Add(placemark)
	}
	return folders
}

func writeAltKML(traffic_data traffic_maps) {
	k := kml.GxKML()
	d := defaultKMLDocument()
	f := defaultKMLFolders()
	for traffic_pos := range traffic_data {
		GxTrack := defaultKMLGxTrack()
		start_alt := time.Date(2016, 5, 28, 0, 0, 0, 0, time.UTC)
		start_alt = start_alt.Add(time.Duration(traffic_data[traffic_pos].minimum_altitude) * time.Hour)
		var length_of_data int = len(traffic_data[traffic_pos].coordinates) * 100
		for _, coordinate := range traffic_data[traffic_pos].coordinates {
			GxTrack.Add(kml.When(start_alt))
			start_alt = start_alt.Add(time.Duration(length_of_data) * time.Microsecond)
			GxTrack.Add(kml.GxCoord(coordinate))
		}

		placemark := defaultKMLPlacemark(traffic_data[traffic_pos])
		placemark.Add(GxTrack)
		f = addToFolder(f, *traffic_data[traffic_pos], placemark)
	}
	for folder := range f {
		d.Add(f[folder])
	}
	k.Add(d)
	writeFile("alt", k)
}

func writeTimeKML(traffic_data traffic_maps) {
	k := kml.GxKML()
	d := defaultKMLDocument()
	f := defaultKMLFolders()
	for traffic_pos := range traffic_data {
		GxTrack := defaultKMLGxTrack()
		for index, coordinate := range traffic_data[traffic_pos].coordinates {
			GxTrack.Add(kml.When(traffic_data[traffic_pos].times[index]))
			GxTrack.Add(kml.GxCoord(coordinate))
		}
		placemark := defaultKMLPlacemark(traffic_data[traffic_pos])
		placemark.Add(GxTrack)
		f = addToFolder(f, *traffic_data[traffic_pos], placemark)
	}
	for folder := range f {
		d.Add(f[folder])
	}
	k.Add(d)
	writeFile("time", k)
}

func dataLogReader(db *sql.DB, query string) (rows *sql.Rows) {
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(fmt.Sprintf("func dataLogReader Query Error: %v", err))
	}
	return rows

}

func build_traffic_maps(db *sql.DB, traffic_type string) (maps traffic_maps) {
	var query string
	var traffic_row traffic_map
	switch traffic_type {
	case "ownship":
		query = "ownship"
		traffic_row.reg = query
		traffic_row.tail = query
		traffic_row.target_type = 99
		traffic_row.icao_address = 0
	default:
		query = "traffic"
	}
	rows := dataLogReader(db, build_query(query))
	defer rows.Close()
	maps = make(traffic_maps)
	for rows.Next() {
		var lat, lng, alt float64
		var GPSClock_value string
		scan_err := rows.Scan(&traffic_row.reg, &traffic_row.tail, &traffic_row.icao_address, &traffic_row.target_type, &lng, &lat, &alt, &GPSClock_value)
		if scan_err != nil && traffic_type == "ownship" {
			rows.Scan(&lng, &lat, &alt, &GPSClock_value)
		}
		if traffic_row.tail == "" || traffic_row.reg == "" {
			//Give UAT or malformed 1090es traffic clean names using ICAO string
			string_icao := fmt.Sprint(traffic_row.icao_address)
			switch {
			case traffic_row.reg == "" && traffic_row.tail == "":
				traffic_row.tail = string_icao
				traffic_row.reg = string_icao
			case traffic_row.reg == "" && traffic_row.tail != "":
				traffic_row.reg = string_icao
			case traffic_row.reg != "" && traffic_row.tail == "":
				traffic_row.tail = string_icao
			}
			if traffic_row.tail == "" {
				traffic_row.tail = fmt.Sprint(traffic_row.icao_address)
			}
			if traffic_row.reg == "" {
				traffic_row.reg = fmt.Sprint(traffic_row.icao_address)
			}
		}
		if _, missing := maps[traffic_row.reg]; !missing {
			//Create maps["N123AB"] with reg, tail and target_type
			maps[traffic_row.reg] = &traffic_map{reg: traffic_row.reg, tail: traffic_row.tail, target_type: traffic_row.target_type}
		}
		if traffic_row.tail != traffic_row.reg {
			maps[traffic_row.reg].tail = traffic_row.tail
		}
		if traffic_row.minimum_altitude == 0 || alt < maps[traffic_row.reg].minimum_altitude {
			maps[traffic_row.reg].minimum_altitude = alt
		}
		if traffic_row.maximum_altitude == 0 || alt > maps[traffic_row.reg].maximum_altitude {
			maps[traffic_row.reg].maximum_altitude = alt
		}
		time_obj, err := time.Parse(StratuxTimeFormat, GPSClock_value)
		if err != nil {
			log.Fatal(fmt.Sprintf("%s - %s: %s \n%s\n", traffic_row.reg, traffic_row.tail, GPSClock_value, err))
		}
		if time_obj.Year() < 1987 {
			//If time is not valid skip writing coordinates and time
			continue
		}
		maps[traffic_row.reg].coordinates = append(maps[traffic_row.reg].coordinates, kml.Coordinate{Lon: lng, Lat: lat, Alt: alt})
		maps[traffic_row.reg].times = append(maps[traffic_row.reg].times, time_obj)
	}
	return maps
}

func build_query(query_type string) string {

	switch query_type {
	case "ownship":
		return "select mySituation.Lng, mySituation.Lat, mySituation.Alt/3.28084, timestamp.GPSClock_value " +
			"from mySituation INNER JOIN timestamp ON mySituation.timestamp_id=timestamp.id"
	case "traffic":
		return "select traffic.Reg, traffic.Tail, traffic.Icao_addr, traffic.TargetType, traffic.Lng, traffic.Lat, " +
			"traffic.Alt/3.28084, timestamp.GPSClock_value FROM traffic " +
			"INNER JOIN timestamp ON traffic.timestamp_id=timestamp.id"
	case "towers":
		return "select Lng, Lat, Alt FROM traffic WHERE Reg = 'N746FD'"
	}
	return "select Lng, Lat, Alt from mySituation"
}

func main() {
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

	if _, err := os.Stat(dataLogFile); os.IsNotExist(err) {
		log.Fatal(fmt.Sprintf("No database exists at '%s', record a replay log first.\n", dataLogFile))
	}
	db, err := sql.Open("sqlite3", dataLogFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ownship_maps := build_traffic_maps(db, "ownship")     //ownship traffic map
	traffic_maps := build_traffic_maps(db, "all_traffic") //all other traffic map
	traffic_maps["ownship"] = ownship_maps["ownship"]     //combine both ownship and other traffic
	writeTimeKML(traffic_maps)                            //Filter based on GPS Time of target
	writeAltKML(traffic_maps)                             //Filter based on Minimum Altitude
}
