package pca9685

import (
	"fmt"
	"math"
)

const (
	MODE1 = byte(iota)
	MODE2

	// The I2C addresses are stored in the 7 MSBs. Addresses must be left-shifted once.
	SUBADR1
	SUBADR2
	SUBADR3
	ALLCALLADR

	// Default for LEDn_...: all zero, except for FULL_OFF_BIT in LEDn_OFF_H.
	LED0_ON_L
	LED0_ON_H
	LED0_OFF_L
	LED0_OFF_H
	LED1_ON_L
	LED1_ON_H
	LED1_OFF_L
	LED1_OFF_H
	LED2_ON_L
	LED2_ON_H
	LED2_OFF_L
	LED2_OFF_H
	LED3_ON_L
	LED3_ON_H
	LED3_OFF_L
	LED3_OFF_H
	LED4_ON_L
	LED4_ON_H
	LED4_OFF_L
	LED4_OFF_H
	LED5_ON_L
	LED5_ON_H
	LED5_OFF_L
	LED5_OFF_H
	LED6_ON_L
	LED6_ON_H
	LED6_OFF_L
	LED6_OFF_H
	LED7_ON_L
	LED7_ON_H
	LED7_OFF_L
	LED7_OFF_H
	LED8_ON_L
	LED8_ON_H
	LED8_OFF_L
	LED8_OFF_H
	LED9_ON_L
	LED9_ON_H
	LED9_OFF_L
	LED9_OFF_H
	LED10_ON_L
	LED10_ON_H
	LED10_OFF_L
	LED10_OFF_H
	LED11_ON_L
	LED11_ON_H
	LED11_OFF_L
	LED11_OFF_H
	LED12_ON_L
	LED12_ON_H
	LED12_OFF_L
	LED12_OFF_H
	LED13_ON_L
	LED13_ON_H
	LED13_OFF_L
	LED13_OFF_H
	LED14_ON_L
	LED14_ON_H
	LED14_OFF_L
	LED14_OFF_H
	LED15_ON_L
	LED15_ON_H
	LED15_OFF_L
	LED15_OFF_H

	LED0  = LED0_ON_L
	LED1  = LED1_ON_L
	LED2  = LED2_ON_L
	LED3  = LED3_ON_L
	LED4  = LED4_ON_L
	LED5  = LED5_ON_L
	LED6  = LED6_ON_L
	LED7  = LED7_ON_L
	LED8  = LED8_ON_L
	LED9  = LED9_ON_L
	LED10 = LED10_ON_L
	LED11 = LED11_ON_L
	LED12 = LED12_ON_L
	LED13 = LED13_ON_L
	LED14 = LED14_ON_L
	LED15 = LED15_ON_L
)

const (
	ALL_ON_L = byte(0xFA + iota)
	ALL_ON_H
	ALL_OFF_L
	ALL_OFF_H
	PRE_SCALE // Only settable in SLEEP mode. Default value: 0x30
	TEST_MODE

	ALL_LEDS = ALL_ON_L
)

// Default values all zero, except ALLCALL and SLEEP
const (
	MODE1_ALLCALL = byte(1 << iota) // 1: Respond to ALLCALL address
	MODE1_SUB3                      // 1: Respond to SUB3 address
	MODE1_SUB2                      // 1: Respond to SUB2 address
	MODE1_SUB1                      // 1: Respond to SUB1 address
	MODE1_SLEEP                     // 0: normal mode 1: oscillator off, low power mode
	MODE1_AI                        // 1: Register auto increment
	MODE1_EXTCLK                    // 1: use EXTCLK pin as clock source. Enable sequence: First set SLEEP, then set (SLEEP | EXTCLK). Can only be cleared by power cycle or software reset.
	MODE1_RESTART                   // Write 1: wake up from SLEEP (write 0 no effect). Only possible if read as 1, after setting SLEEP.
)

// Default values all zero, except OUTDRV
const (
	// Control led state for OE pin = 1 (leds disabled)
	MODE2_OUTNE0 = byte(1 << iota) // (only for OUTNE1=0) 0: leds off 1: [leds on if OUTDRV=1, high-impedance if OUTDRV=0]
	MODE2_OUTNE1                   // 1: high impedance 0: see OUTNE0

	MODE2_OUTDRV // 0: outputs are open drain 1: outputs are totem pole
	MODE2_OCH    // 0: output change on STOP 1: output change on ACK (after writing all 4 registers of an LED)
	MODE2_INVRT  // 1: invert output logic
)

const (
	ADDRESS     = byte(0x40) // 0100 0000
	ADDRESS_MAX = byte(0x7F) // 0111 1111

	DEFAULT_ALLCALL_ADDRESS  = byte(0x70) // 0111 0000
	DEFAULT_SUBCALL1_ADDRESS = byte(0x71) // 0111 0001
	DEFAULT_SUBCALL2_ADDRESS = byte(0x72) // 0111 0010
	DEFAULT_SUBCALL3_ADDRESS = byte(0x74) // 0111 0100
	SOFTWARE_RESET_ADDRESS   = byte(0x03) // 0000 0011 // READ to trigger reset
)

