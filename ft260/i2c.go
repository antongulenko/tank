package ft260

import "fmt"

const (
	ReportID_I2CStatus    = 0xC0 // Feature In
	ReportID_I2CRead      = 0xC2 // Output
	ReportID_I2CInOut     = 0xD0 // 0xD0 - 0xDE, Input, Output
	ReportID_I2CInOut_Max = 0xDE
	// Max size of I2C write payload: (1 + Report ID - 0xD0) * 4 byte

	I2CMaxPayload = (1 + ReportID_I2CInOut_Max - ReportID_I2CInOut) * 4
)

const (
	I2C_StatusControllerBusy = 1 << iota
	I2C_StatusError
	I2C_StatusNoSlaveAck
	I2C_StatusNoDataAck
	I2C_StatusArbitrationLost
	I2C_StatusControllerIdle
	I2C_StatusBusBusy
)

const (
	I2C_MasterNone      = 0x0
	I2C_MasterStart     = 0x2
	I2C_MasterRepStart  = 0x3
	I2C_MasterStop      = 0x4
	I2C_MasterStartStop = 0x6
)

// Result of ReportID_I2CStatus Feature In
type ReportI2cStatus struct {
	BusStatus byte   // Bitmask of I2C_Status...
	BusSpeed  uint16 // 2 byte: LSB+MSB
	// 1 reserved
}

func (r ReportI2cStatus) ReportID() byte {
	return ReportID_I2CInOut
}

func (r ReportI2cStatus) ReportLen() int {
	return 5
}

func (r ReportI2cStatus) Unmarshall(b []byte) error {
	r.BusStatus = b[1]
	r.BusSpeed = uint16(b[2]) + uint16(b[3])<<8
	return nil
}

// Data of ReportID_I2CInOut Interrupt Out
type OperationI2cWrite struct {
	SlaveAddr byte // 0..127
	Condition byte // I2C_Master...
	// 1 byte payload len
	Payload []byte
}

func (r OperationI2cWrite) ReportID() byte {
	return ReportID_I2CInOut + byte(len(r.Payload))/4
}

func (r OperationI2cWrite) ReportLen() int {
	return len(r.Payload) + 4
}

func (r OperationI2cWrite) Marshall(b []byte) error {
	if len(r.Payload) > I2CMaxPayload {
		return fmt.Errorf("Payload len %v exceeds maximum size of %v", len(r.Payload), I2CMaxPayload)
	}
	if r.SlaveAddr&0x80 != 0 {
		return fmt.Errorf("Invalid I2C slave address: %02x", r.SlaveAddr)
	}
	b[1] = r.SlaveAddr
	b[2] = r.Condition
	b[3] = byte(len(r.Payload))
	copy(b[4:], r.Payload)
	return nil
}

// Data of ReportID_I2CRead Interrupt Out
type OperationI2cRead struct {
	SlaveAddr byte   // 0..127
	Condition byte   // I2C_Master...
	Len       uint16 // data length (little endian)
}

func (r OperationI2cRead) ReportID() byte {
	return ReportID_I2CRead
}

func (r OperationI2cRead) ReportLen() int {
	return 5
}

func (r OperationI2cRead) Marshall(b []byte) error {
	if r.SlaveAddr&0x80 != 0 {
		return fmt.Errorf("Invalid I2C slave address: %02x", r.SlaveAddr)
	}
	b[1] = r.SlaveAddr
	b[2] = r.Condition
	b[3], b[4] = byte(r.Len), byte(r.Len>>8)
	return nil
}

// Data of ReportID_I2CInOut Interrupt In
type OperationI2cInput struct {
	Len  byte // Payload length
	Data []byte
}

func (r OperationI2cInput) ReportID() byte {
	// TODO The returned report ID will vary based on the payload size...
	return ReportID_I2CRead
}

func (r OperationI2cInput) ReportLen() int {
	// TODO exact report length will vary...
	return 64 // Max possible report length
}
