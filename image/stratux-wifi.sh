#!/bin/bash


# Preliminaries. Kill off old services.
/usr/bin/killall -9 hostapd hostapd-edimax
/usr/sbin/service isc-dhcp-server stop


# Detect RPi version.
#  Per http://elinux.org/RPi_HardwareHistory

DAEMON_CUSTOM_SSID=/etc/hostapd/ssid.conf
DAEMON_TMP=/tmp/hostapd.conf
DAEMON_CONF=/etc/hostapd/hostapd.conf
DAEMON_SBIN=/usr/sbin/hostapd
EW7811Un=$(lsusb | grep EW-7811Un)
RPI_REV=`cat /proc/cpuinfo | grep 'Revision' | awk '{print $3}' | sed 's/^1000//'`
if [ "$RPI_REV" = "a01041" ] || [ "$RPI_REV" = "a21041" ] || [ "$RPI_REV" = "900092" ] || [ "$RPI_REV" = "900093" ] && [ "$EW7811Un" != '' ]; then
 # This is a RPi2B or RPi0 with Edimax USB Wifi dongle.
 DAEMON_CONF=/etc/hostapd/hostapd-edimax.conf
 DAEMON_SBIN=/usr/sbin/hostapd-edimax
else
 DAEMON_CONF=/etc/hostapd/hostapd.conf
fi

cp -f ${DAEMON_CONF} ${DAEMON_TMP}

SSID="stratux"

if [ -e "${DAEMON_CUSTOM_SSID}" ]; then
	SSID="$(cat ${DAEMON_CUSTOM_SSID})"
fi
#Append the SSID to the configuration file
echo "ssid=$SSID" >> ${DAEMON_TMP}
${DAEMON_SBIN} -B ${DAEMON_TMP}

sleep 5

/usr/sbin/service isc-dhcp-server start
