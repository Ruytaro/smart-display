package display

import (
	"fmt"
	"image"
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

type Display struct {
	port   *serial.Port
	width  uint16
	height uint16
	canvas *gg.Context
	send   chan []byte
}

var font *truetype.Font

func NewDisplay(portName string, width, height uint16, fontData []byte) (*Display, error) {
	config := &serial.Config{
		Name:        portName,
		Baud:        1152000,
		ReadTimeout: time.Second * 5,
	}
	var err error
	font, err = truetype.Parse(fontData)
	utils.Check(err)
	port, err := serial.OpenPort(config)
	utils.Check(err)
	canvas := gg.NewContext(int(width), int(height))
	utils.Check(err)
	display := &Display{port, width, height, canvas, make(chan []byte)}
	go display.senderLoop()
	return display, nil

}

func (d *Display) Fill(r, g, b int) {
}

func (d *Display) senderLoop() {
	for {
		select {
		case data := <-d.send:
			d.port.Write(data)
		}
	}
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
}

func (d *Display) WriteText(text string, color, bg uint16, x, y, size, ax, ay float64) {

	d.canvas.SetRGB(utils.RGB565ToComponents(bg))
	d.canvas.Clear()
	d.canvas.SetRGB(utils.RGB565ToComponents(color))

	face := truetype.NewFace(font, &truetype.Options{Size: size})
	d.canvas.SetFontFace(face)
	d.canvas.DrawStringWrapped(text, x, y, ax, ay, 480, 1.2, gg.AlignLeft)
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

func chunked(data []byte, size int) [][]byte {
	var chunks [][]byte
	for i := 0; i <= len(data); i += size {
		end := min(i+size, len(data))
		chunks = append(chunks, data[i:end])
	}
	return chunks
}

func (d *Display) UpdateDisplay() {
	img := d.canvas.Image()
	rgb565le := toRGB565LE(img)
	d.canvas.SavePNG(fmt.Sprintf("out/%d.png", time.Now().Unix()))
	d.sendCommand(DISPLAY_BITMAP, 0, 0, d.width-1, d.height-1)
	for _, chunk := range chunked(rgb565le, int(480*8)) {
		d.send <- chunk
	}

}

func toRGB565LE(img image.Image) []byte {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	data := make([]byte, width*height*2)
	i := 0
	for y := range height {
		for x := range width {
			r, g, b, _ := img.At(x, y).RGBA()
			data[i] = byte(rgb565(r, g, b) & 0xFF)
			data[i+1] = byte(rgb565(r, g, b) >> 8)
			i += 2
		}
	}
	return data
}

func rgb565(r, g, b uint32) uint16 {
	r5 := uint16((r >> 11) & 0x1F) // 5 bits
	g6 := uint16((g >> 10) & 0x3F) // 6 bits
	b5 := uint16((b >> 11) & 0x1F) // 5 bits
	return (r5 << 11) | (g6 << 5) | b5
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
