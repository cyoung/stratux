#!/usr/bin/env python

from luma.core.interface.serial import i2c
from luma.oled.device import ssd1306, sh1106
from luma.core.render import canvas

from PIL import ImageDraw, ImageFont, Image

import urllib2
import json
import time

from daemon import runner

class StratuxScreen():
    def __init__(self):
        self.stdin_path = '/dev/null'
        self.stdout_path = '/var/log/stratux-screen.log'
        self.stderr_path = '/var/log/stratux-screen.log'
        self.pidfile_path = '/var/run/stratux-screen.pid'
        self.pidfile_timeout = 5
    def run(self):
        font2 = ImageFont.truetype('/etc/stratux-screen/CnC_Red_Alert.ttf', 12)
        serial = i2c(port=1, address=0x3c)
        oled = ssd1306(serial)

        with canvas(oled) as draw:
            logo = Image.open('/etc/stratux-screen/stratux-logo-64x64.bmp')
            draw.bitmap((32, 0), logo, fill=1)

        time.sleep(10)
        n = 0

        while 1:
            time.sleep(1)
            response = urllib2.urlopen('http://localhost/getStatus')
            getStatusHTML = response.read()
            getStatusData = json.loads(getStatusHTML)
            CPUTemp = getStatusData["CPUTemp"]
            uat_current = getStatusData["UAT_messages_last_minute"]
            uat_max = getStatusData["UAT_messages_max"]
            es_current = getStatusData["ES_messages_last_minute"]
            es_max = getStatusData["ES_messages_max"]
    
            response = urllib2.urlopen('http://localhost/getTowers')
            getTowersHTML = response.read()
            getTowersData = json.loads(getTowersHTML)
            NumTowers = 0
            for towerLatLng in getTowersData:
                print getTowersData[towerLatLng]["Messages_last_minute"]
                if (getTowersData[towerLatLng]["Messages_last_minute"] > 0):
                    NumTowers += 1

            with canvas(oled) as draw:
                pad = 2 # Two pixels on the left and right.
                text_margin = 25
                # UAT status.
                draw.text((50, 0), "UAT", font=font2, fill=255)
                # "Status bar", 2 pixels high.
                status_bar_width_max = oled.width - (2 * pad) - (2 * text_margin)
                status_bar_width = 0
                if uat_max > 0:
                    status_bar_width = int((float(uat_current) / uat_max) * status_bar_width_max)
                draw.rectangle((pad + text_margin, 14, pad + text_margin + status_bar_width, 20), outline=255, fill=255) # Top left, bottom right.
                # Draw the current (left) and max (right) numbers.
                draw.text((pad, 14), str(uat_current), font=font2, fill=255)
                draw.text(((2*pad) + text_margin + status_bar_width_max, 14), str(uat_max), font=font2, fill=255)
                # ES status.
                draw.text((44, 24), "1090ES", font=font2, fill=255)
                status_bar_width = 0
                if es_max > 0:
                    status_bar_width = int((float(es_current) / es_max) * status_bar_width_max)
                draw.rectangle((pad + text_margin, 34, pad + text_margin + status_bar_width, 40), outline=255, fill=255) # Top left, bottom right.
                # Draw the current (left) and max (right) numbers.
                draw.text((pad, 34), str(es_current), font=font2, fill=255)
                draw.text(((2*pad) + text_margin + status_bar_width_max, 34), str(es_max), font=font2, fill=255)
                # Other stats.
                seq = (n / 5) % 2
                t = ""
                if seq == 0:
                    t = "CPU: %0.1fC, Towers: %d" % (CPUTemp, NumTowers)
                if seq == 1:
                    t = "GPS Sat: %d/%d/%d" % (getStatusData["GPS_satellites_locked"], getStatusData["GPS_satellites_seen"], getStatusData["GPS_satellites_tracked"])
                    if getStatusData["GPS_solution"] == "3D GPS + SBAS":
                        t = t + " SBAS"
                #print t
                draw.text((pad, 45), t, font=font2, fill=255)

            n = n+1


stratuxscreen = StratuxScreen()
daemon_runner = runner.DaemonRunner(stratuxscreen)
daemon_runner.do_action()
