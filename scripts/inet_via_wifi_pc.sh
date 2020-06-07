#!/bin/bash
# Run this script on your desktop PC which is connected to the Stratux via WiFi and to the internet via cable
# Usage: ./inet_via_wifi_pc.sh enp0s1
# where enp0s1 is the interface name of your outgoing internet connection

sysctl -w net.ipv4.ip_forward=1
iptables -t nat -A POSTROUTING -o $1 -j MASQUERADE