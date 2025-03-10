package display

import (
	"fmt"
	"image/color"
	"os"
	"smart-display/utils"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/tarm/serial"
)

const (
	PORTRAIT          = 0
	LANDSCAPE         = 2
	REVERSE_PORTRAIT  = 1
	REVERSE_LANDSCAPE = 3
	RESET             = 101 // Resets the display
	CLEAR             = 102 // Clears the display to a white screen
	TO_BLACK          = 103 // Makes the screen go black. NOT TESTED
	SCREEN_OFF        = 108 // Turns the screen off
	SCREEN_ON         = 109 // Turns the screen on
	SET_BRIGHTNESS    = 110 // Sets the screen brightness
	SET_ORIENTATION   = 121 // Sets the screen orientation
	DISPLAY_BITMAP    = 197 // Displays an image on the screen
	SET_MIRROR        = 122 //Mirrors the rendering on the screen
	DISPLAY_PIXELS    = 195 //Displays a pixel on the screen
)

const link = "out/latest.png"
const chunk_size = 8

type Display struct {
	port   *serial.Port
	width  uint16
	height uint16
	canvas *gg.Context
	send   chan []byte
	last   [][]color.Color
}

type Chunk struct {
	cx, cy uint16
}

var font *truetype.Font

func NewDisplay(cl chan (any), portName string, width, height uint16, fontData []byte) (*Display, error) {
	config := &serial.Config{
		Name:        portName,
		Baud:        9600,
		ReadTimeout: time.Second * 5,
	}
	var err error
	font, err = truetype.Parse(fontData)
	utils.Check(err)
	port, err := serial.OpenPort(config)
	utils.Check(err)
	canvas := gg.NewContext(int(width), int(height))
	err = nil
	display := &Display{port, width, height, canvas, make(chan []byte), make([][]color.Color, 0)}
	for range width {
		h := make([]color.Color, 0)
		for range height {
			h = append(h, color.Black)
		}
		display.last = append(display.last, h)
	}
	go display.senderLoop(cl)
	display.SetOrientation(LANDSCAPE)
	display.Fill(32, 0, 0)
	return display, nil

}

func (d *Display) Fill(r, g, b uint8) {
	d.canvas.SetColor(color.RGBA{r, g, b, 255})
	d.canvas.Fill()
	d.canvas.DrawRectangle(0, 0, float64(d.width), float64(d.height))
}

func (d *Display) senderLoop(closer chan (any)) {
	ok := true
	var data []byte
	for ok {
		data, ok = <-d.send
		d.port.Write(data)
	}
	close(closer)
}

func (d *Display) Clear() {
	d.SetOrientation(PORTRAIT)
	d.sendCommand(CLEAR, 0, 0, 0, 0)
}

func (d *Display) SetBrightness(level uint8) {
	abs := utils.MapValue(float64(level), 0, 100, 255, 0)
	d.sendCommand(SET_BRIGHTNESS, uint16(abs), 0, 0, 0)
}

func (d *Display) Reset() {
	d.sendCommand(RESET, 0, 0, 0, 0)
	close(d.send)
}

func (d *Display) WriteText(text string, color color.Color, x, y, size, ax, ay float64, al gg.Align) {
	d.canvas.SetRGB255(utils.ColorToComponents(color))
	face := truetype.NewFace(font, &truetype.Options{Size: size})
	d.canvas.SetFontFace(face)
	d.canvas.DrawStringWrapped(text, x, y, ax, ay, 480, 1.2, al)
}

func (d *Display) sendCommand(cmd byte, x, y, ex, ey uint16) {
	buffer := make([]byte, 6)
	buffer[0] = (uint8)(x >> 2)
	buffer[1] = (uint8)(((x & 3) << 6) + (y >> 4))
	buffer[2] = (uint8)(((y & 15) << 4) + (ex >> 6))
	buffer[3] = (uint8)(((ex & 63) << 2) + (ey >> 8))
	buffer[4] = (uint8)(ey & 255)
	buffer[5] = cmd
	d.send <- buffer
}

