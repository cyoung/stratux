#!/usr/bin/python
# @Author Ryan Dewsbury (helno)
#
# This script throttles a fan based on CPU temperature.
#
# It expects a fan that's externally powered, and uses GPIO pin 11 for control.

import RPi.GPIO as GPIO
import time
import os

# Return CPU temperature as float
def getCPUtemp():
	cTemp = os.popen('vcgencmd measure_temp').readline()
	return float(cTemp.replace("temp=","").replace("'C\n",""))

GPIO.setmode(GPIO.BOARD)
GPIO.setup(11,GPIO.OUT)
GPIO.setwarnings(False)
p=GPIO.PWM(11,1000)
PWM = 50

while True:

	CPU_temp = getCPUtemp()
	if CPU_temp > 40.5:
		PWM = min(max(PWM + 1, 0), 100)
		p.start(PWM)
	elif CPU_temp < 39.5:
		PWM = min(max(PWM - 1, 0), 100)
		p.start(PWM)
	time.sleep(5)

GPIO.cleanup()
