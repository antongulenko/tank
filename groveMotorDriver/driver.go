// http://wiki.seeed.cc/Grove-I2C_Motor_Driver_V1.3/
package groveMotorDriver

import (
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// Every I2C command contains 3 bytes
	CommandLength = 3

	// Configuration commands
	Command_SetPWMFrequency = 0x84

	// DC motor commands
	Command_SetMotorSpeed = 0x82 // 2 parameters: 2 byte, 0..255 speed for motor A and B
	Command_SetMotorDir   = 0xaa // 1 parameter: 0x0000bbaa, directions for both motors, 'aa' and 'bb' are Dir* values
	Command_SetMotorA     = 0xa1 // 2 parameters: Dir* value + speed for motor A
	Command_SetMotorB     = 0xa5 // 2 parameters: Dir* value + speed for motor B

	// Stepper motor commands
	Command_StepperSpeed = 0x1a // Param: number of 4ms intervals between steps. Default: 100.
	Command_StepperStep  = 0x1c // Param: number of steps (1..254), 0 -> disable, 255 -> endless (default).
	Command_StepperStop  = 0x1b

	// Parameter for Command_SetPWMFrequency
	// PWM signal Frequency (cycle length = 510, system clock = 16MHz)
	PWM_31372Hz = byte(0x01)
	PWM_3921Hz  = byte(0x02)
	PWM_490Hz   = byte(0x03) // Default
	PWM_122Hz   = byte(0x04)
	PWM_30Hz    = byte(0x05)

	// Parameter for Command_SetMotorSpeed, Command_SetMotorA and Command_SetMotorB
	DirClockwise     = byte(0x02)
	DirAntiClockwise = byte(0x01)
	DirStop          = byte(0)

	// No-op parameter filler (if less than 3 bytes are required)
	emptyParameter = 0x01

	MinStepperInterval = 4 * time.Millisecond
	MaxStepperInterval = 256 * MinStepperInterval
)

func SetPwmFrequency(frequency byte) []byte {
	switch frequency {
	case PWM_31372Hz, PWM_3921Hz, PWM_490Hz, PWM_122Hz, PWM_30Hz:
	default:
		log.Warnf("Invalid PWM motor frequency %02x, using maximum frequency 3921Hz", frequency)
		frequency = PWM_3921Hz
	}
	return []byte{Command_SetPWMFrequency, frequency, emptyParameter}
}

func SetMotorDirections(motorA, motorB byte) []byte {
	// Only use 2 bits from each value (Dir* values)
	dir := ((motorB & 0x3) << 2) | (motorA & 0x3)
	return []byte{Command_SetMotorDir, dir, emptyParameter}
}

func SetMotorSpeed(motorA, motorB byte) []byte {
	return []byte{Command_SetMotorSpeed, motorA, motorB}
}

func SetMotorA(speed byte, dir byte) []byte {
	return []byte{Command_SetMotorA, dir & 0x3, speed}
}

func SetMotorB(speed byte, dir byte) []byte {
	return []byte{Command_SetMotorB, dir & 0x3, speed}
}

func StopStepper() []byte {
	return []byte{Command_StepperStop, emptyParameter, emptyParameter}
}

func SetStepperSpeed(forward bool, speed byte) []byte {
	var dir byte
	if forward {
		dir = 1
	} else {
		dir = 0
	}
	return []byte{Command_StepperSpeed, dir, speed}
}

// interval: 4ms .. 958ms
func SetStepperInterval(forward bool, interval time.Duration) []byte {
	if interval > MaxStepperInterval {
		log.Warnf("Stepper timer interval too large: %v, using maximum interval %v", interval, MaxStepperInterval)
		interval = MaxStepperInterval
	}
	if interval < MinStepperInterval {
		log.Warnf("Stepper timer interval too small: %v, using minimum interval %v", interval, MinStepperInterval)
		interval = MinStepperInterval
	}
	speed := byte(interval/MinStepperInterval - 1)
	return SetStepperSpeed(forward, speed)
}

// 0 -> disable, 255 -> endless
func SetStepCount(count byte) []byte {
	return []byte{Command_StepperStep, count, emptyParameter}
}
