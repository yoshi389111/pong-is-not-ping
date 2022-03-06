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
	BALL_WAIT_INIT  = 9   // 90 [ms] = 9 * TIME_SPAN
	ENEMY_WAIT_MAX  = 8   // 70 [ms]
	MESSAGE_WAIT    = 150 // 1500 [ms]
	PACKET_HEADER   = "ICMP ECHO"
	PADDLE_WIDTH    = 2
	PADDLE_HEIGHT   = 4
	TERM_WIDTH_MIN  = 30
	TERM_HEIGHT_MIN = 15
)

type Status struct {
	width      int
	height     int
	top        int
	bottom     int
	packetData string
	enemy      PongObject
	user       PongObject
}

var status = Status{}

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
	for i, r := range []rune(str) {
		termbox.SetCell(x+i, y, r, termbox.ColorDefault, termbox.ColorDefault)
	}
}

func (o *PongObject) draw() {
	runes := []rune(o.Str())
	size := len(runes)
	x := o.Point().X
	y := o.Point().Y
	width := o.Size().Width
	height := o.Size().Height
	for h := 0; h < height; h++ {
		for w := 0; w < width; w++ {
			ch := runes[(h*width+w)%size]
			termbox.SetCell(x+w, y+h, ch, termbox.ColorDefault, termbox.ColorDefault)
		}
	}
}

func (o *BallObject) draw() {
	shadows := o.Shadows()
	for i := len(shadows) - 1; 0 <= i; i -= 1 {
		shadow := shadows[i]
		termbox.SetCell(shadow.Point().X, shadow.Point().Y, shadow.Char(),
			termbox.ColorDefault, termbox.ColorDefault)
	}
}

func (o *BallObject) reflectPaddle(paddle *PongObject) {
	ballY := int(o.pointF.Y - o.vectorF.Dy)

	dx := -o.vectorF.Dx
	var dy float32
	switch ballY - paddle.Point().Y {
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
		ballX = paddle.Point().X - 1
	} else {
		ballX = paddle.Point().X + paddle.Size().Width
	}
	o.Set(ballX, ballY, dx, dy)
}

func (o *BallObject) reflectWall(wall *PongObject) {
	point := o.shadows[0].point
	if o.vectorF.Dy < 0 {
		o.Set(point.X, wall.Point().Y+wall.Size().Height, o.vectorF.Dx, -o.vectorF.Dy)
	} else {
		o.Set(point.X, wall.Point().Y-1, o.vectorF.Dx, -o.vectorF.Dy)
	}
}

func newBall() BallObject {
	y := rand.Intn(status.height/3) + status.height/3
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
	return NewBallObject(6, y, 1, dy, status.packetData)
}

func moveEnemy(ball *BallObject) {
	ballDx := ball.VectorF().Dx
	ballPoint := ball.Point()

	enemyY := status.enemy.Point().Y + status.enemy.Size().Height/2

	var bestY int
	if ballDx < 0 && ballPoint.X < status.width/2 {
		bestY = ballPoint.Y
	} else if rand.Intn(2) == 1 {
		ballY := ballPoint.Y
		selfY := status.user.Point().Y + status.user.Size().Height/2
		bestY = (selfY + ballY) / 2
	} else {
		return
	}

	if bestY < enemyY && status.top+1 < status.enemy.Point().Y {
		status.enemy.Move(0, -1)
	} else if enemyY < bestY && status.enemy.Point().Y+status.enemy.Size().Height < status.bottom {
		status.enemy.Move(0, 1)
	}
}

