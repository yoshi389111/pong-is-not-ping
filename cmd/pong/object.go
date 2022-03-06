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

func NewPongObject(x, y, w, h int, s string) PongObject {
	return PongObject{
		point: Point{X: x, Y: y},
		size:  Size{Width: w, Height: h},
		str:   s,
	}
}

func (o *PongObject) Point() Point {
	return o.point
}

func (o *PongObject) Size() Size {
	return o.size
}

func (o *PongObject) Move(dx, dy int) {
	o.point.X += dx
	o.point.Y += dy
}

func (o *PongObject) Collision(p Point) bool {
	return (o.point.X <= p.X && p.X < o.point.X+o.size.Width &&
		o.point.Y <= p.Y && p.Y < o.point.Y+o.size.Height)
}

func (o *PongObject) Str() string {
	return o.str
}

type Shadow struct {
	point Point
	ch    rune
}

func (o *Shadow) Point() Point {
	return o.point
}

func (o *Shadow) Char() rune {
	return o.ch
}

type BallObject struct {
	pointF  BallPoint
	vectorF BallVector
	shadows []Shadow
}

func NewBallObject(x, y int, dx, dy float32, s string) BallObject {
	shadows := make([]Shadow, 0, len(s))
	for _, r := range []rune(s) {
		shadows = append(shadows, Shadow{Point{x, y}, r})
	}
	return BallObject{
		pointF:  BallPoint{X: float32(x), Y: float32(y)},
		vectorF: BallVector{Dx: dx, Dy: dy},
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

func (o *BallObject) VectorF() BallVector {
	return o.vectorF
}

func (o *BallObject) Shadows() []Shadow {
	return o.shadows
}

func (o *BallObject) Point() Point {
	return o.shadows[0].point
}

func (o *BallObject) Set(x, y int, dx, dy float32) {
	o.pointF = BallPoint{X: float32(x), Y: float32(y)}
	o.vectorF = BallVector{Dx: dx, Dy: dy}
	o.shadows[0].point = Point{x, y}
}
