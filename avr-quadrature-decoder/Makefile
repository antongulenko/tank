
MCU := atmega1284p
FREQ := 20000000
AVRDUDE_COMMAND := avrdude -P usb -c usbasp -p $(MCU)

ALL: decoder.lss decoder.hex size

decoder.elf: decoder.c
	avr-gcc -O3 -Ofast -mmcu=$(MCU) -std=gnu99 -DAVR -DF_CPU=$(FREQ) -fshort-enums -Wall -Wno-missing-braces -o $@ $<

%.hex: %.elf
	avr-objcopy -O ihex -R .eeprom -R .fuse -R .lock -R .signature $< $@

%.lss: %.elf
	avr-objdump -h -S $< > $@

check_device:
	$(AVRDUDE_COMMAND) -n

flash: decoder.hex
	$(AVRDUDE_COMMAND) -e
	$(AVRDUDE_COMMAND) -U flash:w:$<

size: decoder.elf
	avr-size $< -C --mcu=$(MCU) | grep bytes

clean:
	rm -f *.elf *.hex *.lss

.PHONY: size check_device flash clean
