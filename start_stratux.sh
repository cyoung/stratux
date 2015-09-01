#!/bin/bash

screen -S stratux -d -m /usr/bin/start_uat
screen -S dump1090 -d -m /usr/bin/dump1090 --net --device-index 1
