#!/bin/bash
#####
#####	Wifi/AP control file
#####
#####	Description: All the scripting related to the AP and wireless functions for Stratux should be placed in this file
#####	This file is called when wlan0 is started i.e. "ifup wlan0"
#####	This script is called from /etc/network/interfaces by the line "post-up /usr/sbin/stratux-wifi.sh" under the wlan0 configuration
#####

# common variables
DAEMON_USER_PREF=/etc/hostapd/hostapd.user

#Logging Function
SCRIPT=`basename ${BASH_SOURCE[0]}`
STX_LOG="/var/log/stratux.log"
function wLog () {
       echo "$(date +"%Y/%m/%d %H:%m:%S")  - $SCRIPT - $1" >> ${STX_LOG}
}
wLog "Running Stratux WiFI Script."

##### Function for setting up new file structure for hostapd settings
##### Look for hostapd.user and if found do nothing.
##### If not assume because of previous version and convert to new file structure

function hostapd-upgrade {
	DAEMON_CONF=/etc/hostapd/hostapd.conf
	DAEMON_CONF_EDIMAX=/etc/hostapd/hostapd-edimax.conf
	HOSTAPD_VALUES=('ssid=' 'channel=' 'auth_algs=' 'wpa=' 'wpa_passphrase=' 'wpa_key_mgmt=' 'wpa_pairwise=' 'rsn_pairwise=')

    wLog "Moving existing values from $DAEMON_CONF to $DAEMON_USER_PREF if found"
	for i in "${HOSTAPD_VALUES[@]}"
	do
		if grep -q "^$i" $DAEMON_CONF
        then
			grep "^$i" $DAEMON_CONF >> $DAEMON_USER_PREF
			sed -i '/^'"$i"'/d' $DAEMON_CONF
			sed -i '/^'"$i"'/d' $DAEMON_CONF_EDIMAX
		fi
	done
	sleep 1     #make sure there is time to get the file written before checking for it again
	# If once the code above runs and there is still no hostapd.user file then something is wrong and we will just create the file with basic settings. 
	# Any more then this they somebody was messing with things and its not our fault things are this bad
	wLog "Rechecking if $DAEMON_USER_PREF exists after moving files."
	if [ ! -f $DAEMON_USER_PREF ]; then
	    wLog "File not found. Creating default file. "
		echo "ssid=stratux" > $DAEMON_USER_PREF
		echo "channel=1" >> $DAEMON_USER_PREF
	fi
}
##### End hostapd settings structure function

##### Hostapd Driver check function #####
function ap-start {

	# Preliminaries. Kill off old services.
	wLog "Killing Hostapd services "
    /usr/bin/killall -9 hostapd hostapd-edimax
    wLog "Stopping DHCP services "
	/usr/sbin/service isc-dhcp-server stop

	#EDIMAX Mac Addresses from http://www.adminsub.net/mac-address-finder/edimax
	#for logic check all addresses must be lowercase
	# 74:da:38 is my MAC on my NANO
	edimaxMac=(80:1f:02 74:da:38 00:50:fc 00:1f:1f 00:0e:2e 00:00:b4)

	#Assume PI3 settings
	DAEMON_CONF=/etc/hostapd/hostapd.conf
	DAEMON_SBIN=/usr/sbin/hostapd

	# Location of temporary hostapd.conf built by combining
	# non-editable /etc/hostapd/hostapd.conf or hostapd-edimax.conf
	# and the user configurable /etc/hostapd/hostapd.conf
	DAEMON_TMP=/tmp/hostapd.conf

	#get the first 3 octets of the MAC(XX:XX:XX) at wlan0
	wlan0mac=$(head -c 8 /sys/class/net/wlan0/address)

	# Is there an Edimax Mac Address at wlan0
	if [[ ${edimaxMac[*]} =~ "$wlan0mac" ]]; then
        DAEMON_CONF=/etc/hostapd/hostapd-edimax.conf
        DAEMON_SBIN=/usr/sbin/hostapd-edimax
        wLog "Edimax Dongle found at WLAN0. Using Edimad conf files $DAEMON_SBIN : $DAEMON_CONF"
	fi

	#Make a new hostapd or hostapd-edimax conf file based on logic above
	cat ${DAEMON_USER_PREF} <(echo) ${DAEMON_CONF} > ${DAEMON_TMP}

	${DAEMON_SBIN} -B ${DAEMON_TMP}

	sleep 2

	wLog "Restarting DHCP services"

	/usr/sbin/service isc-dhcp-server start
}
##### End Hostapd driver check function #####

#Do we need to upgrade the hostapd configuration files
wLog "Checking if $DAEMON_USER_PREF file exists"
if [ ! -f $DAEMON_USER_PREF ]; then
    wLog "File not found. Upgrading to new file structure."
	hostapd-upgrade
fi

# function to build /tmp/hostapd.conf and start AP
ap-start
