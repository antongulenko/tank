package main

import (
	"errors"
	"flag"
	"fmt"

	"github.com/antongulenko/golib"
	"github.com/antongulenko/hid"
	"github.com/antongulenko/tank/ft260"
	"github.com/antongulenko/tank/mcp23017"
	log "github.com/sirupsen/logrus"
)

var (
	i2cFreq = uint(400000)
)

func main() {
	flag.UintVar(&i2cFreq, "freq", i2cFreq, "The I2C bus frequency")
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
	dev, err := ft260.Open()
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

	return work(dev)
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
	if status.I2CEnable {
		return errors.New("FT260: I2C is not enabled on the device")
	}
	if status.ChipMode != 0x01 {
		return fmt.Errorf("FT260: unexpected chip mode %02x (expected %02x)", status.ChipMode, 0x01)
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

const gpioConfig = mcp23017.IOCON_BIT_INTPOL | mcp23017.IOCON_BIT_HAEN | mcp23017.IOCON_BIT_SEQOP

func work(dev *ft260.Ft260) error {
	log.Println("Configuring GPIO extension")
	if err := dev.I2cWrite(mcp23017.ADDRESS, mcp23017.IOCON_PAIRED, gpioConfig); err != nil {
		return err
	}

	log.Println("Enabling Ports A and B as output")
	if err := dev.I2cWrite(mcp23017.ADDRESS, mcp23017.IODIR_PAIRED, mcp23017.INPUT, mcp23017.INPUT); err != nil {
		return err
	}

	log.Warnln("Setting Ports A and B high")
	if err := dev.I2cWrite(mcp23017.ADDRESS, mcp23017.GPIO_PAIRED, 0xFF, 0xFF); err != nil {
		return err
	}

	return nil
}
