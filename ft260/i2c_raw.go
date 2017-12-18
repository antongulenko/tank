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
	I2C_StatusControllerBusy = byte(1 << iota)
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

func I2cMasterCodeString(code byte) string {
	switch code {
	case I2C_MasterNone:
		return "Nothing"
	case I2C_MasterStart:
		return "Start"
	case I2C_MasterRepStart:
		return "Repeated Start"
	case I2C_MasterStop:
		return "Stop"
	case I2C_MasterStartStop:
		return "Start + Stop"
	default:
		return fmt.Sprintf("Unknown I2C Master code %v", code)
	}
}

// Result of ReportID_I2CStatus Feature In
type ReportI2cStatus struct {
	BusStatus byte   // Bitmask of I2C_Status...
	BusSpeed  uint16 // 2 byte: LSB+MSB
	// 1 reserved
}

func (r *ReportI2cStatus) ReportID() byte {
	return ReportID_I2CStatus
}

func (r *ReportI2cStatus) ReportLen() int {
	return 4
}

func (r *ReportI2cStatus) Unmarshall(b []byte) error {
	r.BusStatus = b[0]
	r.BusSpeed = uint16(b[1]) + uint16(b[2])<<8
	return nil
}

// Data of ReportID_I2CRead Interrupt Out
type OperationI2cRead struct {
	SlaveAddr byte   // 0..127
	Condition byte   // I2C_Master...
	Len       uint16 // data length (little endian)
}

func (r *OperationI2cRead) IsDataReport() bool {
	return true
}

func (r *OperationI2cRead) ReportID() byte {
	return ReportID_I2CRead
}

func (r *OperationI2cRead) ReportLen() int {
	return 4
}

func (r *OperationI2cRead) Marshall(b []byte) error {
	if r.SlaveAddr&0x80 != 0 {
		return fmt.Errorf("Invalid I2C slave address: %02x", r.SlaveAddr)
	}
	b[0] = r.SlaveAddr
	b[1] = r.Condition
	b[2], b[3] = byte(r.Len), byte(r.Len>>8)
	return nil
}

// Data of ReportID_I2CInOut Interrupt Out
type OperationI2cWrite struct {
	SlaveAddr byte // 0..127
	Condition byte // I2C_Master...
	// 1 byte payload len
	Payload []byte
}

func (r *OperationI2cWrite) IsDataReport() bool {
	return true
}

func (r *OperationI2cWrite) ReportID() byte {
	return ReportID_I2CInOut + byte(len(r.Payload))/4
}

func (r *OperationI2cWrite) ReportLen() int {
	return len(r.Payload) + 3
}

func (r *OperationI2cWrite) Marshall(b []byte) error {
	if len(r.Payload) > I2CMaxPayload {
		return fmt.Errorf("Payload len %v exceeds maximum size of %v", len(r.Payload), I2CMaxPayload)
	}
	if r.SlaveAddr&0x80 != 0 {
		return fmt.Errorf("Invalid I2C slave address: %02x", r.SlaveAddr)
	}
	b[0] = r.SlaveAddr
	b[1] = r.Condition
	b[2] = byte(len(r.Payload))
	copy(b[3:], r.Payload)
	return nil
}

// Data of ReportID_I2CInOut Interrupt In
type OperationI2cInput struct {
	// 1 byte payload length
	Data []byte
}

func (r *OperationI2cInput) IsDataReport() bool {
	return true
}

func (r *OperationI2cInput) IsVariableSize() bool {
	return true
}

func (r *OperationI2cInput) IsVariableReportID() bool {
	return true
}

func (r *OperationI2cInput) ReportID() byte {
	// The report ID probably does not matter here
	return ReportID_I2CInOut
}

func (r *OperationI2cInput) ReportLen() int {
	return I2CMaxPayload - 1 // Max possible report length
}

func (r *OperationI2cInput) Unmarshall(d []byte) error {
	l := d[0]
	if len(d) < int(l)+1 {
		return fmt.Errorf("Short I2C read (%v, needed at least %v)", len(d), l+1)
	}
	copy(r.Data, d[1:1+l])
	return nil
}