func play(kch chan termbox.Key, tch chan bool, seq int) (*PongResult, bool) {

	message := fmt.Sprintf("start icmp_seq=%d", seq)
	messageLabel := NewPongObject((status.width-len(message))/2, status.height/2, len(message), 1, message)
	topWall := NewPongObject(0, status.top, status.width, 1, "=")
	bottomWall := NewPongObject(0, status.bottom, status.width, 1, "=")
	leftWall := NewPongObject(0, status.top+1, 1, status.bottom-status.top-1, "|")
	localhostLabel := NewPongObject(0, (status.height-11)/2, 1, 11, " localhost ")
	ball := newBall()

	ballWaitMax := BALL_WAIT_INIT
	ballWaitTimes := ballWaitMax
	enemyWaitTimes := ENEMY_WAIT_MAX
	msgWaitTimes := MESSAGE_WAIT

	ttl := opts.TimeToLive
	startTiem := time.Now()

	for {
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
		topWall.draw()
		bottomWall.draw()
		leftWall.draw()
		localhostLabel.draw()
		status.enemy.draw()
		status.user.draw()
		if msgWaitTimes == 0 {
			ball.draw()
		} else {
			messageLabel.draw()
		}

		drawString(status.width-20, 0, fmt.Sprintf("icmp_seq=%d ttl=%d ", seq, ttl))
		termbox.Flush()

		select {
		case key := <-kch:
			switch key {
			case termbox.KeyEsc, termbox.KeyCtrlC:
				return nil, true // game end
			case termbox.KeyArrowUp: // UP
				if status.top+1 < status.user.Point().Y {
					status.user.Move(0, -1)
				}
			case termbox.KeyArrowDown: // DOWN
				if status.user.Point().Y+status.user.Size().Height < status.bottom {
					status.user.Move(0, 1)
				}
			}
		case <-tch:

			if 0 < msgWaitTimes {
				msgWaitTimes -= 1
				continue
			}

			enemyWaitTimes--
			if enemyWaitTimes < 0 {
				enemyWaitTimes = ENEMY_WAIT_MAX
				moveEnemy(&ball)
			}

			ballWaitTimes--
			if ballWaitTimes < 0 {
				ballWaitTimes = ballWaitMax
				ball.Next()

				if topWall.Collision(ball.Point()) {
					ball.reflectWall(&topWall)
				}
				if bottomWall.Collision(ball.Point()) {
					ball.reflectWall(&bottomWall)
				}
				if status.user.Collision(ball.Point()) {
					ball.reflectPaddle(&status.user)
					if 1 < ballWaitMax && rand.Intn(2) == 1 {
						// speed up
						ballWaitMax -= 1
					}
				}
				if status.enemy.Collision(ball.Point()) {
					ball.reflectPaddle(&status.enemy)
					ttl -= 1
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
				if ball.Point().X < 1 {
					// win
					time := int(time.Since(startTiem) / time.Second)
					return &PongResult{true, ttl, time}, false
				}
				if status.width-1 <= ball.Point().X {
					// lose
					return &PongResult{}, false
				}
			}
		}
	}
}

func game() ([]PongResult, bool, error) {
	err := termbox.Init()
	if err != nil {
		return nil, false, err
	}
	defer termbox.Close()
	termbox.HideCursor()

	status.width, status.height = termbox.Size()
	if status.width < TERM_WIDTH_MIN || status.height < TERM_HEIGHT_MIN {
		return nil, false, fmt.Errorf("This term(%dx%d) is too narrow. Requires %dx%d area",
			status.width, status.height, TERM_WIDTH_MIN, TERM_HEIGHT_MIN)
	}
	status.top = 1
	status.bottom = status.height - 2

	status.enemy = NewPongObject(3, status.height/2-PADDLE_HEIGHT/2, PADDLE_WIDTH, PADDLE_HEIGHT, "||G|W|||")
	status.user = NewPongObject(status.width-4, status.height/2-PADDLE_HEIGHT/2, PADDLE_WIDTH, PADDLE_HEIGHT, "|")

	results := make([]PongResult, 0, opts.Count)

	kch := make(chan termbox.Key)
	tch := make(chan bool)
	go keyEventLoop(kch)
	go timerEventLoop(tch)

	for i := 0; i < opts.Count; i += 1 {
		seq := i + 1
		result, isEnd := play(kch, tch, seq)
		if isEnd {
			return results, true, nil
		}
		results = append(results, *result)
	}

	// termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	return results, false, nil
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
		status.packetData = PACKET_HEADER + ":" + opts.Padding
	} else {
		status.packetData = PACKET_HEADER
	}
	size := len(status.packetData)

	startTiem := time.Now()

	// start pong
	results, isBrake, err := game()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	time := time.Since(startTiem) / time.Second

	totalPackets := len(results)
	if totalPackets == 0 {
		if isBrake {
			fmt.Println("^C")
		}
		return
	}

	// show results
	fmt.Printf("PONG %s(%s) %d bytes of data.\n", opts.Args.Destination, addr, size)
	for i, r := range results {
		if r.received {
			fmt.Printf("%d bytes from %s(%s): icmp_seq=%d ttl=%d time=%d sec\n",
				size, opts.Args.Destination, addr,
				i+1, r.ttl, r.time)
		} else {
			fmt.Printf("%d bytes from %s(%s): request timed out\n",
				size, opts.Args.Destination, addr)
		}
	}
	if isBrake {
		fmt.Println("^C")
	}
	fmt.Printf("--- %s pong statistics ---\n", opts.Args.Destination)
	nRecv := 0
	for _, r := range results {
		if r.received {
			nRecv++
		}
	}
	lossRatio := 100 - nRecv*100/totalPackets
	fmt.Printf("%d packets transmitted, %d received, %d%% packet loss, time %d sec\n",
		totalPackets, nRecv, lossRatio, time)
}
