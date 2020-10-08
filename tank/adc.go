package tank

import (
	"github.com/antongulenko/tank/ads1115"
	"github.com/antongulenko/tank/ft260"
	log "github.com/sirupsen/logrus"
)

// Measure diff AIN0 to AIN3 continuously in 0..6V, comparator disabled
const adcConfig = ads1115.CONFIG_MUX_03 | ads1115.CONFIG_DR_32 | ads1115.CONFIG_PGA_6V | ads1115.CONFIG_COMP_QUE_OFF

type Adc struct {
	bus ft260.I2cBus

	BatteryMin float64
	BatteryMax float64

	I2cAddr  byte
	Dummy    bool
	SkipInit bool
}

func (a *Adc) Init() error {
	if a.Dummy || a.SkipInit {
		log.Println("Skipping initialization of ADC")
		return nil
	} else {
		log.Printf("Initializing ADC device at %#02x...", a.I2cAddr)
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
		return a.BatteryMax, nil
	}
	val, err := ads1115.ReadRegisterDirectly(a.bus, a.I2cAddr)
	if err != nil {
		return 0, err
	}
	return float64(val) * ads1115.CONVERT_6V, nil
}

func (a *Adc) ConvertVoltageToPercentage(voltage float64) float64 {
	if voltage < a.BatteryMin {
		return 0
	}
	if voltage > a.BatteryMax {
		return 1
	}
	return voltage / (a.BatteryMax - a.BatteryMin)
}

func (a *Adc) GetBatteryPercentage() (float64, error) {
	val, err := a.GetBatteryVoltage()
	if err != nil {
		return 0, err
	}
	return a.ConvertVoltageToPercentage(val), nil
}
