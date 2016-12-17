#!/bin/bash

# Preliminaries. Kill off old services.
/usr/bin/killall -9 hostapd hostapd-edimax
/usr/sbin/service isc-dhcp-server stop

#EDIMAX Mac Addresses from http://www.adminsub.net/mac-address-finder/edimax
#for logic check all addresses must be lowercase
edimaxMac=(80:1f:02 74:da:38 00:50:fc 00:1f:1f 00:0e:2e 00:00:b4)

#Assume PI3 settings
DAEMON_CONF=/etc/hostapd/hostapd.conf
DAEMON_SBIN=/usr/sbin/hostapd

#User settings for hostapd.conf and hostapd-edimax.conf
DAEMON_USER_PREF=/etc/hostapd/hostapd.user

# Temporary hostapd.conf built by combining
# non-editable /etc/hostapd/hostapd.conf or hostapd-edimax.conf
# and the user configurable /etc/hostapd/hostapd.conf
DAEMON_TMP=/tmp/hostapd.conf

#get the first 3 octets of the MAC(XX:XX:XX) for wlan0
wlan0mac=$(head -c 8 /sys/class/net/wlan0/address)

# Is there an Edimax Mac Address at wlan0
if [ ${edimaxMac[*]} =~ "$wlan0mac" ]; then
     # If so then lets see if we have 
     # Detect RPi version. Per http://elinux.org/RPi_HardwareHistory
     #RPI_REV=`cat /proc/cpuinfo | grep 'Revision' | awk '{print $3}' | sed 's/^1000//'`
     #if [ "$RPI_REV" = "a01041" ] || [ "$RPI_REV" = "a21041" ] || [ "$RPI_REV" = "900092" ] || [ "$RPI_REV" = "900093" ]; then
      # This is a RPi2B or RPi0 with Edimax USB Wifi dongle.
      DAEMON_CONF=/etc/hostapd/hostapd-edimax.conf
      DAEMON_SBIN=/usr/sbin//hostapd-edimax
     #fi
fi

#Make a new hostapd or hostapd-edimax conf file based on logic above
cp -f ${DAEMON_CONF} ${DAEMON_TMP}

#inject user settings from file to tmp conf
cat ${DAEMON_USER_PREF} >> ${DAEMON_TMP}

${DAEMON_SBIN} -B ${DAEMON_TMP}

sleep 3

/usr/sbin/service isc-dhcp-server start
