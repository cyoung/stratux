package main

import (
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

func NewMonotonic() *monotonic {
	t := &monotonic{Seconds: 0, Time: time.Time{}, ticker: time.NewTicker(1 * time.Second)}
	go t.Watcher()
	return t
}
