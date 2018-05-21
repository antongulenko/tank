package main

import (
	"bytes"
	"flag"
	"fmt"
	"math"
	"time"

	"github.com/antongulenko/golib"
	"github.com/antongulenko/tank/mcp23017"
	"github.com/antongulenko/tank/pca9685"
	"github.com/antongulenko/tank/tank"
	log "github.com/sirupsen/logrus"
)

var (
	t                 = tank.DefaultTank
	sleepTime         = 400 * time.Millisecond
	benchTime         = 3 * time.Second
	command           = "scan"
	availableCommands = []string{
		"none", "scan", "bench", "gpio", "motors", "motorDirect",
	}
	motorSpeed1 = float64(0)
	motorSpeed2 = float64(0)
)

func main() {
	t.RegisterFlags()
	flag.DurationVar(&sleepTime, "sleep", sleepTime, "Sleep time between GPIO updates (gpio command)")
	flag.DurationVar(&benchTime, "benchTime", sleepTime, "Benchmark time (bench command)")
	flag.StringVar(&command, "c", command, fmt.Sprintf("Command to execute, one of: %v", availableCommands))
	flag.Float64Var(&motorSpeed1, "m1", motorSpeed1, "Speed of motor 1 (-100..100)")
	flag.Float64Var(&motorSpeed2, "m2", motorSpeed2, "Speed of motor 2 (-100..100)")
	golib.RegisterLogFlags()
	flag.Parse()
	golib.ConfigureLogging()
	golib.Checkerr(doMain())
}

func doMain() error {
	if err := t.Setup(); err != nil {
		return err
	}
	log.Println("Successfully opened and configured FT260 device")

	switch command {
	case "none":
		// Nothing
		return nil
	case "scan":
		slaves, err := t.Usb.I2cScan()
		if err != nil {
			return err
		}
		log.Printf("Scanned slaves: %#02v", slaves)
	case "bench":
		return gpioSpeedTest(mcp23017.ADDRESS)
	case "gpio":
		return gpioTest(mcp23017.ADDRESS)
	case "motors":
		return setMotors(mcp23017.ADDRESS, motorSpeed1, motorSpeed2)
	case "motorDirect":
		return setMotorDirect(motorSpeed1)
	default:
		return fmt.Errorf("Unknown command %v, available commands: %v", command, availableCommands)
	}
	return nil
}

func gpioSpeedTest(addr byte) error {
	log.Printf("Configuring GPIO extension %02x", addr)
	if err := t.Usb.I2cWrite(addr, mcp23017.IOCON_PAIRED, gpioConfig); err != nil {
		return err
	}

	// Write/Read every register on the device to overflow and start writing/reading from the first register again
	data := []byte{0, 0xFF, 0, 0xFF, 0, 0, 0xFF, 0xFF, 0, 0, gpioConfig, gpioConfig, 0xFF, 0, 0, 0, 0, 0, 0xFF, 0xFF, 0xFF, 0xFF}

	log.Println("Measuring write...")
	writeData := append([]byte{0}, data...)
	err := bench(func() (int, error) {
		err := t.Usb.I2cWrite(addr, writeData...)
		return len(data), err
	})
	if err != nil {
		return err
	}

	// Read the last byte to set the read address to the first position
	if b, err := t.Usb.I2cGet(addr, byte(len(data)-1), 1); err != nil {
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
		err := t.Usb.I2cRead(addr, receive)
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

func initGpio(addr byte) error {
	log.Printf("Configuring GPIO extension %02x", addr)
	if err := t.Usb.I2cWrite(addr, mcp23017.IOCON_PAIRED, gpioConfig); err != nil {
		return err
	}

	log.Println("Enabling Ports A and B as output")
	if err := t.Usb.I2cWrite(addr, mcp23017.IODIR_PAIRED, mcp23017.OUTPUT, mcp23017.OUTPUT); err != nil {
		return err
	}
	return nil
}

func gpioTest(addr byte) error {
	if err := initGpio(addr); err != nil {
		return err
	}

	val := byte(0xFF)
	for {
		if err := t.Usb.I2cWrite(addr, mcp23017.GPIO_PAIRED, val, val); err != nil {
			return err
		}
		if values, err := t.Usb.I2cGet(addr, mcp23017.GPIO_PAIRED, 2); err != nil {
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

func setMotors(gpioAddr byte, speed1, speed2 float64) error {
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
	log.Printf("Setting GPIO port B to %02x", gpioByte)
	if err := t.Usb.I2cWrite(gpioAddr, mcp23017.GPIO_B_PAIRED, gpioByte); err != nil {
		return err
	}

	pwmAddr := pca9685.ADDRESS

	// Configure PWM signals
	pwmValues := make([]byte, 8)
	pca9685.ValuesInto(pwm1, pwmValues)
	pca9685.ValuesInto(pwm2, pwmValues[4:])
	log.Println("Setting PWM values for motor 1 and 2...")
	pwmValues = append([]byte{pca9685.LED0}, pwmValues...)
	if err := t.Usb.I2cWrite(pwmAddr, pwmValues...); err != nil {
		return err
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

func setMotorDirect(speed float64) error {
	if speed < -100 || speed > 100 {
		return fmt.Errorf("Illegal motor speed1 %v (must be -100..100)", speed)
	}
	if err := t.Motors.Init(); err != nil {
		return err
	}
	return t.Motors.Set(speed, speed)
}
