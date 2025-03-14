package main

import (
	_ "embed"
	"os"
	"os/signal"
	"smart-display/display"
	"smart-display/utils"
	"time"
)

const (
	PORTRAIT          = 0
	LANDSCAPE         = 2
	REVERSE_PORTRAIT  = 1
	REVERSE_LANDSCAPE = 3
)

// embed
//
//go:embed resources/font.ttf
var fontData []byte

func main() {
	// Create a new display
	wch := make(chan (any))
	display, err := display.NewDisplay(wch, "/dev/ttyACM0", 480, 320, fontData)
	//display.SetDebug(true)
	utils.Check(err)
	display.SetBrightness(15)
	tc := time.NewTicker(time.Second * 5)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	run := true
	for run {
		select {
		case <-ch:
			run = false
		case <-tc.C:
			display.Stats()
		}
	}
	display.Reset()
	<-wch
}
