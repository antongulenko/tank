package ft260

import (
	"errors"
	"fmt"
)

const (
	ReportID_ChipCode      = 0xA0 // Feature In
	ReportID_SystemSetting = 0xA1 // Feature In/Out
)

// Requests for ReportID_SystemSetting Feature Out
const (
	SetSystemSetting_Clock               = 0x01 // Clock...
	SetSystemSetting_EnableWakeupInt     = 0x05 // bool
	SetSystemSetting_Interrupt           = 0x0A // 2 byte: InterruptTrigger..., InterruptLevelDuration...
	SetSystemSetting_SuspendOutActiveLow = 0x0B // bool

	SetSystemSetting_GPIO_2 = 0x06 // GPIO_2_...
	SetSystemSetting_GPIO_A = 0x08 // GPIO_A_...
	SetSystemSetting_GPIO_G = 0x09 // GPIO_G_...

	SetSystemSetting_I2CReset    = 0x20 // <empty>
	SetSystemSetting_I2CSetClock = 0x22 // LSB+MSB of clock speed (60K-3400K bps)

	SetSystemSetting_Uart                    = 0x3  // Uart...
	SetSystemSetting_EnableUartDcdRi         = 0x07 // bool
	SetSystemSetting_EnableUartRiWakeup      = 0x0C // bool
	SetSystemSetting_UartRiWakeupFallingEdge = 0x0D // bool
	SetSystemSetting_UartReset               = 0x40 // <empty>
	SetSystemSetting_ConfigureUart           = 0x41 // 9 byte config
	SetSystemSetting_UartBaudRate            = 0x42 // 4 byte
	SetSystemSetting_UartDataBits            = 0x43 // 7..8 (number of bits)
	SetSystemSetting_UartParity              = 0x44 // 0..4 (no/odd/even/high/low parity)
	SetSystemSetting_UartStopBits            = 0x45 // 0 (one stop bit)/2 (two stop bits)
	SetSystemSetting_UartBreaking            = 0x46 // bool
	SetSystemSetting_UartXonXoff             = 0x49 // 2 byte (xon char, xoff char)
)

const (
	Clock12MHz = byte(0)
	Clock24MHz = byte(1)
	Clock48MHz = byte(2)

	// ReportSystemStatus.UartMode
	UartOff           = byte(0)
	UartRTS_CTS       = byte(1)
	UartDTR_DSR       = byte(2)
	UartXON_XOFF      = byte(3)
	UartNoFlowControl = byte(4)

	GPIO_2_Normal    = byte(0)
	GPIO_2_Suspout   = byte(1)
	GPIO_2_ActiveLow = byte(2)
	GPIO_2_TxLed     = byte(4)

	GPIO_A_Normal   = byte(0)
	GPIO_A_TxActive = byte(3)
	GPIO_A_TxLed    = byte(4)

	GPIO_G_Normal    = byte(0)
	GPIO_G_ActiveLow = byte(2)
	GPIO_G_RxLed     = byte(5)
	GPIO_G_BcdDet    = byte(6)

	InterruptTriggerRisingEdge  = byte(0)
	InterruptTriggerLevelHigh   = byte(1)
	InterruptTriggerFallingEdge = byte(2)
	InterruptTriggerLevelLow    = byte(3)

	InterruptLevelDuration1ms  = byte(1)
	InterruptLevelDuration5ms  = byte(2)
	InterruptLevelDuration30ms = byte(3)
)

// Result of ReportID_ChipCode Feature In
type ReportChipCode struct {
	ChipCode uint32 // 02600200
	// 8 reserved byte
}

func (r *ReportChipCode) ReportID() byte {
	return ReportID_ChipCode
}

func (r *ReportChipCode) ReportLen() int {
	return 12
}

func (r *ReportChipCode) Unmarshall(b []byte) error {
	r.ChipCode = uint32(b[0]) + uint32(b[1])<<8 + uint32(b[2])<<16 + uint32(b[3])<<24
	return nil
}

// Result of ReportID_SystemSetting Feature In
type ReportSystemStatus struct {
	ChipMode            byte // Bit 0: DCNF0, Bit 1: DCNF1
	Clock               byte // 0..2 (Clock...MHz)
	Suspended           bool
	PowerStatus         bool // Device Ready?
	I2CEnable           bool
	UartMode            byte // 0..4 (Uart...)
	HidOverI2cEnable    bool
	GPIO2Function       byte // 0..4 (GPIO_2_...)
	GPIOAFunction       byte // 0..4 (GPIO_A_...)
	GPIOGFunction       byte // 0..6 (GPIO_G_...)
	SuspendOutActiveLow bool
	EnableWakeupInt     bool // If disabled: pin acts as GPIO3
	InterruptCond       byte // See Interrupt...() methods
	EnablePowerSaving   bool // Enabled: reduce clock to 30kHz after 5 sec idle
	// 4 reserved byte
}

func (s ReportSystemStatus) InterruptTriggerCondition() byte {
	return s.InterruptCond & 0x3 // 0..4 (InterruptTrigger...)
}

