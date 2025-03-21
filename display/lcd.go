package display

import (
	"fmt"
	"image/color"
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

const chunk_size uint16 = 16

type Display struct {
	port   *serial.Port
	width  uint16
	height uint16
	canvas *gg.Context
	send   chan []byte
	last   [][]color.Color
	debug  bool
}

type Chunk struct {
	cx, cy uint16
}

var font *truetype.Font

func NewDisplay(cl chan (any), portName string, width, height uint16, fontData []byte, debug bool) (*Display, error) {
	var port *serial.Port
	var err error
	if !debug {
		config := &serial.Config{
			Name:        portName,
			Baud:        9600,
			ReadTimeout: time.Second * 5,
		}
		port, err = serial.OpenPort(config)
		utils.Check(err)
	}
	font, err = truetype.Parse(fontData)
	utils.Check(err)
	canvas := gg.NewContext(int(width), int(height))
	err = nil
	display := &Display{port: port, width: width, height: height, canvas: canvas, send: make(chan []byte), last: make([][]color.Color, 0), debug: debug}
	for range width {
		h := make([]color.Color, 0)
		for range height {
			h = append(h, color.Black)
		}
		display.last = append(display.last, h)
	}
	go display.senderLoop(cl, debug)
	display.SetOrientation(LANDSCAPE)
	display.Fill(32, 0, 0)
	return display, nil
}

func (d *Display) Fill(r, g, b uint8) {
	d.canvas.SetColor(color.RGBA{r, g, b, 255})
	d.canvas.DrawRectangle(0, 0, float64(d.width), float64(d.height))
	d.canvas.Fill()
}

func (d *Display) senderLoop(closer chan (any), debug bool) {
	ok := true
	var data []byte
	for ok {
		data, ok = <-d.send
		if debug {
			continue
		}
		d.port.Write(data)
	}
	close(closer)
}

func (d *Display) SetBrightness(level uint8) {
	abs := utils.MapValue(float64(level), 0, 100, 255, 0)
	d.sendCommand(SET_BRIGHTNESS, uint16(abs), 0, 0, 0)
}

func (d *Display) Reset() {
	d.sendCommand(RESET, 0, 0, 0, 0)
	close(d.send)
}

func (d *Display) WriteText(text string, color color.Color, x, y, size, ax, ay, width float64, al gg.Align) {
	d.canvas.SetRGB255(utils.ColorToComponents(color))
	face := truetype.NewFace(font, &truetype.Options{Size: size})
	d.canvas.SetFontFace(face)
	d.canvas.DrawStringWrapped(text, x, y, ax, ay, width, 1.2, al)
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

func (d *Display) Update() {
	if d.debug {
		d.canvas.SavePNG(link)
	} else {
		d.chunkedUpdate()
	}
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
	for _, chunk := range pending {
		d.updateChunk(chunk)
	}
}

func (d *Display) moddedChunk(chunk Chunk) bool {
	for x := range chunk_size {
		for y := range chunk_size {
			px := int(x + chunk.cx*chunk_size)
			py := int(y + chunk.cy*chunk_size)
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
}

func (d *Display) getChunk(c Chunk) []byte {
	data := make([]byte, chunk_size*chunk_size*2)
	i := 0
	for y := range chunk_size {
		py := int(y + c.cy*chunk_size)
		for x := range chunk_size {
			px := int(x + c.cx*chunk_size)
			color := d.canvas.Image().At(px, py)
			data[i] = byte(utils.RGBAToRGB565(color.RGBA()) & 0xFF)
			data[i+1] = byte(utils.RGBAToRGB565(color.RGBA()) >> 8)
			d.last[px][py] = d.canvas.Image().At(px, py)
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

func (d *Display) Stats(paths []string) {
	vmFree, vmUsed, vmTotal := utils.VMStats()
	d.Fill(0, 0, 0)
	red := uint8(utils.MapValue(vmUsed, 0, 100, 0, 0xFF))
	col := utils.RGBAtoColor(red, 0xFF-red, 0, 0xFF)
	d.WriteText(fmt.Sprintf("MemFree: %.0fMB/%.0fMB", vmFree, vmTotal), color.White, 0, 0, 16, 0, 0, float64(d.width), gg.AlignLeft)
	d.WriteText(fmt.Sprintf("Used: %.2f%%", vmUsed), col, 0, 0, 16, 0, 0, float64(d.width), gg.AlignRight)
	for i, path := range paths {
		if i > 4 {
			break
		}
		tf, pf := utils.PathStats(path)
		red := uint8(utils.MapValue(pf, 0, 100, 0, 0xFF))
		color := utils.RGBAtoColor(red, 0xFF-red, 0, 0xFF)
		unit := "MB"
		if tf > 1e4 {
			tf /= 1e3
			unit = "GB"
		}
		h := float64(i+1) * 20
		d.WriteText(fmt.Sprintf("%s  %.0f%% -> %.0f%s", path, pf, tf, unit), color, 0, h, 16, 0, 0, float64(d.width), gg.AlignLeft)
	}
	cpus, err := utils.CPUStats()
	utils.Check(err)
	colwidth := int(d.width) / len(cpus)
	grad := gg.NewLinearGradient(0, 320, 0, 120)
	grad.AddColorStop(0, color.RGBA{0, 255, 0, 255})
	grad.AddColorStop(1, color.RGBA{255, 0, 0, 255})
	//	grad.AddColorStop(1, color.RGBA{0, 0, 0, 255})
	grad.AddColorStop(0.5, color.RGBA{255, 255, 0, 255})
	for i, v := range cpus {
		py := utils.MapValue(v, 0, 100, 320, 120)
		d.canvas.DrawRectangle(0, 0, 480, 320)
		d.canvas.Clip()
		d.canvas.DrawRectangle(float64(i*colwidth)-1, py, float64(colwidth)-2, 320-py)
		d.canvas.SetFillStyle(grad)
		d.canvas.Fill()
		d.WriteText(fmt.Sprintf("%.0f%%", v), color.White, float64(i*colwidth), 120, 16, 0, 0, float64(colwidth), gg.AlignCenter)
	}
	d.Update()
}
func (d *Display) Demo() {
	d.Fill(255, 128, 0)
	d.WriteText("Hello, World!", color.Black, 240, 160, 32, 0.5, 0.5, float64(d.width), gg.AlignCenter)
	d.Update()
	time.Sleep(2 * time.Second)
	d.Fill(0, 255, 128)
	d.WriteText("Hello, World!", color.Black, 240, 160, 32, 0.5, 0.5, float64(d.width), gg.AlignCenter)
	d.Update()
	time.Sleep(2 * time.Second)
	d.Fill(0, 128, 255)
	d.WriteText("Hello, World!", color.Black, 240, 160, 32, 0.5, 0.5, float64(d.width), gg.AlignCenter)
	d.Update()
	time.Sleep(2 * time.Second)
	d.Fill(0, 0, 0)
	d.Update()
	i := 0
	d.canvas.SetColor(color.White)
	for cx := range d.width / chunk_size {
		for cy := range d.height / chunk_size {
			i++
			if i%2 == 0 {
				d.canvas.DrawRectangle(float64(cx*chunk_size), float64(cy*chunk_size), float64(chunk_size), float64(chunk_size))
				d.canvas.Fill()
				d.Update()
			}
		}
		i++
	}
	time.Sleep(time.Second)
}
