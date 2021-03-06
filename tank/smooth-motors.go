package tank

import (
	"flag"
	"math"
	"sync"
	"time"

	"github.com/antongulenko/golib"
)

type Motor interface {
	SetSpeed(val float32)
	GetSpeed() float32
}

type SmoothMotor struct {
	target  float32
	current float32
	tank    *SmoothTank
}

func (m *SmoothMotor) SetSpeed(val float32) {
	m.target = val
	m.tank.notifyChangedPosition()
}

func (m *SmoothMotor) GetSpeed() float32 {
	return m.current
}

type SmoothTank struct {
	Tank
	SleepTime      time.Duration
	AccelSlopeTime time.Duration
	DecelSlopeTime time.Duration
	MinSpeed       float64

	left  SmoothMotor
	right SmoothMotor

	adjustCond *sync.Cond
	stopFlag   bool
}

func (a *SmoothTank) RegisterFlags() {
	a.Tank.RegisterFlags()
	flag.Float64Var(&a.MinSpeed, "minSpeed", a.MinSpeed, "Minimum speed for all motors and directions")
	flag.DurationVar(&a.SleepTime, "adjustSleep", a.SleepTime, "Time to sleep between motor adjustments")
	flag.DurationVar(&a.AccelSlopeTime, "accelSlopeTime", a.AccelSlopeTime, "Maximum time for a motor to ramp up between 0% and 100%")
	flag.DurationVar(&a.DecelSlopeTime, "decelSlopeTime", a.DecelSlopeTime, "Maximum time for a motor to ramp down between 100% and 0%")
}

func (a *SmoothTank) Setup() error {
	a.adjustCond = sync.NewCond(new(sync.Mutex))
	a.left.tank = a
	a.right.tank = a
	if err := a.Tank.Setup(); err != nil {
		return err
	}
	if err := a.Tank.InitI2cPeripherals(); err != nil {
		return err
	}
	go a.adjustSpeedLoop()
	return nil
}

func (a *SmoothTank) Cleanup() {
	a.adjustCond.L.Lock()
	defer a.adjustCond.L.Unlock()
	a.Tank.Cleanup()
	a.left.current = 0
	a.left.target = 0
	a.right.current = 0
	a.right.target = 0
	a.stopFlag = true
	a.adjustCond.Broadcast()
}

func (a *SmoothTank) Left() Motor {
	return &a.left
}

func (a *SmoothTank) Right() Motor {
	return &a.right
}

func (a *SmoothTank) adjustSpeedLoop() {
	accelStep := float32(math.MaxFloat32)
	decelStep := float32(math.MaxFloat32)
	if a.AccelSlopeTime > 0 {
		accelStep = float32(a.SleepTime) / float32(a.AccelSlopeTime)
	}
	if a.DecelSlopeTime > 0 {
		decelStep = float32(a.SleepTime) / float32(a.DecelSlopeTime)
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
			golib.Printerr(a.Motors.Set(leftPos, rightPos))
			time.Sleep(a.SleepTime)
		}
	}
}

func (a *SmoothTank) adjustSpeed(m *SmoothMotor, accelStep, decelStep float32) {
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

func (a *SmoothTank) calcSpeed(val float32) float64 {
	min := float32(a.MinSpeed)
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

func (a *SmoothTank) notifyChangedPosition() {
	a.adjustCond.L.Lock()
	defer a.adjustCond.L.Unlock()
	a.adjustCond.Broadcast()
}
