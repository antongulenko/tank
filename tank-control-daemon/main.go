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
	controller := tankController{
		joystickIndex:           1,
		joystickRetryDuration:   2 * time.Second,
		toggleControlModeButton: 1,
		ledSequenceButton:       2,
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
			Axis: leftAxis,
		},
		LedAxis: JoystickAxisOneDimension{
			JoystickAxis: JoystickAxis{
				AxisNumber:       5,
				ZeroFrom:         0,
				ZeroTo:           0,
				ScaleZeroFromTo:  true,
				SingleInvertFlag: true,
			},
		},
		ledSequence:           tank.DefaultLedSequence,
		startupSequenceRounds: 2,
		ledControlLoopSleep:   100 * time.Millisecond,
		heartbeatStep:         0.001,
	}
	controller.SingleStick.Axis.SingleInvertFlag = false

	controller.registerFlags()
	golib.RegisterFlags(golib.FlagsAll)
	flag.Parse()
	golib.ConfigureLogging()

	controller.ledSequence.NumLeds = controller.tank.Leds.NumLeds
	red, green, yellow := controller.tank.Leds.Groups()
	controller.heartbeatLeds = controller.tank.Leds.Group(green.To, green.To)
	green.To-- // Reserve one LED for the heartbeat
	controller.batteryLeds = red
	controller.speedLeds = yellow
	controller.manualLeds = green

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
	ledSequenceButton       int
	useSingleStick          bool
	joystickRetryDuration   time.Duration

	tank tank.SmoothTank

	Direct      DirectMotorController
	SingleStick OneStickMotorController

	LedAxis               JoystickAxisOneDimension
	startupSequenceRounds int
	ledSequence           tank.LedSequence
	ledControlLoopSleep   time.Duration
	sequenceRunning       bool
	ledControlTime        uint64
	heartbeatStep         float64

	batteryLeds   tank.LedGroup
	speedLeds     tank.LedGroup
	manualLeds    tank.LedGroup
	heartbeatLeds tank.LedGroup
}

func (c *tankController) registerFlags() {
	c.LedAxis.RegisterFlags("leds", "axis for led control")
	c.Direct.RegisterFlags()
	c.SingleStick.RegisterFlags()
	c.tank.RegisterFlags()
	flag.IntVar(&c.startupSequenceRounds, "startup-sequence", c.startupSequenceRounds, "Number of startup sequence rounds (can be disabled)")
	flag.IntVar(&c.joystickIndex, "js", c.joystickIndex, "Joystick device index")
	flag.DurationVar(&c.joystickRetryDuration, "js-retry", c.joystickRetryDuration, "Time to retry joystick initialization")
	flag.IntVar(&c.ledSequenceButton, "led-sequence-button", c.ledSequenceButton, "Joystick Button index to manually trigger LED sequence")
	flag.IntVar(&c.toggleControlModeButton, "toggleControlModeButton", c.toggleControlModeButton, "Joystick Button index that toggles between one-stick and two-stick control")
	flag.BoolVar(&c.useSingleStick, "singleStick", c.useSingleStick, "Use single stick for controlling motors")
	flag.DurationVar(&c.ledControlLoopSleep, "led-control-sleep", c.ledControlLoopSleep, "Sleep time in LED control loop (displaying motor speed and battery voltage)")
	flag.Float64Var(&c.heartbeatStep, "heartbeat-step", c.heartbeatStep, "Heartbeat progress per LED control loop step")
}

func (c *tankController) run() {
	// Initialize USB/I2C peripherals
	golib.Checkerr(c.tank.Setup())

	go c.waitAndInitJoysticks()

	// Run startup sequence
	if c.startupSequenceRounds > 0 {
		log.Println("Initialization done, running LED startup sequence...")
		if err := c.runLedSequence(c.startupSequenceRounds); err != nil {
			log.Errorf("Startup LED sequence failed: %v", err)
		}
	}
	if err := c.manualLeds.Set(0); err != nil {
		log.Errorf("Failed to disable manual LEDs: %v", err)
	}

	// Display motor speed, battery voltage. Does not return.
	c.ledControlLoop()
}

