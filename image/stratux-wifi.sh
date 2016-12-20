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

#User configurable settings for hostapd.conf and hostapd-edimax.conf
DAEMON_USER_PREF=/etc/hostapd/hostapd.user

# Location of temporary hostapd.conf built by combining
# non-editable /etc/hostapd/hostapd.conf or hostapd-edimax.conf
# and the user configurable /etc/hostapd/hostapd.conf
DAEMON_TMP=/tmp/hostapd.conf

#get the first 3 octets of the MAC(XX:XX:XX) at wlan0
wlan0mac=$(head -c 8 /sys/class/net/wlan0/address)

# Is there an Edimax Mac Address at wlan0
if [[ ${edimaxMac[*]} =~ "$wlan0mac" ]]; then
     DAEMON_CONF=/etc/hostapd/hostapd-edimax.conf
     DAEMON_SBIN=/usr/sbin//hostapd-edimax
fi

#Make a new hostapd or hostapd-edimax conf file based on logic above
cat ${DAEMON_USER_PREF} ${DAEMON_CONF} > ${DAEMON_TMP}

${DAEMON_SBIN} -B ${DAEMON_TMP}

sleep 3

/usr/sbin/service isc-dhcp-server start