const (
	BYTE_PER_OUTPUT  = 4
	TIMER_MAX        = 4095
	TIMER_RESOLUTION = TIMER_MAX + 1

	FULL_ON_BIT  = 0x10 // bit 4 of LEDn_ON_H.
	FULL_OFF_BIT = 0x10 // bit 4 of LEDn_OFF_H. Takes precedence over the FULL_ON_BIT.

	FREQ_MIN          = 23.84185791
	FREQ_MAX          = 1525.87890625
	FREQ_MIN_PRESALE  = byte(0xFF)
	FREQ_MAX_PRESCALE = byte(0x03) // Minimum value asserted by hardware
	DEFAULT_PRESCALE  = byte(0x30) // Default PRE_SCALE value, results in 200Hz with the internal oscillator

	INTERNAL_OSCILLATOR = 25000000 // 25 MHz
)

func ValuesInto(onTime float64, target []byte) {
	ValuesDelayedInto(0, onTime, target)
}

// Target byte slice is suitable to write into an LEDn or ALL_LEDS address
func ValuesDelayedInto(delayTime, onTime float64, target []byte) {
	target[0], target[1], target[2], target[3] = ValuesDelayed(delayTime, onTime)
}

func Values(onTime float64) (byte, byte, byte, byte) {
	return ValuesDelayed(0, onTime)
}

// delay and onTime must be in [0; 1]
func ValuesDelayed(delayTime, onTime float64) (onL, onH, offL, offH byte) {
	if delayTime < 0 || delayTime > 1 || onTime < 0 || onTime > 1 {
		panic(fmt.Sprintf("Invalid timer values delay=%v onTime=%v", delayTime, onTime))
	}
	delayCount := round(delayTime*TIMER_RESOLUTION - 1)
	onCount := round(onTime * TIMER_RESOLUTION) // The onCount is added to delayCount, so the -1 correction is not required anymore
	if delayTime == 0 {
		delayCount = 0
		if onCount > 0 {
			onCount-- // Apply -1 correction since delayCount is zero
		}
	}
	if onTime == 0 {
		onCount = 0
	}

	on := delayCount
	off := on + onCount
	if off > TIMER_RESOLUTION {
		// Because of the delay, the first on-time is pushed into the second PWM cycle, and must be corrected
		off -= TIMER_RESOLUTION
	}
	onL, onH = byte(on), byte(on>>8)
	offL, offH = byte(off), byte(off>>8)
	return
}

func round(f float64) int {
	return int(math.Floor(f + .5))
}

func FullOnValuesInto(target []byte) {
	target[0], target[1], target[2], target[3] = FullOnValues()
}

func FullOnValues() (byte, byte, byte, byte) {
	return 0, FULL_ON_BIT, 0, 0
}

func FullOffValuesInto(target []byte) {
	target[0], target[1], target[2], target[3] = FullOffValues()
}

func FullOffValues() (byte, byte, byte, byte) {
	return 0, 0, 0, FULL_OFF_BIT
}

func FullValuesInto(on bool, target []byte) {
	if on {
		FullOnValuesInto(target)
	} else {
		FullOffValuesInto(target)
	}
}

func FullValues(on bool) (byte, byte, byte, byte) {
	if on {
		return FullOnValues()
	} else {
		return FullOffValues()
	}
}

func PrescalerExternalClock(externalOscillator float64, frequency float64) byte {
	v := externalOscillator / (float64(TIMER_RESOLUTION) * frequency)
	return byte(round(v)) - 1
}

func Prescaler(frequency float64) byte {
	return PrescalerExternalClock(INTERNAL_OSCILLATOR, frequency)
}

type PwmOutput struct {
	CurrentState   []float64
	OptimizeUpdate bool
}

func (m *PwmOutput) FillCurrentState(newState []float64) []float64 {
	if m.CurrentState != nil {
		if len(newState) < len(m.CurrentState) {
			newState = append(newState, m.CurrentState[len(newState):]...)
		} else if len(newState) > len(m.CurrentState) {
			return newState[:len(m.CurrentState)]
		}
	}
	return newState
}

func (m *PwmOutput) Update(firstPwmOutput byte, newState []float64) []byte {
	if len(m.CurrentState) != len(newState) {
		m.CurrentState = make([]float64, len(newState))
		m.OptimizeUpdate = false
	}
	numPwmOutputs := len(newState)

	// Compute smallest possible range of values to be updated
	updateFrom := 0
	updateTo := numPwmOutputs
	if m.OptimizeUpdate {
		for i := range newState {
			if m.CurrentState[i] == newState[i] {
				updateFrom++
			} else {
				break
			}
		}
		for i := range newState {
			if m.CurrentState[numPwmOutputs-1-i] == newState[numPwmOutputs-1-i] {
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
	copy(m.CurrentState, newState)
	m.OptimizeUpdate = true
	numChanges := updateTo - updateFrom

	// Compute raw bytes to be sent to the device
	pwmValues := make([]byte, BYTE_PER_OUTPUT*numChanges)
	for i, val := range newState[updateFrom:updateTo] {
		ValuesInto(val, pwmValues[BYTE_PER_OUTPUT*i:])
	}

	return append([]byte{firstPwmOutput + byte(updateFrom)*BYTE_PER_OUTPUT}, pwmValues...)
}
