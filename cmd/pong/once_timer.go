package main

type OnceTimer struct {
	waitCount int
	ch        chan bool
}

func NewOnceTimer() (*OnceTimer, chan bool) {
	ch := make(chan bool)
	return &OnceTimer{-1, ch}, ch
}

func (o *OnceTimer) Tick() {
	if 0 < o.waitCount {
		o.waitCount--
	} else if o.waitCount == 0 {
		o.waitCount = -1
		go func() {
			o.ch <- true
		}()
	}
}

func (o *OnceTimer) Set(wait int) {
	o.waitCount = wait
}
