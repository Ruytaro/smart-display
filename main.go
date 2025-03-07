package main

import (
	"fmt"
	"log"
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

func main() {
	// Create a new display
	display, err := display.NewDisplay("/dev/ttyACM0", 480, 320)
	utils.Check(err)
	display.SetOrientation(LANDSCAPE)
	if err != nil {
		log.Fatalf("Failed to load font: %v", err)
	}
	time.Sleep(time.Second)
	display.WriteText("Hello, World!", 0xFFFF, 0x0000, 0, 0, 16, 0, 0)
	ts := time.Now()
	display.UpdateDisplay()
	last := time.Since(ts)
	fmt.Printf("Time to draw image: %d, %.2f fps\n", last.Milliseconds(), 1000.0/float64(last.Milliseconds()))
}
