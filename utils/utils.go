package utils

func MapValue(value, inMin, inMax, outMin, outMax float64) float64 {
	return outMin + (value-inMin)*(outMax-outMin)/(inMax-inMin)
}

func Check(err error) {
	if err != nil {
		panic(err)
	}
}

func RGB565ToComponents(color uint16) (r, g, b float64) {
	r = float64((color >> 11 & 0x1F) / 0x1F)
	g = float64((color >> 5 & 0x3F) / 0x3F)
	b = float64((color & 0x1F) / 0x1F)
	return r, g, b
}
