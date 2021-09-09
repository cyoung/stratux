#!/bin/bash
#####
#####	Wifi/AP control file
#####
#####	Description: All the scripting related to the AP and wireless functions for Stratux should be placed in this file
#####	This file is called when wlan0 is started i.e. "ifup wlan0"
#####	This script is called from /etc/network/interfaces by the line "post-up /usr/sbin/stratux-wifi.sh" under the wlan0 configuration
#####


#Logging Function
SCRIPT=`basename ${BASH_SOURCE[0]}`
STX_LOG="/var/log/stratux.log"
function wLog () {
       echo "$(date +"%Y/%m/%d %H:%M:%S")  - $SCRIPT - $1" >> ${STX_LOG}
}
wLog "Running Stratux WiFI Script."

interface=$1 # for dhcp and wpa_supplicant
mode=$2 # 0=ap, 1=wifi-direct, 2=ap+client
pin=$3 # wifi-direct pin

if [ "$1" == "0" ] || [ "$1" == "1" ] || [ "$1" == "2" ]; then
	# compatibility to old /etc/network/interfaces before eu027
        echo "COMPAT MODE"
	interface="wlan0"
	mode=$1
	pin=$2
fi

echo "interface=${interface},mode=${mode}"

function prepare-start {
	# Preliminaries. Kill off old services.
	# For some reason, in buster, hostapd will not start if it was just killed. Wait two seconds..
	wLog "Killing Hostapd services "
	/usr/bin/killall hostapd
	sleep 1
	/usr/bin/killall -9 hostapd
	wLog "Stopping DHCP services "
	/bin/systemctl stop dnsmasq
	/usr/bin/killall dnsmasq

	# Sometimes the PID file seems to remain and dhcpd becomes unable to start again?
	# See https://bugs.debian.org/cgi-bin/bugreport.cgi?bug=868240
	sleep 1
	/usr/bin/killall -9 dnsmasq

	/usr/bin/killall wpa_supplicant
	sleep 1
	/usr/bin/killall -9 wpa_supplicant
}

function ap-start {
	#Assume PI3 settings
	DAEMON_CONF=/etc/hostapd/hostapd.conf
	DAEMON_SBIN=/usr/sbin/hostapd

	${DAEMON_SBIN} -B ${DAEMON_CONF}

	sleep 2

	wLog "Restarting DHCP services"

	dnsmasq -u dnsmasq --conf-dir=/etc/dnsmasq.d -i $interface
}

function wifi-direct-start {
	echo "Starting wifi direct mode"

	/sbin/wpa_supplicant -B -i $interface -c /etc/wpa_supplicant/wpa_supplicant.conf
	wpa_cli -i $interface p2p_group_add persistent=0 freq=2
	(while wpa_cli -i p2p-wlan0-0 wps_pin any $pin > /dev/null; do sleep 1; done) & disown
	ifup p2p-wlan0-0

	dnsmasq -u dnsmasq --conf-dir=/etc/dnsmasq.d -i p2p-wlan0-0
}

# function to build /tmp/hostapd.conf and start AP
prepare-start
if [ "$mode" == "1" ]; then
	wifi-direct-start
else
	ap-start
fi
