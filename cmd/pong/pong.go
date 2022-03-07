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
	MESSAGE_WAIT    = 150 // 1500 [ms]
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
	width      int
	height     int
	top        int
	bottom     int
	packetData string
	enemy      PongObject
	user       PongObject
	results    []PongResult
}

var g = GameInfo{}

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

func drawString(x, y int, str string) {
	for i, ch := range []rune(str) {
		termbox.SetCell(x+i, y, ch, termbox.ColorDefault, termbox.ColorDefault)
	}
}

func (o *PongObject) draw() {
	runes := []rune(o.str)
	size := len(runes)
	x := o.point.X
	y := o.point.Y
	width := o.size.Width
	height := o.size.Height
	for h := 0; h < height; h++ {
		for w := 0; w < width; w++ {
			ch := runes[(h*width+w)%size]
			termbox.SetCell(x+w, y+h, ch, termbox.ColorDefault, termbox.ColorDefault)
		}
	}
}

func (o *BallObject) draw() {
	shadows := o.shadows
	for i := len(shadows) - 1; 0 <= i; i -= 1 {
		shadow := shadows[i]
		termbox.SetCell(shadow.point.X, shadow.point.Y, shadow.ch,
			termbox.ColorDefault, termbox.ColorDefault)
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

func newBall() BallObject {
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
	return NewBallObject(6, y, 1, dy, g.packetData)
}

func moveEnemy(ball *BallObject) {
	ballDx := ball.vectorF.Dx
	ballPoint := ball.Point()

	enemyY := g.enemy.point.Y + g.enemy.size.Height/2

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

	if bestY < enemyY && g.top+1 < g.enemy.point.Y {
		g.enemy.Move(0, -1)
	} else if enemyY < bestY && g.enemy.point.Y+g.enemy.size.Height < g.bottom {
		g.enemy.Move(0, 1)
	}
}

// play(one point). return (PongResult, isBreak)
func play(kch chan termbox.Key, tch chan bool, seq int) (*PongResult, bool) {

	message := fmt.Sprintf("start icmp_seq=%d", seq)
	messageLabel := NewPongObject((g.width-len(message))/2, g.height/2, len(message), 1, message)
	topWall := NewPongObject(0, g.top, g.width, 1, "=")
	bottomWall := NewPongObject(0, g.bottom, g.width, 1, "=")
	leftWall := NewPongObject(0, g.top+1, 1, g.bottom-g.top-1, "|")
	localhostLabel := NewPongObject(0, (g.height-11)/2, 1, 11, " localhost ")
	ball := newBall()

	ballSpeed := 0
	ballWait := BALL_WAIT_MAX
	enemyWait := ENEMY_WAIT_MAX
	msgWait := MESSAGE_WAIT

	ttl := opts.TimeToLive
	startTiem := time.Now()

	for {
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
		topWall.draw()
		bottomWall.draw()
		leftWall.draw()
		localhostLabel.draw()
		g.enemy.draw()
		g.user.draw()
		if msgWait == 0 {
			ball.draw()
		} else {
			messageLabel.draw()
		}

		descLabel := fmt.Sprintf("icmp_seq=%d ttl=%d", seq, ttl)
		drawString(g.width-len(descLabel)-2, 0, descLabel)
		termbox.Flush()

		select {
		case key := <-kch:
			switch key {
			case termbox.KeyCtrlC:
				return nil, true
			case termbox.KeyArrowUp:
				if g.top+1 < g.user.point.Y {
					g.user.Move(0, -1)
				}
			case termbox.KeyArrowDown:
				if g.user.point.Y+g.user.size.Height < g.bottom {
					g.user.Move(0, 1)
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
				moveEnemy(&ball)
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
				if g.user.Collision(ball.Point()) {
					ball.reflectPaddle(&g.user)
					if ballSpeed < BALL_WAIT_MAX && rand.Intn(2) == 1 {
						// speed up
						ballSpeed++
					}
				}
				if g.enemy.Collision(ball.Point()) {
					ball.reflectPaddle(&g.enemy)
					ttl--
					if ttl <= 0 {
						return &PongResult{}, false
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
					return &PongResult{true, ttl, time}, false
				} else if g.width-1 <= ballX {
					// lose
					return &PongResult{}, false
				}
			}
		}
	}
}

// play game. return (isBreak, error)
func game() (bool, error) {
	err := termbox.Init()
	if err != nil {
		return false, err
	}
	defer termbox.Close()
	termbox.HideCursor()

	g.width, g.height = termbox.Size()
	if g.width < TERM_WIDTH_MIN || g.height < TERM_HEIGHT_MIN {
		return false, fmt.Errorf("This term(%dx%d) is too narrow. Requires %dx%d area",
			g.width, g.height, TERM_WIDTH_MIN, TERM_HEIGHT_MIN)
	}
	g.top = 1
	g.bottom = g.height - 2

	g.enemy = NewPongObject(3, g.height/2-PADDLE_HEIGHT/2, PADDLE_WIDTH, PADDLE_HEIGHT, "||G|W|||")
	g.user = NewPongObject(g.width-4, g.height/2-PADDLE_HEIGHT/2, PADDLE_WIDTH, PADDLE_HEIGHT, "|")

	kch := make(chan termbox.Key)
	tch := make(chan bool)
	go keyEventLoop(kch)
	go timerEventLoop(tch)

	for i := 0; i < opts.Count; i += 1 {
		seq := i + 1
		result, isBreak := play(kch, tch, seq)
		if isBreak {
			return true, nil
		}
		g.results = append(g.results, *result)
	}

	return false, nil
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
		g.packetData = PACKET_HEADER + ":" + opts.Padding
	} else {
		g.packetData = PACKET_HEADER
	}
	packetLen := len(g.packetData)

	g.results = make([]PongResult, 0, opts.Count)

	startTiem := time.Now()

	// start pong
	isBrake, err := game()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	time := time.Since(startTiem) / time.Second

	nTotal := len(g.results)
	if nTotal == 0 {
		if isBrake {
			fmt.Println("^C")
		}
		return
	}
	nRecv := 0
	for _, r := range g.results {
		if r.received {
			nRecv++
		}
	}

	// show results
	fmt.Printf("PONG %s(%s) %d bytes of data.\n", opts.Args.Destination, addr, packetLen)
	for i, r := range g.results {
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
