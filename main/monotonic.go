/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New"" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	Modifications (c) 2016 AvSquirrel (https://github.com/AvSquirrel)
	monotonic.go: Create monotonic clock using time.Timer - necessary because of real time clock changes on RPi.
*/

package main

import (
	humanize "github.com/dustin/go-humanize"
	"time"
)

// Timer (since start).

type monotonic struct {
	Milliseconds uint64
	Time         time.Time
	ticker       *time.Ticker
}

func (m *monotonic) Watcher() {
	for {
		<-m.ticker.C
		m.Milliseconds += 10
		m.Time = m.Time.Add(10 * time.Millisecond)
	}
}

func (m *monotonic) Since(t time.Time) time.Duration {
	return m.Time.Sub(t)
}

func (m *monotonic) HumanizeTime(t time.Time) string {
	return humanize.RelTime(t, m.Time, "ago", "from now")
}

func (m *monotonic) Unix() int64 {
	return int64(m.Since(time.Time{}).Seconds())
}

func NewMonotonic() *monotonic {
	t := &monotonic{Milliseconds: 0, Time: time.Time{}, ticker: time.NewTicker(10 * time.Millisecond)}
	go t.Watcher()
	return t
}
