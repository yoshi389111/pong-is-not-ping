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

type GameInfo struct {
	width  int
	height int
	top    int
	bottom int
	enemyY int
	userY  int
}

func timerEventLoop(tch chan bool) {
	for {
		tch <- true
		time.Sleep(time.Duration(TIME_SPAN) * time.Millisecond)
	}
}

func keyEventLoop(kch chan termbox.Key) {
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			kch <- ev.Key
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

func newBall(packetData string, height int) BallObject {
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

func moveEnemy(g *GameInfo, enemy, user *PongObject, ball *BallObject, ttl int) {
	ballDx := ball.vectorF.Dx
	ballPoint := ball.Point()

	var bestY int
	if ballDx < 0 && ballPoint.X < g.width/2 {
		bestY = ballPoint.Y
	} else if rand.Intn(2) == 1 {
		ballY := ballPoint.Y
		selfY := user.point.Y + user.size.Height/2
		bestY = (selfY + ballY) / 2
	} else {
		return
	}

	enemyY := enemy.point.Y + enemy.size.Height/2 - ttl%2

	if bestY < enemyY && g.top+1 < enemy.point.Y {
		enemy.Move(0, -1)
	} else if enemyY < bestY && enemy.point.Y+enemy.size.Height < g.bottom {
		enemy.Move(0, 1)
	}
}

// play service.
func playService(g *GameInfo, packetData string, kch chan termbox.Key, tch chan bool, seq int) (result *PongResult, isBreak bool) {

	message := fmt.Sprintf("start icmp_seq=%d", seq)
	messageLabel := NewPongObject((g.width-len(message))/2, g.height/2, len(message), 1, message)
	topWall := NewPongObject(0, g.top, g.width, 1, "=")
	bottomWall := NewPongObject(0, g.bottom, g.width, 1, "=")
	leftWall := NewPongObject(0, g.top+1, 1, g.bottom-g.top-1, "|")
	localhostLabel := NewPongObject(0, (g.height-11)/2, 1, 11, " localhost ")

	enemy := NewPongObject(3, g.enemyY, PADDLE_WIDTH, PADDLE_HEIGHT, "||G|W|||")
	user := NewPongObject(g.width-4, g.userY, PADDLE_WIDTH, PADDLE_HEIGHT, "|")
	defer func() {
		g.enemyY = enemy.point.Y
		g.userY = user.point.Y
	}()

	ball := newBall(packetData, g.height)

	ballSpeed := 0
	ballWait := BALL_WAIT_MAX
	enemyWait := ENEMY_WAIT_MAX
	msgWait := MSG_WAIT_MAX

	ttl := opts.TimeToLive
	startTiem := time.Now()

	for {
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
		topWall.draw()
		bottomWall.draw()
		leftWall.draw()
		localhostLabel.draw()
		enemy.draw()
		user.draw()
		if msgWait == 0 {
			ball.draw()
		} else {
			messageLabel.draw()
		}

		titleLabel := fmt.Sprintf("pong %s", opts.Args.Destination)
		drawString(1, 0, titleLabel)

		descLabel := fmt.Sprintf("  icmp_seq=%d ttl=%d", seq, ttl)
		drawString(g.width-len(descLabel)-1, 0, descLabel)

		bps := int(8 * time.Second / (time.Millisecond * time.Duration(TIME_SPAN*(BALL_WAIT_MAX-ballSpeed+1))))
		bpsLabel := fmt.Sprintf("Speed: %dbps", bps)
		drawString(1, g.height-1, bpsLabel)

		termbox.Flush()

		select {
		case key := <-kch:
			switch key {
			case termbox.KeyCtrlC:
				isBreak = true
				return
			case termbox.KeyArrowUp:
				if g.top+1 < user.point.Y {
					user.Move(0, -1)
				}
			case termbox.KeyArrowDown:
				if user.point.Y+user.size.Height < g.bottom {
					user.Move(0, 1)
				}
			}
		case <-tch:
			if 0 < msgWait {
				msgWait--
				continue
			}

			enemyWait--
			if enemyWait < 0 {
				enemyWait = ENEMY_WAIT_MAX
				moveEnemy(g, &enemy, &user, &ball, ttl)
			}

			ballWait--
			if ballWait < 0 {
				ballWait = BALL_WAIT_MAX - ballSpeed
				ball.Next()

				if topWall.Collision(ball.Point()) {
					ball.reflectWall(&topWall)
				}
				if bottomWall.Collision(ball.Point()) {
					ball.reflectWall(&bottomWall)
				}
				if user.Collision(ball.Point()) {
					ball.reflectPaddle(&user)
					if ballSpeed < BALL_WAIT_MAX && rand.Intn(2) == 1 {
						// speed up
						ballSpeed++
					}
				}
				if enemy.Collision(ball.Point()) {
					ball.reflectPaddle(&enemy)
					ttl--
					if ttl <= 0 {
						result = &PongResult{}
						return
					}
				}
				if topWall.Collision(ball.Point()) {
					ball.reflectWall(&topWall)
				}
				if bottomWall.Collision(ball.Point()) {
					ball.reflectWall(&bottomWall)
				}
				ballX := ball.Point().X
				if ballX < 1 {
					// win
					time := int(time.Since(startTiem) / time.Second)
					result = &PongResult{true, ttl, time}
					return
				} else if g.width-1 <= ballX {
					// lose
					result = &PongResult{}
					return
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
	g.width, g.height = termbox.Size()
	if g.width < TERM_WIDTH_MIN || g.height < TERM_HEIGHT_MIN {
		err = fmt.Errorf("This term(%dx%d) is too narrow. Requires %dx%d area",
			g.width, g.height, TERM_WIDTH_MIN, TERM_HEIGHT_MIN)
		return
	}
	g.top = 1
	g.bottom = g.height - 2
	g.enemyY = g.height/2 - PADDLE_HEIGHT/2
	g.userY = g.height/2 - PADDLE_HEIGHT/2

	kch := make(chan termbox.Key)
	tch := make(chan bool)
	go keyEventLoop(kch)
	go timerEventLoop(tch)

	results = make([]PongResult, 0, opts.Count)
	for i := 0; i < opts.Count; i += 1 {
		seq := i + 1
		var result *PongResult
		result, isBreak = playService(&g, packetData, kch, tch, seq)
		if isBreak {
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
