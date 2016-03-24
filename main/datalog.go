/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New"" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	datalog.go: Log stratux data as it is received. Bucket data into timestamp time slots.

*/

package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	LOG_TIMESTAMP_RESOLUTION = 50 * time.Millisecond
)

type StratuxTimestamp struct {
	id                   int64
	time_type_preference int // 0 = stratuxClock, 1 = gpsClock, 2 = gpsClock extrapolated via stratuxClock.
	stratuxClock_value   time.Time
	gpsClock_value       time.Time
	preferredTime_value  time.Time
}

var dataLogTimestamp StratuxTimestamp // Current timestamp bucket.

/*
	checkTimestamp().
		Verify that our current timestamp is within the LOG_TIMESTAMP_RESOLUTION bucket.
		 Returns false if the timestamp was changed, true if it is still valid.
*/

//FIXME: time -> stratuxClock
func checkTimestamp() bool {
	if time.Since(dataLogTimestamp.stratuxClock_value) >= LOG_TIMESTAMP_RESOLUTION {
		//FIXME: mutex.
		dataLogTimestamp.id = 0
		dataLogTimestamp.stratuxClock_value = time.Now()
		dataLogTimestamp.time_type_preference = 0

		return false
	}
	return true
}

type SQLiteMarshal struct {
	FieldType string
	Marshal   func(v reflect.Value) string
}

func boolMarshal(v reflect.Value) string {
	b := v.Bool()
	if b {
		return "1"
	}
	return "0"
}

func structCanBeMarshalled(v reflect.Value) bool {
	m := v.MethodByName("String")
	if m.IsValid() && !m.IsNil() {
		return true
	}
	return false
}

func intMarshal(v reflect.Value) string {
	return strconv.FormatInt(v.Int(), 10)
}

func uintMarshal(v reflect.Value) string {
	return strconv.FormatUint(v.Uint(), 10)
}

func floatMarshal(v reflect.Value) string {
	return strconv.FormatFloat(v.Float(), 'f', 10, 64)
}

func stringMarshal(v reflect.Value) string {
	return v.String()
}

func notsupportedMarshal(v reflect.Value) string {
	return ""
}

func structMarshal(v reflect.Value) string {
	if structCanBeMarshalled(v) {
		m := v.MethodByName("String")
		in := make([]reflect.Value, 0)
		ret := m.Call(in)
		if len(ret) > 0 {
			return ret[0].String()
		}
	}
	return ""
}

var sqliteMarshalFunctions = map[string]SQLiteMarshal{
	"bool":         {FieldType: "INTEGER", Marshal: boolMarshal},
	"int":          {FieldType: "INTEGER", Marshal: intMarshal},
	"uint":         {FieldType: "INTEGER", Marshal: uintMarshal},
	"float":        {FieldType: "REAL", Marshal: floatMarshal},
	"string":       {FieldType: "TEXT", Marshal: stringMarshal},
	"struct":       {FieldType: "STRING", Marshal: structMarshal},
	"notsupported": {FieldType: "notsupported", Marshal: notsupportedMarshal},
}

var sqlTypeMap = map[reflect.Kind]string{
	reflect.Bool:          "bool",
	reflect.Int:           "int",
	reflect.Int8:          "int",
	reflect.Int16:         "int",
	reflect.Int32:         "int",
	reflect.Int64:         "int",
	reflect.Uint:          "uint",
	reflect.Uint8:         "uint",
	reflect.Uint16:        "uint",
	reflect.Uint32:        "uint",
	reflect.Uint64:        "uint",
	reflect.Uintptr:       "notsupported",
	reflect.Float32:       "float",
	reflect.Float64:       "float",
	reflect.Complex64:     "notsupported",
	reflect.Complex128:    "notsupported",
	reflect.Array:         "notsupported",
	reflect.Chan:          "notsupported",
	reflect.Func:          "notsupported",
	reflect.Interface:     "notsupported",
	reflect.Map:           "notsupported",
	reflect.Ptr:           "notsupported",
	reflect.Slice:         "notsupported",
	reflect.String:        "string",
	reflect.Struct:        "struct",
	reflect.UnsafePointer: "notsupported",
}

