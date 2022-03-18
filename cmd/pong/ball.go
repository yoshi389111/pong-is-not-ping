package main

type Ball struct {
	fx, fy  float32
	dx, dy  float32
	shadows []Shadow
}

func NewBall(x, y int, dx, dy float32, str string) *Ball {
	shadows := make([]Shadow, 0, len(str))
	for _, ch := range []rune(str) {
		shadows = append(shadows, Shadow{x, y, ch})
	}
	return &Ball{
		fx:      float32(x),
		fy:      float32(y),
		dx:      dx,
		dy:      dy,
		shadows: shadows,
	}
}

func (o *Ball) moveNext() {
	o.fx += o.dx
	o.fy += o.dy

	for i := len(o.shadows) - 1; 0 < i; i -= 1 {
		o.shadows[i].x = o.shadows[i-1].x
		o.shadows[i].y = o.shadows[i-1].y
	}
	o.shadows[0].x = int(o.fx)
	o.shadows[0].y = int(o.fy)
}

func (o *Ball) Point() Point {
	s := o.shadows[0]
	return Point{s.x, s.y}
}

func (o *Ball) Set(x, y int, dx, dy float32) {
	o.fx, o.fy = float32(x), float32(y)
	o.dx, o.dy = dx, dy
	o.shadows[0].x = x
	o.shadows[0].y = y
}

func (o *Ball) Draw() {
	for i := len(o.shadows) - 1; 0 <= i; i -= 1 {
		o.shadows[i].Draw()
	}
}
