package tank

import (
	"math"

	"github.com/antongulenko/tank/ft260"
	"github.com/antongulenko/tank/pca9685"
	log "github.com/sirupsen/logrus"
)

const (
	numPwmOutputs = 4
	bytePerOutput = 4
)

type mainMotors struct {
	usb *ft260.Ft260

	I2cAddr byte

	InvertRightDir, InvertLeftDir bool

	// Starting from the first PWM output, the order of outputs must be:
	// left motor, right motor, left direction, right direction
	PwmStart byte // pca9685.LED0

	optimizeUpdate bool
	currentState   [numPwmOutputs]float64
}

func (m *mainMotors) Init() error {
	log.Printf("Initializing PWM driver at %02x...", m.I2cAddr)
	return m.usb.I2cWrite(m.I2cAddr, pca9685.MODE1, pca9685.MODE1_ALLCALL|pca9685.MODE1_AI)
}

func (m *mainMotors) ForceSet(left, right float64) error {
	m.optimizeUpdate = false
	return m.Set(left, right)
}

// Input values in -100..100
func (m *mainMotors) Set(left, right float64) error {
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
	newState := [numPwmOutputs]float64{
		leftSpeed, rightSpeed, dirToFloat(leftDir), dirToFloat(rightDir),
	}

	// Compute smallest possible range of values to be updated
	updateFrom := 0
	updateTo := numPwmOutputs
	if m.optimizeUpdate {
		for i := range newState {
			if m.currentState[i] == newState[i] {
				updateFrom++
			} else {
				break
			}
		}
		for i := range newState {
			if m.currentState[numPwmOutputs-1-i] == newState[numPwmOutputs-1-i] {
				updateTo--
			} else {
				break
			}
		}
		if updateFrom >= updateTo {
			// The desired state is already deployed
			return nil
		}
	}
	m.currentState = newState
	m.optimizeUpdate = true
	numChanges := updateTo - updateFrom

	// Compute raw bytes to be sent to the device
	pwmValues := make([]byte, bytePerOutput*numChanges)
	for i, val := range newState[updateFrom:updateTo] {
		pca9685.ValuesInto(val, pwmValues[bytePerOutput*i:])
	}
	dirToText := func(dir bool) string {
		if dir {
			return "forward"
		} else {
			return "backward"
		}
	}
	log.Printf("Setting motors to %.2f%% (%v) and %.2f%% (%v) (Updating %v PWM value(s) starting at %v)",
		leftSpeed*100, dirToText(leftDir), rightSpeed*100, dirToText(rightDir), updateTo-updateFrom, updateFrom)
	pwmValues = append([]byte{m.PwmStart + byte(updateFrom)*bytePerOutput}, pwmValues...)

	// Send the changed values to the device
	return m.usb.I2cWrite(m.I2cAddr, pwmValues...)
}
