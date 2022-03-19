package main

const (
	MODE_OPENING_MSG = iota
	MODE_PLAYING
	MODE_RESULT_MSG
	MODE_END
)

type PongMode struct {
	mode    int
	msgWait int
}

func (m *PongMode) Update() {
	if m.mode == MODE_PLAYING {
		return
	}
	m.msgWait--
	if 0 < m.msgWait {
		return
	}
	if m.mode == MODE_OPENING_MSG {
		m.Set(MODE_PLAYING)
	} else if m.mode == MODE_RESULT_MSG {
		m.Set(MODE_END)
	}
}

func (m *PongMode) Set(mode int) {
	m.mode = mode
	switch mode {
	case MODE_OPENING_MSG, MODE_RESULT_MSG:
		m.msgWait = MSG_WAIT_MAX
	case MODE_PLAYING, MODE_END:
		m.msgWait = 0
	}
}

func (m *PongMode) Get() int {
	return m.mode
}
