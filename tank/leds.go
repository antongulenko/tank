package tank

import (
	"log"

	"github.com/antongulenko/tank/ft260"
	"github.com/antongulenko/tank/pca9685"
)

const NumLeds = 15

type MainLeds struct {
	bus ft260.I2cBus

	I2cAddr byte

	PwmStart  byte // pca9685.LED0
	pwmOutput pca9685.PwmOutput
}

func (m *MainLeds) Init() error {
	log.Printf("Initializing LED PWM driver at %02x...", m.I2cAddr)
	if err := m.bus.I2cWrite(m.I2cAddr, pca9685.MODE1, pca9685.MODE1_ALLCALL|pca9685.MODE1_AI); err != nil {
		return err
	}
	return m.update(make([]float64, NumLeds)) // Set all Leds to zero
}

func (m *MainLeds) Set(values []float64) error {
	values = m.pwmOutput.FillCurrentState(values)
	return m.update(values)
}

func (m *MainLeds) update(values []float64) error {
	pwmValues := m.pwmOutput.Update(m.PwmStart, values)
	return m.bus.I2cWrite(m.I2cAddr, pwmValues...)
}
