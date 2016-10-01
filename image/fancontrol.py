#!/usr/bin/python
# @Author Ryan Dewsbury (helno)
#
# This script throttles a fan based on CPU temperature.
#
# It expects a fan that's externally powered, and uses GPIO pin 11 for control.

import RPi.GPIO as GPIO
import time
import os

from daemon import runner

class FanControl():
    # Return CPU temperature as float
    def getCPUtemp(self):
        cTemp = os.popen('vcgencmd measure_temp').readline()
        return float(cTemp.replace("temp=","").replace("'C\n",""))

    def __init__(self):
        self.stdin_path = '/dev/null'
        self.stdout_path = '/var/log/fancontrol.log'
        self.stderr_path = '/var/log/fancontrol.log'
        self.pidfile_path = '/var/run/fancontrol.pid'
        self.pidfile_timeout = 5
    def run(self):
        GPIO.setmode(GPIO.BOARD)
        GPIO.setup(12, GPIO.OUT)
        GPIO.setwarnings(False)
        p=GPIO.PWM(12, 1000)
        PWM = 50
        while True:
            CPU_temp = self.getCPUtemp()
            if CPU_temp > 40.5:
                PWM = min(max(PWM + 1, 0), 100)
                p.start(PWM)
            elif CPU_temp < 39.5:
                PWM = min(max(PWM - 1, 0), 100)
                p.start(PWM)
            time.sleep(5)
        GPIO.cleanup()

fancontrol = FanControl()
daemon_runner = runner.DaemonRunner(fancontrol)
daemon_runner.do_action()
