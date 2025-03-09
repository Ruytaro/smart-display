package main

import (
	_ "embed"
	"image/color"
	"log"
	"os"
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
	utils.Check(err)
	//display.SetOrientation(LANDSCAPE)
	if err != nil {
		log.Fatalf("Failed to load font: %v", err)
	}
	display.Demo()

	display.Reset()
	<-wch
	os.Exit(0)

	time.Sleep(time.Second)
	display.Fill(128, 0, 255)
	display.UpdateDisplay()
	display.WriteText("Hello, World!", color.White, 0, 0, 16, 0, 0, 0)
	display.UpdateDisplay()
	display.WriteText("Hello, World!", color.White, 240, 160, 32, 0.5, 0.5, 1)
	display.UpdateDisplay()
	time.Sleep(time.Second)
	display.WriteText("Hello, World!", color.RGBA{128, 0, 255, 255}, 240, 160, 32, 0.5, 0.5, 1)
	display.UpdateDisplay()
	time.Sleep(time.Second)
	display.WriteText("Bye bye, World!", color.White, 240, 160, 48, 0.5, 0.5, 2)
	display.UpdateDisplay()
	time.Sleep(time.Second * 3)
	display.Fill(128, 0, 255)
	display.UpdateDisplay()
	time.Sleep(time.Second)
}
