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
)


type traffic_position struct{

	reg string
	tail string
	targettype int
	coordinates []kml.Coordinate
}

var dataLogFilef string

const (
	dataLogFile    = "/var/log/stratux.sqlite"
)

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
	if err := k.WriteIndent(os.Stdout, "", "  "); err != nil {
		log.Fatal(err)
	}
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

func build_query(target_type, target_id string)(string){
	switch {
	    case "ownship" == target_type:
		return "select Lng, Lat, Alt from mySituation"
	    case "traffic" == target_type:
		return fmt.Sprintf("select Reg, Tail, TargetType, Lng, Lat, Alt FROM traffic WHERE Reg = '%s'", target_id)
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
	flag.Parse()
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
		ownship_coords := readCoordinates(db, build_query("ownship", ""))
		traffic_coords := build_traffic_coords(db, get_traffic_list(db, build_query("traffic_list", "")))
		traffic_coords = append(traffic_coords, traffic_position{reg: "ownship", tail: "ownship", targettype: 99, coordinates: ownship_coords})
		writeKML_multi(traffic_coords)
	}
}
