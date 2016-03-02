#!/bin/bash

while true ; do
   if ifconfig wlan0 | grep -q "UP BROADCAST RUNNING MULTICAST" ; then
      sleep 30
   else
      echo "Wi-Fi connection down! Attempting reconnection."
      service hostapd restart
      service isc-dhcp-server restart
      sleep 10
   fi
done
