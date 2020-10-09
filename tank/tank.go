package tank

import (
	"errors"
	"flag"
	"fmt"

	"github.com/antongulenko/hid"
	"github.com/antongulenko/tank/ads1115"
	"github.com/antongulenko/tank/ft260"
	"github.com/antongulenko/tank/pca9685"
	log "github.com/sirupsen/logrus"
)

var DefaultTank = Tank{
	UsbDevice:       "",
	I2cFreq:         uint(400),
	I2cRequestQueue: 20,
	Motors: MainMotors{
		I2cAddr:        pca9685.ADDRESS,
		PwmStart:       pca9685.LED0,
		InvertLeftDir:  false,
		InvertRightDir: false,
	},
	Leds: MainLeds{
		I2cAddr:  pca9685.ADDRESS + 4, // A2 pin set
		PwmStart: pca9685.LED0,
		NumLeds:  15,
	},
	Adc: Adc{
		I2cAddr:    ads1115.ADDR_GND,
		BatteryMin: 2.60,
		BatteryMax: 3.24,
	},
}

type Tank struct {
	UsbDevice       string
	I2cFreq         uint
	I2cRequestQueue int
	NoI2cSequencer  bool
	Dummy           bool
	SkipInit        bool

	Motors MainMotors
	Leds   MainLeds
	Adc    Adc

	usb       *ft260.Ft260
	sequencer sequencedI2cBus
}

func (t *Tank) RegisterFlags() {
	flag.StringVar(&t.UsbDevice, "dev", t.UsbDevice, "Specify a USB path for FT260")
	flag.UintVar(&t.I2cFreq, "freq", t.I2cFreq, "The I2C bus frequency (60 - 3400)")
	flag.BoolVar(&t.NoI2cSequencer, "no-i2c-sequencer", t.NoI2cSequencer, "Disable the extra goroutine for sequencing I2C commands")
	flag.BoolVar(&t.Dummy, "dummy", t.Dummy, "Disable USB/I2C peripherals")
	flag.BoolVar(&t.SkipInit, "skip-init", t.SkipInit, "Do not initialize USB/I2C peripherals, but use for subsequent commands")

	// Motors
	flag.BoolVar(&t.Motors.Dummy, "dummy-motors", t.Motors.Dummy, "Disable real motor control, only output commands")
	flag.BoolVar(&t.Motors.SkipInit, "skip-init-motors", t.Motors.SkipInit, "Do not initialize motor I2C device, but use for subsequent commands")

	// LEDs
	flag.BoolVar(&t.Leds.Dummy, "dummy-leds", t.Leds.Dummy, "Disable real LED control, only output values")
	flag.BoolVar(&t.Leds.SkipInit, "skip-init-leds", t.Leds.SkipInit, "Do not initialize LED I2C device, but use for subsequent commands")
	flag.IntVar(&t.Leds.NumLeds, "num-leds", t.Leds.NumLeds, "Number of main leds")

	// ADC, Battery
	flag.BoolVar(&t.Adc.Dummy, "dummy-adc", t.Adc.Dummy, "Disable real ADC control, only output values")
	flag.BoolVar(&t.Adc.SkipInit, "skip-init-adc", t.Adc.SkipInit, "Do not initialize ADC I2C device, but use for subsequent commands")
	flag.Float64Var(&t.Adc.BatteryMin, "battery-min", t.Adc.BatteryMin, "Minimum value for battery voltage")
	flag.Float64Var(&t.Adc.BatteryMax, "battery-max", t.Adc.BatteryMax, "Minimum value for battery voltage")
}

