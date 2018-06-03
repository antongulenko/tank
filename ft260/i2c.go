package ft260

import (
	"bytes"
	"fmt"
	"time"
)

var (
	// This results in an I2C operation timeout of 500 milliseconds (plus USB transfer time)
	I2cWaitTime  = 50 * time.Microsecond
	I2cNumChecks = 5000
)

type I2cError struct {
	TimedOut             bool
	BusStatus            byte
	OperationTime        time.Duration
	OperationDescription string
}

func (e *I2cError) Error() string {
	var errBuf bytes.Buffer
	if e.BusStatus&I2C_StatusNoSlaveAck != 0 {
		errBuf.WriteString("no slave ack")
	}
	if e.BusStatus&I2C_StatusNoDataAck != 0 {
		if errBuf.Len() > 0 {
			errBuf.WriteString(", ")
		}
		errBuf.WriteString("no data ack")
	}
	if e.BusStatus&I2C_StatusArbitrationLost != 0 {
		if errBuf.Len() > 0 {
			errBuf.WriteString(", ")
		}
		errBuf.WriteString("arbitration lost")
	}
	if e.BusStatus&I2C_StatusBusBusy != 0 {
		if errBuf.Len() > 0 {
			errBuf.WriteString(", ")
		}
		errBuf.WriteString("bus busy")
	}
	if e.TimedOut {
		if errBuf.Len() > 0 {
			errBuf.WriteString(", ")
		}
		errBuf.WriteString("I2C operation timed out")
	}
	if errBuf.Len() == 0 {
		errBuf.WriteString("Unknown I2C error")
	}
	fmt.Fprintf(&errBuf, " (duration: %v, bus status: %02x)", e.OperationTime, e.BusStatus)
	if e.OperationDescription != "" {
		fmt.Fprintf(&errBuf, " (%v)", e.OperationDescription)
	}
	return "I2C: " + errBuf.String()
}

func (d *Ft260) I2cWrite(addr byte, data ...byte) error {
	return d.i2cWrite(addr, I2C_MasterStartStop, false, data)
}

func (d *Ft260) I2cRead(addr byte, data []byte) error {
	return d.i2cRead(addr, I2C_MasterStartStop, false, data)
}

func (d *Ft260) I2cWriteRead(addr byte, out, in []byte) error {
	err := d.i2cWrite(addr, I2C_MasterStart, true, out) // No STOP!
	if err == nil {
		err = d.i2cRead(addr, I2C_MasterStartStop, false, in)
	}
	return err
}

func (d *Ft260) I2cGet(addr byte, registerAddr byte, size int) ([]byte, error) {
	send := []byte{registerAddr}
	receive := make([]byte, size)
	err := d.I2cWriteRead(addr, send, receive)
	return receive, err
}

func (d *Ft260) i2cWrite(addr byte, condition byte, busBusy bool, data []byte) error {
	op := OperationI2cWrite{
		SlaveAddr: addr,
		Condition: condition,
		Payload:   data,
	}
	err := d.Write(&op)
	if err != nil {
		return err
	}
	err = d.i2cWait(busBusy)
	return d.extendError(err, "writing", addr, condition, data)
}

func (d *Ft260) i2cRead(addr byte, condition byte, busBusy bool, data []byte) error {
	op := OperationI2cRead{
		SlaveAddr: addr,
		Condition: condition,
		Len:       uint16(len(data)),
	}
	err := d.Write(&op)
	if err != nil {
		return err
	}
	if err := d.i2cWait(busBusy); err != nil {
		return d.extendError(err, "reading", addr, condition, data)
	}

	op2 := OperationI2cInput{
		Data: data,
	}
	return d.Read(&op2)
}

func (d *Ft260) extendError(err error, operation string, addr byte, condition byte, data []byte) error {
	if err != nil {
		desc := fmt.Sprintf("I2C Master code %v, %s %v byte from %02x", I2cMasterCodeString(condition), operation, len(data), addr)
		if i2cErr, ok := err.(*I2cError); ok {
			i2cErr.OperationDescription = desc
		} else {
			err = fmt.Errorf("%v (%v)", err, desc)
		}
	}
	return err
}

const i2c_any_error = I2C_StatusError | I2C_StatusNoSlaveAck | I2C_StatusNoDataAck | I2C_StatusArbitrationLost

func (d *Ft260) i2cWait(busBusy bool) error {
	start := time.Now()
	var op ReportI2cStatus
	expectedStatus := I2C_StatusControllerIdle
	if busBusy {
		expectedStatus = I2C_StatusBusBusy
	}
	for i := 0; i < I2cNumChecks; i++ {
		if err := d.Read(&op); err != nil {
			return fmt.Errorf("Failed to check I2C status while waiting for operation to complete: %v", err)
		}
		s := op.BusStatus

		// Wait until the I2C controller is finished
		if s&I2C_StatusControllerBusy != 0 {
			continue
		}
		if s&i2c_any_error != 0 {
			return &I2cError{
				OperationTime: time.Now().Sub(start),
				BusStatus:     op.BusStatus,
			}
		}
		if s == expectedStatus {
			return nil
		}
		time.Sleep(I2cWaitTime)
	}
	return &I2cError{
		TimedOut:      true,
		OperationTime: time.Now().Sub(start),
		BusStatus:     op.BusStatus,
	}
}

type I2cBus interface {
	I2cWrite(addr byte, data ...byte) error
	I2cRead(addr byte, data []byte) error
	I2cWriteRead(addr byte, out, in []byte) error
	I2cGet(addr byte, registerAddr byte, size int) ([]byte, error)
}

type I2cScanner struct {
}

func I2cScan(bus I2cBus) ([]byte, error) {
	// Reserved addresses (https://www.i2c-bus.org/addressing):
	// 0x00: start byte, general call
	// 0x01: CBUS Address
	// 0x02: Different bus formats
	// 0x03: Future purposes
	// 0x04 - 0x07: High speed master codes
	// <<----------------------------------- Normal address range (112 addresses)
	// 0x78 - 0x7B: 10 bit slave addressing
	// 0x7C - 0x7F: Future purposes
	return I2cScanRange(bus, 0x08, 0x77)
}

const i2c_status_error_mask = I2C_StatusError | I2C_StatusNoSlaveAck | I2C_StatusNoDataAck | I2C_StatusArbitrationLost

func I2cScanRange(bus I2cBus, from, to byte) ([]byte, error) {
	var slaves []byte
	if from > to {
		return nil, fmt.Errorf("Illegal address scan range %v - %v", from, to)
	}
	for addr := from; addr <= to; addr++ {
		err := bus.I2cRead(addr, []byte{0})
		if err != nil {
			if i2cErr, ok := err.(*I2cError); ok {
				if !i2cErr.TimedOut &&
					0 == i2cErr.BusStatus&(I2C_StatusBusBusy|I2C_StatusArbitrationLost|I2C_StatusControllerBusy) &&
					0 != i2cErr.BusStatus&I2C_StatusError {
					// Slave NACK or data NACK are expected
					continue
				}
			}
			return nil, err
		}
		slaves = append(slaves, addr)
	}
	return slaves, nil
}
