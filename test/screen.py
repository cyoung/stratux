#!/usr/bin/env python

from oled.device import ssd1306, sh1106
from oled.render import canvas

from PIL import ImageDraw, ImageFont, Image

import urllib2
import json
import time

font2 = ImageFont.truetype('/root/ssd1306/fonts/C&C Red Alert [INET].ttf', 12)
oled = ssd1306(port=1, address=0x3C)

with canvas(oled) as draw:
    logo = Image.open('/root/logo.bmp')
    draw.bitmap((32, 0), logo, fill=1)

time.sleep(10)

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
    NumTowers = len(getTowersData)
    
    
    
    
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
        t = "CPU: %0.1fC, Towers: %d" % (CPUTemp, NumTowers)
        draw.text((pad, 45), t, font=font2, fill=255)