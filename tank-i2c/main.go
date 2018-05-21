package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"math"
	"time"

	"github.com/antongulenko/golib"
	"github.com/antongulenko/hid"
	"github.com/antongulenko/tank/ft260"
	"github.com/antongulenko/tank/mcp23017"
	"github.com/antongulenko/tank/pca9685"
	log "github.com/sirupsen/logrus"
)

var (
	usbDevice         = ""
	i2cFreq           = uint(400)
	sleepTime         = 400 * time.Millisecond
	benchTime         = 3 * time.Second
	command           = "scan"
	availableCommands = []string{
		"none", "scan", "bench", "gpio", "motors",
	}
	motorSpeed1 = float64(0)
	motorSpeed2 = float64(0)
    debugMotors bool
)

func main() {
	flag.StringVar(&usbDevice, "dev", usbDevice, "Specify a USB path for FT260")
	flag.UintVar(&i2cFreq, "freq", i2cFreq, "The I2C bus frequency (60 - 3400)")
	flag.DurationVar(&sleepTime, "sleep", sleepTime, "Sleep time between GPIO updates (gpio command)")
	flag.DurationVar(&benchTime, "benchTime", sleepTime, "Benchmark time (bench command)")
	flag.StringVar(&command, "c", command, fmt.Sprintf("Command to execute, one of: %v", availableCommands))
	flag.Float64Var(&motorSpeed1, "m1", motorSpeed1, "Speed of motor 1 (-100..100)")
	flag.Float64Var(&motorSpeed2, "m2", motorSpeed2, "Speed of motor 2 (-100..100)")
	flag.BoolVar(&debugMotors, "debugMotors", false, "Ouput values that would be written, instead of writing them")
	golib.RegisterLogFlags()
	flag.Parse()
	golib.ConfigureLogging()
	golib.Checkerr(doMain())
}

func doMain() error {
	// Prepare USB HID library, open FT260 device
	if err := hid.Init(); err != nil {
		return err
	}
	defer func() {
		golib.Printerr(hid.Shutdown())
	}()
	dev, err := ft260.OpenPath(usbDevice)
	if err != nil {
		return err
	}
	defer func() {
		golib.Printerr(dev.Close())
	}()

	// Configure and validate system settings
	if err := validateFt260ChipCode(dev); err != nil {
		return err
	}
	if err := configureFt260(dev); err != nil {
		return err
	}
	if err := validateFt260(dev); err != nil {
		return err
	}
	log.Println("Successfully opened and configured FT260 device")

	switch command {
	case "none":
		// Nothing
		return nil
	case "scan":
		slaves, err := dev.I2cScan()
		if err != nil {
			return err
		}
		log.Printf("Scanned slaves: %#02v", slaves)
	case "bench":
		return gpioSpeedTest(dev, mcp23017.ADDRESS)
	case "gpio":
		return gpioTest(dev, mcp23017.ADDRESS)
	case "motors":
		return setMotors(dev, mcp23017.ADDRESS, pca9685.ADDRESS, motorSpeed1, motorSpeed2)
	default:
		return fmt.Errorf("Unknown command %v, available commands: %v", command, availableCommands)
	}
	return nil
}

func validateFt260ChipCode(dev *ft260.Ft260) error {
	var code ft260.ReportChipCode
	if err := dev.Read(&code); err != nil {
		return err
	}
	if code.ChipCode != ft260.FT260_CHIP_CODE {
		return fmt.Errorf("Unexpected chip code %04x (expected 0%04x)", code.ChipCode, ft260.FT260_CHIP_CODE)
	}
	return nil
}

