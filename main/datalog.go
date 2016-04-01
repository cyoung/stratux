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
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	LOG_TIMESTAMP_RESOLUTION = 250 * time.Millisecond
)

type StratuxTimestamp struct {
	id                   int64
	Time_type_preference int // 0 = stratuxClock, 1 = gpsClock, 2 = gpsClock extrapolated via stratuxClock.
	StratuxClock_value   time.Time
	GPSClock_value       time.Time // The value of this is either from the GPS or extrapolated from the GPS via stratuxClock if pref is 1 or 2. It is time.Time{} if 0.
	PreferredTime_value  time.Time
}

var dataLogTimestamps map[int64]StratuxTimestamp
var dataLogCurTimestamp int64 // Current timestamp bucket. This is an index on dataLogTimestamps which is not necessarily the db id.

/*
	checkTimestamp().
		Verify that our current timestamp is within the LOG_TIMESTAMP_RESOLUTION bucket.
		 Returns false if the timestamp was changed, true if it is still valid.
		 This is where GPS timestamps are extrapolated, if the GPS data is currently valid.
*/

func checkTimestamp() bool {
	if stratuxClock.Since(dataLogTimestamps[dataLogCurTimestamp].StratuxClock_value) >= LOG_TIMESTAMP_RESOLUTION {
		//FIXME: mutex.
		var ts StratuxTimestamp
		ts.id = 0
		ts.Time_type_preference = 0 // stratuxClock.
		ts.StratuxClock_value = stratuxClock.Time
		ts.GPSClock_value = time.Time{}
		ts.PreferredTime_value = stratuxClock.Time

		// Extrapolate from GPS timestamp, if possible.
		if isGPSClockValid() && dataLogCurTimestamp > 0 {
			// Was the last timestamp either extrapolated or GPS time?
			last_ts := dataLogTimestamps[dataLogCurTimestamp]
			if last_ts.Time_type_preference == 1 || last_ts.Time_type_preference == 2 {
				// Extrapolate via stratuxClock.
				timeSinceLastTS := ts.StratuxClock_value.Sub(last_ts.StratuxClock_value) // stratuxClock ticks since last timestamp.
				extrapolatedGPSTimestamp := last_ts.PreferredTime_value.Add(timeSinceLastTS)

				// Re-set the preferred timestamp type to '2' (extrapolated time).
				ts.Time_type_preference = 2
				ts.PreferredTime_value = extrapolatedGPSTimestamp
				ts.GPSClock_value = extrapolatedGPSTimestamp
			}
		}

		dataLogCurTimestamp++
		dataLogTimestamps[dataLogCurTimestamp] = ts

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

func structCanBeMarshalled(v reflect.Value) bool {
	m := v.MethodByName("String")
	if m.IsValid() && !m.IsNil() {
		return true
	}
	return false
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

	// Add the timestamp_id field to link up with the timestamp table.
	if tbl != "timestamp" {
		fields = append(fields, "timestamp_id INTEGER")
	}

	tblCreate := fmt.Sprintf("CREATE TABLE %s (id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, %s)", tbl, strings.Join(fields, ", "))
	_, err := db.Exec(tblCreate)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
	}
}

/*
	bulkInsert().
		Reads insertBatch and insertBatchIfs. This is called after a group of insertData() calls.
*/

func bulkInsert(tbl string, db *sql.DB) (res sql.Result, err error) {
	if _, ok := insertString[tbl]; !ok {
		return nil, errors.New("no insert statement")
	}

	batchVals := insertBatchIfs[tbl]
	for len(batchVals) > 0 {
		i := int(0) // Maximum of 10 rows per INSERT statement.
		stmt := ""
		vals := make([]interface{}, 0)
		for len(batchVals) > 0 && i < 10 {
			if len(stmt) == 0 { // The first set will be covered by insertString.
				stmt = insertString[tbl]
			} else {
				stmt += ", (" + strings.Join(strings.Split(strings.Repeat("?", len(batchVals[0])), ""), ",") + ")"
			}
			vals = append(vals, batchVals[0]...)
			batchVals = batchVals[1:]
			i++
		}
		res, err = db.Exec(stmt, vals...)
		if err != nil {
			return
		}
	}

	// Clear the buffers.
	delete(insertString, tbl)
	delete(insertBatchIfs, tbl)

	return
}

/*
	insertData().
		Inserts an arbitrary struct into an SQLite table.
		 Inserts the timestamp first, if its 'id' is 0.

*/

// Cached 'VALUES' statements. Indexed by table name.
var insertString map[string]string // INSERT INTO tbl (col1, col2, ...) VALUES(?, ?, ...). Only for one value.
var insertBatchIfs map[string][][]interface{}

func insertData(i interface{}, tbl string, db *sql.DB, ts_num int64) int64 {
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

	// Add the timestamp_id field to link up with the timestamp table.
	if tbl != "timestamp" {
		keys = append(keys, "timestamp_id")
		if dataLogTimestamps[ts_num].id == 0 {
			//FIXME: This is somewhat convoluted. When insertData() is called for a ts_num that corresponds to a timestamp with no database id,
			// then it inserts that timestamp via the same interface and the id is updated in the structure via the below lines
			// (dataLogTimestamps[ts_num].id = id).
			insertData(dataLogTimestamps[ts_num], "timestamp", db, ts_num) // Updates dataLogTimestamps[ts_num].id.
		}
		values = append(values, strconv.FormatInt(dataLogTimestamps[ts_num].id, 10))
	}

	if _, ok := insertString[tbl]; !ok {
		// Prepare the statement.
		tblInsert := fmt.Sprintf("INSERT INTO %s (%s) VALUES(%s)", tbl, strings.Join(keys, ","),
			strings.Join(strings.Split(strings.Repeat("?", len(keys)), ""), ","))
		insertString[tbl] = tblInsert
	}

	// Make the values slice into a slice of interface{}.
	ifs := make([]interface{}, len(values))
	for i := 0; i < len(values); i++ {
		ifs[i] = values[i]
	}

	insertBatchIfs[tbl] = append(insertBatchIfs[tbl], ifs)

	if tbl == "timestamp" { // Immediate insert always for "timestamp" table.
		res, err := bulkInsert("timestamp", db) // Bulk insert of 1, always.
		if err == nil {
			id, err := res.LastInsertId()
			if err == nil {
				ts := dataLogTimestamps[ts_num]
				ts.id = id
				dataLogTimestamps[ts_num] = ts
			}
			return id
		}
	}

	return 0
}

type DataLogRow struct {
	tbl    string
	data   interface{}
	ts_num int64
}

var dataLogChan chan DataLogRow
var shutdownDataLog chan bool

var dataLogWriteChan chan DataLogRow

func dataLogWriter(db *sql.DB) {
	dataLogWriteChan = make(chan DataLogRow, 10240)
	// The write queue. As data comes in via dataLogChan, it is timestamped and stored.
	//  When writeTicker comes up, the queue is emptied.
	writeTicker := time.NewTicker(10 * time.Second)
	rowsQueuedForWrite := make([]DataLogRow, 0)
	for {
		select {
		case r := <-dataLogWriteChan:
			// Accept timestamped row.
			rowsQueuedForWrite = append(rowsQueuedForWrite, r)
		case <-writeTicker.C:
			// Write the buffered rows. This will block while it is writing.
			// Save the names of the tables affected so that we can run bulkInsert() on after the insertData() calls.
			tblsAffected := make(map[string]bool)
			// Start transaction.
			tx, err := db.Begin()
			if err != nil {
				log.Printf("db.Begin() error: %s\n", err.Error())
				break // from select {}
			}
			for _, r := range rowsQueuedForWrite {
				tblsAffected[r.tbl] = true
				insertData(r.data, r.tbl, db, r.ts_num)
			}
			// Do the bulk inserts.
			for tbl, _ := range tblsAffected {
				bulkInsert(tbl, db)
			}
			// Close the transaction.
			tx.Commit()
			rowsQueuedForWrite = make([]DataLogRow, 0) // Zero the queue.
		}
	}
}

func dataLog() {
	dataLogChan = make(chan DataLogRow, 10240)
	shutdownDataLog = make(chan bool)
	dataLogTimestamps = make(map[int64]StratuxTimestamp, 0)

	// Check if we need to create a new database.
	createDatabase := false

	if _, err := os.Stat(dataLogFile); os.IsNotExist(err) {
		createDatabase = true
		log.Printf("creating new database '%s'.\n", dataLogFile)
	}

	db, err := sql.Open("sqlite3", dataLogFile)
	if err != nil {
		log.Printf("sql.Open(): %s\n", err.Error())
	}
	defer db.Close()

	go dataLogWriter(db)

	// Do we need to create the database?
	if createDatabase {
		makeTable(StratuxTimestamp{}, "timestamp", db)
		makeTable(mySituation, "mySituation", db)
		makeTable(globalStatus, "status", db)
		makeTable(globalSettings, "settings", db)
		makeTable(TrafficInfo{}, "traffic", db)
		makeTable(msg{}, "messages", db)
		makeTable(Dump1090TermMessage{}, "dump1090_terminal", db)
	}

	for {
		select {
		case r := <-dataLogChan:
			// When data is input, the first step is to timestamp it.
			// Check if our time bucket has expired or has never been entered.
			checkTimestamp()
			// Mark the row with the current timestamp ID, in case it gets entered later.
			r.ts_num = dataLogCurTimestamp
			// Queue it for the scheduled write.
			dataLogWriteChan <- r
		case <-shutdownDataLog: // Received a message on the channel (anything). Graceful shutdown (defer statement).
			return
		}
	}
}

/*
	setDataLogTimeWithGPS().
		Create a timestamp entry using GPS time.
*/

func setDataLogTimeWithGPS(sit SituationData) {
	if isGPSClockValid() {
		//FIXME: mutex.
		var ts StratuxTimestamp
		// Piggyback a GPS time update from this update.
		ts.id = 0
		ts.Time_type_preference = 1 // gpsClock.
		ts.StratuxClock_value = stratuxClock.Time
		ts.GPSClock_value = sit.GPSTime
		ts.PreferredTime_value = sit.GPSTime
		dataLogCurTimestamp++
		dataLogTimestamps[dataLogCurTimestamp] = ts
	}
}

func logSituation() {
	if globalSettings.ReplayLog {
		dataLogChan <- DataLogRow{tbl: "mySituation", data: mySituation}
	}
}

func logStatus() {
	dataLogChan <- DataLogRow{tbl: "status", data: globalStatus}
}

func logSettings() {
	dataLogChan <- DataLogRow{tbl: "settings", data: globalSettings}
}

func logTraffic(ti TrafficInfo) {
	if globalSettings.ReplayLog {
		dataLogChan <- DataLogRow{tbl: "traffic", data: ti}
	}
}

func logMsg(m msg) {
	if globalSettings.ReplayLog {
		dataLogChan <- DataLogRow{tbl: "messages", data: m}
	}
}

func logDump1090TermMessage(m Dump1090TermMessage) {
	dataLogChan <- DataLogRow{tbl: "dump1090_terminal", data: m}
}

func initDataLog() {
	insertString = make(map[string]string)
	insertBatchIfs = make(map[string][][]interface{})
	go dataLog()
}
