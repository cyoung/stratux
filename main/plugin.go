package main

import (
	"time"
)

type StratuxPlugin struct {
	InitFunc     func() bool
	ShutdownFunc func() bool
	Name         string
	Clock        time.Time
	Input        chan string
}
