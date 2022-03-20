package main

type Paddle struct {
	x, y int
	w, h int
	str  string
}

func (o *Paddle) MoveY(dy int) {
	o.y += dy
}

func (o Paddle) collision(pos Point) bool {
	return (o.x <= pos.X && pos.X < o.x+o.w &&
		o.y <= pos.Y && pos.Y < o.y+o.h)
}

func (o Paddle) Reflect(ball *Ball) bool {
	if !o.collision(ball.Point()) {
		return false
	}

	ballY := int(ball.fy - ball.dy)

	dx := -ball.dx
	dy := DecideDy(ballY - o.y)

	var ballX int
	if dx < 0 {
		ballX = o.x - 1
	} else {
		ballX = o.x + o.w
	}
	ball.Set(ballX, ballY, dx, dy)
	return true
}

func (o Paddle) Draw() {
	runes := []rune(o.str)
	size := len(runes)
	for y, ymax := 0, o.h; y < ymax; y++ {
		for x, xmax := 0, o.w; x < xmax; x++ {
			drawChar(o.x+x, o.y+y, runes[(y*xmax+x)%size])
		}
	}
}