func configureFt260(dev *ft260.Ft260) (err error) {
	writeConfigValue(dev, &err, ft260.SetSystemSetting_Clock, ft260.Clock48MHz)
	writeConfigValue(dev, &err, ft260.SetSystemSetting_I2CReset, nil) // Reset i2c bus in case it was disturbed
	writeConfigValue(dev, &err, ft260.SetSystemSetting_I2CSetClock, uint16(i2cFreq))
	writeConfigValue(dev, &err, ft260.SetSystemSetting_GPIO_2, ft260.GPIO_2_Normal) // Set all GPIO pins to normal operation
	writeConfigValue(dev, &err, ft260.SetSystemSetting_GPIO_A, ft260.GPIO_A_Normal)
	writeConfigValue(dev, &err, ft260.SetSystemSetting_GPIO_G, ft260.GPIO_G_Normal)
	writeConfigValue(dev, &err, ft260.SetSystemSetting_EnableWakeupInt, false)
	return
}

func writeConfigValue(dev *ft260.Ft260, outErr *error, address byte, val interface{}) {
	if *outErr == nil {
		*outErr = dev.Write(&ft260.SetSystemStatus{
			Request: address,
			Value:   val,
		})
	}
}

func validateFt260(dev *ft260.Ft260) error {
	var status ft260.ReportSystemStatus
	if err := dev.Read(&status); err != nil {
		return err
	}
	if status.ChipMode != 0x01 {
		return fmt.Errorf("FT260: unexpected chip mode %02x (expected %02x)", status.ChipMode, 0x01)
	}
	if status.Clock != ft260.Clock48MHz {
		return fmt.Errorf("FT260: unexpected clock value %02x (expected %02x)", status.Clock, ft260.Clock48MHz)
	}
	if status.GPIO2Function != ft260.GPIO_2_Normal {
		return fmt.Errorf("FT260: unexpected GPIO 2 function %02x (expected %02x)", status.GPIO2Function, ft260.GPIO_2_Normal)
	}
	if status.GPIOAFunction != ft260.GPIO_A_Normal {
		return fmt.Errorf("FT260: unexpected GPIO A function %02x (expected %02x)", status.GPIOAFunction, ft260.GPIO_A_Normal)
	}
	if status.GPIOGFunction != ft260.GPIO_G_Normal {
		return fmt.Errorf("FT260: unexpected GPIO G function %02x (expected %02x)", status.GPIOGFunction, ft260.GPIO_G_Normal)
	}
	if status.EnableWakeupInt {
		return fmt.Errorf("FT260: unexpected clock value %02x (expected %02x)", status.EnableWakeupInt, false)
	}
	if status.Suspended {
		return errors.New("FT260: device is suspended")
	}
	if !status.PowerStatus {
		return errors.New("FT260: device is powered off")
	}
	if !status.I2CEnable {
		return errors.New("FT260: I2C is not enabled on the device")
	}

	var i2cStatus ft260.ReportI2cStatus
	if err := dev.Read(&i2cStatus); err != nil {
		return err
	}
	if i2cStatus.BusSpeed != uint16(i2cFreq) {
		return fmt.Errorf("FT260: unexpected I2C bus speed %v (expected %v)", i2cStatus.BusSpeed, i2cFreq)
	}
	return nil
}

func gpioSpeedTest(dev *ft260.Ft260, addr byte) error {
	log.Printf("Configuring GPIO extension %02x", addr)
	if err := dev.I2cWrite(addr, mcp23017.IOCON_PAIRED, gpioConfig); err != nil {
		return err
	}

	// Write/Read every register on the device to overflow and start writing/reading from the first register again
	data := []byte{0, 0xFF, 0, 0xFF, 0, 0, 0xFF, 0xFF, 0, 0, gpioConfig, gpioConfig, 0xFF, 0, 0, 0, 0, 0, 0xFF, 0xFF, 0xFF, 0xFF}

	log.Println("Measuring write...")
	writeData := append([]byte{0}, data...)
	err := bench(func() (int, error) {
		err := dev.I2cWrite(addr, writeData...)
		return len(data), err
	})
	if err != nil {
		return err
	}

	// Read the last byte to set the read address to the first position
	if b, err := dev.I2cGet(addr, byte(len(data)-1), 1); err != nil {
		return err
	} else {
		expect := []byte{data[len(data)-1]}
		if !bytes.Equal(b, expect) {
			return fmt.Errorf("Expected to read %02x, but got %02x", expect, b)
		}
	}

	log.Println("Measuring read...")
	receive := make([]byte, len(data)*2)
	expectedReadData := make([]byte, len(receive))
	copy(expectedReadData, data)
	copy(expectedReadData[len(data):], data)
	err = bench(func() (int, error) {
		err := dev.I2cRead(addr, receive)
		return len(receive), err
	})
	if !bytes.Equal(expectedReadData, receive) {
		log.Warnln("Received unexpected bytes:", receive)
		log.Warnln(" --------------- Expected:", expectedReadData)
	}
	return err
}

