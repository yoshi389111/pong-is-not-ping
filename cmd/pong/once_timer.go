package main

type OnceTimer struct {
	waitCount int
	nextMode  Mode
	Ch        chan Mode
}

func NewOnceTimer() OnceTimer {
	ch := make(chan Mode)
	return OnceTimer{-1, MODE_EXIT, ch}
}

func (o *OnceTimer) Tick() {
	if 0 < o.waitCount {
		o.waitCount--
	} else if o.waitCount == 0 {
		o.waitCount = -1
		go func() {
			o.Ch <- o.nextMode
		}()
	}
}

func (o *OnceTimer) Set(wait int, nextMode Mode) {
	o.waitCount = wait
	o.nextMode = nextMode
}
