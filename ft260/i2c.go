package ft260

import (
	"bytes"
	"errors"
	"fmt"
	"time"
)

var (
	// This results in an I2C operation timeout of 500 milliseconds (plus USB transfer time)
	I2cWaitTime  = 50 * time.Microsecond
	I2cNumChecks = 10000
)

const i2c_any_error = I2C_StatusError | I2C_StatusNoSlaveAck | I2C_StatusNoDataAck |
	I2C_StatusArbitrationLost | I2C_StatusBusBusy

type I2cError struct {
	TimedOut        bool
	BusBusy         bool
	NoSlaveAck      bool
	NoDataAck       bool
	ArbitrationLost bool

	OperationTime time.Duration
}

func (e I2cError) Error() string {
	var errBuf bytes.Buffer
	if e.NoSlaveAck {
		errBuf.WriteString("no slave ack")
	}
	if e.NoDataAck {
		if errBuf.Len() > 0 {
			errBuf.WriteString(", ")
		}
		errBuf.WriteString("no data ack")
	}
	if e.ArbitrationLost {
		if errBuf.Len() > 0 {
			errBuf.WriteString(", ")
		}
		errBuf.WriteString("arbitration lost")
	}
	if e.BusBusy {
		if errBuf.Len() > 0 {
			errBuf.WriteString(", ")
		}
		errBuf.WriteString("bus busy")
	}
	if e.TimedOut {
		if errBuf.Len() > 0 {
			errBuf.WriteString(", ")
		}
		errBuf.WriteString("operation timed out")
	}
	if errBuf.Len() == 0 {
		errBuf.WriteString("Unknown I2C error")
	}
	fmt.Fprintf(&errBuf, " (duration: %v)", e.OperationTime)
	return errBuf.String()
}

func (d *Ft260) I2cWrite(addr byte, data []byte) error {
	op := OperationI2cWrite{
		SlaveAddr: addr,
		Condition: I2C_MasterStartStop,
		Payload:   data,
	}
	err := d.Write(&op)
	if err != nil {
		return err
	}
	return d.i2cWait()
}

func (d *Ft260) I2cRead(addr byte, data []byte) error {
	op := OperationI2cRead{
		SlaveAddr: addr,
		Condition: I2C_MasterStartStop,
		Len:       uint16(len(data)),
	}
	err := d.Write(&op)
	if err != nil {
		return err
	}
	if err := d.i2cWait(); err != nil {
		return err
	}

	op2 := OperationI2cInput{
		Data: data,
	}
	return d.Read(&op2)
}

func (d *Ft260) I2cWriteRead(addr byte, out, in []byte) error {
	return errors.New("I2c write+read not implemented")
}

func (d *Ft260) i2cWait() error {
	start := time.Now()
	var op ReportI2cStatus
	for i := 0; i < I2cNumChecks; i++ {
		if err := d.Read(&op); err != nil {
			return fmt.Errorf("Failed to check I2C status while waiting for operation to complete: %v", err)
		}
		s := op.BusStatus

		// Wait until the I2C controller is idle
		if s&I2C_StatusControllerBusy != 0 || s&I2C_StatusControllerIdle == 0 {
			continue
		}

		if s&i2c_any_error == 0 {
			return nil
		} else {
			return I2cError{
				BusBusy:         s&I2C_StatusBusBusy != 0,
				NoSlaveAck:      s&I2C_StatusNoSlaveAck != 0,
				NoDataAck:       s&I2C_StatusNoDataAck != 0,
				ArbitrationLost: s&I2C_StatusArbitrationLost != 0,
				OperationTime:   time.Now().Sub(start),
			}
		}
		time.Sleep(I2cWaitTime)
	}
	return I2cError{
		TimedOut:      true,
		OperationTime: time.Now().Sub(start),
	}
}
