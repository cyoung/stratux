package main

import (
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

const invalidCpuTemp = float32(-99.0)

type CpuTempUpdateFunc func(cpuTemp float32)

/* cpuTempMonitor() reads the RPi board temperature every second and
calls a callback.  This is broken out into its own function (run as
its own goroutine) because the RPi temperature monitor code is buggy,
and often times reading this file hangs quite some time.  */

func cpuTempMonitor(updater CpuTempUpdateFunc) {
	timer := time.NewTicker(1 * time.Second)
	for {
		// Update CPUTemp.
		temp, err := ioutil.ReadFile("/sys/class/thermal/thermal_zone0/temp")
		tempStr := strings.Trim(string(temp), "\n")
		t := invalidCpuTemp
		if err == nil {
			tInt, err := strconv.Atoi(tempStr)
			if err == nil {
				if tInt > 1000 {
					t = float32(tInt) / float32(1000.0)
				} else {
					t = float32(tInt) // case where Temp is returned as simple integer
				}
			}
		}
		if t >= invalidCpuTemp { // Only update if valid value was obtained.
			updater(t)
		}
		<-timer.C
	}
}

// Check if CPU temperature is valid. Assume <= 0 is invalid.
func isCPUTempValid(cpuTemp float32) bool {
	return cpuTemp > 0
}
