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
	TIME_SPAN       = 10  // [ms]
	BALL_WAIT_MAX   = 9   // 90 [ms] = 9 * TIME_SPAN
	ENEMY_WAIT_MAX  = 8   // 70 [ms]
	MSG_WAIT_MAX    = 150 // 1500 [ms]
	PACKET_HEADER   = "ICMP ECHO"
	PADDLE_WIDTH    = 2
	PADDLE_HEIGHT   = 4
	TERM_WIDTH_MIN  = 30
	TERM_HEIGHT_MIN = 15
)

type PongResult struct {
	received bool
	ttl      int
	time     int
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
		default:
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

func (o *PongObject) draw() {
	runes := []rune(o.str)
	size := len(runes)
	x := o.point.X
	y := o.point.Y
	for h, hmax := 0, o.size.Height; h < hmax; h++ {
		for w, wmax := 0, o.size.Width; w < wmax; w++ {
			drawChar(x+w, y+h, runes[(h*wmax+w)%size])
		}
	}
}

func (s *Shadow) draw() {
	drawChar(s.point.X, s.point.Y, s.ch)
}

func (o *BallObject) draw() {
	for i := len(o.shadows) - 1; 0 <= i; i -= 1 {
		o.shadows[i].draw()
	}
}

func (o *BallObject) reflectPaddle(paddle *PongObject) {
	ballY := int(o.pointF.Y - o.vectorF.Dy)

	dx := -o.vectorF.Dx
	var dy float32
	switch ballY - paddle.point.Y {
	case -1:
		dy = -1
	case 0:
		dy = -0.5
	case 1:
		dy = -0.25
	case 2:
		dy = 0.25
	case 3:
		dy = 0.5
	default:
		dy = 1
	}

	var ballX int
	if dx < 0 {
		ballX = paddle.point.X - 1
	} else {
		ballX = paddle.point.X + paddle.size.Width
	}
	o.Set(ballX, ballY, dx, dy)
}

func (o *BallObject) reflectWall(wall *PongObject) {
	point := o.Point()
	if o.vectorF.Dy < 0 {
		o.Set(point.X, wall.point.Y+wall.size.Height, o.vectorF.Dx, -o.vectorF.Dy)
	} else {
		o.Set(point.X, wall.point.Y-1, o.vectorF.Dx, -o.vectorF.Dy)
	}
}

func newBall(packetData string, height int) *BallObject {
	y := rand.Intn(height/3) + height/3
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
	return NewBallObject(6, y, 1, dy, packetData)
}

type GameInfo struct {
	width  int
	height int
	top    int
	bottom int

	enemyY int
	userY  int
	enemy  *PongObject
	user   *PongObject

	ball           *BallObject
	messageLabel   *PongObject
	topWall        *PongObject
	bottomWall     *PongObject
	leftWall       *PongObject
	localhostLabel *PongObject

	ballSpeed int
	ballWait  int
	enemyWait int
	msgWait   int
	ttl       int
	startTiem time.Time
}

func (g *GameInfo) moveEnemy() {
	ballDx := g.ball.vectorF.Dx
	ballPoint := g.ball.Point()

	var bestY int
	if ballDx < 0 && ballPoint.X < g.width/2 {
		bestY = ballPoint.Y
	} else if rand.Intn(2) == 1 {
		ballY := ballPoint.Y
		selfY := g.user.point.Y + g.user.size.Height/2
		bestY = (selfY + ballY) / 2
	} else {
		return
	}

	nowEnemyY := g.enemy.point.Y + g.enemy.size.Height/2 - g.ttl%2

	if bestY < nowEnemyY && g.top+1 < g.enemy.point.Y {
		g.enemy.Move(0, -1)
	} else if nowEnemyY < bestY && g.enemy.point.Y+g.enemy.size.Height < g.bottom {
		g.enemy.Move(0, 1)
	}
	g.enemyY = g.enemy.point.Y
}

func (g *GameInfo) moveUser(dy int) {
	if dy < 0 && g.top+1 < g.user.point.Y {
		g.user.Move(0, -1)
		g.userY = g.user.point.Y
	} else if 0 < dy && g.user.point.Y+g.user.size.Height < g.bottom {
		g.user.Move(0, 1)
		g.userY = g.user.point.Y
	}
}

func (g *GameInfo) init(packetData string, seq int) {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	g.width, g.height = termbox.Size()
	g.top = 1
	g.bottom = g.height - 2

	maxY := g.bottom - PADDLE_HEIGHT
	var enemyY int
	if g.enemyY == 0 {
		enemyY = maxY / 2
	} else if maxY < g.enemyY {
		enemyY = maxY
	} else {
		enemyY = g.enemyY
	}
	var userY int
	if g.userY == 0 {
		userY = maxY / 2
	} else if maxY < g.userY {
		userY = maxY
	} else {
		userY = g.userY
	}

	var message string
	if g.width < TERM_WIDTH_MIN || g.height < TERM_HEIGHT_MIN {
		message = "This term is too narrow."
	} else {
		message = fmt.Sprintf("start icmp_seq=%d", seq)
	}
	g.messageLabel = NewPongObject((g.width-len(message))/2, g.height/2, len(message), 1, message)

	g.topWall = NewPongObject(0, g.top, g.width, 1, "=")
	g.bottomWall = NewPongObject(0, g.bottom, g.width, 1, "=")
	g.leftWall = NewPongObject(0, g.top+1, 1, g.bottom-g.top-1, "|")
	g.localhostLabel = NewPongObject(0, (g.height-11)/2, 1, 11, " localhost ")

	g.enemy = NewPongObject(3, enemyY, PADDLE_WIDTH, PADDLE_HEIGHT, "||G|W|||")
	g.user = NewPongObject(g.width-4, userY, PADDLE_WIDTH, PADDLE_HEIGHT, "|")
	g.ball = newBall(packetData, g.height)

	g.ballSpeed = 0
	g.ballWait = BALL_WAIT_MAX
	g.enemyWait = ENEMY_WAIT_MAX
	g.msgWait = MSG_WAIT_MAX

	g.ttl = opts.TimeToLive
	g.startTiem = time.Now()
}

// play service.
func (g *GameInfo) playService(packetData string, seq int, kch chan termbox.Event, tch chan bool) (*PongResult, error) {

	g.init(packetData, seq)

	for {
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
		g.topWall.draw()
		g.bottomWall.draw()
		g.leftWall.draw()
		g.localhostLabel.draw()
		g.enemy.draw()
		g.user.draw()
		if g.msgWait == 0 {
			g.ball.draw()
		} else {
			g.messageLabel.draw()
		}

		titleLabel := fmt.Sprintf("pong %s", opts.Args.Destination)
		drawString(1, 0, titleLabel)

		descLabel := fmt.Sprintf("  icmp_seq=%d ttl=%d", seq, g.ttl)
		drawString(g.width-len(descLabel)-1, 0, descLabel)

		bps := int(8 * time.Second / (time.Millisecond * time.Duration(TIME_SPAN*(BALL_WAIT_MAX-g.ballSpeed))))
		bpsLabel := fmt.Sprintf("Speed: %dbps", bps)
		drawString(1, g.height-1, bpsLabel)

		termbox.Flush()

		select {
		case ev := <-kch:
			switch {
			case ev.Type == termbox.EventKey && ev.Key == termbox.KeyCtrlC:
				return nil, nil
			case ev.Type == termbox.EventKey && ev.Key == termbox.KeyArrowUp:
				g.moveUser(-1)
			case ev.Type == termbox.EventKey && ev.Key == termbox.KeyArrowDown:
				g.moveUser(1)
			case ev.Type == termbox.EventResize:
				g.init(packetData, seq)
				continue
			}
		case <-tch:
			if 0 < g.msgWait {
				g.msgWait--
				continue
			} else if g.width < TERM_WIDTH_MIN || g.height < TERM_HEIGHT_MIN {
				err := fmt.Errorf("This term(%dx%d) is too narrow. Requires %dx%d area",
					g.width, g.height, TERM_WIDTH_MIN, TERM_HEIGHT_MIN)
				return nil, err
			}

			g.enemyWait--
			if g.enemyWait < 0 {
				g.enemyWait = ENEMY_WAIT_MAX
				g.moveEnemy()
			}

			g.ballWait--
			if g.ballWait < 0 {
				g.ballWait = BALL_WAIT_MAX - g.ballSpeed
				g.ball.Next()

				if g.topWall.Collision(g.ball.Point()) {
					g.ball.reflectWall(g.topWall)
				}
				if g.bottomWall.Collision(g.ball.Point()) {
					g.ball.reflectWall(g.bottomWall)
				}
				if g.user.Collision(g.ball.Point()) {
					g.ball.reflectPaddle(g.user)
					if g.ballSpeed+1 < BALL_WAIT_MAX && rand.Intn(2) == 1 {
						// speed up
						g.ballSpeed++
					}
				}
				if g.enemy.Collision(g.ball.Point()) {
					g.ball.reflectPaddle(g.enemy)
					g.ttl--
					if g.ttl <= 0 {
						// lose(ttl=0)
						return &PongResult{}, nil
					}
				}
				if g.topWall.Collision(g.ball.Point()) {
					g.ball.reflectWall(g.topWall)
				}
				if g.bottomWall.Collision(g.ball.Point()) {
					g.ball.reflectWall(g.bottomWall)
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
	packetLen := len(packetData)

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