func (t *Tank) Setup() error {
	if t.Dummy {
		log.Println("Dummy tank: not using USB/I2C peripherals")
		t.Leds.Dummy = true
		t.Motors.Dummy = true
		t.Adc.Dummy = true
	} else {
		if t.SkipInit {
			log.Println("Not initializing USB/I2C peripherals")
			t.Leds.SkipInit = true
			t.Motors.SkipInit = true
			t.Adc.SkipInit = true
		}

		t.sequencer.i2cQueue = make(chan *I2cRequest, t.I2cRequestQueue)
		if !t.NoI2cSequencer {
			go t.sequencer.handleI2cRequests()
		}

		// Prepare Usb HID library, open FT260 device
		if err := hid.Init(); err != nil {
			return err
		}
		usb, err := ft260.OpenPath(t.UsbDevice)
		if err != nil {
			return err
		}
		t.usb = usb
		t.sequencer.usb = t.usb
		t.Motors.bus = t.Bus()
		t.Leds.bus = t.Bus()
		t.Adc.bus = t.Bus()

		// Configure and validate system settings
		if err := t.validateFt260ChipCode(); err != nil {
			return err
		}
		if err := t.configureFt260(); err != nil {
			return err
		}
		if err := t.validateFt260(); err != nil {
			return err
		}
	}
	return nil
}

func (t *Tank) InitI2cPeripherals() error {
	if err := t.Motors.Init(); err != nil {
		return err
	}
	if err := t.Leds.Init(); err != nil {
		return err
	}
	if err := t.Adc.Init(); err != nil {
		return err
	}
	return nil
}

func (t *Tank) Bus() ft260.I2cBus {
	if t.Dummy {
		return new(dummyI2cBus)
	} else if t.NoI2cSequencer {
		return t.usb
	} else {
		return &t.sequencer
	}
}

func (t *Tank) Cleanup() {
	if err := t.Motors.Set(0, 0); err != nil {
		log.Errorf("Cleanup: Failed to disable motors: %v", err)
	}
	if err := t.Leds.DisableAll(); err != nil {
		log.Errorf("Cleanup: Failed to disabled LEDs: %v", err)
	}
	if err := hid.Shutdown(); err != nil {
		log.Errorf("Cleanup: Failed to stop USB HID device: %v", err)
	}
	if err := t.usb.Close(); err != nil {
		log.Errorf("Cleanup: Failed to close USB connection: %v", err)
	}
}

func (t *Tank) validateFt260ChipCode() error {
	var code ft260.ReportChipCode
	if err := t.usb.Read(&code); err != nil {
		return err
	}
	if code.ChipCode != ft260.FT260_CHIP_CODE {
		return fmt.Errorf("Unexpected chip code %04x (expected 0%04x)", code.ChipCode, ft260.FT260_CHIP_CODE)
	}
	return nil
}

func (t *Tank) configureFt260() (err error) {
	t.writeConfigValue(&err, ft260.SetSystemSetting_Clock, ft260.Clock48MHz)
	t.writeConfigValue(&err, ft260.SetSystemSetting_I2CReset, nil) // Reset i2c bus in case it was disturbed
	t.writeConfigValue(&err, ft260.SetSystemSetting_I2CSetClock, uint16(t.I2cFreq))
	t.writeConfigValue(&err, ft260.SetSystemSetting_GPIO_2, ft260.GPIO_2_Normal) // Set all GPIO pins to normal operation
	t.writeConfigValue(&err, ft260.SetSystemSetting_GPIO_A, ft260.GPIO_A_Normal)
	t.writeConfigValue(&err, ft260.SetSystemSetting_GPIO_G, ft260.GPIO_G_Normal)
	t.writeConfigValue(&err, ft260.SetSystemSetting_EnableWakeupInt, false)
	return
}

func (t *Tank) writeConfigValue(outErr *error, address byte, val interface{}) {
	if *outErr == nil {
		*outErr = t.usb.Write(&ft260.SetSystemStatus{
			Request: address,
			Value:   val,
		})
	}
}

func (t *Tank) validateFt260() error {
	var status ft260.ReportSystemStatus
	if err := t.usb.Read(&status); err != nil {
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
		return fmt.Errorf("FT260: unexpected clock value %v (expected %v)", status.EnableWakeupInt, false)
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
	if err := t.usb.Read(&i2cStatus); err != nil {
		return err
	}
	if i2cStatus.BusSpeed != uint16(t.I2cFreq) {
		return fmt.Errorf("FT260: unexpected I2C bus speed %v (expected %v)", i2cStatus.BusSpeed, t.I2cFreq)
	}
	return nil
}
