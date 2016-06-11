#!/bin/bash

echo "adding wlan1 information to etc/network/interfaces..."
echo "  "
printf '\n ' >> /etc/network/interfaces
printf 'allow-hotplug wlan1\n' >> /etc/network/interfaces
printf 'iface wlan1 inet dhcp \n' >> /etc/network/interfaces
printf '%s  wireless-essid 6719 ' >> /etc/network/interfaces

echo " "
echo "Backing up original stratux-wifi.sh... "
mv /usr/sbin/stratux-wifi.sh /usr/sbin/stratux-wifi.sh.ori
echo "Moving new stratux-wifi.sh... "
cp /boot/stx/stratux-wifi.sh /usr/sbin/stratux-wifi.sh
echo "chmod script"
chmod 755 /usr/sbin/stratux-wifi.sh

echo " "

echo "Moving Edimax Binary..."
cp /boot/stx/hostapd-edimax /usr/sbin/hostapd-edimax
echo "chmod binary"
chmod +x /usr/sbin/hostapd-edimax

echo " "

echo "Moving hostapd-edimax.conf..."
cp /boot/stx/hostapd-edimax.conf /etc/hostapd/hostapd-edimax.conf

echo "Complete..."
