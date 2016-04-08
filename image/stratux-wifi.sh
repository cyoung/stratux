#!/bin/bash


# Preliminaries. Kill off old services.
/usr/bin/killall -9 hostapd hostapd-edimax
/usr/sbin/service isc-dhcp-server stop


# Detect RPi version.
#  Per http://elinux.org/RPi_HardwareHistory

DAEMON_CONF=/etc/hostapd/hostapd.conf
DAEMON_SBIN=/usr/sbin/hostapd
EW7811Un=$(head -c 8 /sys/class/net/wlan0/address)
#echo 'WLAN0 MAC '$EW7811Un
RPI_REV=`cat /proc/cpuinfo | grep 'Revision' | awk '{print $3}' | sed 's/^1000//'`

if [ "$RPI_REV" = "a01041" ] || [ "$RPI_REV" = "a21041" ]  || [ "$RPI_REV" = "900092" ]   || [ "$RPI_REV" = "a02082" ]    || [ "$RPI_REV" = "a22082" ]  &&   [ "$EW7811Un" = '74:da:38' ]; then
 # This is a RPi2B.
 DAEMON_CONF=/etc/hostapd/hostapd-edimax.conf
 DAEMON_SBIN=/usr/sbin/hostapd-edimax
fi
#if [ "$RPI_REV" = "a02082" ] || [ "$RPI_REV" = "a22082" ]; then
 # This is a RPi3B.
# DAEMON_CONF=/etc/hostapd/hostapd.conf
#fi

#Example:
#/usr/sbin/hostapd-edimax -B /etc/hostapd/hostapd-edimax.conf
${DAEMON_SBIN} -B ${DAEMON_CONF}



sleep 5

/usr/sbin/service isc-dhcp-server start
