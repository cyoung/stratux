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

func checkTimestamp() bool {
	if stratuxClock.Since(dataLogTimestamp.stratuxClock_value) >= LOG_TIMESTAMP_RESOLUTION {
		//FIXME: mutex.
		dataLogTimestamp.id = 0
		dataLogTimestamp.stratuxClock_value = stratuxClock.Time
		dataLogTimestamp.time_type_preference = 0

		return false
	}
	return true
}

type SQLiteMarshal struct {
	FieldType string
	Marshal   func(i interface{}) string
}

func boolMarshal(i interface{}) string {
	return ""
}

func intMarshal(i interface{}) string {
	return ""
}

func uintMarshal(i interface{}) string {
	return ""
}

func floatMarshal(i interface{}) string {
	return ""
}

func stringMarshal(i interface{}) string {
	return ""
}

func notsupportedMarshal(i interface{}) string {
	return ""
}

var sqliteMarshalFunctions = map[string]SQLiteMarshal{
	"bool":         {FieldType: "INTEGER", Marshal: boolMarshal},
	"int":          {FieldType: "INTEGER", Marshal: intMarshal},
	"uint":         {FieldType: "INTEGER", Marshal: uintMarshal},
	"float":        {FieldType: "REAL", Marshal: floatMarshal},
	"string":       {FieldType: "TEXT", Marshal: stringMarshal},
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
	reflect.Struct:        "notsupported",
	reflect.UnsafePointer: "notsupported",
}

func makeTable(i interface{}, tbl string) {
	val := reflect.ValueOf(i)

	fields := make([]string, 0)
	for i := 0; i < val.NumField(); i++ {
		kind := val.Field(i).Kind()
		fieldName := val.Type().Field(i).Name
		sqlTypeAlias := sqlTypeMap[kind]
		if sqlTypeAlias == "notsupported" || fieldName == "id" {
			continue
		}
		sqlType := sqliteMarshalFunctions[sqlTypeAlias].FieldType
		s := fieldName + " " + sqlType
		fields = append(fields, s)
	}

	if len(fields) > 0 {
		tblCreate := fmt.Sprintf("CREATE TABLE %s (id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, %s)", "test", strings.Join(fields, ", "))
		fmt.Printf("%s\n", tblCreate)
	}
}

func logData(i interface{}, tbl string) {
	val := reflect.ValueOf(i)
	for i := 0; i < val.NumField(); i++ {
		fmt.Printf("%d, %s\n", val.Field(i).Kind(), val.Type().Field(i).Name)
	}
}
