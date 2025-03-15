package utils

import (
	"image/color"
	"log"
	"syscall"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

func MapValue(value, inMin, inMax, outMin, outMax float64) float64 {
	return outMin + (value-inMin)*(outMax-outMin)/(inMax-inMin)
}

func Check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func RGB565ToComponents(color uint16) (r, g, b uint8) {
	r = uint8(color >> 11 & 0x1F)
	g = uint8(color >> 5 & 0x3F)
	b = uint8(color & 0x1F)
	return r, g, b
}

func ColorToComponents(color color.Color) (r, g, b int) {
	cr, cg, cb, _ := color.RGBA()
	r = int(cr >> 8)
	g = int(cg >> 8)
	b = int(cb >> 8)
	return r, g, b
}

func RGBAToRGB565(r, g, b, _ uint32) uint16 {
	r5 := uint16((r >> 11) & 0x1F) // 5 bits
	g6 := uint16((g >> 10) & 0x3F) // 6 bits
	b5 := uint16((b >> 11) & 0x1F) // 5 bits
	return (r5 << 11) | (g6 << 5) | b5
}

func ColorToRGB565(c color.Color) uint16 {
	r, g, b, _ := c.RGBA()
	r5 := uint16((r >> 11) & 0x1F) // 5 bits
	g6 := uint16((g >> 10) & 0x3F) // 6 bits
	b5 := uint16((b >> 11) & 0x1F) // 5 bits
	return (r5 << 11) | (g6 << 5) | b5
}

func RGBAtoColor(r, g, b, a uint8) color.Color {
	return color.RGBA{r, g, b, a}
}

func CPUStats() ([]float64, error) {
	return cpu.Percent(0, true)
}

func VMStats() (float64, float64, float64) {
	vmstat, err := mem.VirtualMemory()
	Check(err)
	mbFree := float64(vmstat.Available) / 1e6
	mbUsed := float64(vmstat.Used) / float64(vmstat.Total) * 100
	mbTotal := float64(vmstat.Total) / 1e6
	return mbFree, mbUsed, mbTotal
}
func PathStats(path string) (float64, float64) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	Check(err)

	total := float64(stat.Blocks * uint64(stat.Bsize))
	free := float64(stat.Bfree * uint64(stat.Bsize))
	percentage := total / free
	free /= 1e6
	return free, percentage
}
