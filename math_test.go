package main

import (
	"log"
	"testing"
)

func TestFit(t *testing.T) {
	cases := []struct {
		x      float32
		oldmin float32
		oldmax float32
		newmin float32
		newmax float32
		want   float32
	}{
		{
			-50,
			-50,
			50,
			0,
			2,
			0,
		},
		{
			50,
			-50,
			50,
			0,
			2,
			2,
		},
		{
			0,
			-50,
			50,
			0,
			2,
			1,
		},
	}
	for _, c := range cases {
		got := fit(c.x, c.oldmin, c.oldmax, c.newmin, c.newmax)
		if got != c.want {
			log.Fatalf("fit(%v, %v, %v, %v, %v): got %v, want %v", c.x, c.oldmin, c.oldmax, c.newmin, c.newmax, c.want)
		}
	}
}
