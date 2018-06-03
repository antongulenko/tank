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
	controller := tankController{
		joystickIndex:           1,
		toggleControlModeButton: 1,
		useSingleStick:          false,
		tank: tank.SmoothTank{
			Tank:           tank.DefaultTank,
			SleepTime:      50 * time.Millisecond,
			AccelSlopeTime: 400 * time.Millisecond,
			DecelSlopeTime: 300 * time.Millisecond,
		},
		Direct: DirectMotorController{
			LeftAxis: JoystickAxisOneDimension{
				JoystickAxis: leftAxis,
				UseY:         true,
			},
			RightAxis: JoystickAxisOneDimension{
				JoystickAxis: rightAxis,
				UseY:         false,
			},
		},
		SingleStick: OneStickMotorController{
			AxisX: JoystickAxisOneDimension{
				JoystickAxis: leftAxis,
				UseY:         false,
			},
			AxisY: JoystickAxisOneDimension{
				JoystickAxis: leftAxis,
				UseY:         true,
			},
		},
	}

	controller.registerFlags()
	golib.RegisterFlags(golib.FlagsAll)
	flag.Parse()
	golib.ConfigureLogging()

	// "Clean" shutdown with Ctrl-C signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(controller.stop)
	}
	defer cleanup()
	go func() {
		fmt.Println("Received signal", <-c)
		cleanup()
		os.Exit(0)
	}()

	controller.run() // Does not return
}

type tankController struct {
	joystickIndex           int
	toggleControlModeButton int
	useSingleStick          bool

	tank tank.SmoothTank

	Direct      DirectMotorController
	SingleStick OneStickMotorController
}

func (c *tankController) registerFlags() {
	c.Direct.RegisterFlags()
	c.SingleStick.RegisterFlags()
	c.tank.RegisterFlags()
	flag.IntVar(&c.joystickIndex, "js", c.joystickIndex, "Joystick device index")
	flag.IntVar(&c.toggleControlModeButton, "toggleControlModeButton", c.toggleControlModeButton, "Joystick Button index that toggles between one-stick and two-stick control")
	flag.BoolVar(&c.useSingleStick, "singleStick", c.useSingleStick, "Use single stick for controlling motors")
}

func (c *tankController) run() {
	// Initialize USB/I2C peripherals
	golib.Checkerr(c.tank.Setup())

	// Connect Joystick
	js := joysticks.Connect(c.joystickIndex)
	if js == nil {
		log.Fatalln("Failed to open joystick with index", c.joystickIndex)
	}
	log.Printf("Opened device index %v (%v buttons, %v axes, %v events)",
		c.joystickIndex, len(js.Buttons), len(js.HatAxes), len(js.Events))

	// Setup joystick controls
	controlButton := uint8(c.toggleControlModeButton)
	if js.ButtonExists(controlButton) {
		toggleMode := js.OnLong(controlButton)
		go func() {
			for range toggleMode {
				c.toggleMotorController(js)
			}
		}()
	} else {
		log.Fatalf("Button for toggling control mode (index %v) does not exist on joystick", controlButton)
	}

	// Setup motor control
	c.SingleStick.Setup(js, c.tank.Left(), c.tank.Right())
	c.Direct.Setup(js, c.tank.Left(), c.tank.Right())
	c.useSingleStick = !c.useSingleStick // Make sure the first toggle initializes the wanted controller
	c.toggleMotorController(js)

	// Start receiving joystick events
	js.ParcelOutEvents() // Does not return
}

func (c *tankController) stop() {
	c.tank.Stop()
}

func (c *tankController) toggleMotorController(js *joysticks.HID) {
	c.useSingleStick = !c.useSingleStick
	if c.useSingleStick {
		log.Println("Setting control mode to SINGLE stick")
		c.Direct.Enabled = false
		c.SingleStick.Enabled = true
	} else {
		log.Println("Setting control mode to DOUBLE stick")
		c.Direct.Enabled = true
		c.SingleStick.Enabled = false
	}
}
