package main

import (
	"flag"
	"fmt"
	"math"
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

func main() {
	leftAxis := JoystickAxis{
		AxisNumber:       1,
		ZeroFrom:         -0.2,
		ZeroTo:           0.1,
		ScaleZeroFromTo:  true,
		InvertX:          true,
		SingleInvertFlag: true,
	}
	rightAxis := leftAxis
	rightAxis.AxisNumber = 3
	directController := DirectMotorController{
		LeftAxis: JoystickAxisOneDimension{
			JoystickAxis: leftAxis,
			UseY:         true,
		},
		RightAxis: JoystickAxisOneDimension{
			JoystickAxis: rightAxis,
			UseY:         false,
		},
	}
	singleStickController := OneStickMotorController{
		Axis: leftAxis,
	}
	singleStickController.Axis.SingleInvertFlag = false
	adjuster := speedAdjuster{
		tank:           tank.DefaultTank,
		adjustCond:     sync.NewCond(new(sync.Mutex)),
		sleepTime:      50 * time.Millisecond,
		accelSlopeTime: 400 * time.Millisecond,
		decelSlopeTime: 300 * time.Millisecond,
		dummyMode:      false,
	}

	joystickIndex := flag.Int("js", 1, "Joystick device index")
	useSingleTick := flag.Bool("singleStick", false, "Use single stick for controlling motors")
	adjuster.registerFlags()
	singleStickController.RegisterFlags()
	directController.RegisterFlags()
	golib.RegisterFlags(golib.FlagsAll)
	flag.Parse()
	golib.ConfigureLogging()

	// "Clean" shutdown with Ctrl-C signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			adjuster.stop()
		})
	}
	defer cleanup()
	go func() {
		fmt.Println("Received signal", <-c)
		cleanup()
		os.Exit(0)
	}()

	adjuster.setup()
	js := joysticks.Connect(*joystickIndex)
	if js == nil {
		log.Fatalln("Failed to open joystick with index", *joystickIndex)
	}
	log.Printf("Opened device index %v (%v buttons, %v axes, %v events)",
		*joystickIndex, len(js.Buttons), len(js.HatAxes), len(js.Events))

	go adjuster.adjustSpeedLoop()
	if *useSingleTick {
		singleStickController.Setup(js, &adjuster.left, &adjuster.right)
	} else {
		directController.Setup(js, &adjuster.left, &adjuster.right)
	}
	js.ParcelOutEvents() // Does not return
}

type Motor struct {
	target   float32
	current  float32
	adjuster *speedAdjuster
}

func (m *Motor) SetSpeed(val float32) {
	m.target = val
	m.adjuster.notifyChangedPosition()
}

type speedAdjuster struct {
	tank           tank.Tank
	sleepTime      time.Duration
	accelSlopeTime time.Duration
	decelSlopeTime time.Duration
	dummyMode      bool
	minSpeed       float64

	// Current position
	left  Motor
	right Motor

	adjustCond *sync.Cond
	stopFlag   bool
}

func (a *speedAdjuster) registerFlags() {
	a.tank.RegisterFlags()
	flag.Float64Var(&a.minSpeed, "minSpeed", a.minSpeed, "Minimum speed for all motors and directions")
	flag.DurationVar(&a.sleepTime, "adjustSleep", a.sleepTime, "Time to sleep between motor adjustments")
	flag.DurationVar(&a.accelSlopeTime, "accelSlopeTime", a.accelSlopeTime, "Maximum time for a motor to ramp up between 0% and 100%")
	flag.DurationVar(&a.decelSlopeTime, "decelSlopeTime", a.decelSlopeTime, "Maximum time for a motor to ramp down between 100% and 0%")
	flag.BoolVar(&a.dummyMode, "dummy", a.dummyMode, "Dummy mode: do not use USB/I2C peripherals, just print motor values")
}

func (a *speedAdjuster) adjustSpeedLoop() {
	accelStep := float32(math.MaxFloat32)
	decelStep := float32(math.MaxFloat32)
	if a.accelSlopeTime > 0 {
		accelStep = float32(a.sleepTime) / float32(a.accelSlopeTime)
	}
	if a.decelSlopeTime > 0 {
		decelStep = float32(a.sleepTime) / float32(a.decelSlopeTime)
	}
	for !a.stopFlag {
		// Wait for incorrect position of any motor
		a.adjustCond.L.Lock()
		for a.left.target == a.left.current && a.right.target == a.right.current && !a.stopFlag {
			a.adjustCond.Wait()
		}
		a.adjustCond.L.Unlock()
		if !a.stopFlag {
			a.adjustSpeed(&a.left, accelStep, decelStep)
			a.adjustSpeed(&a.right, accelStep, decelStep)
			leftPos := a.calcSpeed(a.left.current)
			rightPos := a.calcSpeed(a.right.current)
			if !a.dummyMode {
				golib.Printerr(a.tank.Motors.Set(leftPos, rightPos))
			} else {
				log.Printf("Setting motors to left: %v right: %v", leftPos, rightPos)
			}
			time.Sleep(a.sleepTime)
		}
	}
}

func (a *speedAdjuster) adjustSpeed(m *Motor, accelStep, decelStep float32) {
	cur := m.current
	forward := cur > 0           // Currently driving forward
	increasing := m.target > cur // Target momentum is more forward-oriented than currently

	adjustStep := decelStep
	if forward == increasing {
		adjustStep = accelStep
	}
	if !increasing {
		adjustStep = -adjustStep
	}

	if math.Abs(float64(cur-m.target)) <= math.Abs(float64(adjustStep)) {
		m.current = m.target
	} else {
		m.current = cur + adjustStep
	}
}

func (a *speedAdjuster) calcSpeed(val float32) float64 {
	min := float32(a.minSpeed)
	if min > 0 && min < 1 && val != 0 {
		if val < 0 {
			val = -min + (1-min)*val
		} else {
			val = min + (1-min)*val
		}
	}
	if val == -0 {
		return 0
	}
	return float64(val * 100)
}

func (a *speedAdjuster) setup() {
	a.left.adjuster = a
	a.right.adjuster = a
	if a.dummyMode {
		log.Println("Dummy mode: not connecting USB/I2C peripherals")
	} else {
		golib.Checkerr(a.tank.Setup())
		golib.Checkerr(a.tank.Motors.Init())
		log.Println("Successfully initialized USB/I2C peripherals, now connecting joystick...")
	}
}

func (a *speedAdjuster) stop() {
	a.adjustCond.L.Lock()
	defer a.adjustCond.L.Unlock()
	if !a.dummyMode {
		a.tank.Motors.Set(0, 0)
		a.tank.Cleanup()
	}
	a.left.current = 0
	a.left.target = 0
	a.right.current = 0
	a.right.target = 0
	a.stopFlag = true
	a.adjustCond.Broadcast()
}

func (a *speedAdjuster) notifyChangedPosition() {
	a.adjustCond.L.Lock()
	defer a.adjustCond.L.Unlock()
	a.adjustCond.Broadcast()
}
