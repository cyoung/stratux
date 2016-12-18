#!/bin/bash

# Preliminaries. Kill off old services.
/usr/bin/killall -9 hostapd hostapd-edimax
/usr/sbin/service isc-dhcp-server stop

#Assume PI3 settings
DAEMON_CONF=/etc/hostapd/hostapd.conf
DAEMON_SBIN=/usr/sbin/hostapd

#User settings for hostapd.conf and hostapd-edimax.conf
DAEMON_USER_PREF=/etc/hostapd/hostapd.user

# Temporary hostapd.conf built by combining
# non-editable /etc/hostapd/hostapd.conf or hostapd-edimax.conf
# and the user configurable /etc/hostapd/hostapd.conf
DAEMON_TMP=/tmp/hostapd.conf

# Detect RPi version.
#  Per http://elinux.org/RPi_HardwareHistory
EW7811Un=$(lsusb | grep EW-7811Un)
RPI_REV=`cat /proc/cpuinfo | grep 'Revision' | awk '{print $3}' | sed 's/^1000//'`
if [ "$RPI_REV" = "a01041" ] || [ "$RPI_REV" = "a21041" ] || [ "$RPI_REV" = "900092" ] || [ "$RPI_REV" = "900093" ] && [ "$EW7811Un" != '' ]; then
 # This is a RPi2B or RPi0 with Edimax USB Wifi dongle.
 DAEMON_CONF=/etc/hostapd/hostapd-edimax.conf
 DAEMON_SBIN=/usr/sbin/hostapd-edimax
fi

#Make a new hostapd or hostapd-edimax conf file based on logic above
cat ${DAEMON_USER_PREF} ${DAEMON_CONF} > ${DAEMON_TMP}

${DAEMON_SBIN} -B ${DAEMON_TMP}

sleep 3

/usr/sbin/service isc-dhcp-server start
