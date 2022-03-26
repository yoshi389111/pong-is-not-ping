package main

type ClockWait struct {
	waitMax   int
	waitNow   int
	waitCount int
	ch        chan bool
}

func NewClockWait(wait int) (*ClockWait, chan bool) {
	ch := make(chan bool)
	return &ClockWait{wait, wait, wait, ch}, ch
}

func (o *ClockWait) Tick() {
	o.waitCount--
	if o.waitCount <= 0 {
		o.waitCount = o.waitNow
		go func() {
			o.ch <- true
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
