package tank

import (
	"log"

	"github.com/antongulenko/tank/ads1115"
	"github.com/antongulenko/tank/ft260"
)

// Measure diff AIN0 to AIN3 continuously in 0..6V, comparator disabled
const adcConfig = ads1115.CONFIG_MUX_03 | ads1115.CONFIG_DR_32 | ads1115.CONFIG_PGA_6V | ads1115.CONFIG_COMP_QUE_OFF

type Adc struct {
	bus ft260.I2cBus

	BatteryMin int64
	BatteryMax int64

	I2cAddr byte
	Dummy   bool
}

func (a *Adc) Init() error {
	if a.Dummy {
		log.Println("Skipping initialization of dummy ADC")
		return nil
	} else {
		log.Printf("Initializing ADC device at %02x...", a.I2cAddr)
		err := ads1115.WriteRegister(a.bus, a.I2cAddr, ads1115.REG_CONFIG, adcConfig)
		if err == nil {
			// Configure the address of the register to be read by future reads
			err = a.bus.I2cWrite(a.I2cAddr, ads1115.REG_CONVERSION)
		}
		return err
	}
}

func (a *Adc) GetBatteryVoltage() (float64, error) {
	if a.Dummy {
		return 1, nil
	}
	val, err := ads1115.ReadRegisterDirectly(a.bus, a.I2cAddr)
	if err != nil {
		return 0, err
	}
	if int64(val) < a.BatteryMin {
		return 0, nil
	}
	if int64(val) > a.BatteryMax {
		return 1, nil
	}
	return float64(val) / float64(a.BatteryMax-a.BatteryMin), nil
}
