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
	CPU_WAIT_MAX  = 8   // 70 [ms]
	MSG_WAIT_MAX  = 150 // 1500 [ms]

	PACKET_HEADER = "ICMP ECHO"

	PADDLE_WIDTH  = 2
	PADDLE_HEIGHT = 4

	TERM_WIDTH_MIN  = 30
	TERM_HEIGHT_MIN = 15
)

type PongResult struct {
	received bool
	ttl      int
	time     int
}

type GameInfo struct {
	// terminal

	width  int
	height int
	top    int
	bottom int
	narrow bool

	// paddle positions & ball

	cpuY int
	usrY int
	ball *Ball

	// status

	ballSpeed int
	ballWait  int
	cpuWait   int
	msgWait   int
	ttl       int
	startTiem time.Time
}

func timerEventLoop(tch chan bool) {
	for {
		tch <- true
		time.Sleep(time.Duration(TIME_SPAN) * time.Millisecond)
	}
}

func keyEventLoop(kch chan termbox.Event) {
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey, termbox.EventResize:
			kch <- ev
		}
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

func (g *GameInfo) newBall(packetData string) *Ball {
	y := rand.Intn(g.height/3) + g.height/3
	var dy float32
	switch rand.Intn(4) {
	case 0:
		dy = -0.5
	case 1:
		dy = -0.25
	case 2:
		dy = 0.25
	default:
		dy = 0.5
	}
	return NewBall(6, y, 1, dy, packetData)
}

func (g *GameInfo) isInsideCourt(paddleY int) bool {
	return g.top < paddleY && paddleY+PADDLE_HEIGHT <= g.bottom
}

func (g *GameInfo) correctPaddleY(y int) int {
	if g.isInsideCourt(y) {
		return y
	} else {
		return (g.bottom - PADDLE_HEIGHT) / 2
	}
}

func (g *GameInfo) moveCpu(p *Paddle) {
	ballDx := g.ball.dx
	ballPoint := g.ball.Point()

	var bestY int
	if ballDx < 0 && ballPoint.X < g.width/2 {
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

	paddleCenterY := p.y + p.h/2 - g.ttl%2
	var cpuDy int
	if bestY < paddleCenterY && g.isInsideCourt(p.y-1) {
		cpuDy = -1
	} else if paddleCenterY < bestY && g.isInsideCourt(p.y+1) {
		cpuDy = 1
	} else {
		return
	}
	p.MoveY(cpuDy)
	g.cpuY = p.y
}

func (g *GameInfo) moveUsr(p *Paddle, dy int) {
	var usrDy int
	if dy < 0 && g.isInsideCourt(p.y-1) {
		usrDy = -1
	} else if 0 < dy && g.isInsideCourt(p.y+1) {
		usrDy = 1
	} else {
		return
	}
	p.MoveY(usrDy)
	g.usrY = p.y
}

func (g *GameInfo) adjustSize() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	g.width, g.height = termbox.Size()
	g.top = 1
	g.bottom = g.height - 2
	g.narrow = g.width < TERM_WIDTH_MIN || g.height < TERM_HEIGHT_MIN
}

func (g *GameInfo) initStatus() {
	g.ballSpeed = 0
	g.ballWait = BALL_WAIT_MAX
	g.cpuWait = CPU_WAIT_MAX
	g.msgWait = MSG_WAIT_MAX
	g.ttl = opts.TimeToLive
	g.startTiem = time.Now()
}

