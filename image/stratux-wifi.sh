#!/bin/bash

# Preliminaries. Kill off old services.
/usr/bin/killall -9 hostapd hostapd-edimax
/usr/sbin/service isc-dhcp-server stop

#Assume PI3 settings
DAEMON_CONF=/etc/hostapd/hostapd.conf
DAEMON_SBIN=/usr/sbin/hostapd

DAEMON_CONF_EDIMAX=/etc/hostapd/hostapd-edimax.conf

#User settings for hostapd.conf and hostapd-edimax.conf
DAEMON_USER_PREF=/etc/hostapd/hostapd.user

DAEMON_TMP=/tmp/hostapd.conf

# values to move
HOSTAPD_VALUES=('ssid=' 'channel=' 'auth_algs=' 'wpa=' 'wpa_passphrase=' 'wpa_key_mgmt=' 'wpa_pairwise=' 'rsn_pairwise=')

#values to remove
HOSTAPD_VALUES_RM=('#auth_algs=' '#wpa=' '#wpa_passphrase=' '#wpa_key_mgmt=' '#wpa_pairwise=' '#rsn_pairwise=')


# This code checks for the existence of ostapd.user and if it exists it leaves it alone.
# If the file does not exist it copys over the values from the existing hostapd.conf to hostapd.user and removes them
# check for hostapd.user and if needed create properly
if [ ! -f $DAEMON_USER_PREF ]; then 
# move any custom values
	for i in "${HOSTAPD_VALUES[@]}"
	do
    	if grep -q "^$i" $DAEMON_CONF
		then
        	grep "^$i" $DAEMON_CONF >> $DAEMON_USER_PREF
        	sed -i '/^'"$i"'/d' $DAEMON_CONF
		sed -i '/^'"$i"'/d' $DAEMON_CONF_EDIMAX
		fi
	done
# just remove commented values
	for i in "${HOSTAPD_VALUES_RM[@]}"
	do
    	if grep -q "^$i" $DAEMON_CONF
		then
        	sed -i '/^'"$i"'/d' $DAEMON_CONF
            	sed -i '/^'"$i"'/d' $DAEMON_CONF_EDIMAX
		fi
	done
	# If once the code above runs and there is still no hostapd.user file then something is wrong and we will just create the file with basic settings. 
	#Any more then this they somebody was messing with things and its not our fault things are this bad
	if [ ! -f $DAEMON_USER_PREF ]; then 
		echo "ssid=stratux" >> $DAEMON_USER_PREF
		echo "channel=1" >> $DAEMON_USER_PREF
	fi
fi

# Detect RPi version.
#  Per http://elinux.org/RPi_HardwareHistory
EW7811Un=$(lsusb | grep EW-7811Un)
RPI_REV=`cat /proc/cpuinfo | grep 'Revision' | awk '{print $3}' | sed 's/^1000//'`
if [ "$RPI_REV" = "a01041" ] || [ "$RPI_REV" = "a21041" ] || [ "$RPI_REV" = "900092" ] || [ "$RPI_REV" = "900093" ] && [ "$EW7811Un" != '' ]; then
 # This is a RPi2B or RPi0 with Edimax USB Wifi dongle.
 DAEMON_CONF=/etc/hostapd/hostapd-edimax.conf
 DAEMON_SBIN=/etc/hostapd/hostapd-edimax
# else
#  DAEMON_CONF=/etc/hostapd/hostapd.conf
fi

#Make a new hostapd or hostapd-edimax conf file based on logic above
cp -f ${DAEMON_CONF} ${DAEMON_TMP}

#inject user settings from file to tmp conf
cat ${DAEMON_USER_PREF} >> ${DAEMON_TMP}

${DAEMON_SBIN} -B ${DAEMON_TMP}

sleep 5

/usr/sbin/service isc-dhcp-server start
