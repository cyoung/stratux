#!/usr/bin/env python

# Temporary (until sunxi PWM is set up) script. Turns fan on 100% on Stratux start.


from pyA13.gpio import gpio
from pyA13.gpio import port

gpio.init() #Initialize module. Always called first.

gpio.setcfg(port.PB2, gpio.OUTPUT) #Configure LED1 as output.

gpio.output(port.PB2, gpio.HIGH) #Turn on fan.
