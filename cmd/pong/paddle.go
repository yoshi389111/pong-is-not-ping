package main

type Paddle struct {
	x, y int
	w, h int
	str  string
}

func (o *Paddle) MoveY(dy int) {
	o.y += dy
}

func (o *Paddle) Collision(p Point) bool {
	return (o.x <= p.X && p.X < o.x+o.w &&
		o.y <= p.Y && p.Y < o.y+o.h)
}

func (p *Paddle) Reflect(ball *Ball) {
	ballY := int(ball.fy - ball.dy)

	dx := -ball.dx
	var dy float32
	switch ballY - p.y {
	case -1:
		dy = -1
	case 0:
		dy = -0.5
	case 1:
		dy = -0.25
	case 2:
		dy = 0.25
	case 3:
		dy = 0.5
	default:
		dy = 1
	}

	var ballX int
	if dx < 0 {
		ballX = p.x - 1
	} else {
		ballX = p.x + p.w
	}
	ball.Set(ballX, ballY, dx, dy)
}

func (p *Paddle) Draw() {
	runes := []rune(p.str)
	size := len(runes)
	for h, hmax := 0, p.h; h < hmax; h++ {
		for w, wmax := 0, p.w; w < wmax; w++ {
			drawChar(p.x+w, p.y+h, runes[(h*wmax+w)%size])
		}
	}
}
