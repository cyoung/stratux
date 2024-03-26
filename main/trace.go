// Copyright (c) 2023 Adrian Batzill
// Distributable under the terms of The "BSD New" License
// that can be found in the LICENSE file, herein included
// as part of this header.
// trace.go: record all incoming messages (GPS, dump1090, ...) for future replay

package main

import (
	"compress/gzip"
	"encoding/csv"
	"log"
	"os"
	"sync"
	"time"

	"github.com/ricochet2200/go-disk-usage/du"
	"golang.org/x/exp/slices"
)


type TraceLogger struct {
	fileHandle        *os.File
	gzWriter          *gzip.Writer
	csvWriter         *csv.Writer
	fileName          string
	hasProperFilename bool
	traceMutex        sync.Mutex
	isReplaying       bool
}

const (
	CONTEXT_AIS = "ais"
	CONTEXT_NMEA = "nmea"
	CONTEXT_APRS = "aprs"
	CONTEXT_OGN_RX = "ogn-rx"
	CONTEXT_DUMP1090 = "dump1090"
	CONTEXT_GODUMP978 = "godump978"
	CONTEXT_LOWPOWERUAT = "lowpower_uat"
)

var TraceLog TraceLogger


// At startup, we usually don't know the precise time to generate a good filename.
// Therefore, once we receive our first valid timestamp, we will rename the file to something more appropriate
func (tracer *TraceLogger) OnTimestamp(ts time.Time) {
	tracer.traceMutex.Lock()
	defer tracer.traceMutex.Unlock()
	if tracer.fileHandle == nil || tracer.hasProperFilename {
		return
	}
	formatted := ts.Format(time.RFC3339)
	fname := "/var/log/stratux/" + formatted + "_trace.txt.gz"
	if formatted != fname {
		err := os.Rename(tracer.fileName, fname)
		if err == nil {
			tracer.fileName = fname
		}
	}
	tracer.hasProperFilename = true
}

// Context for now may be one of
func (tracer *TraceLogger) Record(context string, data []byte) {
	go func() {
		tracer.traceMutex.Lock()
		defer tracer.traceMutex.Unlock()
		if tracer.fileHandle == nil || tracer.isReplaying {
			return
		}
		ts := stratuxClock.Time.Format(time.RFC3339Nano)
		tracer.csvWriter.Write([]string {ts, context, string(data)})
	}()
}

func (tracer *TraceLogger) Flush() {
	tracer.traceMutex.Lock()
	defer tracer.traceMutex.Unlock()
	if tracer.fileHandle != nil {
		tracer.gzWriter.Flush()
		tracer.fileHandle.Sync()
	}
}

func (tracer *TraceLogger) Start() {
	tracer.traceMutex.Lock()
	defer tracer.traceMutex.Unlock()
	ts := time.Now().UTC().Format(time.RFC3339)
	os.MkdirAll("/var/log/stratux", os.ModePerm)
	fname := "/var/log/stratux/" + ts + "_trace.txt.gz"

	fileHandle, err := os.OpenFile(fname, os.O_CREATE | os.O_WRONLY, 0666)
	if err != nil {
		log.Printf("Failed to open trace log file: %s", err.Error())
		return
	}
	tracer.gzWriter = gzip.NewWriter(fileHandle)
	tracer.csvWriter = csv.NewWriter(tracer.gzWriter)
	tracer.fileHandle = fileHandle
	tracer.fileName = fname
	tracer.hasProperFilename = false
}

func (tracer *TraceLogger) Stop() {
	tracer.traceMutex.Lock()
	defer tracer.traceMutex.Unlock()
	tracer.csvWriter.Flush()
	tracer.gzWriter.Close()
	tracer.fileHandle.Close()
	tracer.fileHandle = nil
	tracer.csvWriter = nil
	tracer.gzWriter = nil
}

func (tracer *TraceLogger) IsActive() bool {
	return tracer.fileHandle != nil
}

func (tracer *TraceLogger) IsReplaying() bool {
	return tracer.isReplaying
}

func (tracer *TraceLogger) Replay(fname string, speedMultiplier float64, traceSkip int64, msgtypes []string) {
	fhandle, err := os.Open(fname)
	if err != nil {
		log.Printf("Failed to open trace file %s: %s", fname, err.Error())
		return
	}
	tracer.isReplaying = true
	gzReader, err := gzip.NewReader(fhandle)
	if err != nil {
		log.Printf("Failed to open gzip stream for file %s: %s", fname, err.Error())
		return
	}
	startTs := time.Time{}.Add(time.Duration(traceSkip) * time.Minute)
	csvReader := csv.NewReader(gzReader)
	for {
		fields, err := csvReader.Read()
		if err != nil {
			log.Printf("Trace replay stopped: %s", err)
			break
		}
		if len(fields) != 3 {
			log.Printf("Failed to parse trace line")
			continue
		}
		context := fields[1]
		if len(msgtypes) > 0 && !slices.Contains(msgtypes, context) {
			continue
		}
		ts, err := time.Parse(time.RFC3339Nano, fields[0])
		if ts.Before(startTs) {
			continue
		}
		ts = ts.Add(-time.Duration(traceSkip) * time.Minute)
		

		millis := float64(ts.Sub(time.Time{}).Milliseconds()) / speedMultiplier
		ts = time.Time{}.Add(time.Duration(millis) * time.Millisecond)
		if err != nil {
			log.Printf("Failed to parse trace log timestamp %s", fields[0])
			continue
		}
		injectTraceMessage(context, ts, []byte(fields[2]))
	}
}

func injectTraceMessage(context string, ts time.Time, data []byte) {
	toWait := float64(ts.Sub(stratuxClock.Time).Nanoseconds())
	time.Sleep(time.Duration(toWait) * time.Nanosecond)

	if context == CONTEXT_AIS {
		parseAisMessage(string(data))
	} else if context == CONTEXT_NMEA {
		globalStatus.GPS_connected = true
		processNMEALineLow(string(data), true)
	} else if context == CONTEXT_APRS {
		parseAprsMessage(string(data), true)
	} else if context == CONTEXT_OGN_RX {
		parseOgnMessage(string(data), true)
	} else if context == CONTEXT_DUMP1090 {
		parseDump1090Message(string(data))
	} else if context == CONTEXT_GODUMP978 {
		handleUatMessage(string(data))
	} else if context == CONTEXT_LOWPOWERUAT {
		processRadioMessage(data)
	}
}

func traceLoggerWatchdog() {
	for {
		if TraceLog.isReplaying {
			time.Sleep(1 * time.Second)
			continue
		}

		if TraceLog.IsActive() {
			usage := du.NewDiskUsage("/")
			if usage.Free() < 1024 * 1024 * 50 {
				// less than 50mb free? deactivate
				log.Printf("Space running out - disable trace logging for this run")
				TraceLog.Stop()
				break
			}
		}

		if TraceLog.IsActive() && !globalSettings.TraceLog {
			TraceLog.Stop()
		} else if !TraceLog.IsActive() && globalSettings.TraceLog {
			TraceLog.Start()
		}
		time.Sleep(1 * time.Second)
		TraceLog.Flush()
	}
}