func (c *tankController) waitAndInitJoysticks() {
	// Wait until Joysticks can be initialized successfully
	var js *joysticks.HID
	var err error
	for {
		if js, err = c.setupJoysticks(); err != nil {
			log.Errorf("Failed to setup Joysticks: %v. Retrying in %v...", err, c.joystickRetryDuration)
			time.Sleep(c.joystickRetryDuration)
		} else {
			log.Printf("Opened joystick device index %v (%v buttons, %v axes, %v events)", c.joystickIndex, len(js.Buttons), len(js.HatAxes), len(js.Events))
			break
		}
	}

	// Setup motor control
	c.useSingleStick = !c.useSingleStick // Make sure the first toggle initializes the wanted controller
	c.toggleMotorController(js)

	// Start receiving joystick events
	js.ParcelOutEvents() // Does not return
}

func (c *tankController) setupJoysticks() (*joysticks.HID, error) {
	js := joysticks.Connect(c.joystickIndex)
	if js == nil {
		return nil, fmt.Errorf("Failed to open joystick with index %v, retrying in %v...", c.joystickIndex, c.joystickRetryDuration)
	}

	controlButton := uint8(c.toggleControlModeButton)
	if js.ButtonExists(controlButton) {
		toggleMode := js.OnLong(controlButton)
		go func() {
			for range toggleMode {
				c.toggleMotorController(js)
			}
		}()
	} else {
		return nil, fmt.Errorf("Button for toggling control mode (index %v) does not exist on joystick", controlButton)
	}
	sequenceButton := uint8(c.ledSequenceButton)
	if js.ButtonExists(sequenceButton) {
		runSequence := js.OnButton(sequenceButton)
		go func() {
			for range runSequence {
				if err := c.runLedSequence(1); err != nil {
					log.Errorf("Triggered LED sequence failed: %v")
				}
			}
		}()
	} else {
		return nil, fmt.Errorf("Button for manually triggering LED sequence (index %v) does not exist on joystick", sequenceButton)
	}
	c.LedAxis.Notify(js, func(val float32) {
		if !c.sequenceRunning {
			log.Println("Led axis value:", val)
			golib.Printerr(c.manualLeds.Set(float64(val)))
		}
	})
	return js, nil
}

func (c *tankController) stop() {
	c.tank.Cleanup()
}

func (c *tankController) toggleMotorController(js *joysticks.HID) {
	c.useSingleStick = !c.useSingleStick
	if c.useSingleStick {
		log.Println("Setting control mode to SINGLE stick")
		c.Direct.Enabled = false
		c.SingleStick.Enabled = true
		c.SingleStick.Setup(js, c.tank.Left(), c.tank.Right())
	} else {
		log.Println("Setting control mode to DOUBLE stick")
		c.Direct.Enabled = true
		c.SingleStick.Enabled = false
		c.Direct.Setup(js, c.tank.Left(), c.tank.Right())
	}
}

func (c *tankController) runLedSequence(numRounds int) error {
	c.sequenceRunning = true
	defer func() {
		c.sequenceRunning = false
	}()
	return c.ledSequence.Run(numRounds, func(sleepTime time.Duration, values []float64) (err error) {
		err = c.tank.Leds.SetAll(values)
		if err == nil {
			time.Sleep(sleepTime)
		}
		return
	})
}

func (c *tankController) ledControlLoop() {
	for {
		if !c.sequenceRunning {
			// Display battery
			batt, err := c.tank.Adc.GetBatteryPercentage()
			if err != nil {
				log.Errorln("Error querying battery voltage:", err)
			} else {
				if err := c.batteryLeds.Set(batt); err != nil {
					log.Errorln("Error displaying battery voltage:", err)
				}
			}

			// Display speed
			left := math.Abs(float64(c.tank.Left().GetSpeed()))
			right := math.Abs(float64(c.tank.Right().GetSpeed()))
			avgSpeed := (left + right) / 2
			if err := c.speedLeds.Set(avgSpeed); err != nil {
				log.Errorln("Error displaying motor speed:", err)
			}

			// Progress heartbeat
			heartbeatVal := math.Sin(float64(c.ledControlTime) * c.heartbeatStep * math.Pi)
			heartbeatVal = (heartbeatVal + 1) / 2
			if err := c.heartbeatLeds.Set(heartbeatVal); err != nil {
				log.Errorln("Error displaying heartbeat:", err)
			}

			c.ledControlTime++
			log.Debugf("Battery: %v, avg speed: %v, heartbeat: %v, control time: %v", batt, avgSpeed, heartbeatVal, c.ledControlTime)
		}
		time.Sleep(c.ledControlLoopSleep)
	}
}
