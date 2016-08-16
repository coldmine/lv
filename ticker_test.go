package main

import (
	"testing"
	"time"
)

func TestTickerDelta(t *testing.T) {
	tick := time.NewTicker(time.Millisecond)
	start := time.Now()
	for i := 0; i < 1000; i++ {
		<-tick.C
	}
	end := time.Now()

	d := time.Duration(end.Sub(start) - 1000*time.Millisecond)
	if d < 0 {
		d = -d
	}
	if d > time.Millisecond {
		t.Fatal("Delta of time.Ticker is too big")
	}
}
