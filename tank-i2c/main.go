package main

import (
	"github.com/antongulenko/golib"
	"github.com/antongulenko/tank/ft260"
	"github.com/karalabe/hid"
)

func main() {
	dev, err := ft260.Open()
	golib.Checkerr(err)
	defer func() {
		golib.Printerr(dev.Close())
		hid.Shutdown()
	}()

	golib.Checkerr(dev.Write(ft260.SetSystemStatus{
		Request: ft260.SetSystemSetting_Clock,
		Value:   ft260.Clock24MHz,
	}))
	golib.Checkerr(dev.Write(ft260.SetSystemStatus{
		Request: ft260.SetSystemSetting_I2CSetClock,
		Value:   uint16(50000),
	}))
}