// play service.
func (g *GameInfo) playService(packetData string, seq int, kch chan termbox.Event, tch chan bool) (*PongResult, error) {

	g.adjustSize()
	g.initStatus()
	g.ball = g.newBall(packetData)

	title := fmt.Sprintf("pong %s", opts.Args.Destination)
	titleLabel := Message{1, 0, title}
	startMessage := fmt.Sprintf("start icmp_seq=%d", seq)

	for {
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

		topWall := Wall{g.top, g.width, '='}
		bottomWall := Wall{g.bottom, g.width, '='}
		leftWall := VMessage{0, g.top + 1, g.bottom - g.top - 1, "|"}
		localhostLabel := VMessage{0, (g.height - 11) / 2, 11, " localhost "}

		cpuY := g.correctPaddleY(g.cpuY)
		usrY := g.correctPaddleY(g.usrY)
		cpuPaddle := Paddle{3, cpuY, PADDLE_WIDTH, PADDLE_HEIGHT, "||G|W|||"}
		usrPaddle := Paddle{g.width - 4, usrY, PADDLE_WIDTH, PADDLE_HEIGHT, "|"}

		if g.narrow {
			if g.msgWait == 0 {
				err := fmt.Errorf("This term(%dx%d) is too narrow. Requires %dx%d area",
					g.width, g.height, TERM_WIDTH_MIN, TERM_HEIGHT_MIN)
				return nil, err
			} else {
				warningMessage := "This term is too narrow."
				drawString((g.width-len(warningMessage))/2, g.height/2, warningMessage)
			}
		} else {
			topWall.Draw()
			bottomWall.Draw()
			leftWall.Draw()
			localhostLabel.Draw()
			cpuPaddle.Draw()
			usrPaddle.Draw()
			if g.msgWait == 0 {
				g.ball.Draw()
			} else {
				drawString((g.width-len(startMessage))/2, g.height/2, startMessage)
			}
			titleLabel.Draw()

			descLabel := fmt.Sprintf("  icmp_seq=%d ttl=%d", seq, g.ttl)
			drawString(g.width-len(descLabel)-1, 0, descLabel)

			bps := int(8 * time.Second / (time.Millisecond * time.Duration(TIME_SPAN*(BALL_WAIT_MAX-g.ballSpeed))))
			bpsLabel := fmt.Sprintf("Speed: %dbps", bps)
			drawString(1, g.height-1, bpsLabel)
		}

		termbox.Flush()

		select {
		case ev := <-kch:
			switch {
			case ev.Type == termbox.EventKey && ev.Key == termbox.KeyCtrlC:
				return nil, nil
			case ev.Type == termbox.EventKey && ev.Key == termbox.KeyArrowUp:
				g.moveUsr(&usrPaddle, -1)
			case ev.Type == termbox.EventKey && ev.Key == termbox.KeyArrowDown:
				g.moveUsr(&usrPaddle, 1)
			case ev.Type == termbox.EventResize:
				g.adjustSize()
				g.initStatus()
				g.ball = g.newBall(packetData)
				continue
			}
		case <-tch:
			if 0 < g.msgWait {
				g.msgWait--
				continue
			}

			g.cpuWait--
			if g.cpuWait < 0 {
				g.cpuWait = CPU_WAIT_MAX
				g.moveCpu(&cpuPaddle)
			}

			g.ballWait--
			if g.ballWait < 0 {
				g.ballWait = BALL_WAIT_MAX - g.ballSpeed
				g.ball.moveNext()

				if topWall.Collision(g.ball.Point()) {
					topWall.Reflect(g.ball)
				}
				if bottomWall.Collision(g.ball.Point()) {
					bottomWall.Reflect(g.ball)
				}
				if usrPaddle.Collision(g.ball.Point()) {
					usrPaddle.Reflect(g.ball)
					if g.ballSpeed+1 < BALL_WAIT_MAX && rand.Intn(2) == 1 {
						// speed up
						g.ballSpeed++
					}
				}
				if cpuPaddle.Collision(g.ball.Point()) {
					cpuPaddle.Reflect(g.ball)
					g.ttl--
					if g.ttl <= 0 {
						// lose(ttl=0)
						return &PongResult{}, nil
					}
				}
				if topWall.Collision(g.ball.Point()) {
					topWall.Reflect(g.ball)
				}
				if bottomWall.Collision(g.ball.Point()) {
					bottomWall.Reflect(g.ball)
				}
				ballX := g.ball.Point().X
				if ballX < 1 {
					// win
					time := int(time.Since(g.startTiem) / time.Second)
					return &PongResult{true, g.ttl, time}, nil
				} else if g.width-1 <= ballX {
					// lose
					return &PongResult{}, nil
				}
			}
		}
	}
}

// play game.
func playGame(packetData string) (results []PongResult, isBreak bool, err error) {
	err = termbox.Init()
	if err != nil {
		return
	}
	defer termbox.Close()
	termbox.HideCursor()

	g := GameInfo{}

	kch := make(chan termbox.Event)
	tch := make(chan bool)
	go keyEventLoop(kch)
	go timerEventLoop(tch)

	results = make([]PongResult, 0, opts.Count)
	for i := 0; i < opts.Count; i += 1 {

		seq := i + 1
		var result *PongResult
		result, err = g.playService(packetData, seq, kch, tch)
		if err != nil {
			return
		}
		if result == nil {
			isBreak = true
			return
		}
		results = append(results, *result)
	}
	return
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
	results, isBrake, err := playGame(packetData)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	time := time.Since(startTiem) / time.Second

	nTotal := len(results)
	if nTotal == 0 {
		if isBrake {
			fmt.Println("^C")
		}
		return
	}
	nRecv := 0
	for _, r := range results {
		if r.received {
			nRecv++
		}
	}

	// show results
	packetLen := len(packetData)
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
	if isBrake {
		fmt.Println("^C")
	}
	fmt.Printf("--- %s pong statistics ---\n", opts.Args.Destination)
	lossRatio := 100 - nRecv*100/nTotal
	fmt.Printf("%d packets transmitted, %d received, %d%% packet loss, time %d sec\n",
		nTotal, nRecv, lossRatio, time)
}
