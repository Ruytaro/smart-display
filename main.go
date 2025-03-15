package main

import (
	_ "embed"
	"flag"
	"os"
	"os/signal"
	"smart-display/display"
	"smart-display/utils"
	"strings"
	"time"
)

// embed
//
//go:embed resources/font.ttf
var fontData []byte

var (
	port  string
	test  bool
	debug bool
	paths Paths
	rate  uint
)

type Paths []string

func (s *Paths) String() string {
	return strings.Join(*s, ",")
}

func (s *Paths) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func init() {
	flag.StringVar(&port, "tty", "ttyACM0", "tty to use")
	flag.UintVar(&rate, "rate", 5, "sets the update interval")
	flag.BoolVar(&test, "test", false, "Test the display")
	flag.BoolVar(&debug, "d", false, "set debug mode")
	flag.Var(&paths, "path", "list of paths to monitor disk usage")
	flag.Parse()
}

func main() {
	// Create a new display
	wch := make(chan (any))
	display, err := display.NewDisplay(wch, "/dev/"+port, 480, 320, fontData, debug)
	utils.Check(err)
	display.SetBrightness(20)
	if test {
		display.Demo()
		display.Reset()
		<-wch
		os.Exit(0)
	}
	tc := time.NewTicker(time.Second * time.Duration(rate))
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	run := true
	display.Fill(0, 0, 0)
	display.Update()
	for run {
		select {
		case <-ch:
			run = false
		case <-tc.C:
			display.Stats(paths)
		}
	}
	display.Reset()
	<-wch
}
