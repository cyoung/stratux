#!/bin/bash

echo "adding wlan1 information to etc/network/interfaces..."
echo "  "
printf '\n ' >> /etc/network/interfaces
printf 'allow-hotplug wlan1\n' >> /etc/network/interfaces
printf 'iface wlan1 inet dhcp \n' >> /etc/network/interfaces
printf '%s  wireless-essid 6719 ' >> /etc/network/interfaces

echo " "
echo "Fetching stratux-wifi.sh for dual WiFi Dongle pi3 from github"
echo " "
# wget https://raw.githubusercontent.com/peepsnet/stratux/9e034bd09cf6eb730a1b207e82247626a66c0d7a/image/stratux-wifi.sh
echo "Backing up original stratux-wifi.sh... "
mv /usr/sbin/stratux-wifi.sh /usr/sbin/stratux-wifi.sh.ori
echo "Moving new stratux-wifi.sh... "
# mv /root/temp/stratux-wifi.sh /usr/sbin/stratux-wifi.sh
mv /boot/stx/stratux-wifi.sh /usr/sbin/stratux-wifi.sh
echo "chmod script"
chmod 755 /usr/sbin/stratux-wifi.sh


echo "Fetching hostapd-edimax binary..."
# wget https://github.com/cyoung/stratux/blob/master/image/hostapd-edimax?raw=true
echo "Moving Edimax Binary..."
# mv /root/temp/hostapd-edimax?raw=true /usr/sbin/hostapd-edimax
mv /boot/stx/hostapd-edimax?raw=true /usr/sbin/hostapd-edimax
echo "chmod binary"
chmod +x /usr/sbin/hostapd-edimax

echo " "

echo "Fetching hostapd-edimax.conf..."
# wget https://raw.githubusercontent.com/cyoung/stratux/master/image/hostapd-edimax.conf
echo "Moving hostapd-edimax.conf..."
# mv /root/temp/hostapd-edimax.conf /etc/hostapd/hostapd-edimax.conf
mv /boot/stx/hostapd-edimax.conf /etc/hostapd/hostapd-edimax.conf

echo "Complete..."
