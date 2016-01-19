/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New"" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	monotonic.go: Create monotonic clock using time.Timer - necessary because of real time clock changes on RPi.
*/

package main

import (
	humanize "github.com/dustin/go-humanize"
	"time"
)

// Timer (since start).

type monotonic struct {
	Seconds uint64
	Time    time.Time
	ticker  *time.Ticker
}

func (m *monotonic) Watcher() {
	for {
		<-m.ticker.C
		m.Seconds++
		m.Time = m.Time.Add(1 * time.Second)
	}
}

func (m *monotonic) Since(t time.Time) time.Duration {
	return m.Time.Sub(t)
}

func (m *monotonic) HumanizeTime(t time.Time) string {
	return humanize.RelTime(m.Time, t, "ago", "from now")
}

func NewMonotonic() *monotonic {
	t := &monotonic{Seconds: 0, Time: time.Time{}, ticker: time.NewTicker(1 * time.Second)}
	go t.Watcher()
	return t
}
