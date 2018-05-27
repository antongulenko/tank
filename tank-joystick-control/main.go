package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/antongulenko/golib"
	"github.com/antongulenko/tank/tank"
	log "github.com/sirupsen/logrus"
	"github.com/splace/joysticks"
)

var (
	t    = tank.DefaultTank
	left = motor{
		name:     "left",
		axis:     1,
		useY:     true,
		zeroFrom: -0.15,
		zeroTo:   0.1,
		invert:   true,
	}
	right = motor{
		name:     "right",
		axis:     3,
		zeroFrom: -0.15,
		zeroTo:   0.1,
		invert:   true,
	}
	adjuster = speedAdjuster{
		adjustCond:   sync.NewCond(new(sync.Mutex)),
		sleepTime:    50 * time.Millisecond,
		maxSlopeTime: 500 * time.Millisecond,
	}
)

type motor struct {
	name   string
	axis   int
	useY   bool
	invert bool

	// Positions between these values are bound to zero
	zeroFrom, zeroTo float64

	// The smallest speed will be translated to this, speed scaled linearly between this and 100%
	minSpeed float64

	position float32
}

func (m *motor) registerFlags() {
	flag.IntVar(&m.axis, m.name, m.axis, "Index for "+m.name+" motor")
	flag.BoolVar(&m.useY, m.name+"Y", m.useY, "Use Y instead of X axis for "+m.name+" motor")
	flag.BoolVar(&m.invert, m.name+"Invert", m.invert, "Invert "+m.name+" motor direction")
	flag.Float64Var(&m.zeroFrom, m.name+"ZeroFrom", m.zeroFrom, "Start of the zero interval of the "+m.name+" motor")
	flag.Float64Var(&m.zeroTo, m.name+"ZeroTo", m.zeroTo, "End of the zero interval of the "+m.name+" motor")
}

func main() {
	var index int
	flag.IntVar(&index, "js", 1, "Joystick device index")
	minSpeed := flag.Float64("minSpeed", 0, "Minimum speed for all motors and directions")
	flag.DurationVar(&adjuster.sleepTime, "adjustSleep", adjuster.sleepTime, "Time to sleep between motor adjustments")
	flag.DurationVar(&adjuster.maxSlopeTime, "maxSlopeTime", adjuster.maxSlopeTime, "Maximum time for a motor to ramp up between 0% and 100%")
	left.registerFlags()
	right.registerFlags()
	t.RegisterFlags()
	golib.RegisterFlags(golib.FlagsAll)
	flag.Parse()
	golib.ConfigureLogging()
	left.minSpeed = *minSpeed
	right.minSpeed = *minSpeed

	// "Clean" shutdown with Ctrl-C signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			// TODO fix synchronization, cleanly stop the adjuster routine
			adjuster.left = 0
			adjuster.right = 0
			t.Motors.Set(0, 0)
			t.Cleanup()
		})
	}
	defer cleanup()
	go func() {
		fmt.Println("Received signal", <-c)
		cleanup()
		os.Exit(0)
	}()

	golib.Checkerr(t.Setup())
	golib.Checkerr(t.Motors.Init())
	log.Println("Successfully initialized USB/I2C peripherals, now connecting joystick...")

	js := joysticks.Connect(index)
	if js == nil {
		log.Fatalln("Failed to open joystick with index", index)
	}
	log.Printf("Opened device index %v (%v buttons, %v axes, %v events)",
		index, len(js.Buttons), len(js.HatAxes), len(js.Events))
	if !js.HatExists(uint8(left.axis)) {
		log.Fatalf("Left motor stick (%v) does not exist on device %v", left.axis, index)
	}
	if !js.HatExists(uint8(right.axis)) {
		log.Fatalf("Right motor stick (%v) does not exist on device %v", right.axis, index)
	}

	go adjuster.adjustSpeedLoop()
	leftMoved := js.OnMove(uint8(left.axis))
	rightMoved := js.OnMove(uint8(right.axis))
	go js.ParcelOutEvents()
	for {
		select {
		case event := <-leftMoved:
			left.handleAxis(event)
		case event := <-rightMoved:
			right.handleAxis(event)
		}
	}
}

type speedAdjuster struct {
	adjustCond   *sync.Cond
	sleepTime    time.Duration
	maxSlopeTime time.Duration

	// Current position
	left  float32
	right float32
}

func (a *speedAdjuster) adjustSpeedLoop() {
	for {
		// Wait for incorrect position of any motor
		a.adjustCond.L.Lock()
		for a.left == left.position && a.right == right.position {
			a.adjustCond.Wait()
		}
		a.adjustCond.L.Unlock()
		a.adjustSpeed(&left, &a.left)
		a.adjustSpeed(&right, &a.right)
		golib.Printerr(t.Motors.Set(float64(left.position*100), float64(right.position*100)))
		time.Sleep(a.sleepTime)
	}
}

func (a *speedAdjuster) adjustSpeed(m *motor, current *float32) {
	adjustStep := float32(a.sleepTime) / float32(a.maxSlopeTime)
	cur := *current
	diff := cur - m.position
	if diff < 0 {
		diff = -diff
	}
	if diff < adjustStep {
		*current = m.position
	} else if m.position < cur {
		*current = cur - adjustStep
	} else if m.position > cur {
		*current = cur + adjustStep
	}
}

func (a *speedAdjuster) notifyChangedPosition() {
	a.adjustCond.L.Lock()
	a.adjustCond.Broadcast()
	a.adjustCond.L.Unlock()
}

func (m *motor) handleAxis(event joysticks.Event) {
	coords := event.(joysticks.CoordsEvent)
	val := coords.X
	if m.useY {
		val = coords.Y
	}
	if m.invert {
		val = -val
	}
	if float64(val) >= m.zeroFrom && float64(val) <= m.zeroTo {
		val = 0
	}
	min := float32(m.minSpeed)
	if min > 0 && min < 1 {
		val = min + (1-min)*val
	}
	m.position = val
	adjuster.notifyChangedPosition()
}
