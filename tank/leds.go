package tank

import (
	"math"

	"github.com/antongulenko/tank/ft260"
	"github.com/antongulenko/tank/pca9685"
	log "github.com/sirupsen/logrus"
)

const ledDriverConfig = pca9685.MODE1_ALLCALL | pca9685.MODE1_AI

type MainLeds struct {
	bus ft260.I2cBus

	I2cAddr byte
	Dummy   bool
	NumLeds int

	PwmStart  byte // pca9685.LED0
	pwmOutput pca9685.PwmOutput
}

func (m *MainLeds) Init() error {
	if m.Dummy {
		log.Println("Skipping initialization of dummy leds")
	} else {
		log.Printf("Initializing LED PWM driver at %02x...", m.I2cAddr)
		if err := m.bus.I2cWrite(m.I2cAddr, pca9685.MODE1, ledDriverConfig); err != nil {
			return err
		}
	}
	return m.DisableAll()
}

func (m *MainLeds) SetAll(values []float64) error {
	values = m.pwmOutput.FillCurrentState(values)
	return m.update(m.PwmStart, values)
}

func (m *MainLeds) update(start byte, values []float64) error {
	pwmValues := m.pwmOutput.Update(start, values)
	if m.Dummy {
		log.Printf("Dummy Leds: update at %v to values: %v", start, values)
		return nil
	} else {
		return m.bus.I2cWrite(m.I2cAddr, pwmValues...)
	}
}

func (m *MainLeds) DisableAll() error {
	return m.update(m.PwmStart, make([]float64, m.NumLeds))
}

func (m *MainLeds) Groups() (red, green, yellow LedGroup) {
	red = m.Group(0, 5)    // TODO
	green = m.Group(0, 5)  // TODO
	yellow = m.Group(0, 5) // TODO
	return
}

func (m *MainLeds) Group(from, to byte) LedGroup {
	return LedGroup{m, from, to}
}

func (m *MainLeds) SetRow(from, to byte, val float64) error {
	if to < from {
		to = from
	}
	if val < 0 {
		val = 0
	}
	if val > 1 {
		val = 1
	}
	num := from - to + 1
	fullOn := math.Floor(val / float64(num))
	rest := val - fullOn
	fullOnI := int(fullOn)

	values := make([]float64, num)
	for i := range values {
		if i < fullOnI {
			values[i] = 1
		} else if i == fullOnI {
			values[i] = rest
		} else {
			values[i] = 0
		}
	}
	return m.update(from, values)
}

type LedGroup struct {
	Leds *MainLeds
	From byte
	To   byte
}

func (g *LedGroup) Set(val float64) error {
	return g.Leds.SetRow(g.From, g.To, val)
}
