package main

import (
	"flag"
	"fmt"
	"sync"

	"github.com/splace/joysticks"
)

type JoystickAxis struct {
	AxisNumber int

	// Positions between these values are bound to zero
	// TODO individual values for x/y axes
	ZeroFrom, ZeroTo float64

	SingleInvertFlag bool
	InvertX, InvertY bool

	// If true, scale the value range to adjust for zeroFrom/zeroTo and make the entire value range -1..1 available
	ScaleZeroFromTo bool

	currentHook    func(x, y float32)
	notifyLoopOnce sync.Once
}

func (m *JoystickAxis) RegisterFlags(prefix string, desc string) {
	flag.IntVar(&m.AxisNumber, prefix, m.AxisNumber, "Index for joystick axis for "+desc)
	flag.Float64Var(&m.ZeroFrom, prefix+"ZeroFrom", m.ZeroFrom, "Start of the zero interval of "+desc)
	flag.Float64Var(&m.ZeroTo, prefix+"ZeroTo", m.ZeroTo, "End of the zero interval of "+desc)
	flag.BoolVar(&m.ScaleZeroFromTo, prefix+"ScaleZeroFromTo", m.ScaleZeroFromTo, "Can be used to disable the value range adjustment after filtering based on zeroFrom/zeroTo for "+desc)

	if m.SingleInvertFlag {
		flag.BoolVar(&m.InvertX, prefix+"Invert", m.InvertX, "Invert axis direction of "+desc)
	} else {
		flag.BoolVar(&m.InvertX, prefix+"InvertX", m.InvertX, "Invert X axis direction of "+desc)
		flag.BoolVar(&m.InvertY, prefix+"InvertY", m.InvertY, "Invert Y axis direction of "+desc)
	}
}

func (a *JoystickAxis) Notify(js *joysticks.HID, newHook func(x, y float32)) {
	if !js.HatExists(uint8(a.AxisNumber)) {
		panic(fmt.Sprintf("Joystick axis (%v) does not exist on device %v", a.AxisNumber, js))
	}
	a.currentHook = newHook
	a.notifyLoopOnce.Do(func() {
		moved := js.OnMove(uint8(a.AxisNumber))
		go func() {
			for event := range moved {
				coords := event.(joysticks.CoordsEvent)
				x, y := coords.X, coords.Y
				if a.InvertX {
					x = -x
				}
				if a.InvertY || (a.SingleInvertFlag && a.InvertX) {
					y = -y
				}
				x, y = a.convert(x), a.convert(y)
				if hook := a.currentHook; hook != nil {
					hook(x, y)
				}
			}
		}()
	})
}

func (a *JoystickAxis) convert(val float32) float32 {
	zeroFrom := float32(a.ZeroFrom)
	zeroTo := float32(a.ZeroTo)
	if val >= zeroFrom && val <= zeroTo {
		val = 0
	} else if a.ScaleZeroFromTo {
		// Scale the value range from [-1..zeroFrom] and [zeroTo..0] to [-1..0] and [0..1]
		if val > 0 {
			val = (val - zeroTo) / (1 - zeroTo)
		} else if val < 0 {
			val = (zeroFrom - val) / (-1 - zeroFrom)
		}
	}
	return val
}

type JoystickAxisOneDimension struct {
	JoystickAxis
	UseY bool
}

func (m *JoystickAxisOneDimension) RegisterFlags(prefix string, desc string) {
	m.JoystickAxis.RegisterFlags(prefix, desc)
	flag.BoolVar(&m.UseY, prefix+"Y", m.UseY, "Use Y instead of X axis for "+desc)
}

func (a *JoystickAxisOneDimension) Notify(js *joysticks.HID, hook func(val float32)) {
	a.JoystickAxis.Notify(js, func(x, y float32) {
		val := x
		if a.UseY {
			val = y
		}
		hook(val)
	})
}
