package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/gdamore/tcell/v2/termbox"
)

const (
	TIME_SPAN     = 10  // [ms]
	BALL_WAIT_MAX = 9   // 90 [ms] = 9 * TIME_SPAN
	CPU_WAIT_MAX  = 9   // 90 [ms]
	MSG_WAIT_MAX  = 150 // 1500 [ms]
	STARTING_WAIT = 100 // 1000 [ms]

	PACKET_HEADER = "ICMP ECHO"

	LOCALHOST = " localhost "

	PADDLE_WIDTH  = 2
	PADDLE_HEIGHT = 4

	TERM_WIDTH_MIN  = 30
	TERM_HEIGHT_MIN = 15

	TOP_HEIGHT = 1
)

type Mode int

const (
	// show opening message
	MODE_OPENING_MSG Mode = iota
	// serve ball
	MODE_STARTING
	// playing
	MODE_PLAYING
	// show result message
	MODE_RESULT_MSG
	// exit
	MODE_EXIT
)

type PongResult struct {
	received bool
	ttl      int
	time     int
}

var (
	lastCpuY int
	lastUsrY int

	ball                     Ball
	topWall, bottomWall      Wall
	leftWall, localhostLabel VMessage
	cpuPaddle, usrPaddle     Paddle

	startTime     time.Time
	ttl           int
	width, height int
	packetData    string
	mode          Mode

	ballWait ClockWait
	timer    OnceTimer

	// key & resize chanel
	kch chan termbox.Event
)

func keyEventLoop() {
	for {
		kch <- termbox.PollEvent()
	}
}

func drawChar(x, y int, ch rune) {
	termbox.SetCell(x, y, ch, termbox.ColorDefault, termbox.ColorDefault)
}

func drawString(x, y int, str string) {
	for i, ch := range []rune(str) {
		drawChar(x+i, y, ch)
	}
}

func bottomHeight() int { return height - 2 }

func isNarrow() bool { return width < TERM_WIDTH_MIN || height < TERM_HEIGHT_MIN }

func isInsidePaddleY(y int) bool {
	return TOP_HEIGHT < y && y+PADDLE_HEIGHT <= bottomHeight()
}

func correctPaddleY(y int) int {
	if isInsidePaddleY(y) {
		return y
	} else {
		// Centered the first time or if it overflows during resizing
		return (height - PADDLE_HEIGHT) / 2
	}
}

func sign(num int) int {
	switch {
	case 0 < num:
		return 1
	case num < 0:
		return -1
	default:
		return 0
	}
}

func moveCpu() {
	ballPoint := ball.Point()
	var bestY int
	if ball.dx < 0 && ballPoint.X < width/2 {
		// Chase the ball
		bestY = ballPoint.Y
	} else if rand.Intn(2) == 1 {
		// Move between user and ball
		ballY := ballPoint.Y
		usrY := lastUsrY + PADDLE_HEIGHT/2
		bestY = (usrY + ballY) / 2
	} else {
		// Not moving
		return
	}

	centerY := cpuPaddle.y + cpuPaddle.h/2 - ttl%2
	dy := sign(bestY - centerY)
	if dy != 0 && isInsidePaddleY(cpuPaddle.y+dy) {
		cpuPaddle.MoveY(dy)
		lastCpuY = cpuPaddle.y
	}
}

func moveUsr(dy int) {
	if isInsidePaddleY(usrPaddle.y + dy) {
		usrPaddle.MoveY(dy)
		lastUsrY = usrPaddle.y
	}
}

func setMode(m Mode) {
	mode = m
	switch mode {
	case MODE_OPENING_MSG:
		timer.Set(MSG_WAIT_MAX, MODE_STARTING)
	case MODE_STARTING:
		timer.Set(STARTING_WAIT, MODE_PLAYING)
	case MODE_RESULT_MSG:
		timer.Set(MSG_WAIT_MAX, MODE_EXIT)
	}
}

func initGame() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	width, height = termbox.Size()

	ballY := rand.Intn(height/3) + height/3
	ballDy := DecideDy(rand.Intn(4))
	ball = NewBall(1, ballY, 1, ballDy, packetData)
	ballWait.Reset()

	topWall = Wall{1, width, '='}
	bottomWall = Wall{bottomHeight(), width, '='}
	leftWall = VMessage{0, TOP_HEIGHT + 1, bottomHeight() - 1 - TOP_HEIGHT, "|"}
	localhostLabel = VMessage{0, (height - len(LOCALHOST)) / 2, len(LOCALHOST), LOCALHOST}

	cpuY := correctPaddleY(lastCpuY)
	usrY := correctPaddleY(lastUsrY)
	cpuPaddle = Paddle{3, cpuY, PADDLE_WIDTH, PADDLE_HEIGHT, "||G|W|||"}
	usrPaddle = Paddle{width - 4, usrY, PADDLE_WIDTH, PADDLE_HEIGHT, "|"}

	ttl = opts.TimeToLive
	startTime = time.Now()
	setMode(MODE_OPENING_MSG)
}

