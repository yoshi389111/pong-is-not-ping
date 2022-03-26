package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/nsf/termbox-go"
)

const (
	TIME_SPAN     = 10  // [ms]
	BALL_WAIT_MAX = 9   // 90 [ms] = 9 * TIME_SPAN
	CPU_WAIT_MAX  = 9   // 90 [ms]
	MSG_WAIT_MAX  = 150 // 1500 [ms]

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
	MODE_OPENING_MSG Mode = iota
	MODE_PLAYING
	MODE_RESULT_MSG
)

type PongResult struct {
	received bool
	ttl      int
	time     int
}

type GameInfo struct {
	cpuY int
	usrY int
}

func timerEventLoop(tch chan bool) {
	for {
		tch <- true
		time.Sleep(time.Duration(TIME_SPAN) * time.Millisecond)
	}
}

func keyEventLoop(kch chan termbox.Event) {
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

// play service.
func (g *GameInfo) playService(packetData string, seq int, kch chan termbox.Event, tch chan bool) (result *PongResult, err error) {

	ballWait, bch := NewClockWait(BALL_WAIT_MAX)
	cpuWait, cch := NewClockWait(CPU_WAIT_MAX)
	timer, och := NewOnceTimer()

	var ball *Ball
	var topWall, bottomWall Wall
	var leftWall, localhostLabel VMessage
	var cpuPaddle, usrPaddle Paddle
	var startTime time.Time
	var ttl int
	var width, height int
	var narrow bool

	bottomHeight := func() int { return height - 2 }

	isInsidePaddleY := func(y int) bool {
		return TOP_HEIGHT < y && y+PADDLE_HEIGHT <= bottomHeight()
	}

	correctPaddleY := func(y int) int {
		if isInsidePaddleY(y) {
			return y
		} else {
			return (height - PADDLE_HEIGHT) / 2
		}
	}

	moveCpu := func() {
		ballPoint := ball.Point()
		var bestY int
		if ball.dx < 0 && ballPoint.X < width/2 {
			// Chase the ball
			bestY = ballPoint.Y
		} else if rand.Intn(2) == 1 {
			// Move between user and ball
			ballY := ballPoint.Y
			usrY := g.usrY + PADDLE_HEIGHT/2
			bestY = (usrY + ballY) / 2
		} else {
			// Not moving
			return
		}

		centerY := cpuPaddle.y + cpuPaddle.h/2 - ttl%2
		var cpuDy int
		if bestY < centerY && isInsidePaddleY(cpuPaddle.y-1) {
			cpuDy = -1
		} else if centerY < bestY && isInsidePaddleY(cpuPaddle.y+1) {
			cpuDy = 1
		} else {
			return
		}
		cpuPaddle.MoveY(cpuDy)
		g.cpuY = cpuPaddle.y
	}

	moveUsr := func(dy int) {
		if isInsidePaddleY(usrPaddle.y + dy) {
			usrPaddle.MoveY(dy)
			g.usrY = usrPaddle.y
		}
	}

	var mode Mode
	setMode := func(m Mode) {
		mode = m
		switch mode {
		case MODE_OPENING_MSG, MODE_RESULT_MSG:
			timer.Set(MSG_WAIT_MAX)
		}
	}

	initGame := func() {
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
		width, height = termbox.Size()
		narrow = width < TERM_WIDTH_MIN || height < TERM_HEIGHT_MIN

		ballY := rand.Intn(height/3) + height/3
		ballDy := DecideDy(rand.Intn(4))
		ball = NewBall(6, ballY, 1, ballDy, packetData)
		ballWait.Reset()

		topWall = Wall{1, width, '='}
		bottomWall = Wall{bottomHeight(), width, '='}
		leftWall = VMessage{0, TOP_HEIGHT + 1, bottomHeight() - 1 - TOP_HEIGHT, "|"}
		localhostLabel = VMessage{0, (height - len(LOCALHOST)) / 2, len(LOCALHOST), LOCALHOST}

		cpuY := correctPaddleY(g.cpuY)
		usrY := correctPaddleY(g.usrY)
		cpuPaddle = Paddle{3, cpuY, PADDLE_WIDTH, PADDLE_HEIGHT, "||G|W|||"}
		usrPaddle = Paddle{width - 4, usrY, PADDLE_WIDTH, PADDLE_HEIGHT, "|"}

		ttl = opts.TimeToLive
		startTime = time.Now()
		setMode(MODE_OPENING_MSG)
	}

	title := fmt.Sprintf("pong %s", opts.Args.Destination)
	openingMessage := fmt.Sprintf("start icmp_seq=%d", seq)
	var resultMessage string

	initGame()

	for {
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

		if narrow {
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
			cpuPaddle.Draw()
			usrPaddle.Draw()

			switch mode {
			case MODE_PLAYING:
				ball.Draw()
			case MODE_OPENING_MSG:
				drawString((width-len(openingMessage))/2, height/2, openingMessage)
			case MODE_RESULT_MSG:
				ball.Draw()
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
		case <-tch:
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

		case <-och:
			switch mode {
			case MODE_OPENING_MSG:
				// end of opening message display
				setMode(MODE_PLAYING)
			case MODE_RESULT_MSG:
				// end of result message display
				return
			}

		case <-cch:
			if mode == MODE_PLAYING {
				// cpu is movable
				moveCpu()
			}

		case <-bch:
			// ball is movable
			switch mode {
			case MODE_OPENING_MSG:
				continue
			case MODE_PLAYING:
				ball.Move()
			case MODE_RESULT_MSG:
				ball.Move()
				continue
			}

			topWall.Reflect(ball)
			bottomWall.Reflect(ball)
			if usrPaddle.Reflect(ball) && rand.Intn(2) == 1 {
				ballWait.SpeedUp()
			}
			if cpuPaddle.Reflect(ball) {
				ttl--
				if ttl <= 0 {
					// lose(ttl=0)
					return &PongResult{}, nil
				}
			}
			topWall.Reflect(ball)
			bottomWall.Reflect(ball)
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
func playGame(packetData string) ([]PongResult, error) {
	err := termbox.Init()
	if err != nil {
		return nil, err
	}
	defer termbox.Close()
	termbox.HideCursor()

	g := GameInfo{}

	kch := make(chan termbox.Event)
	tch := make(chan bool)
	go keyEventLoop(kch)
	go timerEventLoop(tch)

	results := make([]PongResult, 0, opts.Count)
	for i := 0; i < opts.Count; i += 1 {

		seq := i + 1
		var result *PongResult
		result, err = g.playService(packetData, seq, kch, tch)
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

	var packetData string
	if len(opts.Padding) != 0 {
		packetData = PACKET_HEADER + ":" + opts.Padding
	} else {
		packetData = PACKET_HEADER
	}

	startTiem := time.Now()

	// start pong
	results, err := playGame(packetData)
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
