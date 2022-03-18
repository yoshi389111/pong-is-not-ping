package main

type Wall struct {
	y     int
	width int
	ch    rune
}

func (o Wall) Collision(p Point) bool {
	return (0 <= p.X && p.X < o.width &&
		o.y <= p.Y && p.Y < o.y+1)
}

func (w Wall) Reflect(ball *Ball) {
	var newY int
	if ball.dy < 0 {
		newY = w.y + 1
	} else {
		newY = w.y - 1
	}
	ball.Set(ball.Point().X, newY, ball.dx, -ball.dy)
}

func (o Wall) Draw() {
	for w := 0; w < o.width; w++ {
		drawChar(w, o.y, o.ch)
	}
}
