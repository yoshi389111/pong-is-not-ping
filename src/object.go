package main

type Point struct {
	X int
	Y int
}

type Size struct {
	Width  int
	Height int
}

type BallPoint struct {
	X float32
	Y float32
}

type BallVector struct {
	Dx float32
	Dy float32
}

type PongObject struct {
	point Point
	size  Size
	str   string
}

func NewPongObject(x, y, w, h int, str string) PongObject {
	return PongObject{
		point: Point{x, y},
		size:  Size{w, h},
		str:   str,
	}
}

func (o *PongObject) Move(dx, dy int) {
	o.point.X += dx
	o.point.Y += dy
}

func (o *PongObject) Collision(p Point) bool {
	return (o.point.X <= p.X && p.X < o.point.X+o.size.Width &&
		o.point.Y <= p.Y && p.Y < o.point.Y+o.size.Height)
}

type Shadow struct {
	point Point
	ch    rune
}

type BallObject struct {
	pointF  BallPoint
	vectorF BallVector
	shadows []Shadow
}

func NewBallObject(x, y int, dx, dy float32, str string) BallObject {
	shadows := make([]Shadow, 0, len(str))
	for _, ch := range []rune(str) {
		shadows = append(shadows, Shadow{Point{x, y}, ch})
	}
	return BallObject{
		pointF:  BallPoint{float32(x), float32(y)},
		vectorF: BallVector{dx, dy},
		shadows: shadows,
	}
}

func (o *BallObject) Next() {
	o.pointF.X += o.vectorF.Dx
	o.pointF.Y += o.vectorF.Dy

	for i := len(o.shadows) - 1; 0 < i; i -= 1 {
		o.shadows[i].point = o.shadows[i-1].point
	}
	o.shadows[0].point.X = int(o.pointF.X)
	o.shadows[0].point.Y = int(o.pointF.Y)
}

func (o *BallObject) Point() Point {
	return o.shadows[0].point
}

func (o *BallObject) Set(x, y int, dx, dy float32) {
	o.pointF = BallPoint{float32(x), float32(y)}
	o.vectorF = BallVector{dx, dy}
	o.shadows[0].point = Point{x, y}
}