func (d *Display) UpdateDisplay() {
	f := utils.GetOutFile()
	d.canvas.SavePNG("out/" + f)
	os.Remove(link)
	os.Symlink(f, link)
	t := time.Now()
	d.chunkedUpdate()
	fmt.Println("Updated in: ", time.Since(t).Milliseconds())
}

func (d *Display) chunkedUpdate() {
	pending := make([]Chunk, 0)
	for cx := range d.width / chunk_size {
		for cy := range d.height / chunk_size {
			chk := Chunk{cx, cy}
			if d.moddedChunk(chk) {
				pending = append(pending, chk)
			}
		}
	}
	fmt.Printf("Updated %d chunks\n", len(pending))
	for _, chunk := range pending {
		d.updateChunk(chunk)
	}
}

func (d *Display) moddedChunk(chunk Chunk) bool {
	for x := range chunk_size {
		for y := range chunk_size {
			px := x + int(chunk.cx*chunk_size)
			py := y + int(chunk.cy*chunk_size)
			if d.canvas.Image().At(px, py) != d.last[px][py] {
				return true
			}
		}
	}
	return false
}

func (d *Display) updateChunk(chunk Chunk) {
	cx := chunk.cx * chunk_size
	cy := chunk.cy * chunk_size
	d.sendCommand(DISPLAY_BITMAP, cx, cy, cx+chunk_size-1, cy+chunk_size-1)
	d.send <- d.getChunk(chunk)
	d.saveChunk(chunk)
}

func (d *Display) saveChunk(c Chunk) {
	for y := range chunk_size {
		py := y + int(c.cy*chunk_size)
		for x := range chunk_size {
			px := x + int(c.cx*chunk_size)
			d.last[px][py] = d.canvas.Image().At(px, py)
		}
	}
}

func (d *Display) getChunk(c Chunk) []byte {
	data := make([]byte, chunk_size*chunk_size*2)
	i := 0
	for y := range chunk_size {
		py := y + int(c.cy*chunk_size)
		for x := range chunk_size {
			px := x + int(c.cx*chunk_size)
			color := d.canvas.Image().At(px, py)
			data[i] = byte(utils.RGBAToRGB565(color.RGBA()) & 0xFF)
			data[i+1] = byte(utils.RGBAToRGB565(color.RGBA()) >> 8)
			i += 2
		}
	}
	return data
}

func (d *Display) SetOrientation(orientation uint8) {
	var x, y, ex, ey uint16 = 0, 0, 0, 0
	byteBuffer := make([]byte, 16)
	byteBuffer[0] = byte(x >> 2)
	byteBuffer[1] = byte(((x & 3) << 6) + (y >> 4))
	byteBuffer[2] = byte(((y & 15) << 4) + (ex >> 6))
	byteBuffer[3] = byte(((ex & 63) << 2) + (ey >> 8))
	byteBuffer[4] = byte(ey & 255)
	byteBuffer[5] = SET_ORIENTATION
	byteBuffer[6] = (orientation + 100)
	byteBuffer[7] = byte(d.width >> 8)
	byteBuffer[8] = byte(d.width & 255)
	byteBuffer[9] = byte(d.height >> 8)
	byteBuffer[10] = byte(d.height & 255)
	d.send <- byteBuffer
}

func (d *Display) Demo() {
	d.Fill(255, 128, 0)
	d.WriteText("Hello, World!", color.Black, 240, 160, 32, 0.5, 0.5, gg.AlignCenter)
	d.UpdateDisplay()
	time.Sleep(2 * time.Second)
	d.Fill(0, 255, 128)
	d.WriteText("Hello, World!", color.Black, 240, 160, 32, 0.5, 0.5, gg.AlignCenter)
	d.UpdateDisplay()
	time.Sleep(2 * time.Second)
	d.Fill(0, 128, 255)
	d.WriteText("Hello, World!", color.Black, 240, 160, 32, 0.5, 0.5, gg.AlignCenter)
	d.UpdateDisplay()
	time.Sleep(2 * time.Second)
	d.canvas.NewSubPath()
	d.canvas.LineTo(0, 0)
	d.canvas.LineTo(480, 0)
	d.canvas.LineTo(480, 80)
	d.canvas.LineTo(0, 80)
	d.canvas.ClosePath()
	d.canvas.Fill()
}
