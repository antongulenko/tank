package tank

import (
	"math"

	"github.com/antongulenko/tank/ft260"
	"github.com/antongulenko/tank/pca9685"
	log "github.com/sirupsen/logrus"
)

type MainMotors struct {
	bus ft260.I2cBus

	I2cAddr byte

	InvertRightDir, InvertLeftDir bool

	// Starting from the first PWM output, the order of outputs must be:
	// left motor, right motor, left direction, right direction
	PwmStart byte // pca9685.LED0

	pwmOutput pca9685.PwmOutput
}

func (m *MainMotors) Init() error {
	log.Printf("Initializing motor PWM driver at %02x...", m.I2cAddr)
	return m.bus.I2cWrite(m.I2cAddr, pca9685.MODE1, pca9685.MODE1_ALLCALL|pca9685.MODE1_AI)
}

func (m *MainMotors) ForceSet(left, right float64) error {
	m.pwmOutput.OptimizeUpdate = false
	return m.Set(left, right)
}

// Input values in -100..100
func (m *MainMotors) Set(left, right float64) error {
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
		leftSpeed, rightSpeed, dirToFloat(leftDir), dirToFloat(rightDir),
	}
	pwmValues := m.pwmOutput.Update(m.PwmStart, newState)

	dirToText := func(dir bool) string {
		if dir {
			return "forward"
		} else {
			return "backward"
		}
	}
	log.Printf("Setting motors to %.2f%% (%v) and %.2f%% (%v) (Sending %v byte to PWM device)",
		leftSpeed*100, dirToText(leftDir), rightSpeed*100, dirToText(rightDir), len(pwmValues))

	return m.bus.I2cWrite(m.I2cAddr, pwmValues...)
}
