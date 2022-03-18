package main

type Shadow struct {
	x, y int
	ch   rune
}

func (s Shadow) Draw() {
	drawChar(s.x, s.y, s.ch)
}