func makeTable(i interface{}, tbl string, db *sql.DB) {
	val := reflect.ValueOf(i)

	fields := make([]string, 0)
	for i := 0; i < val.NumField(); i++ {
		kind := val.Field(i).Kind()
		fieldName := val.Type().Field(i).Name
		sqlTypeAlias := sqlTypeMap[kind]

		// Check that if the field is a struct that it can be marshalled.
		if sqlTypeAlias == "struct" && !structCanBeMarshalled(val.Field(i)) {
			continue
		}
		if sqlTypeAlias == "notsupported" || fieldName == "id" {
			continue
		}
		sqlType := sqliteMarshalFunctions[sqlTypeAlias].FieldType
		s := fieldName + " " + sqlType
		fields = append(fields, s)
	}

	if len(fields) > 0 {
		tblCreate := fmt.Sprintf("CREATE TABLE %s (id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, %s)", tbl, strings.Join(fields, ", "))
		_, err := db.Exec(tblCreate)
		fmt.Printf("%s\n", tblCreate)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err.Error())
		}
	}
}

func insertData(i interface{}, tbl string, db *sql.DB) {
	checkTimestamp()
	val := reflect.ValueOf(i)

	keys := make([]string, 0)
	values := make([]string, 0)
	for i := 0; i < val.NumField(); i++ {
		kind := val.Field(i).Kind()
		fieldName := val.Type().Field(i).Name
		sqlTypeAlias := sqlTypeMap[kind]

		if sqlTypeAlias == "notsupported" || fieldName == "id" {
			continue
		}

		v := sqliteMarshalFunctions[sqlTypeAlias].Marshal(val.Field(i))

		keys = append(keys, fieldName)
		values = append(values, v)
	}

	if len(keys) > 0 {
		tblInsert := fmt.Sprintf("INSERT INTO %s (%s) VALUES(%s)", tbl, strings.Join(keys, ","),
			strings.Join(strings.Split(strings.Repeat("?", len(keys)), ""), ","))

		fmt.Printf("%s\n", tblInsert)
		ifs := make([]interface{}, len(values))
		for i := 0; i < len(values); i++ {
			ifs[i] = values[i]
		}
		_, err := db.Exec(tblInsert, ifs...)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err.Error())
		}
	}
}

type SituationData struct {
	// From GPS.
	LastFixSinceMidnightUTC float32
	Lat                     float32
	Lng                     float32
	Quality                 uint8
	HeightAboveEllipsoid    float32 // GPS height above WGS84 ellipsoid, ft. This is specified by the GDL90 protocol, but most EFBs use MSL altitude instead. HAE is about 70-100 ft below GPS MSL altitude over most of the US.
	GeoidSep                float32 // geoid separation, ft, MSL minus HAE (used in altitude calculation)
	Satellites              uint16  // satellites used in solution
	SatellitesTracked       uint16  // satellites tracked (almanac data received)
	SatellitesSeen          uint16  // satellites seen (signal received)
	Accuracy                float32 // 95% confidence for horizontal position, meters.
	NACp                    uint8   // NACp categories are defined in AC 20-165A
	Alt                     float32 // Feet MSL
	AccuracyVert            float32 // 95% confidence for vertical position, meters
	GPSVertVel              float32 // GPS vertical velocity, feet per second
	LastFixLocalTime        time.Time
	TrueCourse              uint16
	GroundSpeed             uint16
	LastGroundTrackTime     time.Time
	LastGPSTimeTime         time.Time
	LastNMEAMessage         time.Time // time valid NMEA message last seen

	// From BMP180 pressure sensor.
	Temp              float64
	Pressure_alt      float64
	LastTempPressTime time.Time

	// From MPU6050 accel/gyro.
	Pitch            float64
	Roll             float64
	Gyro_heading     float64
	LastAttitudeTime time.Time
}

func main() {
	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		fmt.Printf("sql.Open(): %s\n", err.Error())
	}
	defer db.Close()

	e := SituationData{}
	//makeTable(e, "situation", db)
	insertData(e, "situation", db)
}
