
#include <avr/io.h>
#include <avr/interrupt.h>

static inline void configure() {
   // Configure all pins as input without internal pull-ups
   DDRA = PORTA = 0;
   DDRB = PORTB = 0;
   DDRC = PORTC = 0;
   DDRD = PORTD = 0;

   // Enable I2C, disable all other special pin functions
   // Disable interrupts except I2C


   // Reduce power consumption (shut down all but TWI: timers 0-2, USART 0-1, ADC, SPI)
   PRR0 = ~PRTWI;
   PRR1 = PRTIM3;

   // Avoid accidental sleep
   SMCR = 0;

   // Clear reset condition (no need to read)
   MCUSR = 0;

   // Disable watchdog interrupts
   WDTCSR = 0;

   // Disable pin interrupts and external interrupts
   PCICR = 0;
   EIMSK = 0;
   PCMSK0 = PCMSK1 = PCMSK2 = PCMSK3 = 0;
}

// Update lookup table for binary decoder
/*
a   b   A   B   VALUE(abAB)  CHANGE
=========================================
0   0   0   0       0           0
0   0   0   1       1           -1
0   0   1   0       2           1
0   0   1   1       3           0 (illegal)
0   1   0   0       4           1
0   1   0   1       5           0
0   1   1   0       6           0 (illegal)
0   1   1   1       7           -1
1   0   0   0       8           -1
1   0   0   1       9           0 (illegal)
1   0   1   0       10          0
1   0   1   1       11          1
1   1   0   0       12          0 (illegal)
1   1   0   1       13          1
1   1   1   0       14          -1
1   1   1   1       15          0
*/

// TODO optimize this into a switch statement?
uint8_t decode_update_table[] = {
   0, -1, 1, 0, 1, 0, 0, -1, -1, 0, 0, 1, 0, 1, -1, 0
};

static inline void decode_pin_pair(uint8_t old, uint8_t new, volatile uint32_t *counter) {
   uint8_t val = (old & 0xC) | (new & 0x3);
   uint8_t change = decode_update_table[val];
   if (change != 0) *counter += (uint32_t) change; // Not sure if the check is necessary for performance
}

static inline void decode_port(uint8_t old, uint8_t new, volatile uint32_t *counters) {
   decode_pin_pair(old<<2, new   , counters+0); // pin/bit 0 and 1
   decode_pin_pair(old   , new>>2, counters+1); // pin/bit 2 and 3
   decode_pin_pair(old>>2, new>>4, counters+2); // pin/bit 4 and 5
   decode_pin_pair(old>>4, new>>6, counters+3); // pin/bit 6 and 7
}

volatile uint32_t countersA[4] = {0};
volatile uint32_t countersB[4] = {0};
volatile uint32_t countersC[4] = {0};
volatile uint32_t countersD[4] = {0};

int main() {
   configure();
   sei(); // Initialization finished, enable interrupts

   uint8_t pinA = PINA;
   uint8_t pinB = PINB;
   uint8_t pinC = PINC;
   uint8_t pinD = PIND;

   // Endlessly loop and decode pairs of input pins
   while (1) {
      uint8_t new = PINA;
      decode_port(pinA, new, countersA);
      pinA = new;

      new = PINB;
      decode_port(pinB, new, countersB);
      pinB = new;

      new = PINC;
      decode_port(pinC, new, countersC);
      pinC = new;

      new = PIND;
      decode_port(pinD, new, countersD);
      pinD = new;
   }
}
