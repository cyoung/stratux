/*
	Copyright (c) 2023 Adrian Batzill
	Distributable under the terms of The "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	logging.go: Initialize go logging, watch log file size and rotate, delete old logs

*/

package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ricochet2200/go-disk-usage/du"
)

const debugLogFile   = "stratux.log"

var debugLogf string    // Set according to OS config.
var logFileHandle *os.File


func getStratuxLogFiles() []string {
	entries, err := os.ReadDir(logDir)
	stratuxLogs := make([]string, 0)
	if err != nil {
		return stratuxLogs
	}
	

	for _, e := range entries {
		if strings.HasPrefix(e.Name(), debugLogFile + ".") {
			stratuxLogs = append(stratuxLogs, filepath.Join(logDirf, e.Name()))
		}
	}
	sort.Strings(stratuxLogs)
	return stratuxLogs
}

func rotateLogs() {
	stratuxLogs := getStratuxLogFiles()

	// rename suffix, remove if > 9
	for i := len(stratuxLogs)-1; i >= 0; i-- {
		parts := strings.Split(stratuxLogs[i], ".")
		logNum, err := strconv.Atoi(parts[len(parts) - 1])
		if err != nil {
			continue
		}
		
		newPath := filepath.Join(logDirf, debugLogFile + "." + strconv.Itoa(logNum + 1))

		if logNum == 9 {
			os.Remove(stratuxLogs[i])
		} else {
			os.Rename(stratuxLogs[i], newPath)
		}
	}

	// Now rename current log file and re-open
	os.Rename(debugLogf, debugLogf + ".1")
	openLogFile()
}

func deleteOldestLog() int64 {
	logs := getStratuxLogFiles()
	if len(logs) == 0 {
		return 0
	}
	oldest := logs[len(logs) - 1]
	stat, err := os.Stat(oldest)
	if err != nil {
		return 0
	}
	os.Remove(oldest)
	return stat.Size()
}



func logFileWatcher() {
	for {
		logSize, _ := os.Stat(debugLogf)
		if logSize.Size() > 10 * 1024 * 1024 { // 10mb limit
			rotateLogs()
		}

		usage := du.NewDiskUsage(logDirf)
		freeBytes := int64(usage.Free())
		for freeBytes < 50 * 1024 * 1024 { // leave 50mb free
			deleted := deleteOldestLog()
			if deleted == 0 {
				break
			}
			freeBytes += deleted
		}

		time.Sleep(30 * time.Second)
	}
}

func openLogFile() {
	oldFp := logFileHandle
	debugLogf = filepath.Join(logDirf, debugLogFile)
	fp, err := os.OpenFile(debugLogf, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		addSingleSystemErrorf(debugLogf, "Failed to open '%s': %s", debugLogf, err.Error())
	} else {
		// Keep the logfile handle for later use
		logFileHandle = fp
		mfp := io.MultiWriter(fp, os.Stdout)
		log.SetOutput(mfp)

		// Make sure crash dumps are written to the log as well
		syscall.Dup3(int(fp.Fd()), 2, 0)
	}
	if oldFp != nil {
		oldFp.Close()
	}
}

func initLogging() {
	openLogFile()
	go logFileWatcher();
}