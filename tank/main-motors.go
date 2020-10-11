package tank

import (
	"fmt"
	"math"

	"github.com/antongulenko/tank/ft260"
	"github.com/antongulenko/tank/pca9685"
	log "github.com/sirupsen/logrus"
)

type MainMotors struct {
	bus ft260.I2cBus

	I2cAddr  byte
	Dummy    bool
	SkipInit bool

	InvertRightDir, InvertLeftDir bool

	// Starting from the first PWM output, the order of outputs must be:
	// Left Direction, Left Speed, Right Direction, Right Speed
	PwmStart byte // pca9685.LED0

	pwmOutput pca9685.PwmOutput
}

func (m *MainMotors) Init() error {
	if m.Dummy || m.SkipInit {
		log.Println("Skipping initialization of motors")
		return nil
	} else {
		log.Printf("Initializing motor PWM driver at %#02x...", m.I2cAddr)
		return m.bus.I2cWrite(m.I2cAddr, pca9685.MODE1, pca9685.MODE1_ALLCALL|pca9685.MODE1_AI)
	}
}

func (m *MainMotors) ForceSet(left, right float64) error {
	m.pwmOutput.OptimizeUpdate = false
	return m.Set(left, right)
}

// Input values in -100..100
func (m *MainMotors) Set(left, right float64) error {
	if left < -100 || left > 100 {
		return fmt.Errorf("Illegal left motor %v (must be -100..100)", left)
	}
	if right < -100 || right > 100 {
		return fmt.Errorf("Illegal right motor %v (must be -100..100)", right)
	}

	// Split the two float values into separate speed and direction
	leftSpeed := math.Abs(left) / 100
	rightSpeed := math.Abs(right) / 100
	leftDir := left > 0 != m.InvertLeftDir
	rightDir := right > 0 != m.InvertRightDir

	// Compute the 4 new PWM values
	dirToFloat := func(dir bool) (res float64) {
		if dir {
			res = 1
		}
		return
	}
	newState := []float64{
		dirToFloat(leftDir), leftSpeed, dirToFloat(leftDir), leftSpeed,
	}
	pwmValues := m.pwmOutput.Update(m.PwmStart, newState)

	dummyText := ""
	if m.Dummy {
		dummyText = "dummy "
	}
	dirToText := func(dir bool) string {
		if dir {
			return "forward"
		} else {
			return "backward"
		}
	}
	log.Printf("Setting %vmotors to %.2f%% (%v) and %.2f%% (%v) (Sending %v byte to PWM device)",
		dummyText, leftSpeed*100, dirToText(leftDir), rightSpeed*100, dirToText(rightDir), len(pwmValues))

	if m.Dummy {
		return nil
	} else {
		return m.bus.I2cWrite(m.I2cAddr, pwmValues...)
	}
}
