package main

type ClockWait struct {
	waitMax   int
	waitNow   int
	waitCount int
	Ch        chan bool
}

func NewClockWait(wait int) ClockWait {
	ch := make(chan bool)
	return ClockWait{wait, wait, wait, ch}
}

func (o *ClockWait) Tick() {
	o.waitCount--
	if o.waitCount <= 0 {
		o.waitCount = o.waitNow
		go func() {
			o.Ch <- true
		}()
	}
}

func (o *ClockWait) SpeedUp() {
	if 1 < o.waitNow {
		o.waitNow--
		if o.waitNow < o.waitCount {
			o.waitCount = o.waitNow
		}
	}
}

func (o ClockWait) Speed() int {
	return o.waitNow
}

func (o *ClockWait) Reset() {
	o.waitNow = o.waitMax
	o.waitCount = o.waitNow
}
