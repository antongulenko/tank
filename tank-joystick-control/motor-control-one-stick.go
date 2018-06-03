package main

import (
	"math"

	"github.com/antongulenko/tank/tank"
	"github.com/splace/joysticks"
)

type OneStickMotorController struct {
	Axis JoystickAxis
}

func (c *OneStickMotorController) RegisterFlags() {
	c.Axis.RegisterFlags("single", "both motors")
}

func (m *OneStickMotorController) Setup(js *joysticks.HID, left, right tank.Motor) {
	m.Axis.Notify(js, func(x, y float32) {
		l, r := convertStickToDirections(float64(x), float64(y))
		left.SetSpeed(float32(l))
		right.SetSpeed(float32(r))
	})
}

func convertStickToDirections(x, y float64) (l, r float64) {
	// Speed/Power determined by distance from the center
	speed := math.Sqrt(x*x + y*y)
	// Safety check: should the values be outside the circle with radius 1, move them closer to the center
	if speed > 1 {
		x /= speed
		y /= speed
		speed = 1
	}

	if x >= 0 {
		if y >= 0 {
			// Turning forward right -> left motor full power
			l = 1
			anglePercent := math.Atan2(y, x) / (math.Pi / 2)
			r = -1 + (2 * anglePercent)
		} else if y < 0 {
			r = -1
			anglePercent := math.Atan2(-y, x) / (math.Pi / 2)
			l = 1 - (2 * anglePercent)
		}
	} else if x < 0 {
		if y >= 0 {
			// Turning forward left -> right motor full power
			r = 1
			anglePercent := math.Atan2(y, -x) / (math.Pi / 2)
			l = -1 + (2 * anglePercent)
		} else if y < 0 {
			l = -1
			anglePercent := math.Atan2(-y, -x) / (math.Pi / 2)
			r = 1 - (2 * anglePercent)
		}
	}

	l *= speed
	r *= speed
	// Should not happen, but avoid crashes
	if l > 1 {
		l = 1
	}
	if r > 1 {
		r = 1
	}
	return
}

func scaleWithin(start, end, percent float32) float32 {
	return start + (end-start)*percent
}

func scaleDownwardsWithin(start, end, percent float32) float32 {
	return end - (end-start)*percent
}