// play service.
func playService(seq int) (result *PongResult, err error) {

	ballWait = NewClockWait(BALL_WAIT_MAX)
	cpuWait := NewClockWait(CPU_WAIT_MAX)
	timer = NewOnceTimer()

	title := fmt.Sprintf("pong %s", opts.Args.Destination)
	openingMessage := fmt.Sprintf("start icmp_seq=%d", seq)
	var resultMessage string

	ticker := time.NewTicker(time.Duration(TIME_SPAN) * time.Millisecond)
	defer ticker.Stop()

	initGame()

	for {
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

		if isNarrow() {
			if mode == MODE_OPENING_MSG {
				warningMessage := "This term is too narrow."
				drawString((width-len(warningMessage))/2, height/2, warningMessage)
			} else {
				err := fmt.Errorf("This term(%dx%d) is too narrow. Requires %dx%d area",
					width, height, TERM_WIDTH_MIN, TERM_HEIGHT_MIN)
				return nil, err
			}
		} else {
			topWall.Draw()
			bottomWall.Draw()
			leftWall.Draw()
			localhostLabel.Draw()
			usrPaddle.Draw()

			switch mode {
			case MODE_OPENING_MSG:
				drawString((width-len(openingMessage))/2, height/2, openingMessage)
			case MODE_STARTING:
				ball.Draw()
			case MODE_PLAYING:
				ball.Draw()
				cpuPaddle.Draw()
			case MODE_RESULT_MSG:
				ball.Draw()
				cpuPaddle.Draw()
				drawString((width-len(resultMessage))/2, height/2, resultMessage)
			}

			drawString(1, 0, title)

			descLabel := fmt.Sprintf("  icmp_seq=%d ttl=%d", seq, ttl)
			drawString(width-len(descLabel)-1, 0, descLabel)

			bps := int(8 * time.Second / (time.Millisecond * time.Duration(TIME_SPAN*ballWait.Speed())))
			bpsLabel := fmt.Sprintf("Speed: %dbps", bps)
			drawString(1, height-1, bpsLabel)
		}

		termbox.Flush()

		select {
		case <-ticker.C:
			ballWait.Tick()
			cpuWait.Tick()
			timer.Tick()

		case ev := <-kch:
			switch ev.Type {
			case termbox.EventKey:
				switch ev.Key {
				case termbox.KeyCtrlC:
					return nil, nil
				case termbox.KeyArrowUp:
					moveUsr(-1)
				case termbox.KeyArrowDown:
					moveUsr(1)
				}
			case termbox.EventResize:
				initGame()
				continue
			}

		case nextMode := <-timer.Ch:
			if nextMode == MODE_EXIT {
				return
			}
			setMode(nextMode)

		case <-cpuWait.Ch:
			if mode == MODE_PLAYING {
				// cpu is movable
				moveCpu()
			}

		case <-ballWait.Ch:
			// ball is movable
			switch mode {
			case MODE_OPENING_MSG:
				continue
			case MODE_STARTING, MODE_PLAYING:
				ball.Move()
			case MODE_RESULT_MSG:
				ball.Move()
				continue
			}

			topWall.Reflect(&ball)
			bottomWall.Reflect(&ball)
			if usrPaddle.Reflect(&ball) && rand.Intn(2) == 1 {
				ballWait.SpeedUp()
			}
			if mode == MODE_PLAYING && cpuPaddle.Reflect(&ball) {
				ttl--
				if ttl <= 0 {
					// lose(ttl=0)
					return &PongResult{}, nil
				}
			}
			topWall.Reflect(&ball)
			bottomWall.Reflect(&ball)
			ballX := ball.Point().X
			if ballX < 1 {
				// win
				time := int(time.Since(startTime) / time.Second)
				result, err = &PongResult{true, ttl, time}, nil
				resultMessage = fmt.Sprintf("received. time=%d", time)
				setMode(MODE_RESULT_MSG)

			} else if width-1 <= ballX {
				// lose
				result, err = &PongResult{}, nil
				resultMessage = "request timed out"
				setMode(MODE_RESULT_MSG)
			}
		}
	}
}

// play game.
func playGame() ([]PongResult, error) {
	err := termbox.Init()
	if err != nil {
		return nil, err
	}
	defer termbox.Close()
	termbox.HideCursor()

	kch = make(chan termbox.Event)
	go keyEventLoop()

	results := make([]PongResult, 0, opts.Count)
	for i := 0; i < opts.Count; i += 1 {

		seq := i + 1
		var result *PongResult
		result, err = playService(seq)
		if err != nil {
			return nil, err
		}
		if result == nil {
			// break the game
			return results, nil
		}
		results = append(results, *result)
	}
	return results, nil
}

// main of pong
func pong() {
	rand.Seed(time.Now().UnixNano())

	addr, err := net.ResolveIPAddr("ip", opts.Args.Destination)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if len(opts.Padding) != 0 {
		packetData = PACKET_HEADER + ":" + opts.Padding
	} else {
		packetData = PACKET_HEADER
	}

	startTiem := time.Now()

	// start pong
	results, err := playGame()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	time := time.Since(startTiem) / time.Second

	nTotal := len(results)
	if nTotal == 0 {
		fmt.Println("^C")
		return
	}
	nRecv := 0
	for _, r := range results {
		if r.received {
			nRecv++
		}
	}
	packetLen := len(packetData)

	// show results
	fmt.Printf("PONG %s(%s) %d bytes of data.\n", opts.Args.Destination, addr, packetLen)
	for i, r := range results {
		if r.received {
			fmt.Printf("%d bytes from %s: icmp_seq=%d ttl=%d time=%d sec\n",
				packetLen, opts.Args.Destination,
				i+1, r.ttl, r.time)
		} else {
			fmt.Printf("%d bytes from %s: request timed out\n",
				packetLen, opts.Args.Destination)
		}
	}
	if nTotal != opts.Count {
		fmt.Println("^C")
	}
	fmt.Printf("--- %s pong statistics ---\n", opts.Args.Destination)
	lossRatio := 100 - nRecv*100/nTotal
	fmt.Printf("%d packets transmitted, %d received, %d%% packet loss, time %d sec\n",
		nTotal, nRecv, lossRatio, time)
}
