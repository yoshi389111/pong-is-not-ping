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
	narrow bool

	// paddle positions

	cpuY int
	usrY int

	// status

	cpuWait   int
	ttl       int
	startTiem time.Time

	mode PongMode
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

func (g GameInfo) newBall(packetData string) *Ball {
	y := rand.Intn(g.height/3) + g.height/3
	dy := DecideDy(rand.Intn(4))
	return NewBall(6, y, 1, dy, packetData)
}

func (g GameInfo) isInsidePaddleY(y int) bool {
	return 1 < y && y+PADDLE_HEIGHT <= g.height-2
}

func (g GameInfo) correctPaddleY(y int) int {
	if g.isInsidePaddleY(y) {
		return y
	} else {
		return (g.height - PADDLE_HEIGHT) / 2
	}
}

func (g *GameInfo) moveCpu(cpu *Paddle, ball Ball) {
	ballDx := ball.dx
	ballPoint := ball.Point()

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

	paddleCenterY := cpu.y + cpu.h/2 - g.ttl%2
	var cpuDy int
	if bestY < paddleCenterY && g.isInsidePaddleY(cpu.y-1) {
		cpuDy = -1
	} else if paddleCenterY < bestY && g.isInsidePaddleY(cpu.y+1) {
		cpuDy = 1
	} else {
		return
	}
	cpu.MoveY(cpuDy)
	g.cpuY = cpu.y
}

func (g *GameInfo) moveUsr(usr *Paddle, dy int) {
	var usrDy int
	if dy < 0 && g.isInsidePaddleY(usr.y-1) {
		usrDy = -1
	} else if 0 < dy && g.isInsidePaddleY(usr.y+1) {
		usrDy = 1
	} else {
		return
	}
	usr.MoveY(usrDy)
	g.usrY = usr.y
}

func (g *GameInfo) adjustSize() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	g.width, g.height = termbox.Size()
	g.narrow = g.width < TERM_WIDTH_MIN || g.height < TERM_HEIGHT_MIN
}

func (g *GameInfo) initStatus() {
	g.cpuWait = CPU_WAIT_MAX
	g.ttl = opts.TimeToLive
	g.startTiem = time.Now()
	g.mode.Set(MODE_OPENING_MSG)
}

// play service.
func (g *GameInfo) playService(packetData string, seq int, kch chan termbox.Event, tch chan bool) (result *PongResult, err error) {

	g.adjustSize()
	g.initStatus()
	ball := g.newBall(packetData)

	title := fmt.Sprintf("pong %s", opts.Args.Destination)
	openingMessage := fmt.Sprintf("start icmp_seq=%d", seq)
	var resultMessage string

	for {
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

		topWall := Wall{1, g.width, '='}
		bottomWall := Wall{g.height - 2, g.width, '='}
		leftWall := VMessage{0, 2, g.height - 4, "|"}
		localhostLabel := VMessage{0, (g.height - len(LOCALHOST)) / 2, 11, LOCALHOST}

		cpuY := g.correctPaddleY(g.cpuY)
		usrY := g.correctPaddleY(g.usrY)
		cpuPaddle := Paddle{3, cpuY, PADDLE_WIDTH, PADDLE_HEIGHT, "||G|W|||"}
		usrPaddle := Paddle{g.width - 4, usrY, PADDLE_WIDTH, PADDLE_HEIGHT, "|"}

		if g.narrow {
			if g.mode.Get() == MODE_OPENING_MSG {
				warningMessage := "This term is too narrow."
				drawString((g.width-len(warningMessage))/2, g.height/2, warningMessage)
			} else {
				err := fmt.Errorf("This term(%dx%d) is too narrow. Requires %dx%d area",
					g.width, g.height, TERM_WIDTH_MIN, TERM_HEIGHT_MIN)
				return nil, err
			}
		} else {
			topWall.Draw()
			bottomWall.Draw()
			leftWall.Draw()
			localhostLabel.Draw()
			cpuPaddle.Draw()
			usrPaddle.Draw()

			switch g.mode.Get() {
			case MODE_PLAYING:
				ball.Draw()
			case MODE_OPENING_MSG:
				drawString((g.width-len(openingMessage))/2, g.height/2, openingMessage)
			case MODE_RESULT_MSG:
				ball.Draw()
				drawString((g.width-len(resultMessage))/2, g.height/2, resultMessage)
			}

			drawString(1, 0, title)

			descLabel := fmt.Sprintf("  icmp_seq=%d ttl=%d", seq, g.ttl)
			drawString(g.width-len(descLabel)-1, 0, descLabel)

			bps := int(8 * time.Second / (time.Millisecond * time.Duration(TIME_SPAN*ball.Speed())))
			bpsLabel := fmt.Sprintf("Speed: %dbps", bps)
			drawString(1, g.height-1, bpsLabel)
		}

		termbox.Flush()

		select {
		case ev := <-kch:
			switch ev.Type {
			case termbox.EventKey:
				switch ev.Key {
				case termbox.KeyCtrlC:
					return nil, nil
				case termbox.KeyArrowUp:
					g.moveUsr(&usrPaddle, -1)
				case termbox.KeyArrowDown:
					g.moveUsr(&usrPaddle, 1)
				}

			case termbox.EventResize:
				g.adjustSize()
				g.initStatus()
				ball = g.newBall(packetData)
				continue
			}

		case <-tch:
			g.mode.Update()
			switch g.mode.Get() {
			case MODE_END:
				return
			case MODE_OPENING_MSG:
				continue
			case MODE_RESULT_MSG:
				ball.Move()
				continue
			}

			g.cpuWait--
			if g.cpuWait <= 0 {
				g.cpuWait = CPU_WAIT_MAX
				g.moveCpu(&cpuPaddle, *ball)
			}

			if ball.Move() {
				topWall.Reflect(ball)
				bottomWall.Reflect(ball)
				if usrPaddle.Reflect(ball) && rand.Intn(2) == 1 {
					ball.SpeedUp()
				}
				if cpuPaddle.Reflect(ball) {
					g.ttl--
					if g.ttl <= 0 {
						// lose(ttl=0)
						return &PongResult{}, nil
					}
				}
				topWall.Reflect(ball)
				bottomWall.Reflect(ball)
				ballX := ball.Point().X
				if ballX < 1 {
					// win
					time := int(time.Since(g.startTiem) / time.Second)
					result, err = &PongResult{true, g.ttl, time}, nil
					resultMessage = fmt.Sprintf("received. time=%d", time)
					g.mode.Set(MODE_RESULT_MSG)

				} else if g.width-1 <= ballX {
					// lose
					result, err = &PongResult{}, nil
					resultMessage = "request timed out"
					g.mode.Set(MODE_RESULT_MSG)
				}
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
