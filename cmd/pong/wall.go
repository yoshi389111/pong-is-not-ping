package main

type Wall struct {
	y     int
	width int
	ch    rune
}

func (o Wall) collision(p Point) bool {
	return (0 <= p.X && p.X < o.width &&
		o.y <= p.Y && p.Y < o.y+1)
}

func (o Wall) Reflect(ball *Ball) bool {
	if !o.collision(ball.Point()) {
		return false
	}

	var newY int
	if ball.dy < 0 {
		newY = o.y + 1
	} else {
		newY = o.y - 1
	}
	ball.Set(ball.Point().X, newY, ball.dx, -ball.dy)
	return true
}

func (o Wall) Draw() {
	for width := 0; width < o.width; width++ {
		drawChar(width, o.y, o.ch)
	}
}
