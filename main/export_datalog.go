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

type traffic struct {
	reg string
	tail string
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

func writeKML_multi(coordinates_slice [][]kml.Coordinate) {
	k := kml.KML()
	d := kml.Document()
	for _, coordinates := range coordinates_slice {
		d.Add(kml.Placemark(
				kml.LineString(
					kml.AltitudeMode("absolute"),
					kml.Extrude(true),
					kml.Tessellate(true),
					kml.Coordinates(coordinates ...),

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

func get_traffic_list(db *sql.DB, query string) (traffic_list []traffic){
	rows := dataLogReader(db, query)
	defer rows.Close()
	//Coords structure https://play.golang.org/p/RLergI3WyN
	for rows.Next() {
		var Reg, Tail string
		rows.Scan( &Reg, &Tail)
		traffic_list = append(traffic_list, traffic{Reg, Tail})
	}
	return traffic_list

}

func build_traffic_coords(db *sql.DB, traffic_list []traffic) (coordinate_list [][]kml.Coordinate) {
	for _, traffic := range traffic_list {
		rows := dataLogReader(db, build_query("traffic", traffic.reg))
		defer rows.Close()
		coordinates := []kml.Coordinate{}
		for rows.Next() {
			var lat, lng, alt float64
			rows.Scan(&lng, &lat, &alt)
			coordinates = append(coordinates, kml.Coordinate{lng, lat, alt / 3.28084})
		}
		coordinate_list = append(coordinate_list,coordinates)
	}
	return coordinate_list
}

func build_query(target_type, target_id string)(string){
	switch {
	    case "ownship" == target_type:
		return "select Lng, Lat, Alt from mySituation"
	    case "traffic" == target_type:
		return fmt.Sprintf("select Lng, Lat, Alt FROM traffic WHERE Reg = '%s' OR Tail = '%s'", target_id, target_id)
	    case "traffic_list" == target_type:
		return "select DISTINCT  Reg, Tail FROM traffic"
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
		traffic_coords = append(traffic_coords, ownship_coords)
		writeKML_multi(traffic_coords)
	}
}