func (s ReportSystemStatus) InterruptLevelDuration() byte {
	return (s.InterruptCond >> 2) & 0x3 // InterruptLevelDuration...
}

func (r *ReportSystemStatus) ReportID() byte {
	return ReportID_SystemSetting
}

func (r *ReportSystemStatus) ReportLen() int {
	// This should be 19 byte, but the device returns an error for less than 25...
	return 24
}

func (r *ReportSystemStatus) Unmarshall(b []byte) (err error) {
	r.ChipMode = b[0]
	r.Clock = b[1]
	r.Suspended = _readBool(b, 2, &err)
	r.PowerStatus = _readBool(b, 3, &err)
	r.I2CEnable = _readBool(b, 4, &err)
	r.UartMode = b[5]
	r.HidOverI2cEnable = _readBool(b, 6, &err)
	r.GPIO2Function = b[7]
	r.GPIOAFunction = b[8]
	r.GPIOGFunction = b[9]
	r.SuspendOutActiveLow = _readBool(b, 10, &err)
	r.EnableWakeupInt = _readBool(b, 11, &err)
	r.InterruptCond = b[12]
	r.EnablePowerSaving = _readBool(b, 13, &err)
	return
}

type SetSystemStatus struct {
	Request byte
	Value   interface{}
}

func (r *SetSystemStatus) ReportID() byte {
	return ReportID_SystemSetting
}

func (r *SetSystemStatus) ReportLen() int {
	res := 1
	switch r.Request {
	case SetSystemSetting_I2CReset, SetSystemSetting_UartReset:
		// No payload

	case SetSystemSetting_Clock, SetSystemSetting_EnableWakeupInt, SetSystemSetting_SuspendOutActiveLow,
		SetSystemSetting_GPIO_2, SetSystemSetting_GPIO_A, SetSystemSetting_GPIO_G, SetSystemSetting_Uart,
		SetSystemSetting_EnableUartDcdRi, SetSystemSetting_EnableUartRiWakeup, SetSystemSetting_UartRiWakeupFallingEdge,
		SetSystemSetting_UartDataBits, SetSystemSetting_UartParity, SetSystemSetting_UartStopBits,
		SetSystemSetting_UartBreaking:
		res++

	case SetSystemSetting_UartXonXoff, SetSystemSetting_Interrupt, SetSystemSetting_I2CSetClock:
		res += 2
	case SetSystemSetting_ConfigureUart:
		res += 9
	case SetSystemSetting_UartBaudRate:
		res += 4
	default:
		panic(fmt.Sprintf("Illegal system status request ID: %v", r.Request))
	}
	return res
}

func (r *SetSystemStatus) Marshall(b []byte) error {
	b[0] = r.Request
	switch r.Request {
	case SetSystemSetting_I2CReset, SetSystemSetting_UartReset:
		// No payload

	case SetSystemSetting_Clock, SetSystemSetting_GPIO_2, SetSystemSetting_GPIO_A,
		SetSystemSetting_GPIO_G, SetSystemSetting_Uart, SetSystemSetting_UartDataBits,
		SetSystemSetting_UartParity, SetSystemSetting_UartStopBits:
		// Single-byte payload
		val, ok := r.Value.(byte)
		if !ok {
			return fmt.Errorf("System Setting Request ID %02x expects type %T, but got value of type %T (%v)", r.Request, byte(0), r.Value, r.Value)
		}
		b[1] = val

	case SetSystemSetting_EnableWakeupInt, SetSystemSetting_SuspendOutActiveLow, SetSystemSetting_EnableUartDcdRi,
		SetSystemSetting_EnableUartRiWakeup, SetSystemSetting_UartRiWakeupFallingEdge, SetSystemSetting_UartBreaking:
		// Bool payload
		val, ok := r.Value.(bool)
		if !ok {
			return fmt.Errorf("System Setting Request ID %02x expects type %T, but got value of type %T (%v)", r.Request, false, r.Value, r.Value)
		}
		if val {
			b[1] = 1
		} else {
			b[1] = 0
		}

	case SetSystemSetting_Interrupt:
		//= 0x0A // 2 byte: InterruptTrigger..., InterruptLevelDuration...
		return errors.New("Configuring intterupt is not implemented")
	case SetSystemSetting_UartXonXoff:
		//= 0x49 // 2 byte (xon char, xoff char)
		return errors.New("Configuring Uart XON/XOFF is not implemented")
	case SetSystemSetting_I2CSetClock:
		val, ok := r.Value.(uint16)
		if !ok {
			return fmt.Errorf("System Setting Request ID %v expects type %T, but got value of type %T (%v)", r.Request, uint16(0), r.Value, r.Value)
		}
		b[1], b[2] = byte(val), byte(val>>16)
	case SetSystemSetting_ConfigureUart:
		return errors.New("Configuring UART is not implemented")
	case SetSystemSetting_UartBaudRate:
		return errors.New("Configuring UART baud rate is not implemented")
	default:
		return fmt.Errorf("Unknown system setting request ID: %v", r.Request)
	}
	return nil
}
