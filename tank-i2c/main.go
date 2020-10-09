package main

import (
	"bytes"
	"flag"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/antongulenko/golib"
	"github.com/antongulenko/tank/ft260"
	"github.com/antongulenko/tank/mcp23017"
	"github.com/antongulenko/tank/pca9685"
	"github.com/antongulenko/tank/tank"
	log "github.com/sirupsen/logrus"
)

type commandFunc func() error

var (
	t          = tank.DefaultTank
	sleepTime  = 400 * time.Millisecond
	benchTime  = 3 * time.Second
	command    = "scan"
	ledI2cAddr = uint(0x44)
	commands   = map[string]commandFunc{
		"none":           func() error { return nil },
		"scan":           scan,
		"bench":          gpioSpeedTest,
		"gpio":           gpioTest,
		"gpio-motors":    setGpioMotors,
		"motors":         setMotors,
		"leds":           setRawLeds,
		"tankLeds":       setTankLeds,
		"tankLedStartup": playTankLedStartup,
		"battery":        readBatteryVoltage,
	}
	motorSpeed1 = float64(0)
	motorSpeed2 = float64(0)
	debugMotors bool
)

func main() {
	t.RegisterFlags()
	flag.UintVar(&ledI2cAddr, "leds", ledI2cAddr, "I2C address of led driver for -c tankLeds")
	flag.DurationVar(&sleepTime, "sleep", sleepTime, "Sleep time between GPIO updates (gpio command)")
	flag.DurationVar(&benchTime, "benchTime", sleepTime, "Benchmark time (bench command)")
	flag.StringVar(&command, "c", command, fmt.Sprintf("Command to execute, one of: %v", commands))
	flag.Float64Var(&motorSpeed1, "l", motorSpeed1, "Speed of motor 1 (-100..100)")
	flag.Float64Var(&motorSpeed2, "r", motorSpeed2, "Speed of motor 2 (-100..100)")
	flag.BoolVar(&debugMotors, "debugMotors", false, "Output values that would be written, instead of writing them")
	golib.RegisterLogFlags()
	flag.Parse()
	golib.ConfigureLogging()
	golib.Checkerr(doMain())
}

func doMain() error {
	if err := t.Setup(); err != nil {
		return err
	}
	log.Println("Successfully initialized USB/I2C peripherals")

	commandFunc, ok := commands[command]
	if !ok {
		allCommandNames := make([]string, 0, len(commands))
		for commandName := range commands {
			allCommandNames = append(allCommandNames, commandName)
		}
		sort.Strings(allCommandNames)
		return fmt.Errorf("Unknown command %v, available commands: %v", command, commands)
	}
	return commandFunc()
}

func scan() error {
	slaves, err := ft260.I2cScan(t.Bus())
	if err != nil {
		return err
	}
	log.Printf("Scanned slaves: %#02v", slaves)
	return nil
}

func gpioSpeedTest() error {
	addr := mcp23017.ADDRESS

	log.Printf("Configuring GPIO extension %02x", addr)
	if err := t.Bus().I2cWrite(addr, mcp23017.IOCON_PAIRED, gpioConfig); err != nil {
		return err
	}

	// Write/Read every register on the device to overflow and start writing/reading from the first register again
	data := []byte{0, 0xFF, 0, 0xFF, 0, 0, 0xFF, 0xFF, 0, 0, gpioConfig, gpioConfig, 0xFF, 0, 0, 0, 0, 0, 0xFF, 0xFF, 0xFF, 0xFF}

	log.Println("Measuring write...")
	writeData := append([]byte{0}, data...)
	err := bench(func() (int, error) {
		err := t.Bus().I2cWrite(addr, writeData...)
		return len(data), err
	})
	if err != nil {
		return err
	}

	// Read the last byte to set the read address to the first position
	if b, err := t.Bus().I2cGet(addr, byte(len(data)-1), 1); err != nil {
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
		err := t.Bus().I2cRead(addr, receive)
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
	if err := t.Bus().I2cWrite(addr, mcp23017.IOCON_PAIRED, gpioConfig); err != nil {
		return err
	}

	log.Println("Enabling Ports A and B as output")
	if err := t.Bus().I2cWrite(addr, mcp23017.IODIR_PAIRED, mcp23017.OUTPUT, mcp23017.OUTPUT); err != nil {
		return err
	}
	return nil
}

func gpioTest() error {
	addr := mcp23017.ADDRESS
	if err := initGpio(addr); err != nil {
		return err
	}

	val := byte(0xFF)
	for {
		if err := t.Bus().I2cWrite(addr, mcp23017.GPIO_PAIRED, val, val); err != nil {
			return err
		}
		if values, err := t.Bus().I2cGet(addr, mcp23017.GPIO_PAIRED, 2); err != nil {
			return err
		} else {
			log.Println("Port values:", values)
		}
		val = ^val
		time.Sleep(sleepTime)
	}
}

func setMotors() error {
	if err := t.Motors.Init(); err != nil {
		return err
	}
	return t.Motors.Set(motorSpeed1, motorSpeed2)
}

func setRawLeds() error {
	var values []float64
	for _, valueStr := range flag.Args() {
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			log.Errorf("Failed to parse argument '%v' as float: %v", valueStr, err)
			continue
		}
		values = append(values, value)
	}
	if len(values) == 0 {
		return fmt.Errorf("No valid values given for setting LEDs")
	}
	log.Printf("Setting %v led value(s): %v", len(values), values)

	log.Printf("Initializing led driver at %v...", ledI2cAddr)
	if err := t.Bus().I2cWrite(byte(ledI2cAddr), pca9685.MODE1, pca9685.MODE1_ALLCALL|pca9685.MODE1_AI); err != nil {
		return err
	}
	log.Println("Success")
	pwmValues := make([]byte, pca9685.BYTE_PER_OUTPUT*len(values))
	for i, val := range values {
		pca9685.ValuesInto(val, pwmValues[pca9685.BYTE_PER_OUTPUT*i:])
	}
	pwmValues = append([]byte{pca9685.LED0}, pwmValues...)
	log.Printf("Writing %v byte to led driver: %v", len(pwmValues), pwmValues)
	return t.Bus().I2cWrite(byte(ledI2cAddr), pwmValues...)
}

func setTankLeds() error {
	var values []float64
	for _, valueStr := range flag.Args() {
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			log.Errorf("Failed to parse argument '%v' as float: %v", valueStr, err)
			continue
		}
		values = append(values, value)
	}
	if len(values) == 0 {
		return fmt.Errorf("No valid values given for setting LEDs")
	}
	log.Printf("Setting %v led value(s): %v", len(values), values)

	if err := t.Leds.Init(); err != nil {
		return err
	}
	return t.Leds.SetAll(values)
}

func playTankLedStartup() error {
	return fmt.Errorf("Startup sequence not yet implemented")
	/*
		return tank.RunLedStartupSequence(math.MaxInt32, func(sleepTime time.Duration, values []float64) error {
			if err := t.Leds.Set(values); err != nil {
				return err
			}
			time.Sleep(sleepTime)
			return nil
		})
	*/
}

func readBatteryVoltage() error {
	if err := t.Adc.Init(); err != nil {
		return err
	}
	volt, err := t.Adc.GetBatteryVoltage()
	if err != nil {
		return err
	}
	percentage := t.Adc.ConvertVoltageToPercentage(volt)
	log.Printf("Battery percentage: %.2f%% (%.2fV)", percentage*100, volt)
	return nil
}
