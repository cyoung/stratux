#!/bin/bash


# Preliminaries. Kill off old services.
/usr/bin/killall -9 hostapd hostapd-edimax
/usr/sbin/service isc-dhcp-server stop

edimaxMac=(00:1f:1f 80:1f:02 74:da:38)

#  Detect RPi2 or 3 with edimax dongle at wlan0
#  Per http://elinux.org/RPi_HardwareHistory

DAEMON_CONF=/etc/hostapd/hostapd.conf
DAEMON_SBIN=/usr/sbin/hostapd
EW7811Un=$(lsusb | grep EW-7811Un)
wlan0mac=$(head -c 8 /sys/class/net/wlan0/address)
echo 'WLAN0 MAC: '$wlan0mac
echo 'LSUSB Returned: '$EW7811Un
RPI_REV=`cat /proc/cpuinfo | grep 'Revision' | awk '{print $3}' | sed 's/^1000//'`

if [ "$RPI_REV" = "a01041" ] || [ "$RPI_REV" = "a21041" ]  || [ "$RPI_REV" = "900092" ]   || [ "$RPI_REV" = "a02082" ]    || [ "$RPI_REV" = "a22082" ]; then
# If this is a PRPi2/3 [ "$wlan0mac" = '74:da:38' ]
  if [ "$EW7811Un" != '' ] && [[ ${edimaxMac[*]} =~ "$wlan0mac" ]]; then
   # If there is an Edimax Nano at wlan0
   DAEMON_CONF=/etc/hostapd/hostapd-edimax.conf
   DAEMON_SBIN=/usr/sbin/hostapd-edimax
 fi
fi

#Example:
#/usr/sbin/hostapd-edimax -B /etc/hostapd/hostapd-edimax.conf
${DAEMON_SBIN} -B ${DAEMON_CONF}

sleep 5

/usr/sbin/service isc-dhcp-server start
