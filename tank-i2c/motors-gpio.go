package main

import (
	"fmt"
	"math"

	"github.com/antongulenko/tank/mcp23017"
	"github.com/antongulenko/tank/pca9685"
	log "github.com/sirupsen/logrus"
)

const (
	motorStop     = 0
	motorForward  = 1
	motorBackward = 2
)

func setGpioMotors() error {
	gpioAddr := mcp23017.ADDRESS
	speed1 := speedLeft
	speed2 := speedRight

	if speed1 < -100 || speed1 > 100 {
		return fmt.Errorf("Illegal motor speed1 %v (must be -100..100)", speed1)
	}
	if speed2 < -100 || speed2 > 100 {
		return fmt.Errorf("Illegal motor speed2 %v (must be -100..100)", speed2)
	}

	// Initialize both devices first
	if err := initGpio(gpioAddr); err != nil {
		return err
	}
	if err := t.Motors.Init(); err != nil {
		return err
	}

	state1 := getMotorState(speed1)
	state2 := getMotorState(speed2)
	pwm1 := math.Abs(speed1) / 100
	pwm2 := math.Abs(speed2) / 100
	log.Printf("Motor 1: dir %v speed %v. Motor 2: dir %v speed %v.", state1, pwm1, state2, pwm2)

	// Set GPIO direction pins
	gpioByte, err := motorGpioByte(state1, state2)
	if err != nil {
		return err
	}
	gpioByteAddr := mcp23017.GPIO_B_PAIRED
	log.Printf("Setting GPIO port B to %#02x (at addr %#02x)", gpioByte, gpioByteAddr)
	if !debugMotors {
		if err := t.Bus().I2cWrite(gpioAddr, gpioByteAddr, gpioByte); err != nil {
			return err
		}
	}

	pwmAddr := pca9685.ADDRESS

	// Configure PWM signals
	pwmValues := make([]byte, 8)
	pca9685.ValuesInto(pwm1, pwmValues)
	pca9685.ValuesInto(pwm2, pwmValues[4:])
	pwmValues = append([]byte{pca9685.LED0}, pwmValues...)
	log.Printf("Setting PWM values for motor 1 and 2 to %#02x...", pwmValues)
	if !debugMotors {
		if err := t.Bus().I2cWrite(pwmAddr, pwmValues...); err != nil {
			return err
		}
	}
	return nil
}

func getMotorState(speed float64) int {
	if speed < 0 {
		return motorBackward
	} else if speed > 0 {
		return motorForward
	}
	return motorStop
}

func motorGpioByte(state1, state2 int) (byte, error) {
	result := byte(0)
	switch state1 {
	case motorStop:
	case motorForward:
		result |= 1
	case motorBackward:
		result |= 2
	default:
		return 0, fmt.Errorf("Illegal motor state %v", state1)
	}
	switch state2 {
	case motorStop:
	case motorForward:
		result |= 4
	case motorBackward:
		result |= 8
	default:
		return 0, fmt.Errorf("Illegal motor state %v", state2)
	}
	return result, nil
}