func bench(benchFunc func() (int, error)) error {
	start := time.Now()
	transmitted := 0
	for i := 0; ; i++ {
		transmittedNew, err := benchFunc()
		if err != nil {
			return err
		}
		transmitted += transmittedNew
		if i%20 == 0 {
			if duration := time.Now().Sub(start); duration > benchTime {
				bps := float64(transmitted) * 8 / duration.Seconds()
				log.Printf("Transmitted %v byte in %v -> %2v bps", transmitted, duration, bps)
				break
			}
		}
	}
	return nil
}

const gpioConfig = mcp23017.IOCON_BIT_INTPOL | mcp23017.IOCON_BIT_HAEN

func initGpio(dev *ft260.Ft260, addr byte) error {
	log.Printf("Configuring GPIO extension %02x", addr)
	if err := dev.I2cWrite(addr, mcp23017.IOCON_PAIRED, gpioConfig); err != nil {
		return err
	}

	log.Println("Enabling Ports A and B as output")
	if err := dev.I2cWrite(addr, mcp23017.IODIR_PAIRED, mcp23017.OUTPUT, mcp23017.OUTPUT); err != nil {
		return err
	}
	return nil
}

func gpioTest(dev *ft260.Ft260, addr byte) error {
	if err := initGpio(dev, addr); err != nil {
		return err
	}

	val := byte(0xFF)
	for {
		if err := dev.I2cWrite(addr, mcp23017.GPIO_PAIRED, val, val); err != nil {
			return err
		}
		if values, err := dev.I2cGet(addr, mcp23017.GPIO_PAIRED, 2); err != nil {
			return err
		} else {
			log.Println("Port values:", values)
		}
		val = ^val
		time.Sleep(sleepTime)
	}
	return nil
}

const (
	motorStop     = 0
	motorForward  = 1
	motorBackward = 2
)

func setMotors(dev *ft260.Ft260, gpioAddr byte, pwmAddr byte, speed1, speed2 float64) error {
	if speed1 < -100 || speed1 > 100 {
		return fmt.Errorf("Illegal motor speed1 %v (must be -100..100)", speed1)
	}
	if speed2 < -100 || speed2 > 100 {
		return fmt.Errorf("Illegal motor speed2 %v (must be -100..100)", speed2)
	}

	// Initialize both devices first
	if err := initGpio(dev, gpioAddr); err != nil {
		return err
	}
	log.Printf("Initializing PWM driver at %02x...", pwmAddr)
	if err := dev.I2cWrite(pwmAddr, pca9685.MODE1, pca9685.MODE1_ALLCALL|pca9685.MODE1_AI); err != nil {
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
        if err := dev.I2cWrite(gpioAddr, gpioByteAddr, gpioByte); err != nil {
            return err
        }
    }

	// Configure PWM signals
	pwmValues := make([]byte, 8)
	pca9685.ValuesInto(pwm1, pwmValues)
	pca9685.ValuesInto(pwm2, pwmValues[4:])
	pwmValues = append([]byte{pca9685.LED0}, pwmValues...)
	log.Printf("Setting PWM values for motor 1 and 2 to %#02x...", pwmValues)
    if !debugMotors {
        if err := dev.I2cWrite(pwmAddr, pwmValues...); err != nil {
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
