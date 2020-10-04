package mcp23017

// Default bits all zero, except IODIR

// ============== General IO configuration
// IODIR: 0: output, 1: input
// IPOL: 1: GPIO reflects inverted value of the pin
// GPIO: Reading reads pin values. Writing modifies to OLAT.
// OLAT: Output values ("latches")
// GPPU: 1: enable internal pull-up for input pins (100 kOhm)

// ============== Interrupt configuration
// GPINTEN: 1: enable interrupt-on-change. Pins must also be input.
// DEFVAL: opposite value on input pin will cause interrupt (if INTCON is set)
// INTCON: for interrupt: 0: pins compared to previous value 1: pins compared to DEFVAL
// INTF: (read only) interrupt flags. Cleared when INTCAP or GPIO is read.
// INTCAP: (read only) state of pins when interrupt occurs. Remains unchanged until read (or GPIO is read)

// Register addresses when the BANK bit in IOCON is set (it is cleared by default)
const (
	IODIR_A_BANK = byte(iota)
	IPOL_A_BANK
	GPINTEN_A_BANK
	DEFVAL_A_BANK
	INTCON_A_BANK
	IOCON_BANK
	GPPU_A_BANK
	INTF_A_BANK
	INTCAP_A_BANK
	GPIO_A_BANK
	OLAT_A_BANK
	IODIR_B_BANK
	IPOL_B_BANK
	GPINTEN_B_BANK
	DEFVAL_B_BANK
	INTCON_B_BANK
	_ // IOCON
	GPPU_B_BANK
	INTF_B_BANK
	INTCAP_B_BANK
	GPIO_B_BANK
	OLAT_B_BANK
)

const (
	IODIR_A_PAIRED = byte(iota)
	IODIR_B_PAIRED
	IPOL_A_PAIRED
	IPOL_B_PAIRED
	GPINTEN_A_PAIRED
	GPINTEN_B_PAIRED
	DEFVAL_A_PAIRED
	DEFVAL_B_PAIRED
	INTCON_A_PAIRED
	INTCON_B_PAIRED
	IOCON_PAIRED
	_ // IOCON
	GPPU_A_PAIRED
	GPPU_B_PAIRED
	INTF_A_PAIRED
	INTF_B_PAIRED
	INTCAP_A_PAIRED
	INTCAP_B_PAIRED
	GPIO_A_PAIRED
	GPIO_B_PAIRED
	OLAT_A_PAIRED
	OLAT_B_PAIRED

	// The starting registers for configuring both ports in sequential mode
	IODIR_PAIRED   = IODIR_A_PAIRED
	IPOL_PAIRED    = IPOL_A_PAIRED
	GPINTEN_PAIRED = GPINTEN_A_PAIRED
	DEFVAL_PAIRED  = DEFVAL_A_PAIRED
	INTCON_PAIRED  = INTCON_A_PAIRED
	GPPU_PAIRED    = GPPU_A_PAIRED
	INTF_PAIRED    = INTF_A_PAIRED
	INTCAP_PAIRED  = INTCAP_A_PAIRED
	GPIO_PAIRED    = GPIO_A_PAIRED
	OLAT_PAIRED    = OLAT_A_PAIRED
)

const (
	_                = byte(1 << iota)
	IOCON_BIT_INTPOL // 1: INT pins active-high 0: INT pins active-low
	IOCON_BIT_ODR    // (overrides INTPOL) 1: INT pins are open-drain 0: active output (INTPOL sets polarity)
	IOCON_BIT_HAEN   // Enable hardware address pins (zero otherwise)
	IOCON_BIT_DISSLW // 0: slew rate control for SDA output enabled 1: disabled
	IOCON_BIT_SEQOP  // 0: sequential operation enabled 1: disabled (address stays after read/write)
	IOCON_BIT_MIRROR // 0: INT pins not mirrored 1: INT pins mirrored (both high if one is high)
	IOCON_BIT_BANK   // 1: registers grouped in banks 0: registers paired
)

const (
	ADDRESS     = byte(0x20) // 0010 0000
	MAX_ADDRESS = byte(0x27) // 0010 0111

	// Values for IODIR registers
	INPUT  = byte(0xFF)
	OUTPUT = byte(0x00)
)
