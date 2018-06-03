package main

import (
	"github.com/antongulenko/tank/tank"
	"github.com/splace/joysticks"
)

type DirectMotorController struct {
	LeftAxis  JoystickAxisOneDimension
	RightAxis JoystickAxisOneDimension
}

func (c *DirectMotorController) RegisterFlags() {
	c.LeftAxis.RegisterFlags("left", "left motor")
	c.RightAxis.RegisterFlags("right", "right motor")
}

func (m *DirectMotorController) Setup(js *joysticks.HID, left, right tank.Motor) {
	m.LeftAxis.Notify(js, func(val float32) {
		left.SetSpeed(val)
	})
	m.RightAxis.Notify(js, func(val float32) {
		right.SetSpeed(val)
	})
}
