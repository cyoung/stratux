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
	StartupID            int64
}

// 'startup' table creates a new entry each time the daemon is started. This keeps track of sequential starts, even if the
//  timestamp is ambiguous (units with no GPS). This struct is just a placeholder for an empty table (other than primary key).
type StratuxStartup struct {
	id   int64
	Fill string
}

var dataLogStarted bool
var dataLogReadyToWrite bool

var stratuxStartupID int64
var dataLogTimestamps []StratuxTimestamp
var dataLogCurTimestamp int64 // Current timestamp bucket. This is an index on dataLogTimestamps which is not necessarily the db id.

/*
	checkTimestamp().
		Verify that our current timestamp is within the LOG_TIMESTAMP_RESOLUTION bucket.
		 Returns false if the timestamp was changed, true if it is still valid.
		 This is where GPS timestamps are extrapolated, if the GPS data is currently valid.
*/

func checkTimestamp() bool {
	thisCurTimestamp := dataLogCurTimestamp
	if stratuxClock.Since(dataLogTimestamps[thisCurTimestamp].StratuxClock_value) >= LOG_TIMESTAMP_RESOLUTION {
		var ts StratuxTimestamp
		ts.id = 0
		ts.Time_type_preference = 0 // stratuxClock.
		ts.StratuxClock_value = stratuxClock.Time
		ts.GPSClock_value = time.Time{}
		ts.PreferredTime_value = stratuxClock.Time

		// Extrapolate from GPS timestamp, if possible.
		if isGPSClockValid() && thisCurTimestamp > 0 {
			// Was the last timestamp either extrapolated or GPS time?
			last_ts := dataLogTimestamps[thisCurTimestamp]
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

		dataLogTimestamps = append(dataLogTimestamps, ts)
		dataLogCurTimestamp = int64(len(dataLogTimestamps) - 1)
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
	if tbl != "timestamp" && tbl != "startup" {
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
	numColsPerRow := len(batchVals[0])
	maxRowBatch := int(999 / numColsPerRow) // SQLITE_MAX_VARIABLE_NUMBER = 999.
	//	log.Printf("table %s. %d cols per row. max batch %d\n", tbl, numColsPerRow, maxRowBatch)
	for len(batchVals) > 0 {
		//     timeInit := time.Now()
		i := int(0) // Variable number of rows per INSERT statement.

		stmt := ""
		vals := make([]interface{}, 0)
		querySize := uint64(0)                                            // Size of the query in bytes.
		for len(batchVals) > 0 && i < maxRowBatch && querySize < 750000 { // Maximum of 1,000,000 bytes per query.
			if len(stmt) == 0 { // The first set will be covered by insertString.
				stmt = insertString[tbl]
				querySize += uint64(len(insertString[tbl]))
			} else {
				addStr := ", (" + strings.Join(strings.Split(strings.Repeat("?", len(batchVals[0])), ""), ",") + ")"
				stmt += addStr
				querySize += uint64(len(addStr))
			}
			for _, val := range batchVals[0] {
				querySize += uint64(len(val.(string)))
			}
			vals = append(vals, batchVals[0]...)
			batchVals = batchVals[1:]
			i++
		}
		//		log.Printf("inserting %d rows to %s. querySize=%d\n", i, tbl, querySize)
		res, err = db.Exec(stmt, vals...)
		//      timeBatch := time.Since(timeInit)                                                                                                                     // debug
		//      log.Printf("SQLite: bulkInserted %d rows to %s. Took %f msec to build and insert query. querySize=%d\n", i, tbl, 1000*timeBatch.Seconds(), querySize) // debug
		if err != nil {
			log.Printf("sqlite INSERT error: '%s'\n", err.Error())
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
	if tbl != "timestamp" && tbl != "startup" {
		keys = append(keys, "timestamp_id")
		if dataLogTimestamps[ts_num].id == 0 {
			//FIXME: This is somewhat convoluted. When insertData() is called for a ts_num that corresponds to a timestamp with no database id,
			// then it inserts that timestamp via the same interface and the id is updated in the structure via the below lines
			// (dataLogTimestamps[ts_num].id = id).
			dataLogTimestamps[ts_num].StartupID = stratuxStartupID
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

	if tbl == "timestamp" || tbl == "startup" { // Immediate insert always for "timestamp" and "startup" table.
		res, err := bulkInsert(tbl, db) // Bulk insert of 1, always.
		if err == nil {
			id, err := res.LastInsertId()
			if err == nil && tbl == "timestamp" { // Special handling for timestamps. Update the timestamp ID.
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
var shutdownDataLogWriter chan bool

var dataLogWriteChan chan DataLogRow

func dataLogWriter(db *sql.DB) {
	dataLogWriteChan = make(chan DataLogRow, 10240)
	shutdownDataLogWriter = make(chan bool)
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
			//			for i := 0; i < 1000; i++ {
			//				logSituation()
			//			}
			timeStart := stratuxClock.Time
			nRows := len(rowsQueuedForWrite)
			if globalSettings.DEBUG {
				log.Printf("Writing %d rows\n", nRows)
			}
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
			timeElapsed := stratuxClock.Since(timeStart)
			if globalSettings.DEBUG {
				rowsPerSecond := float64(nRows) / float64(timeElapsed.Seconds())
				log.Printf("Writing finished. %d rows in %.2f seconds (%.1f rows per second).\n", nRows, float64(timeElapsed.Seconds()), rowsPerSecond)
			}
			if timeElapsed.Seconds() > 10.0 {
				log.Printf("WARNING! SQLite logging is behind. Last write took %.1f seconds.\n", float64(timeElapsed.Seconds()))
				dataLogCriticalErr := fmt.Errorf("WARNING! SQLite logging is behind. Last write took %.1f seconds.\n", float64(timeElapsed.Seconds()))
				addSystemError(dataLogCriticalErr)
			}
		case <-shutdownDataLogWriter: // Received a message on the channel to initiate a graceful shutdown, and to command dataLog() to shut down
			log.Printf("datalog.go: dataLogWriter() received shutdown message with rowsQueuedForWrite = %d\n", len(rowsQueuedForWrite))
			shutdownDataLog <- true
			return
		}
	}
	log.Printf("datalog.go: dataLogWriter() shutting down\n")
}

func dataLog() {
	dataLogStarted = true
	log.Printf("datalog.go: dataLog() started\n")
	dataLogChan = make(chan DataLogRow, 10240)
	shutdownDataLog = make(chan bool)
	dataLogTimestamps = make([]StratuxTimestamp, 0)
	var ts StratuxTimestamp
	ts.id = 0
	ts.Time_type_preference = 0 // stratuxClock.
	ts.StratuxClock_value = stratuxClock.Time
	ts.GPSClock_value = time.Time{}
	ts.PreferredTime_value = stratuxClock.Time
	dataLogTimestamps = append(dataLogTimestamps, ts)
	dataLogCurTimestamp = 0

	// Check if we need to create a new database.
	createDatabase := false

	if _, err := os.Stat(dataLogFilef); os.IsNotExist(err) {
		createDatabase = true
		log.Printf("creating new database '%s'.\n", dataLogFilef)
	}

	db, err := sql.Open("sqlite3", dataLogFilef)
	if err != nil {
		log.Printf("sql.Open(): %s\n", err.Error())
	}

	defer func() {
		db.Close()
		dataLogStarted = false
		//close(dataLogChan)
		log.Printf("datalog.go: dataLog() has closed DB in %s\n", dataLogFilef)
	}()

	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		log.Printf("db.Exec('PRAGMA journal_mode=WAL') err: %s\n", err.Error())
	}
	_, err = db.Exec("PRAGMA synchronous=OFF")
	if err != nil {
		log.Printf("db.Exec('PRAGMA journal_mode=WAL') err: %s\n", err.Error())
	}

	//log.Printf("Starting dataLogWriter\n") // REMOVE -- DEBUG
	go dataLogWriter(db)

	// Do we need to create the database?
	if createDatabase {
		makeTable(StratuxTimestamp{}, "timestamp", db)
		makeTable(mySituation, "mySituation", db)
		makeTable(globalStatus, "status", db)
		makeTable(globalSettings, "settings", db)
		makeTable(TrafficInfo{}, "traffic", db)
		makeTable(msg{}, "messages", db)
		makeTable(esmsg{}, "es_messages", db)
		makeTable(Dump1090TermMessage{}, "dump1090_terminal", db)
		makeTable(gpsPerfStats{}, "gps_attitude", db)
		makeTable(StratuxStartup{}, "startup", db)
	}

	// The first entry to be created is the "startup" entry.
	stratuxStartupID = insertData(StratuxStartup{}, "startup", db, 0)

	dataLogReadyToWrite = true
	//log.Printf("Entering dataLog read loop\n") //REMOVE -- DEBUG
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
		case <-shutdownDataLog: // Received a message on the channel to complete a graceful shutdown (see the 'defer func()...' statement above).
			log.Printf("datalog.go: dataLog() received shutdown message\n")
			return
		}
	}
	log.Printf("datalog.go: dataLog() shutting down\n")
	close(shutdownDataLog)
}

/*
	setDataLogTimeWithGPS().
		Create a timestamp entry using GPS time.
*/

func setDataLogTimeWithGPS(sit SituationData) {
	if isGPSClockValid() {
		var ts StratuxTimestamp
		// Piggyback a GPS time update from this update.
		ts.id = 0
		ts.Time_type_preference = 1 // gpsClock.
		ts.StratuxClock_value = stratuxClock.Time
		ts.GPSClock_value = sit.GPSTime
		ts.PreferredTime_value = sit.GPSTime

		dataLogTimestamps = append(dataLogTimestamps, ts)
		dataLogCurTimestamp = int64(len(dataLogTimestamps) - 1)
	}
}

/*
	logSituation(), logStatus(), ... pass messages from other functions to the logging
		engine. These are only read into `dataLogChan` if the Replay Log is toggled on,
		and if the log system is ready to accept writes.
*/

func isDataLogReady() bool {
	return dataLogReadyToWrite
}

func logSituation() {
	if globalSettings.ReplayLog && isDataLogReady() {
		dataLogChan <- DataLogRow{tbl: "mySituation", data: mySituation}
	}
}

func logStatus() {
	if globalSettings.ReplayLog && isDataLogReady() {
		dataLogChan <- DataLogRow{tbl: "status", data: globalStatus}
	}
}

func logSettings() {
	if globalSettings.ReplayLog && isDataLogReady() {
		dataLogChan <- DataLogRow{tbl: "settings", data: globalSettings}
	}
}

func logTraffic(ti TrafficInfo) {
	if globalSettings.ReplayLog && isDataLogReady() {
		dataLogChan <- DataLogRow{tbl: "traffic", data: ti}
	}
}

func logMsg(m msg) {
	if globalSettings.ReplayLog && isDataLogReady() {
		dataLogChan <- DataLogRow{tbl: "messages", data: m}
	}
}

func logESMsg(m esmsg) {
	if globalSettings.ReplayLog && isDataLogReady() {
		dataLogChan <- DataLogRow{tbl: "es_messages", data: m}
	}
}

func logGPSAttitude(gpsPerf gpsPerfStats) {
	if globalSettings.ReplayLog && isDataLogReady() {
		dataLogChan <- DataLogRow{tbl: "gps_attitude", data: gpsPerf}
	}
}

func logDump1090TermMessage(m Dump1090TermMessage) {
	if globalSettings.DEBUG && globalSettings.ReplayLog && isDataLogReady() {
		dataLogChan <- DataLogRow{tbl: "dump1090_terminal", data: m}
	}
}

func initDataLog() {
	//log.Printf("dataLogStarted = %t. dataLogReadyToWrite = %t\n", dataLogStarted, dataLogReadyToWrite) //REMOVE -- DEBUG
	insertString = make(map[string]string)
	insertBatchIfs = make(map[string][][]interface{})
	go dataLogWatchdog()

	//log.Printf("datalog.go: initDataLog() complete.\n") //REMOVE -- DEBUG
}

/*
	dataLogWatchdog(): Watchdog function to control startup / shutdown of data logging subsystem.
		Called by initDataLog as a goroutine. It iterates once per second to determine if
		globalSettings.ReplayLog has toggled. If logging was switched from off to on, it starts
		datalog() as a goroutine. If the log is running and we want it to stop, it calls
		closeDataLog() to turn off the input channels, close the log, and tear down the dataLog
		and dataLogWriter goroutines.
*/

func dataLogWatchdog() {
	for {
		if !dataLogStarted && globalSettings.ReplayLog { // case 1: sqlite logging isn't running, and we want to start it
			log.Printf("datalog.go: Watchdog wants to START logging.\n")
			go dataLog()
		} else if dataLogStarted && !globalSettings.ReplayLog { // case 2:  sqlite logging is running, and we want to shut it down
			log.Printf("datalog.go: Watchdog wants to STOP logging.\n")
			closeDataLog()
		}
		//log.Printf("Watchdog iterated.\n") //REMOVE -- DEBUG
		time.Sleep(1 * time.Second)
		//log.Printf("Watchdog sleep over.\n") //REMOVE -- DEBUG
	}
}

/*
	closeDataLog(): Handler for graceful shutdown of data logging goroutines. It is called by
		by dataLogWatchdog(), gracefulShutdown(), and by any other function (disk space monitor?)
		that needs to be able to shut down sqlite logging without corrupting data or blocking
		execution.

		This function turns off log message reads into the dataLogChan receiver, and sends a
		message to a quit channel ('shutdownDataLogWriter`) in dataLogWriter(). dataLogWriter()
		then sends a message to a quit channel to 'shutdownDataLog` in dataLog() to close *that*
		goroutine. That function sets dataLogStarted=false once the logfile is closed. By waiting
		for that signal, closeDataLog() won't exit until the log is safely written. This prevents
		data loss on shutdown.
*/

func closeDataLog() {
	//log.Printf("closeDataLog(): dataLogStarted = %t\n", dataLogStarted) //REMOVE -- DEBUG
	dataLogReadyToWrite = false // prevent any new messages from being sent down the channels
	log.Printf("datalog.go: Starting data log shutdown\n")
	shutdownDataLogWriter <- true      //
	defer close(shutdownDataLogWriter) // ... and close the channel so subsequent accidental writes don't stall execution
	log.Printf("datalog.go: Waiting for shutdown signal from dataLog()")
	for dataLogStarted {
		//log.Printf("closeDataLog(): dataLogStarted = %t\n", dataLogStarted) //REMOVE -- DEBUG
		time.Sleep(50 * time.Millisecond)
	}
	log.Printf("datalog.go: Data log shutdown successful.\n")
}
