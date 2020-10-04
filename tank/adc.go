package tank

import (
	"github.com/antongulenko/tank/ads1115"
	"github.com/antongulenko/tank/ft260"
	log "github.com/sirupsen/logrus"
)

// Measure diff AIN0 to AIN3 continuously in 0..6V, comparator disabled
const adcConfig = ads1115.CONFIG_MUX_03 | ads1115.CONFIG_DR_32 | ads1115.CONFIG_PGA_6V | ads1115.CONFIG_COMP_QUE_1

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

func (a *Adc) GetBatteryVoltage() (int16, error) {
	if a.Dummy {
		return int16(a.BatteryMax), nil
	}
	return ads1115.ReadRegisterDirectly(a.bus, a.I2cAddr)
}

func (a *Adc) ConvertVoltageToPercentage(voltage int16) float64 {
	if int64(voltage) < a.BatteryMin {
		return 0
	}
	if int64(voltage) > a.BatteryMax {
		return 1
	}
	return float64(voltage) / float64(a.BatteryMax-a.BatteryMin)
}

func (a *Adc) GetBatteryPercentage() (float64, error) {
	val, err := a.GetBatteryVoltage()
	if err != nil {
		return 0, err
	}
	return a.ConvertVoltageToPercentage(val), nil
}
