package main

type Message struct {
	x   int
	y   int
	str string
}

func (m Message) Draw() {
	drawString(m.x, m.y, m.str)
}
