package main

import (
	"flag"

	"github.com/antongulenko/golib"
	"github.com/antongulenko/hid"
	"github.com/antongulenko/tank/ft260"
	log "github.com/sirupsen/logrus"
)

func main() {
	golib.RegisterLogFlags()
	flag.Parse()
	golib.ConfigureLogging()

	golib.Checkerr(hid.Init())
	defer func() {
		golib.Printerr(hid.Shutdown())
	}()

	dev, err := ft260.Open()
	golib.Checkerr(err)
	defer func() {
		golib.Printerr(dev.Close())
	}()

	var code ft260.ReportChipCode
	golib.Checkerr(dev.Read(&code))
	log.Printf("%#v", code)

	var status ft260.ReportSystemStatus
	golib.Checkerr(dev.Read(&status))
	log.Printf("%#v", status)

	var i2cStatus ft260.ReportI2cStatus
	golib.Checkerr(dev.Read(&i2cStatus))
	log.Printf("%#v", i2cStatus)

	golib.Checkerr(dev.Write(&ft260.SetSystemStatus{
		Request: ft260.SetSystemSetting_Clock,
		Value:   ft260.Clock24MHz,
	}))
	golib.Checkerr(dev.Write(&ft260.SetSystemStatus{
		Request: ft260.SetSystemSetting_I2CSetClock,
		Value:   uint16(50000),
	}))

	golib.Checkerr(dev.Read(&i2cStatus))
	log.Printf("%#v", i2cStatus)

	log.Warnln("Now enabling Port A as output")
	golib.Checkerr(dev.I2cWrite(0x20, []byte{0, 0}))

	log.Warnln("Now setting Port A high")
	golib.Checkerr(dev.I2cWrite(0x20, []byte{0x12, 0xFF}))
}
