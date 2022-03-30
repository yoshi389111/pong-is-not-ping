package main

type VMessage struct {
	x, y int
	h    int
	str  string
}

func (m VMessage) Draw() {
	runes := []rune(m.str)
	size := len(runes)
	for y := 0; y < m.h; y++ {
		drawChar(m.x, m.y+y, runes[y%size])
	}
}
