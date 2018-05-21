package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

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
)

type motor struct {
	name   string
	axis   int
	useY   bool
	invert bool

	// Positions between these values are bound to zero
	zeroFrom, zeroTo float64

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
	left.registerFlags()
	right.registerFlags()
	t.RegisterFlags()
	golib.RegisterFlags(golib.FlagsAll)
	flag.Parse()
	golib.ConfigureLogging()

	// "Clean" shutdown with Ctrl-C signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
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
	if val == m.position {
		return
	}
	m.position = val

	golib.Printerr(t.Motors.Set(float64(left.position*100), float64(right.position*100)))
}
