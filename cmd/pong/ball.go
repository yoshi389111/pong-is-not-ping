package main

type Ball struct {
	fx, fy float32
	dx, dy float32
	points []Point
	str    string

	ballWait  int
	ballSpeed int
}

func DecideDy(y int) float32 {
	switch {
	case y <= -1:
		return -1
	case y == 0:
		return -0.5
	case y == 1:
		return -0.25
	case y == 2:
		return 0.25
	case y == 3:
		return 0.5
	default:
		return 1
	}
}

func NewBall(x, y int, dx, dy float32, str string) *Ball {
	l := len([]rune(str))
	points := make([]Point, 0, l)
	for i := 0; i < l; i += 1 {
		points = append(points, Point{x, y})
	}
	return &Ball{
		fx:     float32(x),
		fy:     float32(y),
		dx:     dx,
		dy:     dy,
		str:    str,
		points: points,

		ballWait:  BALL_WAIT_MAX,
		ballSpeed: 0,
	}
}

func (o *Ball) Move() {
	o.fx += o.dx
	o.fy += o.dy
	point := Point{int(o.fx), int(o.fy)}
	o.points = append([]Point{point}, o.points[:len(o.points)-1]...)
}

func (o Ball) Point() Point {
	return o.points[0]
}

func (o *Ball) Set(x, y int, dx, dy float32) {
	o.fx, o.fy = float32(x), float32(y)
	o.dx, o.dy = dx, dy
	o.points[0].X = x
	o.points[0].Y = y
}

func (o Ball) Draw() {
	runes := []rune(o.str)
	for i := len(o.points) - 1; 0 <= i; i -= 1 {
		ch := runes[i]
		point := o.points[i]
		drawChar(point.X, point.Y, ch)
	}
}
