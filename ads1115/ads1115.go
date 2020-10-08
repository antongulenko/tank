package ads1115

import (
	"fmt"
    "log"

	"github.com/antongulenko/tank/ft260"
)

// TODO Missing: switching to high-speed mode

const (
	// The ADDR pin can be connected to one of the following 4 pins to select the respective I2C address
	ADDR_GND = byte(0x48)
	ADDR_VDD = byte(0x49)
	ADDR_SDA = byte(0x4A)
	ADDR_SCL = byte(0x4B)

	// Can be issued as second byte directly after an I2C general call to power down the device
	GENERAL_CALL_SHUTDOWN = byte(0x06)

	// The addresses of the writable/readable registers
	REG_CONVERSION = byte(iota)
	REG_CONFIG
	REG_LO_THRESH // Default: 0x8000, relevant for CONFIG_COMP_* bits
	REG_HI_THRESH // Default: 0x7FFF

	// Setting MSB of LO_THRESH to 0 and MSB of HI_THRESH to 1 enables the comparison-ready functionality (not default)
)

// Bits for the CONFIG register

const (
	// Operational State. If written, starts a one-short conversion.
	// If read, indicates whether conversion is taking place (1 = NO operation)
	CONFIG_OS = uint16(0x8000)

	// Selection of inputs. First number is the positive input, second number the negative input.
	// Example: CONFIG_MUX_01 compares AIN0 to AIN1. CONFIG_MUX_1GND compares AIN1 to GND
	// Default: CONFIG_MUX_01
	CONFIG_MUX_01 = uint16(iota) << 12
	CONFIG_MUX_03
	CONFIG_MUX_13
	CONFIG_MUX_23
	CONFIG_MUX_0GND
	CONFIG_MUX_1GND
	CONFIG_MUX_2GND
	CONFIG_MUX_3GND
)

const (
	// Selection of Full Scale (max input voltage values when converting)
	// Default: CONFIG_PGA_2V
	CONFIG_PGA_6V     = uint16(iota) << 9 // 6.144V
	CONFIG_PGA_4V                         // 4.096V
	CONFIG_PGA_2V                         // 2.048V
	CONFIG_PGA_1V                         // 1.024V
	CONFIG_PGA_0_5V                       // 0.512V
	CONFIG_PGA_0_25V1                     // 0.256V
	CONFIG_PGA_0_25V2                     // 0.256V
	CONFIG_PGA_0_25V3                     // 0.256V

	// Mode bit: 1 = single-shot mode/power down. 0 = continuous mode
	CONFIG_MODE = uint16(0x100)
)

const (
	// Data conversion rate (samples per second). higher rate -> lower accuracy/resolution
	// Default: CONFIG_DR_128
	CONFIG_DR_8 = uint16(iota) << 5
	CONFIG_DR_16
	CONFIG_DR_32
	CONFIG_DR_64
	CONFIG_DR_128
	CONFIG_DR_250
	CONFIG_DR_475
	CONFIG_DR_860

	// Comparator mode. 0 = traditional comparator (default), 1 = window comparator
	CONFIG_COMP_MODE = 0x10

	// Polarity of ALERT/RDY pin. 0 = active low (default), 1 = active high
	CONFIG_COMP_POL = 0x8

	// 0 = nolatching comparator (ALERT/RDY pin disables when comparison goes back within the thresholds), default
	// 1 = latching comparator (ALERT/RDY remains latched until data is read by master)
	CONFIG_COMP_LAT = 0x4

	// Number of successive comparisons exceeding the thresholds, after which the ALERT/RDY pin will be activated
	// Default: CONFIG_COMP_QUE_OFF (ALERT/RDY pin set to high impedance)
	CONFIG_COMP_QUE_1   = uint16(0)
	CONFIG_COMP_QUE_2   = uint16(1)
	CONFIG_COMP_QUE_4   = uint16(2)
	CONFIG_COMP_QUE_OFF = uint16(3)
)

func WriteRegister(bus ft260.I2cBus, i2cAddr byte, register byte, val uint16) error {
	return bus.I2cWrite(i2cAddr, register, byte(val>>8), byte(val))
}

func ReadRegister(bus ft260.I2cBus, i2cAddr byte, register byte) (int16, error) {
	v, err := bus.I2cGet(i2cAddr, register, 2)
	if err == nil && len(v) != 2 {
		err = fmt.Errorf("ADS1115 read len %v (need 2 byte)", len(v))
	}
	if err != nil {
		return 0, err
	}
	return parseConversionRegister(v), nil
}

func ReadRegisterDirectly(bus ft260.I2cBus, i2cAddr byte) (int16, error) {
	v := make([]byte, 2)
	err := bus.I2cRead(i2cAddr, v)
	if err == nil && len(v) != 2 {
		err = fmt.Errorf("ADS1115 read len %v (need 2 byte)", len(v))
	}
	if err != nil {
		return 0, err
	}
	return parseConversionRegister(v), nil
}

func parseConversionRegister(v []byte) int16 {
    log.Printf("Received %v ADS bytes: %#x %#x", len(v), v[0], v[1])
	result := int16(v[1])      // Least-significant byte
	result |= int16(v[0]) << 8 // Most-significant byte
	return result
}
