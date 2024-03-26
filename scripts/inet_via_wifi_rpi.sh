#!/bin/bash
# Run this script on your stratux and pass it the IP address of the WiFi client to use for internet connection
# Usage: ./inet_via_wifi_rpi.sh 192.168.10.x 

route add default gw $1
echo "nameserver 8.8.8.8" > /etc/resolv.conf