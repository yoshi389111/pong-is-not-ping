package main

type VMessage struct {
	x, y int
	h    int
	str  string
}

func (m VMessage) Draw() {
	runes := []rune(m.str)
	size := len(runes)
	for h := 0; h < m.h; h++ {
		drawChar(m.x, m.y+h, runes[h%size])
	}
}
